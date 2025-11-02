package local

import (
	"fmt"
	"os"
	"path/filepath"
)

func InitRepository(bare bool) error {
	var basePath string
	if bare {
		basePath = "."
	} else {
		basePath = goitDir
	}

	if _, err := os.Stat(basePath); !os.IsNotExist(err) {
		if basePath == "." {
			return fmt.Errorf("repository already exists in current directory")
		}
		return fmt.Errorf("repository already exists")
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
	if err := os.WriteFile(configPath, []byte{}, 0644); err != nil {
		return err
	}

	return nil
}
