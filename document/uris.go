package document // import "twos.dev/winter/document"

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/yargevad/filepathx"
)

const (
	gitCmd                              = "git"
	gitWorktreeOutputWorktreeLinePrefix = "worktree "
	gitWorktreeOutputHEADLinePrefix     = "HEAD "
	gitWorktreeOutputBranchLinePrefix   = "branch "
)

// dist is a loose wrapper around the Git worktree for the dist directory.
// This is how Winter manages the built files that are ultimately deployed:
//
// - the dist directory SHOULD be in .gitignore
// - the dist directory SHOULD be a Git worktree that points to a bare branch
// - said branch MUST be deployed by some other mechanism, like GitHub Pages
type dist struct {
	// branch is the name to use for the Git branch this worktree will have checked out.
	// For GitHub Pages, this is historically gh-pages.
	branch string
	// path is the path to the dist directory.
	// If not supplied, defaults to ./dist.
	path string
	// projectPath is the path to the Winter project.
	projectPath string
}

// verifyExists creates a Git worktree for the dist directory if it doesn't
// already exist.
func (d *dist) verifyExists() error {
	gitArgv := []string{"git", "-C", d.projectPath}

	worktreeExistsCmd := exec.Command(
		gitCmd, append(gitArgv, "worktree", "list", "--porcelain")...,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	worktreeExistsCmd.Stdout = &stdout
	worktreeExistsCmd.Stderr = &stderr

	if err := worktreeExistsCmd.Run(); err != nil {
		return fmt.Errorf(
			"dist directory %q is not a Git worktree: %w",
			d.path,
			err,
		)
	}

	lines := strings.Split(stdout.String(), "\n")
	var i int
	for j, line := range lines {
		if !strings.HasPrefix(line, gitWorktreeOutputWorktreeLinePrefix) {
			continue
		}
		absWorktreePath := line[len(gitWorktreeOutputWorktreeLinePrefix)-1:]
		absExpectedWorktreePath, err := filepath.Abs(d.path)
		if err != nil {
			return fmt.Errorf("cannot calculate expected worktree path: %w", err)
		}
		if absWorktreePath != absExpectedWorktreePath {
			continue
		}

		// determine if worktreePath lies inside d.projectPath,
		// whether either is relative or absolute.
		absWorktreePath, err = filepath.Abs(absWorktreePath)
		if err != nil {
			return err
		}

		absProjectPath, err := filepath.Abs(d.projectPath)
		if err != nil {
			return err
		}

		relpath, err := filepath.Rel(absProjectPath, absWorktreePath)
		if err != nil {
			return fmt.Errorf(
				"cannot calculate whether Git linked worktree dir is inside main worktree dir %q: %w",
				d.path,
				err,
			)
		}
		if strings.HasPrefix(relpath, "..") {
			return fmt.Errorf(
				"Git linked worktree dir is not inside main worktree dir: %s",
				d.path,
			)
		}
		i = j
		break
	}
	if !strings.HasPrefix(lines[i+1], gitWorktreeOutputHEADLinePrefix) {
		return fmt.Errorf(
			"unexpected output from Git worktree command's HEAD line: %s",
			d.path,
		)
	}
	if !strings.HasPrefix(lines[i+2], gitWorktreeOutputBranchLinePrefix) {
		return fmt.Errorf(
			"unexpected output from Git worktree command's branch line: %s",
			d.path,
		)
	}
	branchRef := lines[i+2][len(gitWorktreeOutputBranchLinePrefix):]
	branchName := strings.TrimPrefix(branchRef, "refs/heads/")
	if d.branch != branchName {
		return fmt.Errorf(
			"dist directory %q is not a Git worktree for branch %q",
			d.path,
			d.branch,
		)
	}

	return nil
}

// SaveNewURIs indexes every HTML file in dist and saves their existence to disk.
// Later, validateURIsDidNotChange can read that file and ensure no file is missing.
//
// The database file's location can be customized with winter.yml.
// It should be commited to the repository.
func (s *Substructure) SaveNewURIs(dist string) error {
	dist = filepath.Clean(dist)
	uris := map[url.URL]struct{}{}
	files, err := filepathx.Glob(filepath.Join(dist, "**", "*"))
	if err != nil {
		return fmt.Errorf("cannot glob dist dir %q: %w", dist, err)
	}
	for _, path := range files {
		path = strings.TrimPrefix(path, dist)
		if len(path) == 0 {
			continue
		}
		if stat, err := os.Stat(path); err != nil {
			return fmt.Errorf("cannot stat path to save %q: %w", path, err)
		} else if stat.IsDir() {
			continue
		}
		uris[url.URL{Path: path}] = struct{}{}
	}

	union := map[url.URL]struct{}{}

	currentPaths, err := os.ReadFile(s.cfg.Known.URIs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(s.cfg.Known.URIs, []byte(""), 0o644); err != nil {
				return fmt.Errorf(
					"cannot make known URIs file at %q: %w",
					s.cfg.Known.URIs,
					err,
				)
			}
			return s.SaveNewURIs(dist)
		}
		return fmt.Errorf("cannot get existing URLs: %w", err)
	}
	for _, uri := range bytes.Split(currentPaths, []byte("\n")) {
		if len(uri) == 0 {
			continue
		}
		union[url.URL{Path: string(uri)}] = struct{}{}
	}
	for uri := range uris {
		union[uri] = struct{}{}
	}
	list := make([]string, 0, len(union))
	for uri := range union {
		list = append(list, uri.Path)
	}
	slices.Sort(list)
	f, err := os.Create(s.cfg.Known.URIs)
	if err != nil {
		return fmt.Errorf(
			"cannot open known URIs file %q for writing: %w",
			s.cfg.Known.URIs,
			err,
		)
	}
	for _, uri := range list {
		if _, err := f.WriteString(uri); err != nil {
			return fmt.Errorf(
				"cannot write URI %q to known URIs file %q: %w",
				uri,
				s.cfg.Known.URIs,
				err,
			)
		}
		if _, err := f.WriteString("\n"); err != nil {
			return fmt.Errorf(
				"cannot write newline to known URIs file %q: %w",
				s.cfg.Known.URIs,
				err,
			)
		}
	}
	return nil
}

