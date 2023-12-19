package cmd // import "twos.dev/winter/cmd"

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"twos.dev/winter/cliutils"
	"twos.dev/winter/document"
)

func newConfigCmd(logger *slog.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Interact with Winter configuration",
		Long: cliutils.Sprintf(`
			Interact with Winter configuration for the project in the current directory.

			Configuration in ` + "`" + `./winter.yml` + "`" + ` takes first precedence.
			Otherwise, configuration is stored according to the XDG spec.
			On Linux, this is generally:

			` + "```" + `
			~/.config/winter/winter.yml
			` + "```" + `

			On macOS, this is generally:

			` + "```" + `
			~/Library/Application Support/winter/winter.yml
			` + "```" + `

			For more information on config locations, see the
			[XDG specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html).
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := document.InteractiveConfig(); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.AddCommand(newConfigGetCmd())
	cmd.AddCommand(newConfigClearCmd())
	return cmd
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "Get or list Winter config",
		Long: cliutils.Sprintf(`
			Get the value of the Winter configuration variable named KEY,
			or all configuration if KEY is omitted.

			See winter config --help for information on configuration storage locations.
		`),
		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := document.NewConfig()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				bytes, err := yaml.Marshal(c)
				if err != nil {
					return err
				}
				if _, err := os.Stdout.Write(bytes); err != nil {
					return err
				}
				return nil
			}
			if len(args) > 1 {
				return fmt.Errorf("must take 0–1 arguments")
			}
			for _, arg := range args {
				if err := mapstructure.Decode(arg, &c); err != nil {
					return err
				}
			}
			return c.Save()
		},
	}
}

func newConfigClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Erase all config",
		Long: cliutils.Sprintf(`
			Erase all Winter configuration. Cannot be undone.
			See winter config --help for information on configuration storage locations.
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := document.ConfigPath()
			if err != nil {
				return err
			}
			return os.Remove(p)
		},
	}
}
