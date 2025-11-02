package goit

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Checkout(target string) (string, error) {
	currentHeadRef, err := GetHeadRef()
	if err != nil {
		return "", fmt.Errorf("getting current HEAD ref: %w", err)
	}

	targetCommitHash, targetRefPath, err := ResolveTarget(target)
	if err != nil {
		return "", err
	}
	if targetCommitHash == "" {
		return "", fmt.Errorf("branch '%s' does not point to a commit (no commits yet?)", target)
	}

	isBranch := (targetRefPath != "")

	if isBranch && currentHeadRef == targetRefPath {
		return fmt.Sprintf("Already on '%s'", target), nil
	}
	if !isBranch && currentHeadRef == targetCommitHash {
		return fmt.Sprintf("HEAD is already at %s", targetCommitHash[:7]), nil
	}

	statusSummary, err := GetStatus()
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}

	objType, content, err := CatFile(targetCommitHash)
	if err != nil {
		return "", fmt.Errorf("target commit %s not found: %w", targetCommitHash, err)
	}
	if objType != "commit" {
		return "", fmt.Errorf("target object %s is not a commit: %w", targetCommitHash, err)
	}

	commit, err := ParseCommitObject(content)
	if err != nil {
		return "", fmt.Errorf("parsing target commit %s: %w", targetCommitHash, err)
	}

	targetTree, err := FlattenTree(commit.TreeHash, "")
	if err != nil {
		return "", fmt.Errorf("flattening target tree %s: %w", commit.TreeHash, err)
	}

	var conflicts []string
	if len(statusSummary.Staged) > 0 {
		conflicts = mapKeys(statusSummary.Staged)
	}
	if len(statusSummary.Unstaged) > 0 {
		conflicts = append(conflicts, mapKeys(statusSummary.Unstaged)...)
	}

	for _, untrackedFile := range statusSummary.Untracked {
		if _, existsInTarget := targetTree[untrackedFile]; existsInTarget {
			conflicts = append(conflicts, "\t"+untrackedFile)
		}
	}

	if len(conflicts) > 0 {
		sort.Strings(conflicts)
		return "", fmt.Errorf("error: Your local changes to the following files would be overwritten by checkout:\n%s\nPlease commit your changes or stash them before you switch branches",
			strings.Join(conflicts, "\n"))
	}

	currentIndex := NewIndex()
	if err := currentIndex.Load(); err != nil {
		return "", fmt.Errorf("loading current index: %w", err)
	}

	if err := CleanWorkingDirectory(currentIndex.Entries, targetTree); err != nil {
		return "", fmt.Errorf("cleaning working directory: %w", err)
	}

	if err := RestoreWorkingDirectory(targetTree); err != nil {
		return "", fmt.Errorf("restoring working directory: %w", err)
	}

	if err := UpdateIndexFromTree(targetTree); err != nil {
		return "", fmt.Errorf("updating index from tree: %w", err)
	}

	if err := UpdateHead(targetRefPath, targetCommitHash); err != nil {
		return "", fmt.Errorf("updating HEAD: %w", err)
	}

	if isBranch {
		return fmt.Sprintf("Switched to branch '%s'", target), nil
	} else {
		return fmt.Sprintf("Note: checking out '%s'. You are in 'detached HEAD' state.\nHEAD is now at %s", target, targetCommitHash[:7]), nil
	}
}

func CleanWorkingDirectory(currentIndexEntries map[string]*IndexEntry, targetTree map[string]TreeEntryInfo) error {
	allKnownpaths := make(map[string]bool)

	for path := range currentIndexEntries {
		allKnownpaths[path] = true
	}

	for path := range targetTree {
		allKnownpaths[path] = true
	}

	for path := range allKnownpaths {
		_, inTarget := targetTree[path]
		if _, inIndex := currentIndexEntries[path]; inIndex && !inTarget {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("removing file %s: %w", path, err)
			}
		}
	}
	return nil
}

func RestoreWorkingDirectory(targetTree map[string]TreeEntryInfo) error {
	for path, entry := range targetTree {
		dir := filepath.Dir(path)
		if dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("creating directory %s: %w", dir, err)
			}
		}

		_, content, err := CatFile(entry.Hash)
		if err != nil {
			return fmt.Errorf("getting content for object %s (path %s): %w", entry.Hash, path, err)
		}

		perm := os.FileMode(0644)
		if entry.Mode == 0100755 {
			perm = 0755
		}

		if err := os.WriteFile(path, content, perm); err != nil {
			return fmt.Errorf("writing file %s: %w", path, err)
		}
	}
	return nil
}

func UpdateIndexFromTree(targetTree map[string]TreeEntryInfo) error {
	newIndex := NewIndex()

	for path, entry := range targetTree {
		stat, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("stat file %s for index update: %w", path, err)
		}

		hashBytes, err := hex.DecodeString(entry.Hash)
		if err != nil {
			return fmt.Errorf("decoding hash string %s: %w", entry.Hash, err)
		}
		var hash [sha1.Size]byte
		copy(hash[:], hashBytes)

		newIndex.AddOrUpdateEntry(path, hash, entry.Mode, stat)
	}

	if err := newIndex.Save(); err != nil {
		return fmt.Errorf("saving new index: %w", err)
	}
	return nil
}

func mapKeys(m map[string]StatusChangeType) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, "\t"+k)
	}
	return keys
}
