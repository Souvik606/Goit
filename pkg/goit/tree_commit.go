package goit

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

type TreeEntry struct {
	Mode uint32
	Hash [20]byte
	Name string
}

type ByName []TreeEntry

func (a ByName) Len() int           { return len(a) }
func (a ByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

func buildTreeStructure(entries map[string]*IndexEntry) map[string]interface{} {
	root := make(map[string]interface{})

	for path, entry := range entries {
		parts := strings.Split(path, "/")
		currentLevel := root

		for i, part := range parts {
			isLastPart := (i == len(parts)-1)

			if isLastPart {
				currentLevel[part] = entry
			} else {
				if _, ok := currentLevel[part]; !ok {
					currentLevel[part] = make(map[string]interface{})
				}

				if nextLevel, ok := currentLevel[part].(map[string]interface{}); ok {
					currentLevel = nextLevel
				} else {
					fmt.Fprintf(os.Stderr, "Error: Path conflict detected at %s in %s\n", part, path)
					return nil
				}
			}
		}
	}
	return root
}

func writeTreeRecursive(node map[string]interface{}) (string, error) {
	var entries []TreeEntry

	for name, item := range node {
		var entry TreeEntry
		entry.Name = name

		if indexEntry, ok := item.(*IndexEntry); ok {
			entry.Mode = indexEntry.Mode
			copy(entry.Hash[:], indexEntry.Hash[:])
		} else if subTree, ok := item.(map[string]interface{}); ok {
			subTreeHashStr, err := writeTreeRecursive(subTree)
			if err != nil {
				return "", err
			}
			subTreeHashBytes, err := hex.DecodeString(subTreeHashStr)
			if err != nil {
				return "", fmt.Errorf("invalid hex string from subtree hash %s: %w", subTreeHashStr, err)
			}
			entry.Mode = 040000
			copy(entry.Hash[:], subTreeHashBytes)
		} else {
			return "", fmt.Errorf("unexpected item type in tree structure for name %s", name)
		}
		entries = append(entries, entry)
	}
	sort.Sort(ByName(entries))

	var content bytes.Buffer
	for _, entry := range entries {
		fmt.Fprintf(&content, "%o %s\000", entry.Mode, entry.Name)
		content.Write(entry.Hash[:])
	}

	contentBytes := content.Bytes()
	fullData := FormatObject("tree", contentBytes)
	hashStr := CalculateHash(fullData)
	err := WriteObject(hashStr, fullData)

	if err != nil {
		return "", fmt.Errorf("writing tree object: %w", err)
	}

	return hashStr, nil
}

func WriteTree() (string, error) {
	index := NewIndex()
	if err := index.Load(); err != nil {
		return "", fmt.Errorf("loading index for write-tree: %w", err)
	}

	if len(index.Entries) == 0 {
		fullData := FormatObject("tree", []byte{})
		hashStr := CalculateHash(fullData)
		err := WriteObject(hashStr, fullData)
		if err != nil {
			return "", fmt.Errorf("writing empty tree object: %w", err)
		}
		return hashStr, nil
	}

	treeStructure := buildTreeStructure(index.Entries)
	if treeStructure == nil {
		return "", fmt.Errorf("failed to build internal tree structure")
	}

	rootHash, err := writeTreeRecursive(treeStructure)
	if err != nil {
		return "", err
	}

	return rootHash, nil
}

func CommitTree(treeHash string, parentHashes []string, message string) (string, error) {
	if len(treeHash) != 40 {
		return "", fmt.Errorf("invalid tree hash provided")
	}

	for _, p := range parentHashes {
		if len(p) != 40 {
			return "", fmt.Errorf("invalid parent hash provided: %s", p)
		}
	}

	if message == "" {
		return "", fmt.Errorf("commit message cannot be empty")
	}

	//TODO: To be replaced latter with real logic
	authorName := os.Getenv("GOIT_AUTHOR_NAME")
	if authorName == "" {
		authorName = "Default Author"
	}
	authorEmail := os.Getenv("GOIT_AUTHOR_EMAIL")
	if authorEmail == "" {
		authorEmail = "author@example.com"
	}

	committerName := os.Getenv("GOIT_COMMITTER_NAME")
	if committerName == "" {
		committerName = authorName
	}
	committerEmail := os.Getenv("GOIT_COMMITTER_EMAIL")
	if committerEmail == "" {
		committerEmail = authorEmail
	}

	now := time.Now()
	timestamp := now.Unix()
	_, offsetSeconds := now.Zone()
	timezoneOffset := fmt.Sprintf("%+03d%02d", offsetSeconds/3600, (offsetSeconds%3600)/60)

	var content bytes.Buffer
	fmt.Fprintf(&content, "tree %s\n", treeHash)
	for _, p := range parentHashes {
		fmt.Fprintf(&content, "parent %s\n", p)
	}
	fmt.Fprintf(&content, "author %s <%s> %d %s\n", authorName, authorEmail, timestamp, timezoneOffset)
	fmt.Fprintf(&content, "committer %s <%s> %d %s\n", committerName, committerEmail, timestamp, timezoneOffset)
	fmt.Fprintf(&content, "\n%s\n", message)

	if !strings.HasSuffix(message, "\n") {
		content.WriteString("\n")
	}

	fullData := FormatObject("commit", content.Bytes())
	hashStr := CalculateHash(fullData)
	err := WriteObject(hashStr, fullData)
	if err != nil {
		return "", fmt.Errorf("writing commit object: %w", err)
	}

	return hashStr, nil
}

func ReadMessageFromStdin() (string, error) {
	fmt.Println("Enter commit message (end with Ctrl+D or Ctrl+Z on Windows):")
	scanner := bufio.NewScanner(os.Stdin)
	var message strings.Builder
	for scanner.Scan() {
		message.WriteString(scanner.Text())
		message.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading commit message from stdin: %w", err)
	}
	msg := strings.TrimSpace(message.String())
	if msg == "" {
		return "", fmt.Errorf("aborting commit due to empty commit message")
	}
	return msg, nil
}
