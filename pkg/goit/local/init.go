package local

import (
	"fmt"
	"os"
	"path/filepath"
)

const goitDir = ".goit"

func InitRepository(rootPath string, bare bool) error {
	var basePath string
	if bare {
		basePath = rootPath
	} else {
		basePath = filepath.Join(rootPath, goitDir)
	}

	var checkPath string
	if bare {
		checkPath = filepath.Join(basePath, "HEAD")
	} else {
		checkPath = basePath
	}

	if _, err := os.Stat(checkPath); !os.IsNotExist(err) {
		if bare {
			return fmt.Errorf("repository already exists in %s (found HEAD file)", rootPath)
		}
		return fmt.Errorf("repository already exists in %s", basePath)
	}

	if !bare {
		if err := os.Mkdir(basePath, 0755); err != nil {
			return err
		}
	}

	dirs := []string{"objects", "refs/heads", "refs/tags"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(basePath, dir), 0755); err != nil {
			return err
		}
	}

	headPath := filepath.Join(basePath, "HEAD")
	headContent := []byte("ref: refs/heads/main\n")
	if err := os.WriteFile(headPath, headContent, 0644); err != nil {
		return err
	}

	configPath := filepath.Join(basePath, "config")
	configContent := []byte("[core]\n\trepositoryformatversion = 0\n\tfilemode = true\n")
	if bare {
		configContent = append(configContent, []byte("\tbare = true\n")...)
	} else {
		configContent = append(configContent, []byte("\tbare = false\n")...)
	}

	if err := os.WriteFile(configPath, configContent, 0644); err != nil {
		return err
	}

	return nil
}

func IsValidBareRepo(path string) bool {
	if path == "" {
		path = "."
	}

	headPath := filepath.Join(path, "HEAD")
	if _, err := os.Stat(headPath); os.IsNotExist(err) {
		return false
	}
	configPath := filepath.Join(path, "config")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false
	}
	objectsPath := filepath.Join(path, "objects")
	if _, err := os.Stat(objectsPath); os.IsNotExist(err) {
		return false
	}
	return true
}
