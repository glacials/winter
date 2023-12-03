package cmd // import "twos.dev/winter/cmd"

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"text/template"
	"time"

	"github.com/gorilla/feeds"
	"github.com/spf13/cobra"
	"twos.dev/winter/cliutils"
	"twos.dev/winter/document"
)

//go:embed all:defaults
var defaults embed.FS

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new Winter website",
		Long: cliutils.Sprintf(`
			Interactively create a new Winter project.

			Winter will ask you several questions about the new project's
			name,
			path,
			and other details.
		`),
		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInitCmd(os.Stdin, os.Stdout)
		},
	}
	return cmd
}

// runInitCmd performs execution of the winter init command.
// It is separate from the corresponding [cobra.Command] function for easy testing.
func runInitCmd(in io.Reader, out io.Writer) error {
	destDirPath, err := cliutils.Ask(`
		- What directory should Winter initialize into?
		  Several files and directories will be generated.

		  Directory [.]:
	`, ".", in, out)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destDirPath, 0o755); err != nil {
		return fmt.Errorf("cannot make initial directory %q: %w", destDirPath, err)
	}
	dirName, err := filepath.Abs(destDirPath)
	if err != nil {
		return fmt.Errorf("cannot get dir name for %q: %w", destDirPath, err)
	}
	dirName = filepath.Base(dirName)
	name, err := cliutils.Ask(fmt.Sprintf(`
		- What is the human-readable name for your website?
		  This will be used in Atom feeds, <meta> tags, and the <title> tag.

			This can be changed at any time in winter.yml.

		  Name [%s]:
	`, dirName), dirName, in, out)
	if err != nil {
		return err
	}
	description, err := cliutils.Ask(`
		- What is the human-readable description for your website?
		  This will be used in Atom feeds and <meta> tags.

			This can be changed at any time in winter.yml.

		  Description [(none)]:
	`, "", in, out)
	if err != nil {
		return err
	}
	authorName, err := cliutils.Ask(`
		- What is the primary website author's name?
		  This will be used in Atom feeds and copyright notices.

			This can be changed at any time in winter.yml.

		  Primary Author Name [(none)]:
	`, "", in, out)
	if err != nil {
		return err
	}
	authorEmail, err := cliutils.Ask(`
		- What is the primary website author's email address?
		  This will be used in Atom feeds.

			This can be changed at any time in winter.yml.

		  Primary Author Email Address [(none)]:
	`, "", in, out)
	if err != nil {
		return err
	}
	year, err := cliutils.Ask(fmt.Sprintf(`
		- What year was your website created?
			This will be used in copyright notices.

			This can be changed at any time in winter.yml.

		  Year Established [%d]:
	`, time.Now().Year()), fmt.Sprintf("%d", time.Now().Year()), in, out)
	if err != nil {
		return err
	}
	return fs.WalkDir(
		defaults,
		".",
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			srcFile, err := defaults.Open(path)
			if err != nil {
				return err
			}
			defer srcFile.Close()
			destFilePath := filepath.Join(destDirPath, path)
			destFilePathDir := filepath.Dir(destFilePath)
			if err := os.MkdirAll(destFilePathDir, 0o755); err != nil {
				return fmt.Errorf("cannot make destination directory %q: %w", destFilePathDir, err)
			}
			destFile, err := os.Create(destFilePath)
			if err != nil {
				return err
			}
			defer destFile.Close()
			if destFile.Name() == "winter.yml" {
				text, err := io.ReadAll(srcFile)
				if err != nil {
					return err
				}
				t, err := template.New("winter.yml").Parse(string(text))
				if err != nil {
					return err
				}
				yearInt, err := strconv.Atoi(year)
				if err != nil {
					return fmt.Errorf("cannot convert year %q to integer: %w", year, err)
				}
				config := document.Config{
					Author: feeds.Author{
						Name:  authorName,
						Email: authorEmail,
					},
					Description: description,
					Name:        name,
					Since:       yearInt,
				}
				if err := t.Execute(destFile, config); err != nil {
					return fmt.Errorf("cannot execute winter.yml template: %w", err)
				}
			} else {
				_, err = io.Copy(destFile, srcFile)
				if err != nil {
					return err
				}
			}
			return nil
		},
	)
}
