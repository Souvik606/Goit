package test

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	goit "souvik606/goit/pkg/goit/local"

	"io"
	"os"
	"path/filepath"
	"testing"
)

func setupTestRepo(t *testing.T) (cleanupFunc func()) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	if err := goit.InitRepository(false); err != nil {
		t.Fatalf("Failed to initialize test repository: %v", err)
	}

	return func() {
		os.Chdir(originalWd)
	}
}

func TestHashObjectWrite(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	fileName := "test.txt"
	fileContent := []byte("hello goit test content")
	err := os.WriteFile(fileName, fileContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	header := fmt.Sprintf("blob %d\000", len(fileContent))
	fullData := append([]byte(header), fileContent...)
	expectedHashBytes := sha1.Sum(fullData)
	expectedHash := hex.EncodeToString(expectedHashBytes[:])

	hash, err := goit.HashObject(fileName, true, "blob")
	if err != nil {
		t.Fatalf("HashObject failed: %v", err)
	}

	if hash != expectedHash {
		t.Errorf("Hash mismatch: got %s, want %s", hash, expectedHash)
	}

	objectPath := filepath.Join(".goit", "objects", hash[:2], hash[2:])
	if _, err := os.Stat(objectPath); os.IsNotExist(err) {
		t.Errorf("Object file %s was not created", objectPath)
		return
	}

	file, err := os.Open(objectPath)
	if err != nil {
		t.Fatalf("Could not open object file %s: %v", objectPath, err)
	}
	defer file.Close()

	zr, err := zlib.NewReader(file)
	if err != nil {
		t.Fatalf("Could not create zlib reader for %s: %v", objectPath, err)
	}
	defer zr.Close()

	readData, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("Could not decompress data from %s: %v", objectPath, err)
	}

	if !bytes.Equal(readData, fullData) {
		t.Errorf("Decompressed object data does not match original data with header")
	}
}

func TestCatFile(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	fileName := "cat_test.txt"
	fileContent := []byte("content for cat-file")
	err := os.WriteFile(fileName, fileContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	hash, err := goit.HashObject(fileName, true, "blob")
	if err != nil {
		t.Fatalf("HashObject failed during setup for CatFile: %v", err)
	}

	objType, retrievedContent, err := goit.CatFile(hash)
	if err != nil {
		t.Fatalf("CatFile failed for hash %s: %v", hash, err)
	}

	if objType != "blob" {
		t.Errorf("Expected object type 'blob', got '%s'", objType)
	}

	if !bytes.Equal(retrievedContent, fileContent) {
		t.Errorf("CatFile content mismatch: got %q, want %q", string(retrievedContent), string(fileContent))
	}
}

func TestCatFileInvalidHash(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	_, _, err := goit.CatFile("invalidhash")
	if err == nil {
		t.Error("Expected error for invalid hash length, but got nil")
	}

	nonExistentHash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	_, _, err = goit.CatFile(nonExistentHash)
	if err == nil {
		t.Errorf("Expected error for non-existent hash %s, but got nil", nonExistentHash)
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected a file not found error (os.ErrNotExist) for non-existent hash, but got: %v", err)
	}
}
