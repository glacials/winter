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
	dfault   string
	stdin    string
	expected string
}

func TestAsk(t *testing.T) {
	for _, test := range []testCase{
		{
			name:     "EmptyAnswer",
			question: "Test",
			dfault:   "",
			stdin:    "\n",
			expected: "",
		},
		{
			name:     "SomeText",
			question: "Test",
			dfault:   "",
			stdin:    "abc\n",
			expected: "abc",
		},
		{
			name:     "EmptyQuestion",
			question: "Test",
			dfault:   "",
			stdin:    "\n",
			expected: "",
		},
		{
			name:     "WithDefault",
			question: "Test",
			dfault:   "abc123",
			stdin:    "\n",
			expected: "abc123",
		},
		{
			name:     "WithUnusedDefault",
			question: "Test",
			dfault:   "abc123",
			stdin:    "def456\n",
			expected: "def456",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			out := &bytes.Buffer{}

			actualAnswer, err := Ask(test.question, test.dfault, strings.NewReader(test.stdin), out)
			assert.NilError(t, err)
			assert.Equal(t, actualAnswer, test.expected)

			actualQuestion, err := io.ReadAll(out)
			assert.NilError(t, err)
			assert.Equal(t, strings.Contains(string(actualQuestion), test.question), true)
		})
	}
}
