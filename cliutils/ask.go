// Package cliutils holds helper functions for interacting with a user at a command-line interface.
package cliutils // import "twos.dev/winter/cliutils"

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// Ask writes question to out with a visual treatment indicating a prompt,
// then returns everything read from in until the next line break.
//
// Ask blocks until said line break.
// The line break is not included in answer.
//
// If using Ask with os.Stdin and os.Stdout and no error is expected,
// use [MustAsk] instead.
func Ask(question string, in io.Reader, out io.Writer) (answer string, err error) {
	if _, err := out.Write([]byte(question)); err != nil {
		return "", fmt.Errorf("cannot ask %q: %w", question, err)
	}
	scanner := bufio.NewScanner(in)
	_ = scanner.Scan()
	return scanner.Text(), scanner.Err()
}

// MustAsk writes question to stdout with a visual treatment indicating a prompt,
// then returns everything read from stdin until the next line break.
//
// MustAsk blocks until said line break.
// The line break is not included in answer.
//
// If any issue occurs, MustAsk panics.
// To handle errors or use other readers or writers,
// use [Ask] instead.
func MustAsk(question string) (answer string) {
	answer, err := Ask(question, os.Stdin, os.Stdout)
	if err != nil {
		panic(fmt.Sprintf("Cannot ask question at CLI: %s", err))
	}
	return answer
}
