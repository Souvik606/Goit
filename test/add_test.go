package test

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"souvik606/goit/pkg/goit"
	"testing"
)

func TestAddSingleNewFile(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	fileName := "new_file.txt"
	fileContent := []byte("new content")
	err := os.WriteFile(fileName, fileContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	index := goit.NewIndex()

	err = goit.AddPaths([]string{fileName}, index)
	if err != nil {
		t.Fatalf("AddPaths failed: %v", err)
	}

	if len(index.Entries) != 1 {
		t.Fatalf("Expected 1 entry in index, got %d", len(index.Entries))
	}
	entry, ok := index.Entries[fileName]
	if !ok {
		t.Fatalf("File %s not found in index", fileName)
	}

	header := fmt.Sprintf("blob %d\000", len(fileContent))
	fullData := append([]byte(header), fileContent...)
	expectedHash := sha1.Sum(fullData)
	if !bytes.Equal(entry.Hash[:], expectedHash[:]) {
		t.Errorf("Hash mismatch for %s", fileName)
	}
	if entry.Mode != 0100644 {
		t.Errorf("Mode mismatch for %s: got %o", fileName, entry.Mode)
	}

	err = index.Save()
	if err != nil {
		t.Fatalf("Save index failed: %v", err)
	}

	reloadedIndex := goit.NewIndex()
	err = reloadedIndex.Load()
	if err != nil {
		t.Fatalf("Load index failed: %v", err)
	}
	if _, ok := reloadedIndex.Entries[fileName]; !ok {
		t.Errorf("File %s not found in reloaded index", fileName)
	}
}

func TestAddModifiedFile(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	fileName := "mod_file.txt"
	initialContent := []byte("initial")
	modifiedContent := []byte("modified content")

	err := os.WriteFile(fileName, initialContent, 0644)
	if err != nil {
		t.Fatalf("Write initial file failed: %v", err)
	}
	index := goit.NewIndex()
	err = goit.AddPaths([]string{fileName}, index)
	if err != nil {
		t.Fatalf("Initial AddPaths failed: %v", err)
	}
	err = index.Save()
	if err != nil {
		t.Fatalf("Initial Save failed: %v", err)
	}

	headerInitial := fmt.Sprintf("blob %d\000", len(initialContent))
	fullDataInitial := append([]byte(headerInitial), initialContent...)
	initialHash := sha1.Sum(fullDataInitial)

	err = os.WriteFile(fileName, modifiedContent, 0644)
	if err != nil {
		t.Fatalf("Write modified file failed: %v", err)
	}

	err = index.Load()
	if err != nil {
		t.Fatalf("Reload index failed: %v", err)
	}
	err = goit.AddPaths([]string{fileName}, index)
	if err != nil {
		t.Fatalf("Modified AddPaths failed: %v", err)
	}

	entry, ok := index.Entries[fileName]
	if !ok {
		t.Fatalf("File %s not found in index after modify", fileName)
	}

	headerMod := fmt.Sprintf("blob %d\000", len(modifiedContent))
	fullDataMod := append([]byte(headerMod), modifiedContent...)
	modifiedHash := sha1.Sum(fullDataMod)

	if bytes.Equal(entry.Hash[:], initialHash[:]) {
		t.Errorf("Hash was not updated after modification")
	}
	if !bytes.Equal(entry.Hash[:], modifiedHash[:]) {
		t.Errorf("Hash does not match modified content")
	}
}

func TestAddDot(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	os.Mkdir("subdir", 0755)
	os.WriteFile("file1.txt", []byte("content1"), 0644)
	os.WriteFile("subdir/file2.txt", []byte("content2"), 0644)
	os.WriteFile(".hiddenfile", []byte("hidden"), 0644)

	index := goit.NewIndex()
	err := goit.AddPaths([]string{"."}, index)
	if err != nil {
		t.Fatalf("AddPaths . failed: %v", err)
	}

	if len(index.Entries) != 2 {
		t.Errorf("Expected 2 entries in index after 'add .', got %d", len(index.Entries))
		for p := range index.Entries {
			t.Logf("Found entry: %s", p)
		}
	}
	if _, ok := index.Entries["file1.txt"]; !ok {
		t.Errorf("file1.txt not found after 'add .'")
	}
	if _, ok := index.Entries[filepath.ToSlash("subdir/file2.txt")]; !ok {
		t.Errorf("subdir/file2.txt not found after 'add .'")
	}
	if _, ok := index.Entries[".hiddenfile"]; ok {
		t.Errorf(".hiddenfile should have been ignored but was added")
	}
}

func TestAddDeletesFile(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	fileName := "to_delete.txt"

	os.WriteFile(fileName, []byte("delete me"), 0644)
	index := goit.NewIndex()
	goit.AddPaths([]string{fileName}, index)
	index.Save()

	err := os.Remove(fileName)
	if err != nil {
		t.Fatalf("Failed to remove file: %v", err)
	}

	err = index.Load()
	if err != nil {
		t.Fatalf("Reload index failed: %v", err)
	}
	err = goit.AddPaths([]string{"."}, index)
	if err != nil {
		t.Fatalf("AddPaths . after delete failed: %v", err)
	}

	if len(index.Entries) != 0 {
		t.Errorf("Expected index to be empty after deleting file and running 'add .', got %d entries", len(index.Entries))
	}
	if _, ok := index.Entries[fileName]; ok {
		t.Errorf("File %s should have been removed from index but was found", fileName)
	}
}
