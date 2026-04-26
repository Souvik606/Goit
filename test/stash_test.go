package test

import (
	"os"
	goit "souvik606/goit/pkg/goit/local"
	"strings"
	"testing"
)

func TestStashAndPop(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	fileName := "stash_test.txt"
	initialContent := "Version 1 - Committed"
	modifiedContent := "Version 2 - Uncommitted changes"

	// 1. Create a baseline commit (so we have a HEAD to revert to)
	err := os.WriteFile(fileName, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write initial file: %v", err)
	}
	index := goit.NewIndex()
	goit.AddPaths([]string{fileName}, index)
	index.Save()
	_, _, err = goit.Commit("Initial commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// 2. Modify the file in the workspace (do not stage it)
	err = os.WriteFile(fileName, []byte(modifiedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// 3. Run Stash
	msg, err := goit.Stash()
	if err != nil {
		t.Fatalf("Stash failed: %v", err)
	}
	if !strings.Contains(msg, "Saved working directory") {
		t.Errorf("Unexpected stash success message: %s", msg)
	}

	// 4. Verify the workspace was cleaned (reverted to Version 1)
	currentContent, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatalf("Failed to read file after stash: %v", err)
	}
	if string(currentContent) != initialContent {
		t.Errorf("Workspace not cleaned after stash.\nExpected: '%s'\nGot: '%s'", initialContent, string(currentContent))
	}

	// 5. Run Stash Pop
	popMsg, err := goit.StashPop()
	if err != nil {
		t.Fatalf("StashPop failed: %v", err)
	}
	if !strings.Contains(popMsg, "Dropped") {
		t.Errorf("Unexpected stash pop message: %s", popMsg)
	}

	// 6. Verify the workspace has the uncommitted changes back (Version 2)
	restoredContent, err := os.ReadFile(fileName)
	if err != nil {
		t.Fatalf("Failed to read file after pop: %v", err)
	}
	if string(restoredContent) != modifiedContent {
		t.Errorf("Workspace not restored after pop.\nExpected: '%s'\nGot: '%s'", modifiedContent, string(restoredContent))
	}
}
