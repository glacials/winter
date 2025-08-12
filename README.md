# Winter

[![Go Reference](https://pkg.go.dev/badge/twos.dev/winter.svg)](https://pkg.go.dev/twos.dev/winter)

_Note: Winter is in its alpha stage.
It is not recommended to use Winter for important production content._

Winter is the static website generator for folks who want their content to last decades.
It has three principles:

1. **[Cool URIs don't change](https://www.w3.org/Provider/Style/URI)**.
   Winter remembers the pages you publish,
   and hard-fails if a URL changes.
   Never accidentally break a link again.
2. **Content should outlast generators.**
   Source content and built artifacts are trivial to move to another generator,
   or to none at all.
3. **Special behavior through structure, not symbols.**
   Winter-specific behaviors are based on structure,
   not symbols,
   so degrade gracefully if you change generators.
   For example,
   a figure caption is defined as an italics paragraph following a figure.

Winter has two rules:

1. **Web paths end in `.html`.**
   Migration to as little as a dumb fileserver should be easy,
   even if Winter itself is dead and gone.
   That means no routing layer.
2. **Pages must not be in subdirectories.**
   Shuffling users up and down directory trees creates opportunities for
   invisible link breakages.
   You can organize source trees how you want,
   but built content will always be flat.

Winter enforces these rules itself, so you don't have to remember them.

## Installation

```sh
# CLI
go install twos.dev/winter@latest
winter --help

# Go library
go get -u twos.dev/winter@latest
```

## Documentation

This section details the `winter` CLI documentation. For Go library
documentation, see
[pkg.go.dev/twos.dev/winter](https://pkg.go.dev/twos.dev/winter).

### Directory Structure

- `./src`—Content to be built
  - `./src/cold`—Stable content, optionally [templated](https://pkg.go.dev/text/template)
    - `*.md`—Markdown files
    - `*.html`—HTML files
    - `*.org`—Org mode files
  - `./src/warm`—Unstable content, optionally [templated](https://pkg.go.dev/text/template)
    - `*.md`—Markdown files
    - `*.html`—HTML files
    - `*.org`—Org mode files
  - `./src/templates`—Reusable content
    - `text_document.html.tmpl`—Default page container (from `<html>` to `</html>`)
    - `*.html.tmpl`—HTML templates
  - `./src/img`—Gallery images
    - `...`—Any directory structure
- `./public`—Static files to be copied directly to the build directory without processing
- `./dist`—Build directory

### Commands

Pass `--help` to any command to see detailed usage and behavior.

#### `winter init`

<!-- Don't change manually here. Change `twos.dev/winter/cmd` then paste changes here. -->

Usage: `winter init`

Initialize the current directory for use with Winter. The Winter directory
structure detailed above will be created, and default starting templates will
be populated so that you have a working `index.html` listing posts.

#### `winter build`

<!-- Don't change manually here. Change `twos.dev/winter/cmd` then paste changes here. -->

Usage: `winter build [--serve] [--source directory]`

Build all source content into `dist`.
When `--serve` is passed,
a fileserver is stood up afterwards pointing to `dist`,
content is continually rebuilt as it changes,
and the browser automatically refreshes.

Winter always builds text content from `src/cold` and `src/warm`, gallery
content from `src/img`, and templates from `src/template`. If `--source` is
specified, Winter will also build text content from that file or directory.
`--source` can be specified any number of times.

#### `winter freeze`

<!-- Don't change manually here. Change `twos.dev/winter/cmd` then paste changes here. -->

Usage: `winter freeze shortname...`

Freeze all arguments, specified by shortname. This moves the files from
`src/warm` to `src/cold` and reflects the change in Git.

### Documents

A document is an HTML, Markdown, or Org file with optional frontmatter.
The first level 1 heading
(`<h1>` in HTML, `#` in Markdown, `*` in Org)
will be used as the document title.

Here is an example document called `example.md`:

```markdown
---
date: 2022-07-07
filename: example.html
type: post
---

# My Example Document

This is an example document.
```

### Frontmatter

Frontmatter for HTML and Markdown documents is specified in YAML.
Frontmatter for Org files is specified using Org keywords of
equivalent names (in whatever case you choose). All fields are
optional.

The available frontmatter fields for HTML and Markdown are:

```yaml
filename: example.html
date: 2022-07-07
updated: 2022-11-10
category: arbitrary string
toc: true|false
type: post|page|draft
```

The same fields are available in Org files:

```org
#+FILENAME: example.html
#+DATE: 2022-07-07
#+UPDATED: 2022-11-10
#+CATEGORY: arbitrary string
#+TOC: true|false
#+TYPE: post|page|draft
```

See below for details of each.

#### `category`

The category of the document.
Accepts any string.
This is exposed to templates via the [`{{ .Category }}`](#cat) field.

The default template set provided by `winter init`
gives a minor visual treatment to the listing and display of documents with categories.

Otherwise, the category is semantic.

#### `date`

Date is the publish date of the document written as `YYYY-MM-DD`.
It is available to templates as a Go
[`time.Time`](https://pkg.go.dev/time#Time)
using
[`{{ .CreatedAt }}`](#createdat).

When not set the document will not have a publish date attached to it.

#### `filename`

Filename specifies the desired final location of the built file in `dist`.
This must end in `.html` no matter the source document type and must not be in a subdirectory.
Winter enforces this because if you later move off Winter,
web paths that end in `.html` and do not use subdirectories will be the easiest to migrate.

When not set the filename of the source document is used, with any extensions replaced with `.html`.
For example, `envy.html.tmpl` and `envy.md` would both become `envy.html`
(though if two source files would produce the same destination file, Winter will error).

A document's **web path** is defined as its `filename`.
The web path is accessible to templates using the [`{{ .WebPath }}`](#webpath) template variable.

#### `updated`

Updated is the date the document was last meaningfully updated,
if any,
written as `YYYY-MM-DD`.
It is available to templates as a Go
[`time.Time`](https://pkg.go.dev/time#Time)
using
[`{{ .UpdatedAt }}`](#updatedat).

When not set the document will not have an update date attached to it.

#### `toc`

Whether to render a table of contents (default `false`).
If `true`,
the table of contents will be rendered just before the first level 2 header
(`<h2>` in HTML, `##` in Markdown, `**` in Org)
and will list all level 2, 3, and 4 headers.
See the [top of this page](#toc) for an example.

#### `type`

The kind of document. Possible values are `post`, `page`, `draft`.

`post` documents are programmatically included in template functions.
`page` and `draft` documents have no programmatic action taken on
them and will not be discoverable unless linked to.

The default template set provided by `winter init` gives each a different visual treatment:

- Posts have a publish date and update date at the top and bottom of the page
- Pages have a publish date and update date at the bottom of the page
- Drafts behave like posts but have a "DRAFT" banner at the top of the page

Beyond this, types are only semantic.

### Templates

Any file in `./src` can be templated using the format expressed in the
[`text/template`](https://pkg.go.dev/text/template)
Go library.

#### Document Fields

The following fields are available to templates rendering documents.

##### `{{ .Category }}`

_Type: `string`_

The value specified by the frontmatter [`category`](#category) field.
This is an arbitrary string.

##### `{{ .CreatedAt }}`

_Type: [`time.Time`](https://pkg.go.dev/time#Time)_

The publish date of the document as specified by the frontmatter
[`date`](#date)
attribute.

This value can be formatted using Go's
[`func (time.Time) Format`](https://pkg.go.dev/time#Time.Format)
function:

```template
{{ .CreatedAt.Format "2006 January" }} <!-- Renders 2022 July  -->
{{ .CreatedAt.Format "2006-01-02" }}   <!-- Renders 2022-07-08 -->
```

Use `{{ .CreatedAt.IsZero }}` to see if the date was not set.
You can use this to hide unset dates:

```template
{{ if not .CreatedAt.IsZero }}
  published {{ .CreatedAt.Format "2006 January" }}
{{ end }}
```

##### `{{ .Documents.All }}`

_Type: `[]Document`_

All documents Winter knows about,
ordered by creation date descending.

##### `{{ .IsType "<string>" }}`

_Type: `function(string) bool`_

Returns whether the document is of the given type
(i.e. has frontmatter specifying `type: <string>`).
See [Types](#types) for valid document types.

##### `{{ .Preview }}`

_Type: `string`_

A teaser for the document, such as a summary of its contents.
If unset, one will be inferred from the beginning of the document's content.

##### `{{ .Title }}`

_Type: `string`_

The value of the document's first level 1 heading
(`<h1>` in HTML, `#` in Markdown, `*` in Org).

##### `{{ .UpdatedAt }}`

_Type: [`time.Time`](https://pkg.go.dev/time#Time)_

The date the document was last meaningfully updated,
as specified by the frontmatter
[`updated`](#updated)
attribute.

This value behaves identically to
[`{{ .CreatedAt }}`](#createdat).
For details on dealing with it,
see that documentation.

##### `{{ .WebPath }}`

_Type: `string`_

The path component of the URL that will point to the document.
This is usually the [filename](#filename) frontmatter attribute if set,
or the source document's filename sans extension otherwise,
preceded with a slash and ending in `.html`.

Equivalent to the file's destination location on disk relative to `dist/`.

#### Functions

##### `posts`

Usage: `{{ range posts }} ... {{ end }}`

Returns a list of all documents with type `post`,
from most to least recent.

See [Document Fields](#fields) for a list of fields available to documents.

##### `yearly`

Usage: `{{ range yearly posts }}{{ .Year }}: {{ range .Documents.All }} ... {{ end }}{{ end }}`

Returns a list of `year` types ordered from most to least recent.
A `year` has two fields,
`.Year` (integer)
and
`.Documents` (array of documents).
This allows templates to display posts sectioned by year.

See [Document Fields](#fields) for a list of fields available to documents.
