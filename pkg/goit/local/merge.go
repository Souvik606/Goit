package local

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type MergeAction int

const (
	ActionKeep MergeAction = iota
	ActionOverwriteTarget
	ActionDelete
	ActionContentMerge
	ActionTreeConflict
)

func Merge(target string) (bool, string, error) {
	headRefPath, err := GetHeadRef()
	if err != nil {
		return false, "", fmt.Errorf("failed to get HEAD ref: %w", err)
	}
	headCommitHash, err := GetHeadCommitHash()
	if err != nil {
		return false, "", fmt.Errorf("failed to get HEAD commit hash: %w", err)
	}
	if headCommitHash == "" {
		return false, "", fmt.Errorf("cannot merge into an empty repository")
	}

	targetCommitHash, _, err := ResolveTarget(target)
	if err != nil {
		return false, "", fmt.Errorf("resolving target '%s': %w", target, err)
	}

	status, err := GetStatus()
	if err != nil {
		return false, "", fmt.Errorf("failed to check workspace status: %w", err)
	}

	if len(status.Staged) > 0 || len(status.Unstaged) > 0 || len(status.Untracked) > 0 {
		return false, "", fmt.Errorf("Your local changes would be overwritten by merge.\nPlease commit your changes or stash them before you merge.")
	}

	mergeBaseHash, err := findMergeBase(headCommitHash, targetCommitHash)
	if err != nil {
		return false, "", fmt.Errorf("failed to find merge base: %w", err)
	}

	if mergeBaseHash == targetCommitHash {
		fmt.Println("Already up-to-date.")
		return false, mergeBaseHash, nil
	}

	if mergeBaseHash == headCommitHash {
		fmt.Printf("Updating %s..%s\n", headCommitHash[:7], targetCommitHash[:7])
		fmt.Println("Fast-forward")

		err := executeFastForward(headRefPath, targetCommitHash)
		if err != nil {
			return false, "", fmt.Errorf("fast-forward failed: %w", err)
		}
		return false, mergeBaseHash, nil
	}

	fmt.Printf("Merge base found: %s\n", mergeBaseHash[:7])
	return true, mergeBaseHash, nil
}

func executeFastForward(currentRefPath, targetCommitHash string) error {
	objType, content, err := CatFile(targetCommitHash)
	if err != nil || objType != "commit" {
		return fmt.Errorf("failed to read target commit object")
	}
	commit, err := ParseCommitObject(content)
	if err != nil {
		return err
	}
	targetTree, err := FlattenTree(commit.TreeHash, "")
	if err != nil {
		return err
	}

	currentIndex := NewIndex()
	if err := currentIndex.Load(); err != nil {
		return err
	}

	if err := CleanWorkingDirectory(currentIndex.Entries, targetTree); err != nil {
		return err
	}
	if err := RestoreWorkingDirectory(targetTree); err != nil {
		return err
	}
	if err := UpdateIndexFromTree(targetTree); err != nil {
		return err
	}

	if strings.HasPrefix(currentRefPath, "refs/heads/") {
		if err := UpdateRef(currentRefPath, targetCommitHash); err != nil {
			return err
		}
	} else {
		if err := UpdateHead("", targetCommitHash); err != nil {
			return err
		}
	}

	return nil
}

func findMergeBase(hashA, hashB string) (string, error) {
	if hashA == hashB {
		return hashA, nil
	}

	queueA := []string{hashA}
	queueB := []string{hashB}

	visitedA := map[string]bool{hashA: true}
	visitedB := map[string]bool{hashB: true}

	for len(queueA) > 0 || len(queueB) > 0 {
		if len(queueA) > 0 {
			currA := queueA[0]
			queueA = queueA[1:]

			if visitedB[currA] {
				return currA, nil
			}

			parentsA, err := getParents(currA)
			if err != nil {
				return "", err
			}
			for _, p := range parentsA {
				if !visitedA[p] {
					visitedA[p] = true
					queueA = append(queueA, p)
				}
			}
		}

		if len(queueB) > 0 {
			currB := queueB[0]
			queueB = queueB[1:]

			if visitedA[currB] {
				return currB, nil
			}

			parentsB, err := getParents(currB)
			if err != nil {
				return "", err
			}
			for _, p := range parentsB {
				if !visitedB[p] {
					visitedB[p] = true
					queueB = append(queueB, p)
				}
			}
		}
	}

	return "", fmt.Errorf("fatal: refusing to merge unrelated histories")
}

