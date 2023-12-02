package cmd // import "twos.dev/winter/cmd"

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alecthomas/assert"
)

func TestRunInitCmd(t *testing.T) {
	stdin := strings.NewReader("/tmp/abc\n")
	stdout := &bytes.Buffer{}
	assert.NoError(t, runInitCmd(stdin, stdout))
}
