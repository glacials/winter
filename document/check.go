package document // import "twos.dev/winter/document"

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/net/html"
)

var cssURLPattern = regexp.MustCompile(`(?i)url\(\s*['"]?([^'"\)]+)['"]?\s*\)`)

// CheckSite verifies that local links and resources in a built site resolve.
// It intentionally checks generated output, including hand-maintained legacy
// pages, rather than only checking Winter's source graph.
func CheckSite(dist string) error {
	dist = filepath.Clean(dist)
	pages := map[string]map[string]struct{}{}
	var files []string
	if err := filepath.WalkDir(dist, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dist, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		files = append(files, rel)
		if strings.EqualFold(filepath.Ext(rel), ".html") || strings.EqualFold(filepath.Ext(rel), ".htm") {
			ids, err := collectHTMLIDs(path)
			if err != nil {
				return fmt.Errorf("cannot parse %q: %w", rel, err)
			}
			pages[rel] = ids
		}
		return nil
	}); err != nil {
		return fmt.Errorf("cannot inspect site at %q: %w", dist, err)
	}

	var problems []string
	for _, rel := range files {
		path := filepath.Join(dist, filepath.FromSlash(rel))
		switch strings.ToLower(filepath.Ext(rel)) {
		case ".html", ".htm":
			refs, err := collectHTMLReferences(path)
			if err != nil {
				return fmt.Errorf("cannot parse %q: %w", rel, err)
			}
			for _, ref := range refs {
				if problem := checkReference(dist, rel, ref, pages); problem != "" {
					problems = append(problems, problem)
				}
			}
		case ".css":
			refs, err := collectCSSReferences(path)
			if err != nil {
				return fmt.Errorf("cannot parse %q: %w", rel, err)
			}
			for _, ref := range refs {
				if problem := checkReference(dist, rel, ref, pages); problem != "" {
					problems = append(problems, problem)
				}
			}
		}
	}

	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	problems = compactStrings(problems)
	return fmt.Errorf("site check found %d broken local reference(s):\n- %s", len(problems), strings.Join(problems, "\n- "))
}

type siteReference struct {
	Attribute string
	Raw       string
}

func collectHTMLIDs(path string) (map[string]struct{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	root, err := html.Parse(f)
	if err != nil {
		return nil, err
	}
	ids := map[string]struct{}{}
	walkHTML(root, func(node *html.Node) {
		if node.Type != html.ElementNode {
			return
		}
		for _, attr := range node.Attr {
			if attr.Key == "id" || (node.Data == "a" && attr.Key == "name") {
				ids[attr.Val] = struct{}{}
			}
		}
	})
	return ids, nil
}

func collectHTMLReferences(path string) ([]siteReference, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	root, err := html.Parse(f)
	if err != nil {
		return nil, err
	}
	var refs []siteReference
	walkHTML(root, func(node *html.Node) {
		if node.Type != html.ElementNode {
			return
		}
		for _, attr := range node.Attr {
			if htmlReferenceAttribute(node.Data, attr.Key) {
				refs = append(refs, siteReference{Attribute: node.Data + "[" + attr.Key + "]", Raw: attr.Val})
			}
			if attr.Key == "srcset" && (node.Data == "img" || node.Data == "source") {
				for _, candidate := range parseSrcset(attr.Val) {
					refs = append(refs, siteReference{Attribute: node.Data + "[srcset]", Raw: candidate})
				}
			}
		}
	})
	return refs, nil
}

func htmlReferenceAttribute(element, attribute string) bool {
	switch element + ":" + attribute {
	case "a:href", "area:href", "audio:src", "embed:src", "form:action",
		"iframe:src", "img:src", "input:src", "link:href", "object:data",
		"script:src", "source:src", "track:src", "video:poster", "video:src":
		return true
	default:
		return false
	}
}

func parseSrcset(value string) []string {
	if strings.HasPrefix(strings.TrimSpace(value), "data:") {
		return nil
	}
	var refs []string
	for _, candidate := range strings.Split(value, ",") {
		fields := strings.Fields(strings.TrimSpace(candidate))
		if len(fields) > 0 {
			refs = append(refs, fields[0])
		}
	}
	return refs
}

func collectCSSReferences(path string) ([]siteReference, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var refs []siteReference
	scanner := bufio.NewScanner(f)
	// CSS can contain generated data on very long lines.
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		for _, match := range cssURLPattern.FindAllStringSubmatch(scanner.Text(), -1) {
			refs = append(refs, siteReference{Attribute: "css url()", Raw: strings.TrimSpace(match[1])})
		}
	}
	return refs, scanner.Err()
}

func checkReference(dist, source string, ref siteReference, pages map[string]map[string]struct{}) string {
	raw := strings.TrimSpace(ref.Raw)
	if raw == "" || strings.HasPrefix(raw, "//") {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Sprintf("%s: %s has invalid URL %q: %v", source, ref.Attribute, raw, err)
	}
	if u.Scheme != "" || u.Host != "" {
		return ""
	}
	decodedPath, err := url.PathUnescape(u.Path)
	if err != nil {
		return fmt.Sprintf("%s: %s has invalid escaped path %q: %v", source, ref.Attribute, raw, err)
	}
	target := source
	if decodedPath != "" {
		if strings.HasPrefix(decodedPath, "/") {
			target = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(decodedPath)), "/")
		} else {
			target = filepath.ToSlash(filepath.Clean(filepath.Join(filepath.Dir(source), filepath.FromSlash(decodedPath))))
		}
	}
	if target == "." || target == "" {
		target = "index.html"
	}
	if target == ".." || strings.HasPrefix(target, "../") {
		return fmt.Sprintf("%s: %s escapes the web root: %q", source, ref.Attribute, raw)
	}

	resolved := target
	info, statErr := os.Stat(filepath.Join(dist, filepath.FromSlash(resolved)))
	if statErr == nil && info.IsDir() {
		resolved = filepath.ToSlash(filepath.Join(resolved, "index.html"))
		_, statErr = os.Stat(filepath.Join(dist, filepath.FromSlash(resolved)))
	}
	// GitHub Pages serves /name from name.html. Keep checking this documented
	// Winter compatibility behavior for hyperlinks, but never for resources.
	if errors.Is(statErr, os.ErrNotExist) && (ref.Attribute == "a[href]" || ref.Attribute == "area[href]") && filepath.Ext(resolved) == "" {
		if _, htmlErr := os.Stat(filepath.Join(dist, filepath.FromSlash(resolved+".html"))); htmlErr == nil {
			resolved += ".html"
			statErr = nil
		}
	}
	if statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return fmt.Sprintf("%s: %s %q resolves to missing %q", source, ref.Attribute, raw, target)
		}
		return fmt.Sprintf("%s: cannot inspect %s target %q: %v", source, ref.Attribute, raw, statErr)
	}
	if u.Fragment != "" {
		ids, ok := pages[resolved]
		if ok {
			fragment, err := url.PathUnescape(u.Fragment)
			if err != nil {
				return fmt.Sprintf("%s: %s has invalid fragment %q: %v", source, ref.Attribute, u.Fragment, err)
			}
			if _, ok := ids[fragment]; !ok {
				return fmt.Sprintf("%s: %s %q points to missing fragment #%s in %q", source, ref.Attribute, raw, fragment, resolved)
			}
		}
	}
	return ""
}

func walkHTML(node *html.Node, visit func(*html.Node)) {
	visit(node)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		walkHTML(child, visit)
	}
}

func compactStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}
