package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsurePublishWorktreeAndRejectDeletion(t *testing.T) {
	root := t.TempDir()
	if err := gitRun(root, "init", "-b", "main"); err != nil {
		t.Fatal(err)
	}
	if err := gitRun(root, "config", "user.name", "Winter Test"); err != nil {
		t.Fatal(err)
	}
	if err := gitRun(root, "config", "user.email", "winter@example.com"); err != nil {
		t.Fatal(err)
	}
	writeGitTestFile(t, root, "README.md", "source")
	if err := gitRun(root, "add", "README.md"); err != nil {
		t.Fatal(err)
	}
	if err := gitRun(root, "commit", "-m", "source"); err != nil {
		t.Fatal(err)
	}
	remote := filepath.Join(t.TempDir(), "remote.git")
	if err := gitRun("", "init", "--bare", remote); err != nil {
		t.Fatal(err)
	}
	if err := gitRun(root, "remote", "add", "origin", remote); err != nil {
		t.Fatal(err)
	}
	if err := gitRun(root, "switch", "--orphan", "gh-pages"); err != nil {
		t.Fatal(err)
	}
	writeGitTestFile(t, root, "index.html", "published")
	if err := gitRun(root, "add", "index.html"); err != nil {
		t.Fatal(err)
	}
	if err := gitRun(root, "commit", "-m", "publish"); err != nil {
		t.Fatal(err)
	}
	if err := gitRun(root, "push", "-u", "origin", "gh-pages"); err != nil {
		t.Fatal(err)
	}
	if err := gitRun(root, "switch", "main"); err != nil {
		t.Fatal(err)
	}

	dist := filepath.Join(root, "dist")
	if err := ensurePublishWorktree(root, dist, "origin", "gh-pages"); err != nil {
		t.Fatal(err)
	}
	branch, err := gitOutput(dist, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	if branch != "gh-pages" {
		t.Fatalf("dist branch = %q, want gh-pages", branch)
	}
	if err := os.Remove(filepath.Join(dist, "index.html")); err != nil {
		t.Fatal(err)
	}
	err = ensurePublishWorktree(root, dist, "origin", "gh-pages")
	if err == nil || !strings.Contains(err.Error(), "deleted files") {
		t.Fatalf("expected deleted-file error, got %v", err)
	}
}

func writeGitTestFile(t *testing.T, root, rel, contents string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
