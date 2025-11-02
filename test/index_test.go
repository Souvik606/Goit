package test

import (
	"bytes"
	"crypto/sha1"
	"os"
	"path/filepath"
	goit "souvik606/goit/pkg/goit/local"
	"testing"
	"time"
)

type dummyFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (d dummyFileInfo) Name() string       { return d.name }
func (d dummyFileInfo) Size() int64        { return d.size }
func (d dummyFileInfo) Mode() os.FileMode  { return d.mode }
func (d dummyFileInfo) ModTime() time.Time { return d.modTime }
func (d dummyFileInfo) IsDir() bool        { return d.isDir }
func (d dummyFileInfo) Sys() interface{}   { return nil }

func TestSaveLoadIndexEmpty(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	index := goit.NewIndex()
	err := index.Save()
	if err != nil {
		t.Fatalf("Save failed for empty index: %v", err)
	}

	loadedIndex := goit.NewIndex()
	err = loadedIndex.Load()
	if err != nil {
		t.Fatalf("Load failed after saving empty index: %v", err)
	}

	if len(loadedIndex.Entries) != 0 {
		t.Errorf("Expected loaded index to be empty, but got %d entries", len(loadedIndex.Entries))
	}
}

func TestSaveLoadIndexWithEntries(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	index := goit.NewIndex()

	hash1 := sha1.Sum([]byte("content1"))
	stat1 := dummyFileInfo{name: "file1.txt", size: 8, modTime: time.Now(), mode: 0644}
	index.AddOrUpdateEntry("file1.txt", hash1, 0100644, stat1)

	hash2 := sha1.Sum([]byte("content2-longer"))
	stat2 := dummyFileInfo{name: "a/file2.txt", size: 15, modTime: time.Now().Add(-time.Hour), mode: 0755}
	index.AddOrUpdateEntry("a/file2.txt", hash2, 0100755, stat2)

	err := index.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loadedIndex := goit.NewIndex()
	err = loadedIndex.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loadedIndex.Entries) != 2 {
		t.Errorf("Expected loaded index to have 2 entries, got %d", len(loadedIndex.Entries))
	}

	entry1, ok := loadedIndex.Entries["file1.txt"]
	if !ok {
		t.Fatalf("Entry 'file1.txt' not found in loaded index")
	}
	if entry1.Mode != 0100644 {
		t.Errorf("file1.txt: Mode mismatch, got %o, want %o", entry1.Mode, 0100644)
	}
	if !bytes.Equal(entry1.Hash[:], hash1[:]) {
		t.Errorf("file1.txt: Hash mismatch")
	}
	if entry1.Size != uint64(stat1.Size()) {
		t.Errorf("file1.txt: Size mismatch, got %d, want %d", entry1.Size, stat1.Size())
	}
	if entry1.MTimeSeconds != stat1.ModTime().Unix() {
		t.Errorf("file1.txt: MTimeSeconds mismatch")
	}

	entry2, ok := loadedIndex.Entries["a/file2.txt"]
	if !ok {
		t.Fatalf("Entry 'a/file2.txt' not found in loaded index")
	}
	if entry2.Mode != 0100755 {
		t.Errorf("a/file2.txt: Mode mismatch, got %o, want %o", entry2.Mode, 0100755)
	}
	if !bytes.Equal(entry2.Hash[:], hash2[:]) {
		t.Errorf("a/file2.txt: Hash mismatch")
	}
	if entry2.Size != uint64(stat2.Size()) {
		t.Errorf("a/file2.txt: Size mismatch, got %d, want %d", entry2.Size, stat2.Size())
	}
	if entry2.MTimeSeconds != stat2.ModTime().Unix() {
		t.Errorf("a/file2.txt: MTimeSeconds mismatch")
	}

}

func TestLoadIndexCorruptedChecksum(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	index := goit.NewIndex()
	hash1 := sha1.Sum([]byte("data"))
	stat1 := dummyFileInfo{name: "f.txt", size: 4, modTime: time.Now(), mode: 0644}
	index.AddOrUpdateEntry("f.txt", hash1, 0100644, stat1)

	err := index.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	indexPath := filepath.Join(".goit", "index")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Failed to read index file: %v", err)
	}
	if len(content) < sha1.Size {
		t.Fatalf("Index file too short to corrupt checksum")
	}
	content[len(content)-1]++

	err = os.WriteFile(indexPath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write corrupted index file: %v", err)
	}

	corruptedIndex := goit.NewIndex()
	err = corruptedIndex.Load()
	if err == nil {
		t.Errorf("Expected error when loading index with corrupted checksum, but got nil")
	} else if !bytes.Contains([]byte(err.Error()), []byte("checksum mismatch")) {
		t.Errorf("Expected checksum mismatch error, but got: %v", err)
	}
}

func TestLoadIndexCorruptedSignature(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	index := goit.NewIndex()

	err := index.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	indexPath := filepath.Join(".goit", "index")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Failed to read index file: %v", err)
	}
	if len(content) < 7 {
		t.Fatalf("Index file too short to corrupt signature")
	}
	content[0] = 'X'

	err = os.WriteFile(indexPath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write corrupted index file: %v", err)
	}

	corruptedIndex := goit.NewIndex()
	err = corruptedIndex.Load()
	if err == nil {
		t.Errorf("Expected error when loading index with corrupted signature, but got nil")
	} else if !bytes.Contains([]byte(err.Error()), []byte("invalid index signature")) {
		t.Errorf("Expected signature mismatch error, but got: %v", err)
	}
}

func TestLoadNonExistentIndex(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	indexPath := filepath.Join(".goit", "index")
	os.Remove(indexPath)

	index := goit.NewIndex()
	err := index.Load()

	if err != nil {
		t.Fatalf("Load failed for non-existent index: %v", err)
	}
	if len(index.Entries) != 0 {
		t.Errorf("Expected empty index when file doesn't exist, got %d entries", len(index.Entries))
	}
}
