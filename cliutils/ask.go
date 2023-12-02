// Package cliutils holds helper functions for interacting with a user at a command-line interface.
package cliutils // import "twos.dev/winter/cliutils"

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Ask writes question to out with a visual treatment indicating a prompt,
// then returns everything read from in until the next line break.
// If the input was the empty string, dfault is returned instead.
//
// question is implicitly passed through [Sprintf] for formatting.
//
// Ask blocks until said line break.
// The line break is not included in answer.
//
// If using Ask with os.Stdin and os.Stdout and no error is expected,
// use [MustAsk] instead.
func Ask(question, dfault string, in io.Reader, out io.Writer) (answer string, err error) {
	question = strings.TrimSuffix(Sprintf(question), "\n") + " "
	if _, err := out.Write([]byte(question)); err != nil {
		return "", fmt.Errorf("cannot ask %q: %w", question, err)
	}
	scanner := bufio.NewScanner(in)
	_ = scanner.Scan()
	text := scanner.Text()
	if text == "" {
		text = dfault
	}
	return text, scanner.Err()
}

// MustAsk is like Ask but panics on error and always interacts with stdin/stdout.
//
// It writes question to stdout with a visual treatment indicating a prompt,
// then returns everything read from stdin until the next line break.
// If the input was the empty string, dfault is returned instead.
//
// question is implicitly passed through [Sprintf] for formatting.
//
// MustAsk blocks until said line break.
// The line break is not included in answer.
//
// If any issue occurs, MustAsk panics.
// To handle errors or use other readers or writers,
// use [Ask] instead.
func MustAsk(question, dfault string) (answer string) {
	answer, err := Ask(question, dfault, os.Stdin, os.Stdout)
	if err != nil {
		panic(fmt.Sprintf("Cannot ask question at CLI: %s", err))
	}
	return answer
}
