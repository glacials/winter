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

// AskBool is like Ask but asks the user a yes/no question instead of a generic string question.
//
// AskBool writes question to out with a visual treatment indicating a prompt,
// then reads from in until it reads a line break.
//
// If the read text is "y", "Y", "yes", or similar; true is returned.
// If the read text is "n", "N", "no", or similar; false is returned.
// If the read text is the empty string, dfault is returned.
// Otherwise, the function repeats from the start until a sufficient answer is read.
//
// question is implicitly passed through [Sprintf] for formatting.
func AskBool(question string, dfault bool, in io.Reader, out io.Writer) (answer bool, err error) {
	var dfaultStr string
	if dfault {
		dfaultStr = "Y/n"
	} else {
		dfaultStr = "y/N"
	}
	answerStr, err := Ask(question, dfaultStr, in, out)
	if err != nil {
		return false, err
	}
	if answerStr == "Y/n" {
		return true, nil
	}
	if answerStr == "y/N" {
		return false, nil
	}
	ans := strings.TrimSpace(strings.ToLower(answerStr))
	if ans == "y" || ans == "yes" || ans == "ye" || ans == "1" {
		return true, nil
	}
	if ans == "n" || ans == "no" || ans == "0" {
		return false, nil
	}
	return AskBool(question, dfault, in, out)
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
