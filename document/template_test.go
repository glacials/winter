package document // import "twos.dev/winter/document"

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestTemplate(t *testing.T) {
	for _, test := range []testCase{
		{
			name:     "Noop",
			input:    "abc123",
			expected: "abc123\n",
		},
		{
			name:     "SimpleTemplate",
			input:    "{{ add 1 2 }}",
			expected: "3\n",
		},
		{
			name:     "Template",
			input:    `{{ template "hello_world.tmpl" }}`,
			expected: "Hello, world!\n\n",
		},
		{
			name:     "Image",
			input:    `<img src="/path/to/image.jpg" alt="Alt text" />`,
			expected: "<img src=\"/path/to/image.jpg\" alt=\"Alt text\" />\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			src := fmt.Sprintf("src/test/%s", test.name)
			doc := NewTemplateDocument(
				src,
				NewMetadata(src, filepath.Join("testdata", "templates")),
				nil,
				nil,
				nil,
			)
			if err := doc.Load(strings.NewReader(test.input)); err != nil {
				t.Errorf("load failed: %s", err)
			}
			var actual bytes.Buffer
			if err := doc.Render(&actual); err != nil {
				t.Errorf("render failed: %s", err)
			}
			assert.Equal(t, test.expected, actual.String())
		})
	}
}

func TestTemplateDocumentDraftsFunc(t *testing.T) {
	docs := &documents{}
	docs.add(testDocument("Old Draft", draft, time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)))
	docs.add(testDocument("Post", post, time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)))
	docs.add(testDocument("New Draft", draft, time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)))

	doc := NewTemplateDocument(
		"src/test/drafts",
		NewMetadata("src/test/drafts", filepath.Join("testdata", "templates")),
		docs,
		nil,
		nil,
	)

	got := doc.draftsFunc()
	assert.Equal(t, len(got), 2)
	assert.Equal(t, got[0].Metadata().Title, "New Draft")
	assert.Equal(t, got[1].Metadata().Title, "Old Draft")
}

func TestYearlyGroupsGivenDocuments(t *testing.T) {
	got := yearly([]Document{
		testDocument("Draft", draft, time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)),
		testDocument("Post", post, time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)),
		testDocument("Undated Draft", draft, time.Time{}),
	})

	assert.Equal(t, len(got), 2)
	assert.Equal(t, got[0].Year, 2024)
	assert.Equal(t, got[0].Undated, false)
	assert.Equal(t, len(got[0].Documents.All), 2)
	assert.Equal(t, got[1].Year, 0)
	assert.Equal(t, got[1].Undated, true)
	assert.Equal(t, len(got[1].Documents.All), 1)
}

type testDoc struct {
	meta *Metadata
}

func testDocument(title string, k kind, createdAt time.Time) Document {
	meta := NewMetadata("src/test/"+strings.ToLower(strings.ReplaceAll(title, " ", "-")), filepath.Join("testdata", "templates"))
	meta.CreatedAt = createdAt
	meta.Kind = k
	meta.Title = title
	return testDoc{meta: meta}
}

func (doc testDoc) DependsOn(string) bool {
	return false
}

func (doc testDoc) Load(io.Reader) error {
	return nil
}

func (doc testDoc) Metadata() *Metadata {
	return doc.meta
}

func (doc testDoc) Render(io.Writer) error {
	return nil
}
