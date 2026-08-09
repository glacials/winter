package cmd // import "twos.dev/winter/cmd"

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"twos.dev/winter/cliutils"
	"twos.dev/winter/document"
)

func newPublishCmd() *cobra.Command {
	var allowDirty, noPush bool
	var branch, remote string
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Incrementally build, check, commit, and publish the site",
		Long: cliutils.Sprintf(`
			Publish the Winter project using the configured dist directory as a
			persistent Git worktree. Existing public files are retained, changed and
			new files are overlaid, and deletions are never staged automatically.

			Before pushing, Winter checks all generated HTML and CSS for broken local
			links, resources, srcset candidates, and fragments. The deployment commit
			records the source commit it was built from.
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := document.NewConfig()
			if err != nil {
				return err
			}
			projectRoot, err := gitOutput("", "rev-parse", "--show-toplevel")
			if err != nil {
				return fmt.Errorf("publish must run inside a Git worktree: %w", err)
			}
			if !allowDirty {
				status, err := gitOutput(projectRoot, "status", "--porcelain", "--untracked-files=normal")
				if err != nil {
					return err
				}
				if status != "" {
					return fmt.Errorf("source worktree has uncommitted changes; commit them before publishing (or use --allow-dirty in CI)")
				}
			}
			distPath := cfg.Dist
			if !filepath.IsAbs(distPath) {
				distPath = filepath.Join(projectRoot, distPath)
			}
			if err := ensurePublishWorktree(projectRoot, distPath, remote, branch); err != nil {
				return err
			}
			cfg.Dist = distPath
			s, err := document.NewSubstructure(cfg)
			if err != nil {
				return err
			}
			if err := s.ExecuteAll(distPath); err != nil {
				return err
			}
			if err := document.CheckSite(distPath); err != nil {
				return err
			}
			status, err := gitOutput(distPath, "status", "--porcelain")
			if err != nil {
				return err
			}
			if deleted := deletedStatusPaths(status); len(deleted) != 0 {
				return fmt.Errorf("build deleted public files, refusing to publish:\n%s", strings.Join(deleted, "\n"))
			}
			if err := gitRun(distPath, "add", "--ignore-removal", "."); err != nil {
				return fmt.Errorf("cannot stage published files: %w", err)
			}
			sourceCommit, err := gitOutput(projectRoot, "rev-parse", "HEAD")
			if err != nil {
				return err
			}
			changed, err := gitOutput(distPath, "diff", "--cached", "--name-only")
			if err != nil {
				return err
			}
			if changed != "" {
				shortCommit := sourceCommit
				if len(shortCommit) > 12 {
					shortCommit = shortCommit[:12]
				}
				message := fmt.Sprintf("Publish %s\n\nSource-Commit: %s", shortCommit, sourceCommit)
				if err := gitRun(distPath, "commit", "-m", message); err != nil {
					return fmt.Errorf("cannot commit published files: %w", err)
				}
				fmt.Printf("Published %d pages and %d images into %s.\n", s.DocumentCount(), s.ImageCount(), branch)
			} else {
				fmt.Println("Published output is already up to date.")
			}
			if noPush {
				fmt.Println("Deployment commit was not pushed (--no-push).")
				return nil
			}
			if err := gitRun(distPath, "push", remote, "HEAD:"+branch); err != nil {
				return fmt.Errorf("cannot push deployment branch: %w", err)
			}
			fmt.Printf("Pushed %s to %s.\n", branch, remote)
			return nil
		},
	}
	flags := cmd.Flags()
	flags.BoolVar(&allowDirty, "allow-dirty", false, "allow uncommitted source changes (intended for CI-generated inputs)")
	flags.BoolVar(&noPush, "no-push", false, "create the deployment commit without pushing it")
	flags.StringVar(&branch, "branch", "gh-pages", "deployment branch")
	flags.StringVar(&remote, "remote", "origin", "Git remote to publish to")
	return cmd
}
