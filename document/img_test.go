package document

import (
	"bytes"
	"errors"
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

func TestFormatMissingGearError(t *testing.T) {
	configuredGear := []Gear{
		{
			Make:  "Fujifilm",
			Model: "X-T5",
			Link:  "https://twos.dev/x-t5",
			EXIF: struct {
				Make  string "yaml:\"make,omitempty\""
				Model string "yaml:\"model,omitempty\""
			}{
				Make:  "FUJIFILM",
				Model: "X-T5",
			},
		},
		{
			Make:  "Fujifilm",
			Model: "16-50mm F2.8",
			Link:  "https://twos.dev/16-50",
			EXIF: struct {
				Make  string "yaml:\"make,omitempty\""
				Model string "yaml:\"model,omitempty\""
			}{
				Make:  "FUJIFILM",
				Model: "XF16-50mmF2.8-4.8 R LM WR",
			},
		},
		{
			Make:  "Olympus",
			Model: "M.Zuiko 17mm F1.8",
			Link:  "https://twos.dev/17mm",
		},
	}

	assert.Equal(
		t,
		formatMissingGearError(
			"src/img/photography/2026-japan/Japan 2026 - 9 of 54.jpeg",
			"FUJIFILM",
			"XF35mmF2 R WR",
			configuredGear,
		),
		`no matching gear entry in winter.yml for photo "src/img/photography/2026-japan/Japan 2026 - 9 of 54.jpeg"
wanted EXIF match:
  make: FUJIFILM
  model: XF35mmF2 R WR
configured gear:
  - Fujifilm X-T5 (exif.make="FUJIFILM", exif.model="X-T5")
  - Fujifilm 16-50mm F2.8 (exif.make="FUJIFILM", exif.model="XF16-50mmF2.8-4.8 R LM WR")
  - Olympus M.Zuiko 17mm F1.8 (exif.make=<unset>, exif.model=<unset>)
add a matching item under gear: in winter.yml`,
	)
}

func TestWrapErrorfMultiline(t *testing.T) {
	root := errors.New("line one\nline two")
	err := wrapErrorf(wrapErrorf(root, "inner"), "outer")

	assert.Equal(t, err.Error(), "outer:\n  inner:\n    line one\n    line two")
	assert.Assert(t, errors.Is(err, root))
}
