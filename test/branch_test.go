package test

import (
	"os"
	"path/filepath"
	goit "souvik606/goit/pkg/goit/local"
	"strings"
	"testing"
)

func TestBranchCreateOnEmptyRepo(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	err := goit.CreateBranch("feature")
	if err == nil {
		t.Fatal("Expected error creating branch on empty repo, but got nil")
	}

	if !strings.Contains(err.Error(), "HEAD does not point to a commit") {
		t.Errorf("Expected 'no commit' error, got: %v", err)
	}
}

func TestBranchCreateAndList(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	branches, active, err := goit.ListBranches()
	if err != nil {
		t.Fatalf("ListBranches (initial) failed: %v", err)
	}
	if len(branches) != 1 || branches[0] != "main" {
		t.Errorf("Expected initial list to be ['main'], got %v", branches)
	}
	if active != "main" {
		t.Errorf("Expected active branch to be 'main', got '%s'", active)
	}

	writeFile(t, "file.txt", "content")
	addAll(t)
	commit(t, "Initial commit")

	commitHash1, err := goit.GetRefHash("refs/heads/main")
	if err != nil || commitHash1 == "" {
		t.Fatalf("Failed to get hash of first commit: %v", err)
	}

	newBranchName := "feature-a"
	err = goit.CreateBranch(newBranchName)
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	featureRefPath := filepath.Join(".goit", "refs", "heads", newBranchName)
	if _, err := os.Stat(featureRefPath); os.IsNotExist(err) {
		t.Fatalf("Branch file %s was not created", featureRefPath)
	}

	featureHash, err := goit.GetRefHash(filepath.Join("refs", "heads", newBranchName))
	if err != nil {
		t.Fatalf("Failed to read hash from new branch: %v", err)
	}
	if featureHash != commitHash1 {
		t.Errorf("New branch hash mismatch: got %s, want %s", featureHash, commitHash1)
	}

	branches, active, err = goit.ListBranches()
	if err != nil {
		t.Fatalf("ListBranches (after create) failed: %v", err)
	}
	if len(branches) != 2 {
		t.Errorf("Expected 2 branches, got %d", len(branches))
	}
	if active != "main" {
		t.Errorf("Expected active branch to still be 'main', got '%s'", active)
	}

	err = goit.CreateBranch(newBranchName)
	if err == nil {
		t.Fatal("Expected error creating existing branch, but got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected 'already exists' error, got: %v", err)
	}
}