func Execute3WayMerge(baseHash, headHash, targetHash, targetRefName string) error {
	baseTree, err := getTreeFromCommit(baseHash)
	if err != nil {
		return err
	}
	headTree, err := getTreeFromCommit(headHash)
	if err != nil {
		return err
	}
	targetTree, err := getTreeFromCommit(targetHash)
	if err != nil {
		return err
	}

	allPaths := make(map[string]bool)
	for p := range baseTree {
		allPaths[p] = true
	}
	for p := range headTree {
		allPaths[p] = true
	}
	for p := range targetTree {
		allPaths[p] = true
	}

	hasConflicts := false
	currentIndex := NewIndex()
	currentIndex.Load()

	for path := range allPaths {
		baseEntry, hasBase := baseTree[path]
		headEntry, hasHead := headTree[path]
		targetEntry, hasTarget := targetTree[path]

		var bHash, hHash, tHash string
		if hasBase {
			bHash = baseEntry.Hash
		}
		if hasHead {
			hHash = headEntry.Hash
		}
		if hasTarget {
			tHash = targetEntry.Hash
		}

		action := determineMergeAction(bHash, hHash, tHash)

		switch action {
		case ActionKeep:
			continue

		case ActionOverwriteTarget:
			err := restoreFile(path, tHash, targetEntry.Mode)
			if err != nil {
				return err
			}
			stat, _ := os.Stat(path)
			hashBytes, _ := hex.DecodeString(tHash)
			var h [20]byte
			copy(h[:], hashBytes)
			currentIndex.AddOrUpdateEntry(path, h, targetEntry.Mode, stat)

		case ActionDelete:
			os.Remove(path)
			currentIndex.RemoveEntry(path)

		case ActionContentMerge:
			fmt.Printf("Auto-merging %s\n", path)
			conflict, err := performContentMerge(path, bHash, hHash, tHash, targetRefName)
			if err != nil {
				return err
			}
			if conflict {
				fmt.Printf("CONFLICT (content): Merge conflict in %s\n", path)
				hasConflicts = true
			} else {
				stat, _ := os.Stat(path)
				newHashStr, _ := HashObject(path, true, "blob")
				hashBytes, _ := hex.DecodeString(newHashStr)
				var h [20]byte
				copy(h[:], hashBytes)
				currentIndex.AddOrUpdateEntry(path, h, headEntry.Mode, stat)
			}

		case ActionTreeConflict:
			fmt.Printf("CONFLICT (modify/delete): %s\n", path)
			hasConflicts = true
		}
	}

	currentIndex.Save()

	if hasConflicts {
		os.WriteFile(filepath.Join(goitDir, "MERGE_HEAD"), []byte(targetHash+"\n"), 0644)
		return fmt.Errorf("Automatic merge failed; fix conflicts and then commit the result.")
	}

	fmt.Println("Merge completed cleanly. (Auto-commit logic to be added)")
	return nil
}

func determineMergeAction(b, h, t string) MergeAction {
	if h == t {
		return ActionKeep
	}

	if b == "" {
		if h != "" && t == "" {
			return ActionKeep
		}
		if h == "" && t != "" {
			return ActionOverwriteTarget
		}
		return ActionContentMerge
	}

	if b == h && t == "" {
		return ActionDelete
	}
	if b == t && h == "" {
		return ActionDelete
	}

	if (h != b && h != "" && t == "") || (t != b && t != "" && h == "") {
		return ActionTreeConflict
	}

	if b == h && t != b {
		return ActionOverwriteTarget
	}
	if b == t && h != b {
		return ActionKeep
	}
	return ActionContentMerge
}

func performContentMerge(path, bHash, hHash, tHash, targetRefName string) (bool, error) {
	var bLines, hLines, tLines []string

	if bHash != "" {
		bLines = getLines(bHash)
	}
	if hHash != "" {
		hLines = getLines(hHash)
	}
	if tHash != "" {
		tLines = getLines(tHash)
	}

	mergedText, hasConflict := Diff3(bLines, hLines, tLines, "HEAD", targetRefName)

	err := os.WriteFile(path, []byte(mergedText), 0644)
	return hasConflict, err
}

type Hunk struct {
	BaseStart int
	BaseEnd   int
	Lines     []string
}

