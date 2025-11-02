package test

import (
	"os"
	"path/filepath"
	goit "souvik606/goit/pkg/goit/local"
	"testing"
)

func TestInitRepository(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	defer os.Chdir(originalWd)

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	err = goit.InitRepository()
	if err != nil {
		t.Fatalf("InitRepository() failed: %v", err)
	}

	basePath := ".goit"
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		t.Errorf(".goit directory was not created in %s", tempDir)
	}

	expectedDirs := []string{"objects", "refs/heads", "refs/tags"}
	for _, dir := range expectedDirs {
		fullPath := filepath.Join(basePath, dir)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected directory %s was not created", fullPath)
		}
	}

	headPath := filepath.Join(basePath, "HEAD")
	content, err := os.ReadFile(headPath)
	if err != nil {
		files, _ := os.ReadDir(".")
		t.Logf("Contents of tempDir (%s) before failing HEAD read:", tempDir)
		for _, f := range files {
			t.Logf("- %s", f.Name())
		}
		filesGoit, _ := os.ReadDir(basePath)
		t.Logf("Contents of .goit (%s) before failing HEAD read:", basePath)
		for _, f := range filesGoit {
			t.Logf("- %s", f.Name())
		}

		t.Fatalf("Could not read HEAD file: %v", err)
	}

	expectedContent := "ref: refs/heads/main\n"
	if string(content) != expectedContent {
		t.Errorf("HEAD content mismatch: got %q, want %q", string(content), expectedContent)
	}

	err = goit.InitRepository()
	if err == nil {
		t.Errorf("Expected error when re-initializing repository, but got nil")
	}
}
