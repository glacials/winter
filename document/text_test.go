package document // import "twos.dev/winter/document"

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestTextDocument(t *testing.T) {
	src := "src/test.txt"
	doc := NewTextDocument(
		src,
		NewMetadata(src, filepath.Join("testdata", "templates")),
		nil,
	)
	if err := doc.Load(strings.NewReader("hello world")); err != nil {
		t.Fatalf("load failed: %v", err)
	}
	var buf bytes.Buffer
	if err := doc.Render(&buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	expected := "<!doctype html><html><head><meta charset=\"utf-8\"><style>body{background:white;font-family:monospace;}pre{margin:2em auto;width:fit-content;text-align:left;}#raw{display:block;text-align:center;margin-top:1em;}</style></head><body><a id=\"raw\" href=\"test.txt\">see raw</a><pre>hello world</pre></body></html>"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
	var raw bytes.Buffer
	if err := doc.RenderGemini(&raw); err != nil {
		t.Fatalf("render raw failed: %v", err)
	}
	if raw.String() != "hello world" {
		t.Errorf("expected raw %q, got %q", "hello world", raw.String())
	}
}
