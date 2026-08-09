package cmd // import "twos.dev/winter/cmd"

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"twos.dev/winter/cliutils"
	"twos.dev/winter/document"
)

func newAuditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "audit",
		Short: "Perform a clean rebuild and compare it with the persistent site",
		Long: cliutils.Sprintf(`
			Build the current sources from scratch in a temporary directory, then
			compare every current generated output with the persistent dist directory.
			Preserved files whose sources were intentionally removed are not treated as
			errors. This is deliberately slower than an incremental publish.
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := document.NewConfig()
			if err != nil {
				return err
			}
			if err := document.CheckSite(cfg.Dist); err != nil {
				return fmt.Errorf("persistent site failed its link check: %w", err)
			}
			tmp, err := os.MkdirTemp("", "winter-audit-*")
			if err != nil {
				return err
			}
			defer os.RemoveAll(tmp)
			knownURIs := filepath.Join(tmp, "known-uris.txt")
			if err := os.WriteFile(knownURIs, nil, 0o644); err != nil {
				return err
			}
			cleanCfg := *cfg
			cleanCfg.Dist = tmp
			cleanCfg.Known.URIs = knownURIs
			s, err := document.NewSubstructure(&cleanCfg)
			if err != nil {
				return err
			}
			if err := s.ExecuteAll(tmp); err != nil {
				return fmt.Errorf("clean audit build failed: %w", err)
			}
			var differences []string
			if err := filepath.WalkDir(tmp, func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() {
					return nil
				}
				rel, err := filepath.Rel(tmp, path)
				if err != nil {
					return err
				}
				if filepath.ToSlash(rel) == ".winter/manifest.json" {
					return nil
				}
				cleanBytes, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				publishedPath := filepath.Join(cfg.Dist, rel)
				publishedBytes, err := os.ReadFile(publishedPath)
				if err != nil {
					if errors.Is(err, os.ErrNotExist) {
						differences = append(differences, filepath.ToSlash(rel)+" is missing from persistent dist")
						return nil
					}
					return err
				}
				if !bytes.Equal(cleanBytes, publishedBytes) {
					differences = append(differences, filepath.ToSlash(rel)+" differs from a clean build")
				}
				return nil
			}); err != nil {
				return err
			}
			if len(differences) != 0 {
				sort.Strings(differences)
				return fmt.Errorf("audit found %d difference(s):\n- %s", len(differences), strings.Join(differences, "\n- "))
			}
			fmt.Println("Audit passed: persistent output matches a clean build for all current sources.")
			return nil
		},
	}
}
