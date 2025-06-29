package document // import "twos.dev/winter/document"

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/gomarkdown/markdown/parser"
	gemini "github.com/tdemin/gmnhg"
)

// TextDocument represents a source file written in plain text,
// with optional Go template syntax embedded in it.
//
// TextDocument implements [Document].
//
// The TextDocument is transitory;
// its only purpose is to create a [TemplateDocument].
type TextDocument struct {
	deps map[string]struct{}
	meta *Metadata
	// next holds the incarnations of this document that come after rendering is
	// complete.
	next map[Document]struct{}

	// gemtext is a staging area for rendered gemtext.
	gemtext []byte
	// html is a staging area for rendered HTML.
	html []byte
	// raw is a staging area for raw text.
	raw []byte
}

// NewTextDocument creates a new document whose original source is at path src.
//
// Nothing is read from disk; src is metadata.
// To read from src, call [Load] on the result.
func NewTextDocument(
	src string,
	meta *Metadata,
	next map[Document]struct{},
) *TextDocument {
	return &TextDocument{
		deps: map[string]struct{}{
			src:                {},
			"public/style.css": {},
		},
		meta: meta,
		next: next,
	}
}

func (doc *TextDocument) DependsOn(src string) bool {
	if _, ok := doc.deps[src]; ok {
		return true
	}
	if doc.meta.WebPath == "/archives.html" ||
		doc.meta.WebPath == "/writing.html" ||
		doc.meta.WebPath == "/index.html" {
		return true
	}
	if strings.HasPrefix(filepath.Clean(src), "src/templates/") {
		return true
	}
	for next := range doc.next {
		if next.DependsOn(src) {
			return true
		}
	}
	return false
}

// Load reads text from r and loads it into doc.
//
// If called more than once, the last call wins.
func (doc *TextDocument) Load(r io.Reader) error {
	txtbody1, err := doc.meta.UnmarshalDocument(r)
	if err != nil {
		return fmt.Errorf(
			"cannot load template frontmatter for %q: %w",
			doc.meta.SourcePath,
			err,
		)
	}

	txtbody2 := make([]byte, len(txtbody1))
	txtbody3 := make([]byte, len(txtbody1))
	copy(txtbody2, txtbody1)
	copy(txtbody3, txtbody1)
	if err := doc.loadForGemini(txtbody1); err != nil {
		return err
	}
	if err := doc.loadForHTML(txtbody2); err != nil {
		return err
	}
	if err := doc.loadForRaw(txtbody3); err != nil {
		return err
	}
	return nil
}

func (doc *TextDocument) loadForGemini(mdbody []byte) error {
	gemtext, err := gemini.RenderMarkdown(mdbody, gemini.Defaults)
	if err != nil {
		return fmt.Errorf(
			"cannot convert markdown in %s to gemtext: %w",
			doc.meta.SourcePath,
			err,
		)
	}
	doc.gemtext = gemtext
	for next := range doc.next {
		if geminiDoc, ok := next.(*GeminiDocument); ok {
			if err := geminiDoc.Load(bytes.NewReader(doc.gemtext)); err != nil {
				return fmt.Errorf("cannot load from %T to %T: %w", doc, next, err)
			}
		}
	}
	return nil
}

func (doc *TextDocument) loadForHTML(body []byte) error {
	p := parser.NewWithExtensions(
		parser.Attributes |
			parser.Autolink |
			parser.FencedCode |
			parser.Footnotes |
			parser.HeadingIDs |
			parser.MathJax |
			parser.Strikethrough |
			parser.Tables,
	)
	p.Opts.ParserHook = parserHook

	doc.html = append(
		fmt.Appendf(
			[]byte{},
			"<p><a href='%s'>See raw</a></p>\n",
			doc.meta.RawPath,
		),
		append(
			append(
				[]byte("<pre>"),
				body...,
			),
			[]byte("</pre>")...,
		)...,
	)
	if doc.next == nil {
		return nil
	}
	for next := range doc.next {
		if htmlDoc, ok := next.(*HTMLDocument); ok {
			if err := htmlDoc.Load(bytes.NewReader(doc.html)); err != nil {
				return fmt.Errorf("cannot load from %T to %T: %w", doc, next, err)
			}
		}
	}
	return nil
}

func (doc *TextDocument) loadForRaw(mdbody []byte) error {
	doc.raw = mdbody
	if doc.next == nil {
		return nil
	}
	for next := range doc.next {
		if textDoc, ok := next.(*RawDocument); ok {
			if err := textDoc.Load(bytes.NewReader(doc.raw)); err != nil {
				return fmt.Errorf("cannot load from %T to %T: %w", doc, next, err)
			}
		}
	}
	return nil
}

func (doc *TextDocument) Metadata() *Metadata {
	return doc.meta
}

func (doc *TextDocument) Render(w io.Writer) error {
	if doc.next == nil {
		if _, err := io.Copy(w, bytes.NewReader(doc.html)); err != nil {
			return fmt.Errorf("cannot render Markdown: %w", err)
		}
		return nil
	}
	for next := range doc.next {
		if html, ok := next.(*HTMLDocument); ok {
			if err := html.Render(w); err != nil {
				return fmt.Errorf("cannot render from %T to %T: %w", doc, next, err)
			}
		}
	}
	return nil
}

func (doc *TextDocument) RenderGemini(w io.Writer) error {
	if doc.next == nil {
		if _, err := io.Copy(w, bytes.NewReader(doc.gemtext)); err != nil {
			return fmt.Errorf("cannot render Markdown: %w", err)
		}
		return nil
	}
	for next := range doc.next {
		if gemini, ok := next.(*GeminiDocument); ok {
			if err := gemini.Render(w); err != nil {
				return fmt.Errorf("cannot render from %T to %T: %w", doc, next, err)
			}
		}
	}
	return nil
}

func (doc *TextDocument) RenderRaw(w io.Writer) error {
	if doc.next == nil {
		if _, err := io.Copy(w, bytes.NewReader(doc.raw)); err != nil {
			return fmt.Errorf("cannot render Markdown: %w", err)
		}
		return nil
	}
	for next := range doc.next {
		if raw, ok := next.(*RawDocument); ok {
			if err := raw.Render(w); err != nil {
				return fmt.Errorf("cannot render from %T to %T: %w", doc, next, err)
			}
		}
	}
	return nil
}
