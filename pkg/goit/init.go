package goit

import (
	"fmt"
	"os"
	"path/filepath"
)

func InitRepository() error {
	basePath := ".goit"
	dirs := []string{"objects", "refs/heads", "refs/tags"}

	if _, err := os.Stat(basePath); !os.IsNotExist(err) {
		return fmt.Errorf("repository already exists")
	}

	if err := os.Mkdir(basePath, 0755); err != nil {
		return err
	}

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

	return nil
}
