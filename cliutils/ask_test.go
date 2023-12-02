package cliutils // import "twos.dev/winter/cliutils"

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

type testCase struct {
	name     string
	question string
	stdin    string
	expected string
}

func TestAsk(t *testing.T) {
	for _, test := range []testCase{
		{
			name:     "EmptyAnswer",
			question: "Test",
			stdin:    "\n",
			expected: "",
		},
		{
			name:     "SomeText",
			question: "Test",
			stdin:    "abc\n",
			expected: "abc",
		},
		{
			name:     "EmptyQuestion",
			question: "Test",
			stdin:    "\n",
			expected: "",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			out := &bytes.Buffer{}

			actualAnswer, err := Ask(test.question, strings.NewReader(test.stdin), out)
			assert.NilError(t, err)
			assert.Equal(t, actualAnswer, test.expected)

			actualQuestion, err := io.ReadAll(out)
			assert.NilError(t, err)
			assert.Equal(t, string(actualQuestion), test.question)
		})
	}
}
