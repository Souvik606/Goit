package test

import (
	"os"
	"souvik606/goit/pkg/goit"
	"strings"
	"testing"
)

func TestLog(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	history, err := goit.Log()
	if err != nil {
		t.Fatalf("Log failed on empty repo: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("Expected 0 commits for empty repo, got %d", len(history))
	}

	fileName := "log_test.txt"
	fileContent1 := []byte("log v1")
	err = os.WriteFile(fileName, fileContent1, 0644)
	if err != nil {
		t.Fatalf("Failed to write log file v1: %v", err)
	}
	index := goit.NewIndex()
	goit.AddPaths([]string{fileName}, index)
	index.Save()
	commitMsg1 := "First log commit"
	commitHash1, _, err := goit.Commit(commitMsg1)
	if err != nil {
		t.Fatalf("Commit 1 failed: %v", err)
	}

	fileContent2 := []byte("log v2")
	err = os.WriteFile(fileName, fileContent2, 0644)
	if err != nil {
		t.Fatalf("Failed to write log file v2: %v", err)
	}
	err = index.Load()
	if err != nil {
		t.Fatalf("Failed to reload index: %v", err)
	}
	goit.AddPaths([]string{fileName}, index)
	index.Save()
	commitMsg2 := "Second log commit"
	commitHash2, _, err := goit.Commit(commitMsg2)
	if err != nil {
		t.Fatalf("Commit 2 failed: %v", err)
	}

	history, err = goit.Log()
	if err != nil {
		t.Fatalf("Log failed on populated repo: %v", err)
	}

	if len(history) != 2 {
		t.Fatalf("Expected 2 commits in history, got %d", len(history))
	}

	if history[0].Hash != commitHash2 {
		t.Errorf("Log[0] hash mismatch: expected %s (latest), got %s", commitHash2, history[0].Hash)
	}
	if !strings.Contains(history[0].Commit.Message, commitMsg2) {
		t.Errorf("Log[0] message mismatch: expected '%s', got '%s'", commitMsg2, history[0].Commit.Message)
	}

	if history[1].Hash != commitHash1 {
		t.Errorf("Log[1] hash mismatch: expected %s (first), got %s", commitHash1, history[1].Hash)
	}
	if !strings.Contains(history[1].Commit.Message, commitMsg1) {
		t.Errorf("Log[1] message mismatch: expected '%s', got '%s'", commitMsg1, history[1].Commit.Message)
	}

	if len(history[0].Commit.ParentHashes) != 1 || history[0].Commit.ParentHashes[0] != commitHash1 {
		t.Errorf("Commit 2 parent hash incorrect: expected %s, got %v", commitHash1, history[0].Commit.ParentHashes)
	}
	if len(history[1].Commit.ParentHashes) != 0 {
		t.Errorf("Commit 1 (initial) should have 0 parents, got %d", len(history[1].Commit.ParentHashes))
	}
}
