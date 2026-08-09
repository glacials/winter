package cmd // import "twos.dev/winter/cmd"

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func gitOutput(directory string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if directory != "" {
		cmd.Dir = directory
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return "", fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), message, err)
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func gitRun(directory string, args ...string) error {
	_, err := gitOutput(directory, args...)
	return err
}

func ensurePublishWorktree(projectRoot, distPath, remote, branch string) error {
	remoteRef := remote + "/" + branch
	fetchRefspec := "refs/heads/" + branch + ":refs/remotes/" + remoteRef
	if err := gitRun(projectRoot, "fetch", remote, fetchRefspec); err != nil {
		return fmt.Errorf("cannot fetch deployment branch %s/%s: %w", remote, branch, err)
	}
	absDist, err := canonicalPath(distPath)
	if err != nil {
		return fmt.Errorf("cannot resolve dist path %q: %w", distPath, err)
	}
	worktrees, err := gitOutput(projectRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return err
	}
	found := false
	for _, line := range strings.Split(worktrees, "\n") {
		if !strings.HasPrefix(line, "worktree ") {
			continue
		}
		path := strings.TrimPrefix(line, "worktree ")
		absPath, err := canonicalPath(path)
		if err == nil && absPath == absDist {
			found = true
			break
		}
	}
	if !found {
		entries, err := os.ReadDir(absDist)
		if err == nil && len(entries) != 0 {
			return fmt.Errorf(
				"dist directory %q exists and is not an empty registered Git worktree; move it aside and retry",
				distPath,
			)
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("cannot inspect dist directory %q: %w", distPath, err)
		}
		if err := gitRun(projectRoot, "worktree", "add", "-B", branch, absDist, remoteRef); err != nil {
			return fmt.Errorf("cannot create deployment worktree at %q: %w", distPath, err)
		}
	}
	checkedOutBranch, err := gitOutput(absDist, "branch", "--show-current")
	if err != nil {
		return err
	}
	if checkedOutBranch != branch {
		return fmt.Errorf("dist worktree %q is on branch %q, not deployment branch %q", distPath, checkedOutBranch, branch)
	}
	if staged, err := gitOutput(absDist, "diff", "--cached", "--name-only"); err != nil {
		return err
	} else if staged != "" {
		return fmt.Errorf("dist worktree has staged changes; commit or unstage them before publishing:\n%s", staged)
	}
	status, err := gitOutput(absDist, "status", "--porcelain")
	if err != nil {
		return err
	}
	if deleted := deletedStatusPaths(status); len(deleted) != 0 {
		return fmt.Errorf("dist worktree contains deleted files; restore them before append-only publishing:\n%s", strings.Join(deleted, "\n"))
	}
	localHead, err := gitOutput(absDist, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	remoteHead, err := gitOutput(projectRoot, "rev-parse", remoteRef)
	if err != nil {
		return err
	}
	if localHead == remoteHead {
		return nil
	}
	if status != "" {
		return fmt.Errorf(
			"dist has local build changes and is not synchronized with %s; publish or discard those changes before retrying",
			remoteRef,
		)
	}
	if err := gitRun(absDist, "merge", "--ff-only", remoteRef); err != nil {
		return fmt.Errorf("deployment branch diverged from %s; resolve it manually: %w", remoteRef, err)
	}
	return nil
}

func canonicalPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if evaluated, err := filepath.EvalSymlinks(absPath); err == nil {
		return filepath.Clean(evaluated), nil
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(absPath))
	if err != nil {
		return filepath.Clean(absPath), nil
	}
	return filepath.Clean(filepath.Join(parent, filepath.Base(absPath))), nil
}

func deletedStatusPaths(status string) []string {
	var deleted []string
	for _, line := range strings.Split(status, "\n") {
		if len(line) < 3 {
			continue
		}
		if line[0] == 'D' || line[1] == 'D' {
			deleted = append(deleted, line[3:])
		}
	}
	return deleted
}
