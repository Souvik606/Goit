package local

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Rm(paths []string, cached bool, recursive bool) error {
	index := NewIndex()
	if err := index.Load(); err != nil {
		return fmt.Errorf("loading index: %w", err)
	}

	matchedFiles := make(map[string]bool)

	for _, p := range paths {
		normalizedPath := filepath.ToSlash(filepath.Clean(p))
		foundMatch := false

		if _, exists := index.Entries[normalizedPath]; exists {
			matchedFiles[normalizedPath] = true
			foundMatch = true
		}

		prefix := normalizedPath + "/"
		for idxPath := range index.Entries {
			if strings.HasPrefix(idxPath, prefix) {
				if !recursive {
					return fmt.Errorf("fatal: not removing '%s' recursively without -r", normalizedPath)
				}
				matchedFiles[idxPath] = true
				foundMatch = true
			}
		}

		if !foundMatch {
			return fmt.Errorf("fatal: pathspec '%s' did not match any files", p)
		}
	}

	for file := range matchedFiles {
		delete(index.Entries, file)

		if !cached {
			err := os.Remove(file)
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to physically delete %s: %w", file, err)
			}
		}

		fmt.Printf("rm '%s'\n", file)
	}

	if err := index.Save(); err != nil {
		return fmt.Errorf("saving index after rm: %w", err)
	}

	return nil
}
