package goit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func getRefPath(ref string) string {
	return filepath.Join(goitDir, ref)
}

func GetHeadRef() (string, error) {
	headPath := getRefPath("HEAD")
	content, err := os.ReadFile(headPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("HEAD file not found, repository may be corrupt or uninitialized")
		}
		return "", fmt.Errorf("reading HEAD: %w", err)
	}

	headContent := strings.TrimSpace(string(content))
	if after, ok := strings.CutPrefix(headContent, "ref: "); ok {
		return strings.TrimSpace(after), nil
	}

	if len(headContent) == 40 {
		return headContent, nil
	}

	return "", fmt.Errorf("invalid HEAD content: %s", headContent)
}

func GetRefHash(refPath string) (string, error) {
	fullPath := getRefPath(refPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("reading ref file %s: %w", fullPath, err)
	}
	return strings.TrimSpace(string(content)), nil
}

func UpdateRef(refPath string, hash string) error {
	fullPath := getRefPath(refPath)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("creating directories for ref %s: %w", refPath, err)
	}

	content := []byte(hash + "\n")
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return fmt.Errorf("writing ref file %s: %w", fullPath, err)
	}

	return nil
}
