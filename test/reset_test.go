package test

import (
	"os"
	"testing"

	"souvik606/goit/pkg/goit/local"
)

func TestReset(t *testing.T) {
	// 1. Setup Environment
	// Inject environment variables so commits succeed without a config file
	os.Setenv("GOIT_AUTHOR_NAME", "Test Author")
	os.Setenv("GOIT_AUTHOR_EMAIL", "test@example.com")
	defer os.Unsetenv("GOIT_AUTHOR_NAME")
	defer os.Unsetenv("GOIT_AUTHOR_EMAIL")

	// Create an isolated temporary directory for the test repository
	tempDir := t.TempDir()
	cwd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(cwd)

	if err := local.InitRepository(".", false); err != nil {
		t.Fatalf("Failed to initialize goit repo: %v", err)
	}

	fileName := "data.txt"

	// 2. Create Commit 1 (C1)
	os.WriteFile(fileName, []byte("Version 1"), 0644)
	index := local.NewIndex()
	index.Load()
	local.AddPaths([]string{"."}, index)
	index.Save()

	hash1, _, err := local.Commit("Commit 1")
	if err != nil {
		t.Fatalf("Failed to create Commit 1: %v", err)
	}

	// 3. Create Commit 2 (C2)
	os.WriteFile(fileName, []byte("Version 2"), 0644)
	index.Load()
	local.AddPaths([]string{"."}, index)
	index.Save()

	hash2, _, err := local.Commit("Commit 2")
	if err != nil {
		t.Fatalf("Failed to create Commit 2: %v", err)
	}

	// 4. Create an Untracked File (to test Hard Reset safety)
	untrackedName := "untracked.txt"
	os.WriteFile(untrackedName, []byte("I should survive a hard reset"), 0644)

	// ==========================================
	// Subtest 1: SOFT RESET
	// ==========================================
	t.Run("Soft Reset", func(t *testing.T) {
		if err := local.Reset(hash1, "soft"); err != nil {
			t.Fatalf("Soft reset failed: %v", err)
		}

		// Verify HEAD moved back to C1
		headRef, _ := local.GetHeadRef()
		currentHash, _ := local.GetRefHash(headRef)
		if currentHash != hash1 {
			t.Errorf("Soft reset: HEAD did not move to hash1")
		}

		// Verify Workspace is untouched (should still be "Version 2")
		content, _ := os.ReadFile(fileName)
		if string(content) != "Version 2" {
			t.Errorf("Soft reset: workspace was modified! Expected 'Version 2', got '%s'", string(content))
		}

		// Verify Index is untouched (status should show changes to be committed)
		status, _ := local.GetStatus()
		if _, ok := status.Staged[fileName]; !ok {
			t.Errorf("Soft reset: expected %s to be staged, but it was not", fileName)
		}
	})

	// Restore state to C2 for the next test
	local.Reset(hash2, "hard")

	// ==========================================
	// Subtest 2: MIXED RESET
	// ==========================================
	t.Run("Mixed Reset", func(t *testing.T) {
		if err := local.Reset(hash1, "mixed"); err != nil {
			t.Fatalf("Mixed reset failed: %v", err)
		}

		// Verify Workspace is untouched (should still be "Version 2")
		content, _ := os.ReadFile(fileName)
		if string(content) != "Version 2" {
			t.Errorf("Mixed reset: workspace was modified! Expected 'Version 2', got '%s'", string(content))
		}

		// Verify Index was overwritten (status should now show UNSTAGED changes)
		status, _ := local.GetStatus()
		if _, ok := status.Unstaged[fileName]; !ok {
			t.Errorf("Mixed reset: expected %s to be unstaged, but it was not", fileName)
		}
	})

	// Restore state to C2 for the next test
	local.Reset(hash2, "hard")

	// ==========================================
	// Subtest 3: HARD RESET
	// ==========================================
	t.Run("Hard Reset", func(t *testing.T) {
		// Dirty the workspace intentionally
		os.WriteFile(fileName, []byte("Dirty uncommitted work"), 0644)

		if err := local.Reset(hash1, "hard"); err != nil {
			t.Fatalf("Hard reset failed: %v", err)
		}

		// Verify Workspace was ruthlessly overwritten (must be exactly "Version 1")
		content, _ := os.ReadFile(fileName)
		if string(content) != "Version 1" {
			t.Errorf("Hard reset: workspace not reverted! Expected 'Version 1', got '%s'", string(content))
		}

		// Verify the untracked file was completely ignored and still exists
		if _, err := os.Stat(untrackedName); os.IsNotExist(err) {
			t.Errorf("Hard reset: ruthlessly deleted an untracked file!")
		}

		// Verify Index is completely clean
		status, _ := local.GetStatus()
		if len(status.Staged) > 0 || len(status.Unstaged) > 0 {
			t.Errorf("Hard reset: index is not clean. Staged: %d, Unstaged: %d", len(status.Staged), len(status.Unstaged))
		}
	})
}
