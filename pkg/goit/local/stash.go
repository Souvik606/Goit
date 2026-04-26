package local

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const stashStackFile = "stash_stack"

func Stash() (string, error) {
	status, err := GetStatus()
	if err != nil {
		return "", fmt.Errorf("checking status: %w", err)
	}
	if len(status.Staged) == 0 && len(status.Unstaged) == 0 && len(status.Untracked) == 0 {
		return "No local changes to save", nil
	}

	currentIndex := NewIndex()
	if err := currentIndex.Load(); err != nil {
		return "", fmt.Errorf("loading index: %w", err)
	}

	stashIndex := NewIndex()
	for path, entry := range currentIndex.Entries {
		stashIndex.Entries[path] = entry
	}

	pathsToProcess := make([]string, 0)
	for path := range status.Unstaged {
		pathsToProcess = append(pathsToProcess, path)
	}
	for _, path := range status.Untracked {
		pathsToProcess = append(pathsToProcess, path)
	}

	for _, path := range pathsToProcess {
		hashStr, err := HashObject(path, true, "blob")
		if err != nil {
			return "", fmt.Errorf("hashing file %s: %w", path, err)
		}

		stat, err := os.Stat(path)
		if err != nil {
			return "", err
		}

		hashBytes, err := hex.DecodeString(hashStr)
		if err != nil {
			return "", fmt.Errorf("decoding hash: %w", err)
		}

		var hash [sha1.Size]byte
		copy(hash[:], hashBytes)

		mode := uint32(0100644)
		if stat.Mode()&0111 != 0 {
			mode = 0100755
		}

		stashIndex.AddOrUpdateEntry(path, hash, mode, stat)
	}

	treeStructure := buildTreeStructure(stashIndex.Entries)
	if treeStructure == nil {
		return "", fmt.Errorf("failed to build stash tree structure")
	}
	treeHash, err := writeTreeRecursive(treeStructure)
	if err != nil {
		return "", fmt.Errorf("writing stash tree: %w", err)
	}

	stashMsg := fmt.Sprintf("WIP on %s: %s", time.Now().Format("2006-01-02 15:04:05"), treeHash)
	if err := pushStash(treeHash, stashMsg); err != nil {
		return "", fmt.Errorf("saving to stash stack: %w", err)
	}

	headHash, err := GetHeadCommitHash()
	if err != nil {
		return "", fmt.Errorf("getting HEAD hash: %w", err)
	}

	var headTree map[string]TreeEntryInfo
	if headHash != "" {
		_, content, err := CatFile(headHash)
		if err != nil {
			return "", err
		}
		commit, err := ParseCommitObject(content)
		if err != nil {
			return "", err
		}
		headTree, err = FlattenTree(commit.TreeHash, "")
		if err != nil {
			return "", err
		}
	} else {
		headTree = make(map[string]TreeEntryInfo)
	}

	if err := CleanWorkingDirectory(currentIndex.Entries, headTree); err != nil {
		return "", fmt.Errorf("cleaning workspace: %w", err)
	}
	if err := RestoreWorkingDirectory(headTree); err != nil {
		return "", fmt.Errorf("restoring workspace: %w", err)
	}
	if err := UpdateIndexFromTree(headTree); err != nil {
		return "", fmt.Errorf("resetting index: %w", err)
	}

	return fmt.Sprintf("Saved working directory and index state WIP: %s", treeHash[:7]), nil
}

func StashPop() (string, error) {
	status, err := GetStatus()
	if err != nil {
		return "", err
	}
	if len(status.Staged) > 0 || len(status.Unstaged) > 0 {
		return "", fmt.Errorf("error: Your local changes would be overwritten by stash pop.\nPlease commit or stash them first")
	}

	hash, _, err := popStash()
	if err != nil {
		return "", err
	}
	if hash == "" {
		return "No stash entries found.", nil
	}

	targetTree, err := FlattenTree(hash, "")
	if err != nil {
		return "", fmt.Errorf("reading stashed tree: %w", err)
	}

	if err := RestoreWorkingDirectory(targetTree); err != nil {
		return "", fmt.Errorf("applying stash to workspace: %w", err)
	}

	return fmt.Sprintf("Dropped refs/stash@{0} (%s)\nApplied stash.", hash[:7]), nil
}

func getStashStackPath() string {
	if IsValidBareRepo(".") {
		return stashStackFile
	}
	return filepath.Join(goitDir, stashStackFile)
}

func pushStash(hash string, message string) error {
	path := getStashStackPath()
	var existingContent []byte
	if _, err := os.Stat(path); err == nil {
		existingContent, err = os.ReadFile(path)
		if err != nil {
			return err
		}
	}
	newEntry := fmt.Sprintf("%s %s\n", hash, message)
	finalContent := append([]byte(newEntry), existingContent...)
	return os.WriteFile(path, finalContent, 0644)
}

func popStash() (string, string, error) {
	path := getStashStackPath()
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", nil
		}
		return "", "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return "", "", nil
	}

	firstLine := scanner.Text()
	parts := strings.SplitN(firstLine, " ", 2)
	hash := parts[0]
	msg := ""
	if len(parts) > 1 {
		msg = parts[1]
	}

	var remainingContent strings.Builder
	for scanner.Scan() {
		remainingContent.WriteString(scanner.Text() + "\n")
	}
	file.Close()

	if err := os.WriteFile(path, []byte(remainingContent.String()), 0644); err != nil {
		return "", "", err
	}
	return hash, msg, nil
}
