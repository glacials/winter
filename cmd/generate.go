package cmd // import "twos.dev/winter/cmd"

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"regexp"

	"github.com/invopop/jsonschema"
	"github.com/spf13/cobra"
	"twos.dev/winter/cliutils"
	"twos.dev/winter/document"
)

func newGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate internal Winter code",
		Long: cliutils.Sprintf(`
			Generate secondary artifacts for Winter to integrate with the system it runs on.

			This should never need to be run manually.
			It is run as part of Winter's build process.

			Currently, it only generates a YAML schema for winter.yml.
		`),
		Args:   cobra.NoArgs,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerateCmd(os.Stdin, os.Stdout)
		},
	}
	return cmd
}

// runGenerateCmd performs execution of the winter generate command.
// It is separate from the corresponding [cobra.Command] function for easy testing.
func runGenerateCmd(in io.Reader, out io.Writer) error {
	reflector := jsonschema.Reflector{}
	if err := reflector.AddGoComments("twos.dev/winter", "./document"); err != nil {
		return err
	}
	for comments := range reflector.CommentMap {
		// See https://github.com/invopop/jsonschema/issues/85
		reflector.CommentMap[comments] = regexp.MustCompile(
			`([^\s])[^\S\r\n]*\n[^\S\r\n]*([^\s])`,
		).ReplaceAllString(reflector.CommentMap[comments], "$1 $2")
	}
	schema, err := reflector.Reflect(document.Config{}).MarshalJSON()
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, schema, "", "  "); err != nil {
		return err
	}
	return os.WriteFile(
		"./cmd/winter.schema.yml",
		buf.Bytes(),
		fs.FileMode(0o644),
	)
}
