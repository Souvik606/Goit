package local

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

func Reset(target string, mode string) error {
	targetCommitHash, _, err := ResolveTarget(target)
	if err != nil {
		return fmt.Errorf("resolving target '%s': %w", target, err)
	}
	if targetCommitHash == "" {
		return fmt.Errorf("target '%s' does not point to a commit", target)
	}

	objType, content, err := CatFile(targetCommitHash)
	if err != nil {
		return fmt.Errorf("reading target commit %s: %w", targetCommitHash, err)
	}
	if objType != "commit" {
		return fmt.Errorf("target object %s is not a commit", targetCommitHash)
	}

	commit, err := ParseCommitObject(content)
	if err != nil {
		return fmt.Errorf("parsing target commit: %w", err)
	}

	headRef, err := GetHeadRef()
	if err != nil {
		return fmt.Errorf("getting HEAD ref: %w", err)
	}

	if strings.HasPrefix(headRef, "refs/heads/") {
		if err := UpdateRef(headRef, targetCommitHash); err != nil {
			return fmt.Errorf("updating branch ref: %w", err)
		}
	} else {
		if err := UpdateHead("", targetCommitHash); err != nil {
			return fmt.Errorf("updating detached HEAD: %w", err)
		}
	}

	if mode == "soft" {
		return nil
	}

	targetTree, err := FlattenTree(commit.TreeHash, "")
	if err != nil {
		return fmt.Errorf("flattening target tree: %w", err)
	}

	switch mode {
	case "mixed":
		if err := resetIndexToTree(targetTree); err != nil {
			return fmt.Errorf("resetting index: %w", err)
		}

	case "hard":
		currentIndex := NewIndex()
		if err := currentIndex.Load(); err != nil {
			return fmt.Errorf("loading current index: %w", err)
		}

		if err := CleanWorkingDirectory(currentIndex.Entries, targetTree); err != nil {
			return fmt.Errorf("cleaning workspace: %w", err)
		}
		if err := RestoreWorkingDirectory(targetTree); err != nil {
			return fmt.Errorf("restoring workspace: %w", err)
		}
		if err := UpdateIndexFromTree(targetTree); err != nil {
			return fmt.Errorf("updating index: %w", err)
		}
	}

	return nil
}

func resetIndexToTree(targetTree map[string]TreeEntryInfo) error {
	newIndex := NewIndex()
	if newIndex.Entries == nil {
		newIndex.Entries = make(map[string]*IndexEntry)
	}

	for path, entry := range targetTree {
		stat, err := os.Stat(path)

		var mode uint32 = entry.Mode
		var mtimeSec, mtimeNano int64
		var size uint64

		if err == nil {
			mtimeSec = stat.ModTime().Unix()
			mtimeNano = int64(stat.ModTime().Nanosecond())
			size = uint64(stat.Size())
		}

		hashBytes, _ := hex.DecodeString(entry.Hash)
		var hash [sha1.Size]byte
		copy(hash[:], hashBytes)

		newIndex.Entries[path] = &IndexEntry{
			Mode:         mode,
			Hash:         hash,
			Path:         path,
			MTimeSeconds: mtimeSec,
			MTimeNanos:   mtimeNano,
			Size:         size,
		}
	}
	return newIndex.Save()
}
