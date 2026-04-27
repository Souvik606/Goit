package local

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
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

type ParsedCommit struct {
	TreeHash      string
	ParentHashes  []string
	AuthorLine    string
	CommitterLine string
	Message       string
}

type LogEntry struct {
	Hash   string
	Commit *ParsedCommit
}

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

	config, err := ReadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to read config: %w", err)
	}
	authorName := ""
	authorEmail := ""
	if userSec, ok := config["user"]; ok {
		authorName = userSec["name"]
		authorEmail = userSec["email"]
	}

	if authorName == "" {
		authorName = os.Getenv("GOIT_AUTHOR_NAME")
		if authorName == "" {
			return "", fmt.Errorf("\n*** Please tell me who you are.\n\nRun\n  goit config user.email \"you@example.com\"\n  goit config user.name \"Your Name\"\n\nto set your account's default identity.\nfatal: empty ident name not allowed")
		}
	}

	if authorEmail == "" {
		authorEmail = os.Getenv("GOIT_AUTHOR_EMAIL")
		if authorEmail == "" {
			return "", fmt.Errorf("fatal: empty ident email not allowed")
		}
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
	err = WriteObject(hashStr, fullData)
	if err != nil {
		return "", fmt.Errorf("writing commit object: %w", err)
	}

	return hashStr, nil
}

func Commit(message string) (string, string, error) {
	treeHash, err := WriteTree()
	if err != nil {
		return "", "", fmt.Errorf("failed to write tree: %w", err)
	}

	headRefPath, err := GetHeadRef()
	if err != nil {
		return "", "", err
	}

	parentHash, err := GetRefHash(headRefPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to get parent hash from %s: %w", headRefPath, err)
	}

	var parentHashes []string
	if parentHash != "" {
		parentHashes = append(parentHashes, parentHash)
	}

	mergeHeadPath := filepath.Join(goitDir, "MERGE_HEAD")
	mergeHeadBytes, err := os.ReadFile(mergeHeadPath)
	isMergeCommit := false

	if err == nil {
		mergeHash := strings.TrimSpace(string(mergeHeadBytes))
		if len(mergeHash) == 40 {
			parentHashes = append(parentHashes, mergeHash)
			isMergeCommit = true
		}
	}

	newCommitHash, err := CommitTree(treeHash, parentHashes, message)
	if err != nil {
		return "", "", fmt.Errorf("failed to create commit object: %w", err)
	}

	if err := UpdateRef(headRefPath, newCommitHash); err != nil {
		return "", "", fmt.Errorf("failed to update HEAD ref %s: %w", headRefPath, err)
	}

	if isMergeCommit {
		os.Remove(mergeHeadPath)
	}

	return newCommitHash, headRefPath, nil
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

func ParseCommitObject(content []byte) (*ParsedCommit, error) {
	commit := &ParsedCommit{
		ParentHashes: make([]string, 0),
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	var messageStarted bool
	var message strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		if messageStarted {
			message.WriteString(line)
			message.WriteString("\n")
			continue
		}

		if line == "" {
			messageStarted = true
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			return nil, fmt.Errorf("malformed commit object line: %s", line)
		}
		key, value := parts[0], parts[1]

		switch key {
		case "tree":
			commit.TreeHash = value
		case "parent":
			commit.ParentHashes = append(commit.ParentHashes, value)
		case "author":
			commit.AuthorLine = value
		case "committer":
			commit.CommitterLine = value
		default:

		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning commit object content: %w", err)
	}

	commit.Message = strings.TrimSpace(message.String())
	return commit, nil
}

func Log() ([]LogEntry, error) {
	var history []LogEntry

	headRef, err := GetHeadRef()
	if err != nil {
		return nil, err
	}

	currentHash, err := GetRefHash(headRef)
	if err != nil {
		return nil, err
	}

	for currentHash != "" {
		objType, content, err := CatFile(currentHash)
		if err != nil {
			return nil, fmt.Errorf("failed to read commit object %s: %w", currentHash, err)
		}
		if objType != "commit" {
			return nil, fmt.Errorf("object %s is not a commit, but a %s", currentHash, objType)
		}

		commit, err := ParseCommitObject(content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse commit object %s: %w", currentHash, err)
		}

		history = append(history, LogEntry{
			Hash:   currentHash,
			Commit: commit,
		})

		if len(commit.ParentHashes) > 0 {
			currentHash = commit.ParentHashes[0]
		} else {
			currentHash = ""
		}
	}
	return history, nil
}
