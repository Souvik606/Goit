package test

import (
	"fmt"
	"os"
	goit "souvik606/goit/pkg/goit/local"
	"strings"
	"testing"
)

func TestCommitPorcelain(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	fileName := "file.txt"
	fileContent1 := []byte("test content v1")
	err := os.WriteFile(fileName, fileContent1, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	index := goit.NewIndex()
	err = goit.AddPaths([]string{fileName}, index)
	if err != nil {
		t.Fatalf("AddPaths failed: %v", err)
	}
	err = index.Save()
	if err != nil {
		t.Fatalf("SaveIndex failed: %v", err)
	}

	commitMsg1 := "Initial commit"
	commitHash1, refUpdated, err := goit.Commit(commitMsg1)
	if err != nil {
		t.Fatalf("goit.Commit (1) failed: %v", err)
	}

	if len(commitHash1) != 40 {
		t.Errorf("Expected first commit hash to be 40 chars, got %d", len(commitHash1))
	}
	if refUpdated != "refs/heads/main" {
		t.Errorf("Expected ref updated to be 'refs/heads/main', got '%s'", refUpdated)
	}

	headRef, err := goit.GetRefHash("refs/heads/main")
	if err != nil {
		t.Fatalf("Failed to read head ref after commit 1: %v", err)
	}
	if headRef != commitHash1 {
		t.Errorf("HEAD ref content mismatch: got %s, want %s", headRef, commitHash1)
	}

	objType, contentBytes, err := goit.CatFile(commitHash1)
	if err != nil {
		t.Fatalf("CatFile failed for commit 1: %v", err)
	}
	if objType != "commit" {
		t.Errorf("Expected object type 'commit', got '%s'", objType)
	}
	content1 := string(contentBytes)
	if !strings.Contains(content1, "tree ") {
		t.Errorf("Commit 1 content missing tree line")
	}
	if strings.Contains(content1, "parent ") {
		t.Errorf("Commit 1 (initial) should not have a parent line")
	}
	if !strings.Contains(content1, commitMsg1) {
		t.Errorf("Commit 1 content missing commit message")
	}

	fileContent2 := []byte("test content v2")
	err = os.WriteFile(fileName, fileContent2, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file v2: %v", err)
	}

	err = index.Load()
	if err != nil {
		t.Fatalf("Failed to reload index: %v", err)
	}
	err = goit.AddPaths([]string{fileName}, index)
	if err != nil {
		t.Fatalf("AddPaths (2) failed: %v", err)
	}
	err = index.Save()
	if err != nil {
		t.Fatalf("SaveIndex (2) failed: %v", err)
	}

	commitMsg2 := "Second commit"
	commitHash2, _, err := goit.Commit(commitMsg2)
	if err != nil {
		t.Fatalf("goit.Commit (2) failed: %v", err)
	}
	if commitHash1 == commitHash2 {
		t.Errorf("Commit 2 hash is the same as Commit 1 hash")
	}

	headRef, err = goit.GetRefHash("refs/heads/main")
	if err != nil {
		t.Fatalf("Failed to read head ref after commit 2: %v", err)
	}
	if headRef != commitHash2 {
		t.Errorf("HEAD ref content not updated to commit 2: got %s, want %s", headRef, commitHash2)
	}

	objType, contentBytes, err = goit.CatFile(commitHash2)
	if err != nil {
		t.Fatalf("CatFile failed for commit 2: %v", err)
	}
	if objType != "commit" {
		t.Errorf("Expected object type 'commit', got '%s'", objType)
	}
	content2 := string(contentBytes)

	expectedParentLine := fmt.Sprintf("parent %s", commitHash1)
	if !strings.Contains(content2, expectedParentLine) {
		t.Errorf("Commit 2 content missing correct parent line. Expected: '%s'\nGot:\n%s", expectedParentLine, content2)
	}
	if !strings.Contains(content2, commitMsg2) {
		t.Errorf("Commit 2 content missing commit message")
	}
}
