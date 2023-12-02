package cliutils // import "twos.dev/winter/cliutils"

import (
	"fmt"
	"strings"

	markdown "github.com/MichaelMure/go-term-markdown"
	"github.com/lithammer/dedent"
)

// Printf is like [fmt.Printf],
// but applies several visual treatments to make format pleasant to read at a terminal,
// especially when it spans multiple lines or paragraphs.
//
// Specifically,
// single newlines and common indentation are ignored
// and the text is wrapped to 80 characters
// (allowing strings to be well-formatted in both source code and output),
// and limited Markdown support is applied.
func Printf(format string, args ...any) (int, error) {
	return fmt.Print(Sprintf(format, args...))
}

// Sprintf is like [fmt.Sprintf],
// but applies several visual treatments to make format pleasant to read at a terminal,
// especially when it spans multiple lines or paragraphs.
//
// Specifically,
// single newlines and common indentation are ignored
// and the text is wrapped to 80 characters
// (allowing strings to be well-formatted in both source code and output),
// and limited Markdown support is applied.
func Sprintf(format string, args ...any) string {
	format = fmt.Sprintf(format, args...)
	format = dedent.Dedent(format)
	format = strings.TrimSpace(format)
	format = string(markdown.Render(format, 80, 0))
	return format
}