// validateURIsDidNotChange returns an error if this build neglected to produce
// an HTML file that was previously present on the site.
//
// To update the list validateURIsDidNotChange uses, run:
//
//	winter freeze
//
// For more information about the "cool URIs don't change" rule, see:
// https://www.w3.org/Provider/Style/URI
func (s *Substructure) validateURIsDidNotChange(dist string) error {
	paths, err := os.ReadFile(s.cfg.Known.URIs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(s.cfg.Known.URIs, []byte(""), 0o644); err != nil {
				return fmt.Errorf("cannot create new known URIs file: %w", err)
			}
			return s.validateURIsDidNotChange(dist)
		}
		return fmt.Errorf("cannot read known URLs file: %w", err)
	}
	changedURIs := []string{}
	for _, pathBytes := range bytes.Split(paths, []byte{'\n'}) {
		if len(pathBytes) == 0 {
			continue
		}
		_, err := os.Stat(filepath.Join(dist, string(pathBytes)))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				uri := url.URL{
					Scheme: "https",
					Host:   s.cfg.Production.URL,
					Path:   string(pathBytes),
				}
				changedURIs = append(changedURIs, uri.String())
			} else {
				return fmt.Errorf("cannot stat %q: %w", pathBytes, err)
			}
		}
	}
	if len(changedURIs) > 0 {
		return fmt.Errorf(
			`cool URIs do not change, but these ones would have been removed by this build:

- %s

Please restore these files and try again. You can inspect the results in dist/ for details.

DANGER: If you want to break these URLs and cause them to 404, remove them from src/uris.txt.
`,
			strings.Join(changedURIs, "\n- "),
		)
	}
	return nil
}
