package document

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v2"
)

func TestPublicationDateFrontmatter(t *testing.T) {
	for _, tt := range []struct {
		name, fields, want string
		wantErr            bool
	}{
		{name: "published", fields: "published: 2026-10-25\n", want: "2026-10-25"},
		{name: "date alias", fields: "date: 2026-04-18\n", want: "2026-04-18"},
		{name: "published first", fields: "published: 2026-10-25\ndate: 2026-04-18\n", want: "2026-10-25"},
		{name: "published last", fields: "date: 2026-04-18\npublished: 2026-10-25\n", want: "2026-10-25"},
		{name: "no date"},
		{name: "null published", fields: "published: null\ndate: 2026-04-18\n", want: "2026-04-18"},
		{name: "invalid published", fields: "published: invalid\n", wantErr: true},
		{name: "invalid date", fields: "date: invalid\n", wantErr: true},
		{name: "invalid created", fields: "created: invalid\n", wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			meta := NewMetadata("src/example.md", "templates")
			input := "---\n" + tt.fields + "type: post\nupdated: 2026-10-26\n---\nBody\n"
			body, err := meta.UnmarshalDocument(strings.NewReader(input))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected invalid date to fail")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.want == "" {
				if !meta.PublishedAt.IsZero() {
					t.Fatalf("unexpected publication date: %v", meta.PublishedAt)
				}
			} else if got := meta.PublishedAt.Format(time.DateOnly); got != tt.want {
				t.Fatalf("publication date = %s, want %s", got, tt.want)
			}
			if !meta.IsType("post") || meta.UpdatedAt.Format(time.DateOnly) != "2026-10-26" {
				t.Fatalf("other frontmatter was not preserved: %+v", meta)
			}
			if meta.SourcePath != "src/example.md" || meta.WebPath != "example.html" || meta.Layout != "templates/text_document.html.tmpl" {
				t.Fatalf("metadata defaults were not preserved: %+v", meta)
			}
			if string(body) != "Body\n" {
				t.Fatalf("body = %q", body)
			}
		})
	}
}

func TestPublicationDateReload(t *testing.T) {
	meta := NewMetadata("src/example.md", "templates")
	for _, fields := range []string{"published: 2026-10-25", "date: 2026-10-26"} {
		if _, err := meta.UnmarshalDocument(strings.NewReader("---\n" + fields + "\n---\nBody")); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := meta.UnmarshalDocument(strings.NewReader("Body without frontmatter")); err != nil {
		t.Fatal(err)
	}
	if got := meta.PublishedAt.Format(time.DateOnly); got != "2026-10-26" {
		t.Fatalf("reloaded publication date = %s", got)
	}
}

func TestPublicationDateMarshalsCanonicalKey(t *testing.T) {
	var meta Metadata
	if err := yaml.Unmarshal([]byte("date: 2026-10-25\n"), &meta); err != nil {
		t.Fatal(err)
	}
	encoded, err := yaml.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "published:") || strings.Contains(string(encoded), "\ndate:") || strings.HasPrefix(string(encoded), "date:") {
		t.Fatalf("expected only canonical publication key, got:\n%s", encoded)
	}
}

func TestPublicationDateDocumentFormats(t *testing.T) {
	for _, format := range []string{"markdown", "html", "org"} {
		for _, fields := range []struct{ name, yaml, org string }{
			{"published", "published: 2026-10-25", "#+PUBLISHED: 2026-10-25"},
			{"date alias", "date: 2026-10-25", "#+DATE: 2026-10-25"},
			{"published first", "published: 2026-10-25\ndate: 2026-04-18", "#+PUBLISHED: 2026-10-25\n#+DATE: 2026-04-18"},
			{"published last", "date: 2026-04-18\npublished: 2026-10-25", "#+DATE: 2026-04-18\n#+PUBLISHED: 2026-10-25"},
			{"mixed case org", "published: 2026-10-25", "#+PuBlIsHeD: 2026-10-25"},
		} {
			t.Run(format+"/"+fields.name, func(t *testing.T) {
				fields.yaml += "\ncreated: 2026-04-18\nupdated: 2026-10-26"
				fields.org += "\n#+CrEaTeD: 2026-04-18\n#+UPDATED: 2026-10-26"
				meta := NewMetadata("src/example", "templates")
				var doc Document
				input := "---\n" + fields.yaml + "\n---\n# Example\nBody"
				switch format {
				case "markdown":
					doc = NewMarkdownDocument("src/example.md", meta, nil)
				case "html":
					doc = NewHTMLDocument("src/example.html", meta, nil)
					input = "---\n" + fields.yaml + "\n---\n<h1>Example</h1><p>Body</p>"
				case "org":
					doc = NewOrgDocument("src/example.org", meta, NewHTMLDocument("src/example.org", meta, nil))
					input = fields.org + "\n* Example\nBody"
				}
				if err := doc.Load(strings.NewReader(input)); err != nil {
					t.Fatal(err)
				}
				if got := meta.PublishedAt.Format(time.DateOnly); got != "2026-10-25" {
					t.Fatalf("publication date = %s, want 2026-10-25", got)
				}
				if got := meta.CreatedAt.Format(time.DateOnly); got != "2026-04-18" {
					t.Fatalf("start date = %s, want 2026-04-18", got)
				}
				if got := meta.UpdatedAt.Format(time.DateOnly); got != "2026-10-26" {
					t.Fatalf("updated date = %s, want 2026-10-26", got)
				}
				tmpl := template.Must(template.New("dates").Parse(`{{ .CreatedAt.Format "2006-01-02" }} / {{ .PublishedAt.Format "2006-01-02" }} / {{ .UpdatedAt.Format "2006-01-02" }}`))
				var rendered bytes.Buffer
				if err := tmpl.Execute(&rendered, meta); err != nil {
					t.Fatal(err)
				}
				if got := rendered.String(); got != "2026-04-18 / 2026-10-25 / 2026-10-26" {
					t.Fatalf("rendered dates = %q", got)
				}
			})
		}
	}
}
