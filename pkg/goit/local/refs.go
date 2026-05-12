package local

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	isHexHash = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

func getRefPath(ref string) string {
	if IsValidBareRepo(".") {
		return strings.ReplaceAll(ref, "\\", "/")
	}
	ref = strings.ReplaceAll(ref, "\\", "/")
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
	if strings.HasPrefix(headContent, "ref: ") {
		ref := strings.TrimSpace(strings.TrimPrefix(headContent, "ref: "))
		return strings.ReplaceAll(ref, "\\", "/"), nil
	}

	if isHexHash.MatchString(headContent) {
		return headContent, nil
	}

	return "", fmt.Errorf("invalid HEAD content: %s", headContent)
}

func GetRefHash(refPath string) (string, error) {
	if isHexHash.MatchString(refPath) {
		return refPath, nil
	}

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
	refPath = strings.ReplaceAll(refPath, "\\", "/")
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

func UpdateHead(refPath string, hash string) error {
	var headContent string
	if refPath != "" {
		refPath = strings.ReplaceAll(refPath, "\\", "/")
		headContent = "ref: " + refPath
	} else {
		headContent = hash
	}

	headFilePath := getRefPath("HEAD")
	content := []byte(headContent + "\n")
	if err := os.WriteFile(headFilePath, content, 0644); err != nil {
		return fmt.Errorf("writing HEAD file: %w", err)
	}
	return nil
}

func GetHeadCommitHash() (string, error) {
	headRef, err := GetHeadRef()
	if err != nil {
		return "", err
	}

	headHash, err := GetRefHash(headRef)
	if err != nil {
		return "", err
	}
	if headHash == "" && !isHexHash.MatchString(headRef) {
		return "", nil
	}
	if isHexHash.MatchString(headRef) {
		return headRef, nil
	}

	return headHash, nil
}

func ResolveTarget(target string) (hash string, refPath string, err error) {
	if isHexHash.MatchString(target) {
		return target, "", nil
	}

	if strings.HasPrefix(target, "refs/") {
		fullPath := getRefPath(target)
		if _, err := os.Stat(fullPath); err == nil {
			hash, err := GetRefHash(target)
			return hash, target, err
		}
	}

	refPath = "refs/heads/" + target
	fullPath := getRefPath(refPath)

	if _, err := os.Stat(fullPath); err == nil {
		hash, err := GetRefHash(refPath)
		return hash, refPath, err
	}

	remoteRefPath := "refs/remotes/origin/" + target
	fullRemotePath := getRefPath(remoteRefPath)

	if _, err := os.Stat(fullRemotePath); err == nil {
		hash, _ := GetRefHash(remoteRefPath)
		UpdateRef(refPath, hash)
		return hash, refPath, nil
	}

	return "", "", fmt.Errorf("fatal: '%s' is not a commit and a branch '%s' cannot be created from it", target, target)
}

func ResolveRef(goitDir, refName string) (string, error) {
	refPath := filepath.Join(goitDir, refName)

	data, err := os.ReadFile(refPath)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func UpdateRefRaw(repoRoot, refName, hash string) error {
	refPath := filepath.Join(repoRoot, refName)
	if err := os.MkdirAll(filepath.Dir(refPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(refPath, []byte(hash+"\n"), 0644)
}

func ListBranches() (branches []string, activeBranch string, err error) {
	headsDir := getRefPath("refs/heads")

	branchMap := make(map[string]bool)

	headRef, err := GetHeadRef()
	if err != nil {
		return nil, "", err
	}

	if strings.HasPrefix(headRef, "refs/heads/") {
		activeBranch = strings.TrimPrefix(headRef, "refs/heads/")
		branchMap[activeBranch] = true
	} else if isHexHash.MatchString(headRef) {
		activeBranch = headRef[:7]
	}

	entries, err := os.ReadDir(headsDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, "", fmt.Errorf("reading refs/heads: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			branchMap[entry.Name()] = true
		}
	}

	for branch := range branchMap {
		branches = append(branches, branch)
	}
	sort.Strings(branches)

	return branches, activeBranch, nil
}

func CreateBranch(name string) error {
	branchPath := "refs/heads/" + name
	fullBranchPath := getRefPath(branchPath)

	if _, err := os.Stat(fullBranchPath); err == nil {
		return fmt.Errorf("branch '%s' already exists", name)
	}

	currentCommitHash, err := GetHeadCommitHash()
	if err != nil {
		return fmt.Errorf("getting current commit hash: %w", err)
	}

	if currentCommitHash == "" {
		return fmt.Errorf("cannot create branch: HEAD does not point to a commit (no commits yet?)")
	}

	if err := UpdateRef(branchPath, currentCommitHash); err != nil {
		return fmt.Errorf("creating new branch ref: %w", err)
	}

	return nil
}
