package document

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v2"
)

func TestCreatedDateIsSeparateFromPublication(t *testing.T) {
	for _, format := range []string{"markdown", "html", "org"} {
		for _, tt := range []struct {
			name, yaml, org, published string
			wantErr                    bool
		}{
			{name: "created only", yaml: "created: 2025-08-11", org: "#+CREATED: 2025-08-11"},
			{name: "with published", yaml: "created: 2025-08-11\npublished: 2026-10-25", org: "#+CREATED: 2025-08-11\n#+PUBLISHED: 2026-10-25", published: "2026-10-25"},
			{name: "with date alias", yaml: "date: 2026-10-25\ncreated: 2025-08-11", org: "#+DATE: 2026-10-25\n#+CREATED: 2025-08-11", published: "2026-10-25"},
			{name: "mixed case org", yaml: "created: 2025-08-11", org: "#+CrEaTeD: 2025-08-11"},
			{name: "invalid created", yaml: "created: invalid", org: "#+CREATED: invalid", wantErr: true},
		} {
			t.Run(format+"/"+tt.name, func(t *testing.T) {
				meta := NewMetadata("src/example", "templates")
				var doc Document
				input := "---\n" + tt.yaml + "\n---\n# Example\nBody"
				switch format {
				case "markdown":
					doc = NewMarkdownDocument("src/example.md", meta, nil)
				case "html":
					doc = NewHTMLDocument("src/example.html", meta, nil)
					input = "---\n" + tt.yaml + "\n---\n<h1>Example</h1><p>Body</p>"
				case "org":
					doc = NewOrgDocument("src/example.org", meta, NewHTMLDocument("src/example.org", meta, nil))
					input = tt.org + "\n* Example\nBody"
				}
				err := doc.Load(strings.NewReader(input))
				if tt.wantErr {
					if err == nil {
						t.Fatal("expected invalid created date to fail")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if got := meta.CreatedAt.Format(time.DateOnly); got != "2025-08-11" {
					t.Fatalf("creation date = %s", got)
				}
				if tt.published == "" {
					if !meta.PublishedAt.IsZero() {
						t.Fatalf("created changed publication date: %v", meta.PublishedAt)
					}
				} else if got := meta.PublishedAt.Format(time.DateOnly); got != tt.published {
					t.Fatalf("publication date = %s, want %s", got, tt.published)
				}
				for name, encode := range map[string]func(interface{}) ([]byte, error){"yaml": yaml.Marshal, "json": json.Marshal} {
					encoded, err := encode(meta)
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(string(encoded), "2025-08-11") {
						t.Fatalf("creation date missing from %s: %s", name, encoded)
					}
				}
			})
		}
	}
}
