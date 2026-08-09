package document // import "twos.dev/winter/document"

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

const (
	manifestVersion      = 1
	manifestRelativePath = ".winter/manifest.json"
	documentGeneratorKey = "winter-document-v1"
	imageGeneratorKey    = "winter-cwebp-q75-v1"
	staticGeneratorKey   = "winter-static-v1"
)

// buildManifest is persistent build state stored with the published site.
// Keeping it in dist (and therefore on the deployment branch) makes image and
// static-file reuse work on fresh CI runners as well as on a developer's machine.
type buildManifest struct {
	Version int                      `json:"version"`
	Sources map[string]manifestEntry `json:"sources"`
}

type manifestEntry struct {
	Generator string   `json:"generator"`
	Hash      string   `json:"hash"`
	Kind      string   `json:"kind"`
	Orphaned  bool     `json:"orphaned,omitempty"`
	Outputs   []string `json:"outputs"`
}

func newBuildManifest() *buildManifest {
	return &buildManifest{
		Version: manifestVersion,
		Sources: map[string]manifestEntry{},
	}
}

func loadBuildManifest(dist string) (*buildManifest, error) {
	path := filepath.Join(dist, filepath.FromSlash(manifestRelativePath))
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return newBuildManifest(), nil
		}
		return nil, fmt.Errorf("cannot read build manifest %q: %w", path, err)
	}
	var manifest buildManifest
	if err := json.Unmarshal(b, &manifest); err != nil {
		return nil, fmt.Errorf("cannot parse build manifest %q: %w", path, err)
	}
	if manifest.Version != manifestVersion {
		return nil, fmt.Errorf(
			"build manifest %q has version %d; Winter supports version %d",
			path,
			manifest.Version,
			manifestVersion,
		)
	}
	if manifest.Sources == nil {
		manifest.Sources = map[string]manifestEntry{}
	}
	return &manifest, nil
}

func (m *buildManifest) save(dist string) error {
	path := filepath.Join(dist, filepath.FromSlash(manifestRelativePath))
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot encode build manifest: %w", err)
	}
	b = append(b, '\n')
	if err := writeFileIfChanged(path, b, 0o644); err != nil {
		return fmt.Errorf("cannot save build manifest %q: %w", path, err)
	}
	return nil
}

func (m *buildManifest) markAllOrphaned() {
	for source, entry := range m.Sources {
		entry.Orphaned = true
		m.Sources[source] = entry
	}
}

func (m *buildManifest) isFresh(source, hash, generator string, outputs []string, dist string) bool {
	entry, ok := m.Sources[manifestSourcePath(source)]
	if !ok || entry.Hash != hash || entry.Generator != generator {
		return false
	}
	if len(entry.Outputs) != len(outputs) {
		return false
	}
	want := append([]string(nil), outputs...)
	got := append([]string(nil), entry.Outputs...)
	sort.Strings(want)
	sort.Strings(got)
	for i := range want {
		if want[i] != got[i] {
			return false
		}
	}
	for _, output := range outputs {
		info, err := os.Stat(filepath.Join(dist, filepath.FromSlash(output)))
		if err != nil || info.IsDir() {
			return false
		}
	}
	return true
}

func (m *buildManifest) record(source, hash, generator, kind string, outputs []string) {
	sortedOutputs := append([]string(nil), outputs...)
	sort.Strings(sortedOutputs)
	m.Sources[manifestSourcePath(source)] = manifestEntry{
		Generator: generator,
		Hash:      hash,
		Kind:      kind,
		Outputs:   sortedOutputs,
	}
}

func manifestSourcePath(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("cannot open %q to hash it: %w", path, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("cannot hash %q: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func outputPaths(doc Document) []string {
	paths := []string{filepath.ToSlash(filepath.Clean(doc.Metadata().WebPath))}
	if doc.Metadata().GeminiPath != "" {
		paths = append(paths, filepath.ToSlash(filepath.Clean(doc.Metadata().GeminiPath)))
	}
	return paths
}

func imageOutputPaths(im *img) []string {
	paths := []string{filepath.ToSlash(filepath.Clean(im.WebPath))}
	for _, thumb := range im.Thumbnails {
		paths = append(paths, filepath.ToSlash(filepath.Clean(thumb.WebPath)))
	}
	return paths
}
