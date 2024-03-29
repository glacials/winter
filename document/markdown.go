package document // import "twos.dev/winter/document"

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	mdhtml "github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	gemini "github.com/tdemin/gmnhg"
)

var mdrepl = map[string][]byte{
	"&quot;": []byte("\""),
}

var (
	templateStart = []byte("{{")
	templateEnd   = []byte("}}")
)

// MarkdownDocument represents a source file written in Markdown,
// with optional Go template syntax embedded in it.
//
// MarkdownDocument implements [Document].
//
// The MarkdownDocument is transitory;
// its only purpose is to create a [TemplateDocument].
type MarkdownDocument struct {
	deps map[string]struct{}
	meta *Metadata
	// next holds the incarnations of this document that come after Markdown rendering is complete.
	next map[Document]struct{}

	html    []byte
	gemtext []byte
}

type TemplateNode struct {
	ast.Leaf
	Raw []byte
}

// NewMarkdownDocument creates a new document whose original source is at path src.
//
// Nothing is read from disk; src is metadata.
// To read and parse Markdown, call [Load].
func NewMarkdownDocument(src string, meta *Metadata, next map[Document]struct{}) *MarkdownDocument {
	return &MarkdownDocument{
		deps: map[string]struct{}{
			src:                {},
			"public/style.css": {},
		},
		meta: meta,
		next: next,
	}
}

func (doc *MarkdownDocument) DependsOn(src string) bool {
	if _, ok := doc.deps[src]; ok {
		return true
	}
	if doc.meta.WebPath == "/archives.html" || doc.meta.WebPath == "/writing.html" || doc.meta.WebPath == "/index.html" {
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

// Load reads Markdown from r and loads it into doc.
//
// If called more than once, the last call wins.
func (doc *MarkdownDocument) Load(r io.Reader) error {
	mdbody1, err := doc.meta.UnmarshalDocument(r)
	if err != nil {
		return fmt.Errorf("cannot load template frontmatter for %q: %w", doc.meta.SourcePath, err)
	}

	mdbody2 := make([]byte, len(mdbody1))
	copy(mdbody2, mdbody1)
	if err := doc.loadForGemini(mdbody1); err != nil {
		return err
	}
	if err := doc.loadForHTML(mdbody2); err != nil {
		return err
	}
	return nil
}

func (doc *MarkdownDocument) loadForGemini(mdbody []byte) error {
	gemtext, err := gemini.RenderMarkdown(mdbody, gemini.Defaults)
	if err != nil {
		return fmt.Errorf("cannot convert markdown in %s to gemtext: %w", doc.meta.SourcePath, err)
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

func (doc *MarkdownDocument) loadForHTML(mdbody []byte) error {
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

	byts := markdown.ToHTML(mdbody, p, newRenderer())
	for old, new := range mdrepl {
		byts = bytes.ReplaceAll(byts, []byte(old), new)
	}
	doc.html = byts
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

func (doc *MarkdownDocument) Metadata() *Metadata {
	return doc.meta
}

func (doc *MarkdownDocument) Render(w io.Writer) error {
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

func (doc *MarkdownDocument) RenderGemini(w io.Writer) error {
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

func newRenderer() *mdhtml.Renderer {
	opts := mdhtml.RendererOptions{
		Flags: mdhtml.FlagsNone,
	}
	return mdhtml.NewRenderer(opts)
}

func parserHook(data []byte) (ast.Node, []byte, int) {
	if !bytes.HasPrefix(data, templateStart) {
		return nil, nil, 0
	}
	start := bytes.Index(data, templateStart)
	if start < 0 {
		return nil, nil, 0
	}
	end := bytes.Index(data, templateEnd)
	if end < 0 {
		return nil, data, 0
	}
	return &ast.Text{Leaf: ast.Leaf{Literal: data[0 : end+len(templateEnd)]}}, nil, end + len(templateEnd)
}
