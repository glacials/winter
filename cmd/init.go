package cmd // import "twos.dev/winter/cmd"

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"
	"twos.dev/winter/cliutils"
)

const (
	starterDirPath    = "cmd/defaults"
	winterYMLFilename = "winter.yml"
)

var (
	starterWinterYMLPath = path.Join(starterDirPath, winterYMLFilename)
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [DIR]",
		Short: "Start a new Winter project",
		Long: cliutils.Sprintf(`
			Initialize a new Winter project by generating a fresh %s.

			The %s file is placed in DIR if supplied,
			or the working directory otherwise.
		`,
			winterYMLFilename,
			winterYMLFilename,
		),
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
			winterYMLPath := filepath.Join(dir, winterYMLFilename)
			if _, err := os.Stat(winterYMLPath); err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf(
						"cannot check for existing %s: %w",
						winterYMLPath,
						err,
					)
				}
			} else {
				cliutils.Printf(`
					%s already detected at %s.
					To generate a new one,
					move or remove the existing one.
				`, winterYMLFilename, winterYMLPath)
				return nil
			}

			winterYML, err := os.ReadFile(starterWinterYMLPath)
			if err != nil {
				return fmt.Errorf("cannot read default %s: %w", winterYMLFilename, err)
			}

			if err := os.WriteFile(winterYMLPath, []byte(winterYML), 0644); err != nil {
				return fmt.Errorf("cannot write winter.yml: %w", err)
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
