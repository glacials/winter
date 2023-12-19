package cmd // import "twos.dev/winter/cmd"

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"twos.dev/winter/cliutils"
)

func newInitCmd(logger *slog.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [DIR]",
		Short: "Start a new Winter project",
		Long: cliutils.Sprintf(`
			Initialize a new Winter project by generating a fresh winter.yml.

			The winter.yml file is placed in DIR if supplied,
			or the working directory otherwise.
		`),
		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			fmt.Println(`
 _      ___      __
| | /| / (_)__  / /____ ____
| |/ |/ / / _ \/ __/ -_) __/
|__/|__/_/_//_/\__/\__/_/`)
			winterYMLPath := filepath.Join(dir, "winter.yml")
			if _, err := os.Stat(winterYMLPath); err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("cannot check for existing winter.yml: %w", err)
				}
			} else {
				cliutils.Printf(`
					winter.yml already detected in working directory.
					To generate a new one,
					move or remove the existing one.
				`)
				return nil
			}

			cliutils.Printf(`
				winter.yml generated.
				Open it and follow the instructions to set up your project.
			`)
			return nil
		},
	}
	return cmd
}
