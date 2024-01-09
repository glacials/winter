package document // import "twos.dev/winter/document"

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// GeminiRenderer is a type that can render itself into gemtext,
// the markup language for Geminispace,
// an alternate web.
type GeminiRenderer interface {
	// RenderGemini converts the document into gemtext,
	// then writes the result to w.
	RenderGemini(w io.Writer) error
}

// GeminiDocument represents a document written in or converted to gemtext,
// the markup language for Geminispace (https://geminiprotocol.net/docs/gemtext.gmi).
//
// GeminiDocument implements [Document].
type GeminiDocument struct {
	deps    map[string]struct{}
	meta    *Metadata
	gemtext []byte
}

// NewGeminiDocument creates a new gemtext document whose original source is at path src.
//
// Nothing is read from disk;
// src is metadata.
// To read and parse gemtext, call [Load].
func NewGeminiDocument(src string, meta *Metadata) *GeminiDocument {
	return &GeminiDocument{
		deps: map[string]struct{}{
			src: {},
		},
		meta: meta,
	}
}

func (doc *GeminiDocument) DependsOn(src string) bool {
	if _, ok := doc.deps[src]; ok {
		return true
	}
	if doc.meta.WebPath == "/archives.html" || doc.meta.WebPath == "/writing.html" || doc.meta.WebPath == "/index.html" {
		return true
	}
	if strings.HasPrefix(filepath.Clean(src), "src/templates/") {
		return true
	}
	return false
}

// Load reads Gemini from r and loads it into doc.
//
// If called more than once, the last call wins.
func (doc *GeminiDocument) Load(r io.Reader) error {
	gemtext, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("cannot read gemtext for %s into document: %w", doc.meta.SourcePath, err)
	}
	doc.gemtext = gemtext
	return nil
}

func (doc *GeminiDocument) Metadata() *Metadata {
	return doc.meta
}

func (doc *GeminiDocument) Render(w io.Writer) error {
	if _, err := io.Copy(w, bytes.NewReader(doc.gemtext)); err != nil {
		return fmt.Errorf("cannot render Markdown: %w", err)
	}
	return nil
}
