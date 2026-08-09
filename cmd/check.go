package cmd // import "twos.dev/winter/cmd"

import (
	"fmt"

	"github.com/spf13/cobra"
	"twos.dev/winter/cliutils"
	"twos.dev/winter/document"
)

func newCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check the built site for broken local links and resources",
		Long: cliutils.Sprintf(`
			Check every HTML and CSS file in the configured dist directory.
			Local links, images, responsive image candidates, scripts, stylesheets,
			and fragments must resolve. External URLs are not requested.
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := document.NewConfig()
			if err != nil {
				return err
			}
			if err := document.CheckSite(cfg.Dist); err != nil {
				return err
			}
			fmt.Println("Site check passed.")
			return nil
		},
	}
}
