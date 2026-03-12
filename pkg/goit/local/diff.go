package local

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

type EditOperation int

const (
	OpKeep EditOperation = iota
	OpInsert
	OpDelete
)

type DiffLine struct {
	Op   EditOperation
	Text string
}

func DiffWorkspaceIndex() error {
	idx := NewIndex()
	if err := idx.Load(); err != nil {
		return fmt.Errorf("failed to load index: %w", err)
	}

	for path, entry := range idx.Entries {
		workspaceData, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("\033[31mDeleted: %s\033[0m\n", path)
				continue
			}
			return fmt.Errorf("reading workspace file %s: %w", path, err)
		}

		workspaceHash, err := GetBlobHash(path)
		if err != nil {
			return fmt.Errorf("hashing workspace file %s: %w", path, err)
		}

		stagedHashHex := hex.EncodeToString(entry.Hash[:])
		if workspaceHash == stagedHashHex {
			continue
		}

		_, stagedData, err := CatFile(stagedHashHex)
		if err != nil {
			return fmt.Errorf("reading staged object %s: %w", stagedHashHex, err)
		}

		fmt.Printf("\n--- a/%s\n+++ b/%s\n", path, path)

		stagedLines := strings.Split(string(stagedData), "\n")
		workspaceLines := strings.Split(string(workspaceData), "\n")

		diffs := computeLCSDiff(stagedLines, workspaceLines)
		printDiff(diffs)
	}

	return nil
}

func computeLCSDiff(a, b []string) []DiffLine {
	m, n := len(a), len(b)
	L := make([][]int, m+1)
	for i := range L {
		L[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				L[i][j] = L[i-1][j-1] + 1
			} else {
				if L[i-1][j] > L[i][j-1] {
					L[i][j] = L[i-1][j]
				} else {
					L[i][j] = L[i][j-1]
				}
			}
		}
	}

	var diffs []DiffLine
	i, j := m, n
	for i > 0 && j > 0 {
		if a[i-1] == b[j-1] {
			diffs = append(diffs, DiffLine{OpKeep, a[i-1]})
			i--
			j--
		} else if L[i-1][j] > L[i][j-1] {
			diffs = append(diffs, DiffLine{OpDelete, a[i-1]})
			i--
		} else {
			diffs = append(diffs, DiffLine{OpInsert, b[j-1]})
			j--
		}
	}

	for i > 0 {
		diffs = append(diffs, DiffLine{OpDelete, a[i-1]})
		i--
	}

	for j > 0 {
		diffs = append(diffs, DiffLine{OpInsert, b[j-1]})
		j--
	}

	for i, j := 0, len(diffs)-1; i < j; i, j = i+1, j-1 {
		diffs[i], diffs[j] = diffs[j], diffs[i]
	}

	return diffs
}

func printDiff(diffs []DiffLine) {
	colorReset := "\033[0m"
	colorRed := "\033[31m"
	colorGreen := "\033[32m"

	for _, d := range diffs {
		switch d.Op {
		case OpKeep:
			fmt.Printf("  %s\n", d.Text)
		case OpInsert:
			fmt.Printf("%s+ %s%s\n", colorGreen, d.Text, colorReset)
		case OpDelete:
			fmt.Printf("%s- %s%s\n", colorRed, d.Text, colorReset)
		}
	}
}
