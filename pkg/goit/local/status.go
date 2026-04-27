package local

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type StatusChangeType int

const (
	ChangeNone StatusChangeType = iota
	ChangeStagedNew
	ChangeStagedModified
	ChangeStagedDeleted
	ChangeModified
	ChangeDeleted
)

type FileStatus struct {
	Path     string
	Staged   StatusChangeType
	Unstaged StatusChangeType
}

type StatusSummary struct {
	Staged    map[string]StatusChangeType
	Unstaged  map[string]StatusChangeType
	Untracked []string
}

type TreeEntryInfo struct {
	Mode uint32
	Hash string
}

func GetStatus() (*StatusSummary, error) {
	summary := &StatusSummary{
		Staged:    make(map[string]StatusChangeType),
		Unstaged:  make(map[string]StatusChangeType),
		Untracked: make([]string, 0),
	}

	ignoreRules, _ := ReadIgnoreFile()

	headMap, err := loadHeadTreeAsMap()
	if err != nil {
		return nil, fmt.Errorf("loading HEAD tree: %w", err)
	}

	index := NewIndex()
	if err := index.Load(); err != nil {
		return nil, fmt.Errorf("loading index: %w", err)
	}
	indexMap := index.Entries

	workDirMap, err := scanWorkingDirectory(ignoreRules)
	if err != nil {
		return nil, fmt.Errorf("scanning working directory: %w", err)
	}

	allPaths := make(map[string]bool)
	for path := range headMap {
		allPaths[path] = true
	}
	for path := range indexMap {
		allPaths[path] = true
	}
	for path := range workDirMap {
		allPaths[path] = true
	}

	for path := range allPaths {
		headEntry, inHead := headMap[path]
		indexEntry, inIndex := indexMap[path]
		workDirHash, inWorkDir := workDirMap[path]

		if !inHead && !inIndex && inWorkDir {
			summary.Untracked = append(summary.Untracked, path)
			continue
		}

		indexHash := ""
		if inIndex {
			indexHash = hex.EncodeToString(indexEntry.Hash[:])
		}

		stagedChange := ChangeNone
		unstagedChange := ChangeNone

		if inHead && !inIndex {
			stagedChange = ChangeStagedDeleted
		} else if !inHead && inIndex {
			stagedChange = ChangeStagedNew
		} else if inHead && inIndex && headEntry.Hash != indexHash {
			stagedChange = ChangeStagedModified
		}

		if inIndex && !inWorkDir {
			unstagedChange = ChangeDeleted
		} else if !inIndex && inWorkDir {
		} else if inIndex && inWorkDir && indexHash != workDirHash {
			unstagedChange = ChangeModified
		}

		if stagedChange != ChangeNone {
			summary.Staged[path] = stagedChange
		}
		if unstagedChange != ChangeNone {
			summary.Unstaged[path] = unstagedChange
		}
	}

	return summary, nil
}

func loadHeadTreeAsMap() (map[string]TreeEntryInfo, error) {
	headHash, err := GetHeadCommitHash()
	if err != nil {
		return nil, err
	}
	if headHash == "" {
		return make(map[string]TreeEntryInfo), nil
	}

	objType, content, err := CatFile(headHash)
	if err != nil {
		return nil, err
	}
	if objType != "commit" {
		return nil, fmt.Errorf("HEAD points to a non-commit object: %s", objType)
	}

	commit, err := ParseCommitObject(content)
	if err != nil {
		return nil, err
	}

	return FlattenTree(commit.TreeHash, "")
}

func FlattenTree(treeHash string, prefix string) (map[string]TreeEntryInfo, error) {
	fileMap := make(map[string]TreeEntryInfo)
	if treeHash == "" {
		return fileMap, nil
	}

	objType, content, err := CatFile(treeHash)
	if err != nil {
		return nil, fmt.Errorf("reading tree object %s: %w", treeHash, err)
	}
	if objType != "tree" {
		return nil, fmt.Errorf("object %s is not a tree (got %s)", treeHash, objType)
	}

	reader := bufio.NewReader(bytes.NewReader(content))
	for {
		modeBytes, err := reader.ReadBytes(' ')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("parsing tree %s: reading mode: %w", treeHash, err)
		}
		modeStr := strings.TrimSpace(string(modeBytes))
		mode, _ := strconv.ParseUint(modeStr, 8, 32)

		nameBytes, err := reader.ReadBytes(0)
		if err != nil {
			return nil, fmt.Errorf("parsing tree %s: reading name: %w", treeHash, err)
		}
		name := strings.TrimRight(string(nameBytes), "\000")

		var hashBytes [sha1.Size]byte
		_, err = io.ReadFull(reader, hashBytes[:])
		if err != nil {
			if err == io.ErrUnexpectedEOF || err == io.EOF {
				return nil, fmt.Errorf("parsing tree %s: unexpected end of file while reading hash for %s", treeHash, name)
			}
			return nil, fmt.Errorf("parsing tree %s: reading hash: %w", treeHash, err)
		}
		hashHex := hex.EncodeToString(hashBytes[:])

		fullPath := prefix + name

		isDir := (mode & 0040000) != 0
		if isDir {
			subMap, err := FlattenTree(hashHex, fullPath+"/")
			if err != nil {
				return nil, err
			}
			for k, v := range subMap {
				fileMap[k] = v
			}
		} else {
			fileMap[fullPath] = TreeEntryInfo{
				Mode: uint32(mode),
				Hash: hashHex,
			}
		}
	}

	return fileMap, nil
}

func scanWorkingDirectory(ignoreRules []IgnoreRule) (map[string]string, error) {
	workDirMap := make(map[string]string)

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		normalizedPath := filepath.ToSlash(path)

		if normalizedPath == "." {
			return nil
		}

		if IsIgnored(ignoreRules, normalizedPath, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() && normalizedPath == goitDir {
			return filepath.SkipDir
		}
		if strings.HasPrefix(normalizedPath, goitDir+"/") && normalizedPath != goitDir {
			return nil
		}
		baseName := filepath.Base(normalizedPath)
		if strings.HasPrefix(baseName, ".") && baseName != ".goitignore" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !d.IsDir() {
			hash, err := GetBlobHash(path)
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return fmt.Errorf("hashing file %s: %w", path, err)
			}
			workDirMap[normalizedPath] = hash
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return workDirMap, nil
}