func Diff3(base, head, target []string, headName, targetName string) (string, bool) {
	headDiff := computeLCSDiff(base, head)
	targetDiff := computeLCSDiff(base, target)

	headHunks := extractHunks(headDiff)
	targetHunks := extractHunks(targetDiff)

	for _, h := range headHunks {
		for _, t := range targetHunks {
			if h.BaseStart <= t.BaseEnd && t.BaseStart <= h.BaseEnd {
				if !hunksIdentical(h, t) {
					return simpleDiff3(base, head, target, headName, targetName)
				}
			}
		}
	}

	var output []string
	hIdx, tIdx := 0, 0
	baseIdx := 0

	for baseIdx < len(base) || hIdx < len(headHunks) || tIdx < len(targetHunks) {
		var hHunk, tHunk *Hunk
		if hIdx < len(headHunks) {
			hHunk = &headHunks[hIdx]
		}
		if tIdx < len(targetHunks) {
			tHunk = &targetHunks[tIdx]
		}

		if hHunk != nil && tHunk != nil && hHunk.BaseStart == baseIdx && tHunk.BaseStart == baseIdx {
			output = append(output, hHunk.Lines...)
			baseIdx = hHunk.BaseEnd
			hIdx++
			tIdx++
			continue
		}

		if hHunk != nil && hHunk.BaseStart == baseIdx {
			output = append(output, hHunk.Lines...)
			baseIdx = hHunk.BaseEnd
			hIdx++
			continue
		}

		if tHunk != nil && tHunk.BaseStart == baseIdx {
			output = append(output, tHunk.Lines...)
			baseIdx = tHunk.BaseEnd
			tIdx++
			continue
		}

		if baseIdx < len(base) {
			output = append(output, base[baseIdx])
			baseIdx++
		}
	}

	return strings.Join(output, "\n"), false
}

func extractHunks(diffs []DiffLine) []Hunk {
	var hunks []Hunk
	baseIdx := 0
	inHunk := false
	var current Hunk

	for _, d := range diffs {
		switch d.Op {
		case OpKeep:
			if inHunk {
				current.BaseEnd = baseIdx
				hunks = append(hunks, current)
				inHunk = false
			}
			baseIdx++
		case OpDelete:
			if !inHunk {
				inHunk = true
				current = Hunk{BaseStart: baseIdx, Lines: []string{}}
			}
			baseIdx++
		case OpInsert:
			if !inHunk {
				inHunk = true
				current = Hunk{BaseStart: baseIdx, Lines: []string{}}
			}
			current.Lines = append(current.Lines, d.Text)
		}
	}
	if inHunk {
		current.BaseEnd = baseIdx
		hunks = append(hunks, current)
	}
	return hunks
}

func hunksIdentical(a, b Hunk) bool {
	if a.BaseStart != b.BaseStart || a.BaseEnd != b.BaseEnd || len(a.Lines) != len(b.Lines) {
		return false
	}
	for i := range a.Lines {
		if a.Lines[i] != b.Lines[i] {
			return false
		}
	}
	return true
}

func simpleDiff3(base, head, target []string, headName, targetName string) (string, bool) {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("<<<<<<< %s\n", headName))
	for _, line := range head {
		buffer.WriteString(line + "\n")
	}
	buffer.WriteString("=======\n")
	for _, line := range target {
		buffer.WriteString(line + "\n")
	}
	buffer.WriteString(fmt.Sprintf(">>>>>>> %s\n", targetName))

	return buffer.String(), true
}

func getParents(commitHash string) ([]string, error) {
	objType, content, err := CatFile(commitHash)
	if err != nil {
		return nil, err
	}
	if objType != "commit" {
		return nil, fmt.Errorf("expected commit object, got %s", objType)
	}
	commit, err := ParseCommitObject(content)
	if err != nil {
		return nil, err
	}
	return commit.ParentHashes, nil
}

func getTreeFromCommit(commitHash string) (map[string]TreeEntryInfo, error) {
	if commitHash == "" {
		return make(map[string]TreeEntryInfo), nil
	}
	objType, content, err := CatFile(commitHash)
	if err != nil || objType != "commit" {
		return nil, fmt.Errorf("invalid commit hash")
	}
	commit, _ := ParseCommitObject(content)
	return FlattenTree(commit.TreeHash, "")
}

func getLines(hash string) []string {
	_, content, err := CatFile(hash)
	if err != nil {
		return []string{}
	}
	return strings.Split(string(content), "\n")
}

func restoreFile(path, hash string, mode uint32) error {
	os.MkdirAll(filepath.Dir(path), 0755)
	_, content, err := CatFile(hash)
	if err != nil {
		return err
	}
	perm := os.FileMode(0644)
	if mode == 0100755 {
		perm = 0755
	}
	return os.WriteFile(path, content, perm)
}
