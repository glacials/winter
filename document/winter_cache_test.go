package document

import (
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/adrg/xdg"
	"gotest.tools/v3/assert"
)

func TestExecuteAllUsesCachedImageMetadata(t *testing.T) {
	tmp := t.TempDir()

	cwd, err := os.Getwd()
	assert.NilError(t, err)
	assert.NilError(t, os.Chdir(tmp))
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))

	srcImg := filepath.Join("src", "img", "2023", "trip", "IMG_0385.JPG")
	assert.NilError(t, os.MkdirAll(filepath.Dir(srcImg), 0o755))
	assert.NilError(t, copyFile(filepath.Join(cwd, "testdata", "IMG_0385.JPG"), srcImg))

	distImg := filepath.Join("dist", "img", "2023", "trip", "IMG_0385.webp")
	assert.NilError(t, os.MkdirAll(filepath.Dir(distImg), 0o755))
	assert.NilError(t, copyFile(filepath.Join(cwd, "testdata", "IMG_0385.webp"), distImg))

	oldTime := time.Unix(1, 0)
	assert.NilError(t, os.Chtimes(distImg, oldTime, oldTime))

	sumPath, err := xdg.CacheFile(fmt.Sprintf("%s.sum", filepath.Join(AppName, "generated", "img", filepath.Base(srcImg))))
	assert.NilError(t, err)
	sum := fnv.New32()
	data, err := os.ReadFile(srcImg)
	assert.NilError(t, err)
	_, err = sum.Write(data)
	assert.NilError(t, err)
	assert.NilError(t, os.MkdirAll(filepath.Dir(sumPath), 0o755))
	assert.NilError(t, os.WriteFile(sumPath, []byte(fmt.Sprintf("%d", sum.Sum32())), 0o644))

	cfg := &Config{
		Description: "desc",
		Dist:        "dist",
		Name:        "Test Site",
		Since:       2024,
	}
	cfg.Author.Name = "Tester"
	cfg.Author.Email = "tester@example.com"
	cfg.Gear = []Gear{{
		Link:  "https://example.com/canon-eos-rebel-t7",
		Make:  "Canon",
		Model: "EOS Rebel T7",
	}}
	cfg.Gear[0].EXIF.Make = "Canon"
	cfg.Gear[0].EXIF.Model = "Canon EOS Rebel T7"
	cfg.Development.URL = "http://localhost:8100"
	cfg.Known.URIs = filepath.Join("src", "uris.txt")
	cfg.Production.URL = "example.com"

	s, err := NewSubstructure(cfg)
	assert.NilError(t, err)

	before, err := os.Stat(distImg)
	assert.NilError(t, err)

	assert.NilError(t, s.ExecuteAll(cfg.Dist))

	after, err := os.Stat(distImg)
	assert.NilError(t, err)
	assert.Assert(t, after.ModTime().Equal(before.ModTime()), "cached build should not rewrite image")

	gallery := s.galleries["trip"]
	assert.Assert(t, len(gallery) == 1)
	assert.Assert(t, len(gallery[0].Thumbnails) > 0)
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
