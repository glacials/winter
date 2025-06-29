package document // import "twos.dev/winter/document"

import "io"

// RawRenderer is a type that can render itself into raw text.
//
// For example, a .txt file can be rendered either to HTML,
// which embeds it into the standard set of templates,
// or into a raw text file,
// which evaluates frontmatter and templates but otherwise copies the file.
type RawRenderer interface {
	// RenderRaw writes the rendered document to w.
	RenderRaw(io.Writer) error
}

// RawDocument represents a document with no special processing.
// It is virtually a passthrough from source to render.
//
// RawDocument implements [Document].
type RawDocument struct {
	deps map[string]struct{}
	meta *Metadata
	data []byte
}

// NewRawDocument creates a new document whose original source is at path src.
//
// Nothing is read from disk; src is metadata.
// To read from src, call [Load] on the result.
func NewRawDocument(src string, meta *Metadata) *RawDocument {
	return &RawDocument{
		deps: map[string]struct{}{
			src: {},
		},
		meta: meta,
	}
}

func (doc *RawDocument) DependsOn(src string) bool {
	if _, ok := doc.deps[src]; ok {
		return true
	}
	return false
}

// Load reads text from r and loads it into doc.
//
// If called more than once, the last call wins.
func (doc *RawDocument) Load(r io.Reader) error {
	doc.data, _ = io.ReadAll(r)
	return nil
}

func (doc *RawDocument) Metadata() *Metadata {
	return doc.meta
}

func (doc *RawDocument) Render(w io.Writer) error {
	_, err := w.Write(doc.data)
	return err
}
