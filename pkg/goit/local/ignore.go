package local

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type IgnoreRule struct {
	Pattern string
	IsDir   bool
}

func ReadIgnoreFile() ([]IgnoreRule, error) {
	var rules []IgnoreRule

	file, err := os.Open(".goitignore")
	if err != nil {
		if os.IsNotExist(err) {
			return rules, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	firstLine := true

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if firstLine {
			line = strings.TrimPrefix(line, "\xef\xbb\xbf")
			firstLine = false
		}

		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		isDir := false
		if strings.HasSuffix(line, "/") {
			isDir = true
			line = strings.TrimSuffix(line, "/")
		}

		rules = append(rules, IgnoreRule{
			Pattern: line,
			IsDir:   isDir,
		})
	}
	return rules, nil
}

func IsIgnored(rules []IgnoreRule, path string, isDir bool) bool {
	if path == goitDir || strings.HasPrefix(path, goitDir+"/") {
		return true
	}

	baseName := filepath.Base(path)

	for _, rule := range rules {
		matchedBase, _ := filepath.Match(rule.Pattern, baseName)
		matchedPath, _ := filepath.Match(rule.Pattern, path)
		matchedPrefix := strings.HasPrefix(path, rule.Pattern+"/")

		if matchedBase || matchedPath || matchedPrefix {
			if rule.IsDir && !isDir {
				continue
			}
			return true
		}
	}
	return false
}
