package test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	goit "souvik606/goit/pkg/goit/local"
	"strings"
	"testing"
)

func TestWriteTreeEmpty(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	index := goit.NewIndex()
	err := index.Save()
	if err != nil {
		t.Fatalf("Failed to save empty index: %v", err)
	}

	treeHash, err := goit.WriteTree()
	if err != nil {
		t.Fatalf("WriteTree failed for empty index: %v", err)
	}

	expectedEmptyTreeHash := "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
	if treeHash != expectedEmptyTreeHash {
		t.Errorf("Expected empty tree hash %s, but got %s", expectedEmptyTreeHash, treeHash)
	}

	objType, content, err := goit.CatFile(treeHash)
	if err != nil {
		t.Fatalf("CatFile failed for empty tree hash %s: %v", treeHash, err)
	}
	if objType != "tree" {
		t.Errorf("Expected object type 'tree', got '%s'", objType)
	}
	if len(content) != 0 {
		t.Errorf("Expected empty tree content, but got %d bytes", len(content))
	}
}

func TestWriteTreeSingleFile(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	fileName := "file.txt"
	content := []byte("hello")
	os.WriteFile(fileName, content, 0644)
	index := goit.NewIndex()
	goit.AddPaths([]string{fileName}, index)
	index.Save()

	blobData := goit.FormatObject("blob", content)
	blobHash := goit.CalculateHash(blobData)

	treeHash, err := goit.WriteTree()
	if err != nil {
		t.Fatalf("WriteTree failed: %v", err)
	}

	objType, treeContentBytes, err := goit.CatFile(treeHash)
	if err != nil {
		t.Fatalf("CatFile failed for tree hash %s: %v", treeHash, err)
	}
	if objType != "tree" {
		t.Errorf("Expected object type 'tree', got '%s'", objType)
	}

	expectedEntryStart := fmt.Sprintf("100644 %s\000", fileName)
	if !bytes.HasPrefix(treeContentBytes, []byte(expectedEntryStart)) {
		t.Errorf("Tree content prefix mismatch. Got '%s', expected start '%s'", string(treeContentBytes), expectedEntryStart)
	}

	if len(treeContentBytes) != len(expectedEntryStart)+20 {
		t.Errorf("Unexpected tree content length: %d", len(treeContentBytes))
	} else {
		hashBytesFromTree := treeContentBytes[len(expectedEntryStart):]
		blobHashBytes, _ := hex.DecodeString(blobHash)
		if !bytes.Equal(hashBytesFromTree, blobHashBytes) {
			t.Errorf("Blob hash in tree content mismatch. Got %x, expected %x", hashBytesFromTree, blobHashBytes)
		}
	}
}

func TestWriteTreeNested(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	os.Mkdir("dir", 0755)
	os.WriteFile("root.txt", []byte("root"), 0644)
	os.WriteFile("dir/sub.txt", []byte("sub"), 0644)
	index := goit.NewIndex()
	goit.AddPaths([]string{"."}, index)
	index.Save()

	rootTreeHash, err := goit.WriteTree()
	if err != nil {
		t.Fatalf("WriteTree failed: %v", err)
	}

	_, rootContentBytes, err := goit.CatFile(rootTreeHash)
	if err != nil {
		t.Fatalf("CatFile failed for root tree: %v", err)
	}

	entriesStr := string(rootContentBytes)
	if !strings.Contains(entriesStr, "40000 dir\000") {
		t.Errorf("Root tree missing 'dir' entry or wrong mode")
	}
	if !strings.Contains(entriesStr, "100644 root.txt\000") {
		t.Errorf("Root tree missing 'root.txt' entry or wrong mode")
	}
	if strings.Index(entriesStr, "dir") > strings.Index(entriesStr, "root.txt") {
		t.Errorf("Root tree entries not sorted correctly ('dir' should come before 'root.txt')")
	}

	dirEntryPrefix := "40000 dir\000"
	dirEntryIndex := strings.Index(entriesStr, dirEntryPrefix)
	if dirEntryIndex == -1 {
		t.Fatalf("'dir' entry not found in root tree content")
	}
	dirHashBytes := rootContentBytes[dirEntryIndex+len(dirEntryPrefix) : dirEntryIndex+len(dirEntryPrefix)+20]
	dirTreeHash := hex.EncodeToString(dirHashBytes)

	_, dirContentBytes, err := goit.CatFile(dirTreeHash)
	if err != nil {
		t.Fatalf("CatFile failed for dir tree %s: %v", dirTreeHash, err)
	}

	if !strings.Contains(string(dirContentBytes), "100644 sub.txt\000") {
		t.Errorf("'dir' subtree missing 'sub.txt' entry or wrong mode")
	}
}
