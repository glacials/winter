// Package cliutils holds helper functions for interacting with a user at a command-line interface.
package cliutils // import "twos.dev/winter/cliutils"

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	NoDefault = "[[twos.dev/winter/cliutils.NoDefault]]"
)

// Ask writes question to out with a visual treatment indicating a prompt,
// then reads from in until it reads a line break.
// It returns all text read until the line break (excluding it).
//
// If dfault is [NoDefault],
// when the returned text would be an empty string,
// the function repeats from the start until the returned text would be nonempty.
//
// If dfault is any other string,
// when the returned text would be an empty string,
// dfault is returned instead.
//
// question is implicitly passed through [Sprintf] for formatting.
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
	if _, err := out.Write([]byte("\n")); err != nil {
		return "", err
	}
	return text, scanner.Err()
}

// MustAsk is like Ask but panics on error and always interacts with stdin/stdout.
//
// MustAsk writes question to stdout with a visual treatment indicating a prompt,
// then reads from stdin until it reads a line break.
// It returns all text read until the line break (excluding it).
//
// If dfault is [NoDefault],
// when the returned text would be an empty string,
// the function repeats from the start until the returned text would be nonempty.
//
// If dfault is any other string,
// when the returned text would be an empty string,
// dfault is returned instead.
//
// question is implicitly passed through [Sprintf] for formatting.
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
