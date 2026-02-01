package remote

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"souvik606/goit/pkg/goit/local"
	"strconv"
	"strings"
)

func FindCommitsToSync(wants []string, haves []string) (map[string]bool, error) {
	queue := make([]string, 0, len(wants))
	queue = append(queue, wants...)

	wantsMap := make(map[string]bool)
	for _, hash := range wants {
		wantsMap[hash] = true
	}

	havesMap := make(map[string]bool)
	for _, hash := range haves {
		havesMap[hash] = true
	}

	neededCommits := make(map[string]bool)

	for len(queue) > 0 {
		hash := queue[0]
		queue = queue[1:]

		if _, alreadyNeeded := neededCommits[hash]; alreadyNeeded {
			continue
		}
		if _, weHave := havesMap[hash]; weHave {
			continue
		}

		neededCommits[hash] = true

		objType, content, err := local.CatFile(hash)
		if err != nil {
			return nil, fmt.Errorf("reading commit %s: %w", hash, err)
		}
		if objType != "commit" {
			return nil, fmt.Errorf("object %s is not a commit", hash)
		}

		commit, err := local.ParseCommitObject(content)
		if err != nil {
			return nil, fmt.Errorf("parsing commit %s: %w", hash, err)
		}

		queue = append(queue, commit.ParentHashes...)
	}

	return neededCommits, nil
}

func FindRequiredObjects(commitHashes map[string]bool) (map[string]bool, error) {
	allObjects := make(map[string]bool)
	queue := make([]string, 0, len(commitHashes))

	for hash := range commitHashes {
		allObjects[hash] = true
		queue = append(queue, hash)
	}

	for len(queue) > 0 {
		hash := queue[0]
		queue = queue[1:]

		objType, content, err := local.CatFile(hash)
		if err != nil {
			return nil, fmt.Errorf("reading object %s: %w", hash, err)
		}
		if objType == "commit" {
			commit, err := local.ParseCommitObject(content)
			if err != nil {
				return nil, fmt.Errorf("parsing commit %s: %w", hash, err)
			}
			if _, exists := allObjects[commit.TreeHash]; !exists {
				allObjects[commit.TreeHash] = true
				queue = append(queue, commit.TreeHash)
			}
		} else if objType == "tree" {
			treeEntries, err := ParseTreeObject(content)
			if err != nil {
				return nil, fmt.Errorf("parsing tree %s: %w", hash, err)
			}
			for _, entry := range treeEntries {
				entryHash := hex.EncodeToString(entry.Hash[:])
				if _, exists := allObjects[entryHash]; !exists {
					allObjects[entryHash] = true
					queue = append(queue, entryHash)
				}
			}
		}
	}

	return allObjects, nil
}

func ParseTreeObject(content []byte) ([]local.TreeEntry, error) {
	var entries []local.TreeEntry
	reader := bufio.NewReader(bytes.NewReader(content))

	for {
		modeBytes, err := reader.ReadBytes(' ')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("parsing tree: reading mode: %w", err)
		}
		modeStr := strings.TrimSpace(string(modeBytes))
		mode, _ := strconv.ParseUint(modeStr, 8, 32)

		nameBytes, err := reader.ReadBytes(0)
		if err != nil {
			return nil, fmt.Errorf("parsing tree: reading name: %w", err)
		}
		name := strings.TrimRight(string(nameBytes), "\000")

		var hashBytes [20]byte
		_, err = io.ReadFull(reader, hashBytes[:])
		if err != nil {
			return nil, fmt.Errorf("parsing tree: reading hash: %w", err)
		}

		entries = append(entries, local.TreeEntry{
			Mode: uint32(mode),
			Name: name,
			Hash: hashBytes,
		})
	}
	return entries, nil
}
