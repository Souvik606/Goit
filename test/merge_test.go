package test

import (
	"os"
	"strings"
	"testing"

	goit "souvik606/goit/pkg/goit/local"
)

// Helper function to simulate what cmd/add.go does
func testAdd(t *testing.T, paths ...string) {
	index := goit.NewIndex()
	if err := index.Load(); err != nil {
		t.Fatalf("failed to load index: %v", err)
	}
	if err := goit.AddPaths(paths, index); err != nil {
		t.Fatalf("failed to add paths: %v", err)
	}
	if err := index.Save(); err != nil {
		t.Fatalf("failed to save index: %v", err)
	}
}

// Helper to handle Windows directory locking
func setupTestDir(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}

	tempDir := t.TempDir()
	os.Chdir(tempDir)

	// DEFER: As soon as the test finishes, jump back to the original directory.
	// This releases the Windows file lock so t.TempDir() can safely delete it.
	t.Cleanup(func() {
		os.Chdir(originalWD)
	})
}

func TestMergeFastForward(t *testing.T) {
	setupTestDir(t)
	goit.InitRepository(".", false)

	// 1. Commit A on main
	os.WriteFile("file.txt", []byte("base"), 0644)
	testAdd(t, ".")
	goit.Commit("Initial commit")

	// 2. Create and switch to feature branch
	goit.CreateBranch("feature")
	goit.Checkout("feature")

	// 3. Commit B on feature
	os.WriteFile("file.txt", []byte("base + feature"), 0644)
	testAdd(t, ".")
	targetHash, _, _ := goit.Commit("Feature commit")

	// 4. Switch back to main and merge feature
	goit.Checkout("main")
	requires3Way, _, err := goit.Merge("feature")
	if err != nil {
		t.Fatalf("Fast-forward merge failed: %v", err)
	}

	if requires3Way {
		t.Errorf("Expected fast-forward, but got requires3Way = true")
	}

	// Verify HEAD moved to feature's commit
	headHash, _ := goit.GetHeadCommitHash()
	if headHash != targetHash {
		t.Errorf("Expected HEAD to be %s, got %s", targetHash, headHash)
	}
}

func TestMerge3WayClean(t *testing.T) {
	setupTestDir(t)
	goit.InitRepository(".", false)

	// 1. Base Commit (main)
	os.WriteFile("base.txt", []byte("base file"), 0644)
	testAdd(t, ".")
	goit.Commit("Base commit")

	// 2. Feature Branch (Adds a new file)
	goit.CreateBranch("feature")
	goit.Checkout("feature")
	os.WriteFile("feature.txt", []byte("feature file"), 0644)
	testAdd(t, ".")
	featureHash, _, _ := goit.Commit("Feature commit")

	// 3. Main Branch (Adds a different file)
	goit.Checkout("main")
	os.WriteFile("main.txt", []byte("main file"), 0644)
	testAdd(t, ".")
	headHash, _, _ := goit.Commit("Main commit")

	// 4. Perform 3-Way Merge
	requires3Way, mergeBaseHash, err := goit.Merge("feature")
	if !requires3Way {
		t.Fatalf("Expected 3-way merge, got fast-forward")
	}

	err = goit.Execute3WayMerge(mergeBaseHash, headHash, featureHash, "feature")
	if err != nil {
		t.Fatalf("Expected clean 3-way merge, got error: %v", err)
	}

	// Verify all files exist
	if _, err := os.Stat("feature.txt"); os.IsNotExist(err) {
		t.Errorf("Clean merge failed: feature.txt is missing")
	}
	if _, err := os.Stat("main.txt"); os.IsNotExist(err) {
		t.Errorf("Clean merge failed: main.txt is missing")
	}
}

func TestMerge3WayConflict(t *testing.T) {
	setupTestDir(t)
	goit.InitRepository(".", false)

	// 1. Base Commit (main)
	os.WriteFile("conflict.txt", []byte("line 1\n"), 0644)
	testAdd(t, ".")
	goit.Commit("Base commit")

	// 2. Feature Branch (Modifies line 1)
	goit.CreateBranch("feature")
	goit.Checkout("feature")
	os.WriteFile("conflict.txt", []byte("line 1 from feature\n"), 0644)
	testAdd(t, ".")
	featureHash, _, _ := goit.Commit("Feature commit")

	// 3. Main Branch (Modifies line 1 differently)
	goit.Checkout("main")
	os.WriteFile("conflict.txt", []byte("line 1 from main\n"), 0644)
	testAdd(t, ".")
	headHash, _, _ := goit.Commit("Main commit")

	// 4. Perform 3-Way Merge
	requires3Way, mergeBaseHash, _ := goit.Merge("feature")
	if !requires3Way {
		t.Fatalf("Expected 3-way merge, got fast-forward")
	}

	err := goit.Execute3WayMerge(mergeBaseHash, headHash, featureHash, "feature")

	// We EXPECT an error here because of the conflict
	if err == nil {
		t.Fatalf("Expected conflict error, got nil")
	}
	if !strings.Contains(err.Error(), "Automatic merge failed") {
		t.Errorf("Expected specific conflict error message, got: %v", err)
	}

	// Verify MERGE_HEAD exists
	if _, err := os.Stat(".goit/MERGE_HEAD"); os.IsNotExist(err) {
		t.Errorf("Expected .goit/MERGE_HEAD to be created during conflict, but it is missing")
	}

	// Verify conflict markers are in the file
	content, _ := os.ReadFile("conflict.txt")
	contentStr := string(content)
	if !strings.Contains(contentStr, "<<<<<<< HEAD") || !strings.Contains(contentStr, "=======") || !strings.Contains(contentStr, ">>>>>>> feature") {
		t.Errorf("Conflict markers missing from file. File content:\n%s", contentStr)
	}
}
