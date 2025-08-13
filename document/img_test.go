package document

import (
	"bytes"
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

const sampleJPGPath = "testdata/IMG_0385.JPG"
const sampleWebpPath = "testdata/IMG_0385.webp"
const sampleWinterYML = `
production:
  url: twos.dev
gear:
  - make: Canon
    model: EOS Rebel T7
    link: https://twos.dev
    exif:
      make: Canon
      model: Canon EOS Rebel T7
`

func TestNewIMG(t *testing.T) {
	c, err := newConfigFromBytes([]byte(sampleWinterYML))
	assert.NilError(t, err)

	im, err := NewIMG(sampleJPGPath, c)
	assert.NilError(t, err)

	srcf, err := os.Open(im.SourcePath)
	assert.NilError(t, err)
	defer srcf.Close()

	assert.NilError(t, im.Load(srcf))

	var b bytes.Buffer
	assert.NilError(t, im.Render(&b))
}
