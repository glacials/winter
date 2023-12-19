package document // import "twos.dev/winter/document"

import (
	"fmt"
	"io"
	"sort"

	"golang.org/x/net/html/atom"
)

const (
	draft kind = iota
	post
	page
	static
)

const (
	tocEl     = atom.Ol
	toc       = "<ol id=\"toc\">{{.Entries}}</ol>"
	tocEntry  = "<li><a href=\"#{{.Anchor}}\">{{.Section}}</a></li>"
	tocMax    = 5
	tocMin    = 2
	tocReturn = `
<span style="margin-left:0.5em">
	<a href="#{{.Anchor}}" style="text-decoration:none">#</a>
	<a href="#toc" style="text-decoration:none">&uarr;</a>
</span>
`
)

var hi = map[atom.Atom]int{
	atom.H1: 1,
	atom.H2: 2,
	atom.H3: 3,
	atom.H4: 4,
	atom.H5: 5,
	atom.H6: 6,
}

// Document is something that can be built,
// usually from a source file on disk to a destination file on disk.
//
// After a document has been built by calling [Build],
// it can be passed to a template during execution:
//
//	var buf bytes.Buffer
//	t.Execute(&buf, d)
type Document interface {
	// DependsOn returns true if and only if the given source path,
	// when changed,
	// should cause this document to be rebuilt.
	DependsOn(src string) bool
	// Load reads or re-reads the source file from disk,
	// overwriting any previously stored or parsed contents.
	Load(r io.Reader) error
	// Metadata returns data about the document,
	// which may have been inferred automatically or set by frontmatter.
	Metadata() *Metadata
	// Render generates the final HTML for the document and writes it to w.
	Render(w io.Writer) error
}

type kind int

func (k *kind) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	var tmp kind

	tmp, err := parseKind(s)
	if err != nil {
		return err
	}

	*k = tmp

	return nil
}

func parseKind(s string) (kind, error) {
	switch s {
	case "draft", "":
		return draft, nil
	case "post":
		return post, nil
	case "page":
		return page, nil
	case "gallery":
		return static, nil
	}
	return -1, fmt.Errorf("unknown kind %q", s)
}

// documents is sortable collection of Documents.
type documents struct {
	// All is the collection of documents that defines the documents struct.
	// It is nested in a struct,
	// rather than directly defining documents,
	// because non-pointer references to slices cannot have their headers modified
	// (therefore operations like append don't propagate back).
	All []Document
}

func (d *documents) add(doc Document) {
	if d.All == nil {
		d.All = []Document{}
	}
	d.All = append(d.All, doc)
}

func (d *documents) addOrUpdate(doc Document) {
	if d.All == nil {
		d.All = []Document{}
	}
	for i, existing := range d.All {
		if existing.Metadata().SourcePath == doc.Metadata().SourcePath {
			d.All[i] = doc
			// Sort again in case d's creation date changed.
			sort.Sort(d)
			return
		}
	}
	d.add(doc)
}

func (d *documents) Len() int {
	return len(d.All)
}

func (d *documents) Less(i, j int) bool {
	return d.All[i].Metadata().CreatedAt.After(d.All[j].Metadata().CreatedAt)
}

func (d *documents) Swap(i, j int) {
	d.All[i], d.All[j] = d.All[j], d.All[i]
}
