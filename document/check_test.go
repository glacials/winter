package document

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckSite(t *testing.T) {
	dist := t.TempDir()
	writeTestSiteFile(t, dist, "index.html", `<html><body id="top">
<a href="/about.html#bio">About</a>
<img src="/img/photo.webp" srcset="/img/photo-small.webp 320w, /img/photo.webp 1024w">
</body></html>`)
	writeTestSiteFile(t, dist, "about.html", `<h1 id="bio">Biography</h1>`)
	writeTestSiteFile(t, dist, "img/photo.webp", "photo")
	writeTestSiteFile(t, dist, "img/photo-small.webp", "small photo")
	writeTestSiteFile(t, dist, "style.css", `body { background: url('/img/photo.webp'); }`)

	if err := CheckSite(dist); err != nil {
		t.Fatalf("CheckSite returned an error for a valid site: %v", err)
	}
}

func TestCheckSiteReportsBrokenSrcsetAndFragment(t *testing.T) {
	dist := t.TempDir()
	writeTestSiteFile(t, dist, "index.html", `<html><body>
<a href="/about.html#missing">About</a>
<img srcset="/img/missing.webp 320w">
</body></html>`)
	writeTestSiteFile(t, dist, "about.html", `<h1 id="bio">Biography</h1>`)

	err := CheckSite(dist)
	if err == nil {
		t.Fatal("CheckSite unexpectedly accepted broken references")
	}
	for _, want := range []string{"missing fragment #missing", `missing "img/missing.webp"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err, want)
		}
	}
}

func writeTestSiteFile(t *testing.T, root, rel, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
