// Package cmd contains the commands for the winter CLI.
package cmd // import "twos.dev/winter/cmd"

import (
	"context"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"twos.dev/winter/cliutils"
)

const (
	dist = "dist"
)

// Execute sets up the root command and all attached subcommands,
// then runs them according to the CLI arguments supplied.
func Execute(version string) {
	opts := slog.HandlerOptions{Level: slog.LevelWarn}
	var moreVerbose, lessVerbose int
	rootCmd := &cobra.Command{
		Use:   "winter",
		Short: cliutils.Sprintf("Build or serve a static website locally"),
		Long: cliutils.Sprintf(`
			Winter is a careful, conscientious static website generator.

			It powers websites that aim to stay online for decades,
			including the off ramps one might need to migrate away from it later.

			Winter offers three main benefits for these types of websites:

			- **Self-testing.**
			  When Winter publishes a new URL,
			  it remembers it.
				If any future generation would remove that URL from the internet,
				Winter automatically stops and errors.
			- **Invisible cherries.**
			  Although Winter adds some conveniences to Markdown
			  ("cherries on top")
				all new syntax gracefully degrades into standard page content when not parsed.
				For example,
				to create a photo grid simply place one photo per line in a paragraph containing no text;
				to caption it simply write a paragraph just below it in all italics.
			- **Forcibly flat.**
			  Winter allows source files to be organized in any fashion,
				but will always output a website with no directory structure.
				Directory trees are a hidden nuisance for aging websites,
				invisibly breaking images and stylesheets while producing little value.
				If Winter detects conflicts in the resulting tree,
				it automatically stops and errors.
		`),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			verbosity := moreVerbose - lessVerbose
			switch verbosity {
			case -1:
				opts.Level = slog.LevelError
			case 0:
				opts.Level = slog.LevelWarn
			case 1:
				opts.Level = slog.LevelInfo
			case 2:
				opts.Level = slog.LevelDebug
			}
			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &opts)))
			return nil
		},
		Version: version,
	}
	f := rootCmd.PersistentFlags()
	_ = *f.StringArrayP("source", "i", []string{}, "supplemental source file or directory to build (can be specified multiple times)")
	f.CountVarP(
		&moreVerbose,
		"verbose",
		"v",
		"output more details",
	)
	f.CountVarP(
		&lessVerbose,
		"quiet",
		"q",
		"output fewer details",
	)

	rootCmd.AddCommand(newBuildCmd())
	rootCmd.AddCommand(newCleanCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newFreezeCmd())
	rootCmd.AddCommand(newGenerateCommand())
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newServeCmd())
	rootCmd.AddCommand(newTestCmd())
	err := rootCmd.ExecuteContext(context.Background())
	if err != nil {
		os.Exit(1)
	}
}

func init() {
}
