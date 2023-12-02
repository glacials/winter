package cmd // import "twos.dev/winter/cmd"

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"twos.dev/winter/cliutils"
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
		  The directory should be empty or mostly empty.
		  Several files and directories will be generated.

			This can be changed at any time by simply moving the directory.

		  Directory [.]:
	`, ".", in, out)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destDirPath, 0o755); err != nil {
		return fmt.Errorf("cannot make initial directory %q: %w", destDirPath, err)
	}
	name, err := cliutils.Ask(fmt.Sprintf(`
		- What is the human-readable name for your project?
		  This will be displayed in several places on the final website,
			for example in the <title> tag.

			This can be changed at any time in winter.yml.

		  Name [%s]:
	`, filepath.Base(destDirPath)), filepath.Base(destDirPath), in, out)
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
			_, err = io.Copy(destFile, srcFile)
			if err != nil {
				return err
			}
			return nil
		},
	)
}
