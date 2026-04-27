package test

import (
	"os"
	"path/filepath"
	"reflect"
	goit "souvik606/goit/pkg/goit/local"
	"strings"
	"testing"
)

func TestReadWriteConfig(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	config, err := goit.ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig (initial) failed: %v", err)
	}

	config["core"]["test"] = "true"
	config["remote \"origin\""] = make(map[string]string)
	config["remote \"origin\""]["url"] = "http://test.com"

	err = config.Save()
	if err != nil {
		t.Fatalf("config.Save failed: %v", err)
	}

	loadedConfig, err := goit.ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig (after save) failed: %v", err)
	}

	if !reflect.DeepEqual(config, loadedConfig) {
		t.Errorf("Loaded config does not match saved config.\nSaved: %v\nLoaded: %v", config, loadedConfig)
	}

	url := loadedConfig["remote \"origin\""]["url"]
	if url != "http://test.com" {
		t.Errorf("Loaded config has incorrect URL: got %s, want http://test.com", url)
	}
}

func TestConfigFindsRepoRoot(t *testing.T) {
	cleanup := setupTestRepo(t)
	defer cleanup()

	subDir := "a/b/c"
	err := os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create sub-directory: %v", err)
	}

	err = os.Chdir(subDir)
	if err != nil {
		t.Fatalf("Failed to cd to sub-directory: %v", err)
	}

	config, err := goit.ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig from sub-directory failed: %v", err)
	}

	if _, ok := config["core"]; !ok {
		t.Errorf("Config read from sub-directory did not find [core] section")
	}
}

func TestReadConfigNotExist(t *testing.T) {
	tempDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}
	defer os.Chdir(originalWd)

	_, err = goit.ReadConfig()
	if err == nil {
		t.Fatalf("Expected ReadConfig to fail in non-repo dir, but it did not")
	}
	if !strings.Contains(err.Error(), "not a goit repository") {
		t.Fatalf("ReadConfig on non-repo dir returned unexpected error: %v", err)
	}
}

func TestGetConfigPathInBareRepo(t *testing.T) {
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

	config, err := goit.ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig failed in bare repo: %v", err)
	}

	config["test"] = make(map[string]string)
	config["test"]["bare"] = "true"
	err = config.Save()
	if err != nil {
		t.Fatalf("config.Save failed in bare repo: %v", err)
	}

	content, err := os.ReadFile("config")
	if err != nil {
		t.Fatalf("Failed to read root 'config' file in bare repo: %v", err)
	}

	if !strings.Contains(string(content), "[test]") {
		t.Errorf("Bare repo config file content is incorrect")
	}
}
