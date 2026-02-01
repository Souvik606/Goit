package test

import (
	"os"
	"path/filepath"
	goit "souvik606/goit/pkg/goit/local"
	"strings"
	"testing"
)

func TestInitRepository(t *testing.T) {
	tempDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	repoPath := filepath.Join(tempDir, "test-repo")
	os.Mkdir(repoPath, 0755)

	if err := os.Chdir(repoPath); err != nil {
		t.Fatalf("Failed to change to repo dir: %v", err)
	}
	defer os.Chdir(originalWd)

	err = goit.InitRepository(repoPath, false)
	if err != nil {
		t.Fatalf("InitRepository(false) failed: %v", err)
	}

	basePath := ".goit"
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		t.Errorf(".goit directory was not created")
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
		t.Fatalf("Could not read HEAD file: %v", err)
	}

	expectedContent := "ref: refs/heads/main\n"
	if string(content) != expectedContent {
		t.Errorf("HEAD content mismatch: got %q, want %q", string(content), expectedContent)
	}

	configPath := filepath.Join(basePath, "config")
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file was not created at %s", configPath)
	}
	if !strings.Contains(string(configContent), "bare = false") {
		t.Errorf("config file missing 'bare = false'")
	}

	err = goit.InitRepository(repoPath, false)
	if err == nil {
		t.Errorf("Expected error when re-initializing repository, but got nil")
	}
}

func TestInitBareRepository(t *testing.T) {
	tempDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	bareRepoPath := filepath.Join(tempDir, "bare-repo")
	os.Mkdir(bareRepoPath, 0755)

	if err := os.Chdir(bareRepoPath); err != nil {
		t.Fatalf("Failed to change to bare repo dir: %v", err)
	}
	defer os.Chdir(originalWd)

	err = goit.InitRepository(bareRepoPath, true)
	if err != nil {
		t.Fatalf("InitRepository(true) failed: %v", err)
	}

	if _, err := os.Stat(".goit"); err == nil {
		t.Errorf(".goit directory was created, but should not have been in bare repo")
	}

	expectedRootFiles := []string{"HEAD", "config"}
	for _, file := range expectedRootFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Expected root file %s was not created in bare repo", file)
		}
	}

	expectedDirs := []string{"objects", "refs/heads", "refs/tags"}
	for _, dir := range expectedDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Expected directory %s was not created in bare repo", dir)
		}
	}

	configPath := "config"
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file was not created at %s", configPath)
	}
	if !strings.Contains(string(configContent), "bare = true") {
		t.Errorf("config file missing 'bare = true'")
	}
}
