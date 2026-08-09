package document

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildManifestPersistsFreshAndOrphanedEntries(t *testing.T) {
	dist := t.TempDir()
	output := filepath.Join(dist, "photo.webp")
	if err := os.WriteFile(output, []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := newBuildManifest()
	manifest.record("src/img/photo.jpg", "abc", imageGeneratorKey, "image", []string{"photo.webp"})
	if !manifest.isFresh("src/img/photo.jpg", "abc", imageGeneratorKey, []string{"photo.webp"}, dist) {
		t.Fatal("newly recorded output was not fresh")
	}
	manifest.markAllOrphaned()
	if err := manifest.save(dist); err != nil {
		t.Fatal(err)
	}

	loaded, err := loadBuildManifest(dist)
	if err != nil {
		t.Fatal(err)
	}
	entry := loaded.Sources["src/img/photo.jpg"]
	if !entry.Orphaned {
		t.Fatal("entry was not persisted as orphaned")
	}
	if !loaded.isFresh("src/img/photo.jpg", "abc", imageGeneratorKey, []string{"photo.webp"}, dist) {
		t.Fatal("a source that reappears with identical output should be reusable")
	}
}

func TestWriteFileIfChangedPreservesModificationTime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "page.html")
	if err := writeFileIfChanged(path, []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Unix(1234, 0)
	if err := os.Chtimes(path, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := writeFileIfChanged(path, []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(oldTime) {
		t.Fatalf("unchanged output was rewritten: got mtime %v, want %v", info.ModTime(), oldTime)
	}
}
