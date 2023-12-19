package cmd // import "twos.dev/winter/cmd"

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
	"twos.dev/winter/cliutils"
)

func newTestCmd(logger *slog.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "test <environment>",
		Short: "Run site-specific integration tests",
		Long: cliutils.Sprintf(`
			Run site-specific integration tests.

			Ensures the second rule of Winter is followed: cool URLs don't change.
			Tests to make sure several human-level assumptions are true about the published website.
		`),
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"local", "production"},
		RunE: func(cmd *cobra.Command, args []string) error {
			client := http.Client{}
			winterResp, err := client.Do(&http.Request{
				URL: &url.URL{Scheme: "https", Host: "twos.dev", Path: "/winter"},
			})
			if err != nil {
				return err
			}
			winterHTMLResp, err := client.Do(&http.Request{
				URL: &url.URL{Scheme: "https", Host: "twos.dev", Path: "/winter.html"},
			})
			if err != nil {
				return err
			}

			winterBody, err := io.ReadAll(winterResp.Body)
			if err != nil {
				return err
			}
			winterHTMLBody, err := io.ReadAll(winterHTMLResp.Body)
			if err != nil {
				return err
			}

			// In case I ever move off GitHub Pages, make sure we continue this implementation detail;
			// twos.dev/winter is a required path because that's the Go import path.
			if string(winterBody) != string(winterHTMLBody) {
				return fmt.Errorf("twos.dev/winter and twos.dev/winter.html must be the same page")
			}
			return nil
		},
	}
}
