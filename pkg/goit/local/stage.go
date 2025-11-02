package local

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func AddPaths(paths []string, index *Index) error {
	if len(paths) == 0 {
		return fmt.Errorf("no paths specified for add")
	}

	fullScan := false
	specificPaths := make(map[string]bool)
	for _, p := range paths {
		cleanPath := filepath.Clean(p)
		if cleanPath == "." {
			fullScan = true
			break
		}
		specificPaths[cleanPath] = true
	}

	if fullScan {
		touchedPaths, err := scanAndStageDirectory(".", index)
		if err != nil {
			return fmt.Errorf("scanning working directory: %w", err)
		}
		if err := checkDeletions(index, touchedPaths); err != nil {
			return fmt.Errorf("checking for deletions: %w", err)
		}
	} else {
		touchedPaths := make(map[string]bool)
		for path := range specificPaths {
			stat, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					if _, exists := index.Entries[path]; exists {
						index.RemoveEntry(path)
						touchedPaths[path] = true
					} else {
						fmt.Fprintf(os.Stderr, "warning: pathspec '%s' did not match any files\n", path)
					}
					continue
				}
				return fmt.Errorf("stat path %s: %w", path, err)
			}

			if stat.IsDir() {
				dirTouched, err := scanAndStageDirectory(path, index)
				if err != nil {
					return fmt.Errorf("scanning directory %s: %w", path, err)
				}
				for p := range dirTouched {
					touchedPaths[p] = true
				}
			} else {
				err := stageSingleFile(path, stat, index)
				if err != nil {
					return fmt.Errorf("staging file %s: %w", path, err)
				}
				touchedPaths[path] = true
			}
		}
	}

	return nil
}

func scanAndStageDirectory(dirPath string, index *Index) (map[string]bool, error) {
	touchedPaths := make(map[string]bool)

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accessing path %s: %v\n", path, err)
			if d != nil && d.IsDir() && path != dirPath {
				return filepath.SkipDir
			}
			return nil
		}

		normalizedPath := filepath.ToSlash(path)

		if d.IsDir() && normalizedPath == goitDir {
			return filepath.SkipDir
		}
		if strings.HasPrefix(normalizedPath, goitDir+string(filepath.Separator)) {
			return nil
		}
		baseName := filepath.Base(normalizedPath)
		if normalizedPath != "." && strings.HasPrefix(baseName, ".") && baseName != ".goignore" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !d.IsDir() {
			fileInfo, err := d.Info()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting info for %s: %v\n", path, err)
				return nil
			}
			err = stageSingleFile(normalizedPath, fileInfo, index)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error staging file %s: %v\n", path, err)
				return nil
			}
			touchedPaths[normalizedPath] = true
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking directory %s: %w", dirPath, err)
	}
	return touchedPaths, nil
}

func stageSingleFile(path string, stat os.FileInfo, index *Index) error {
	isMod, calculatedHashBytes, err := isFileModified(path, stat, index)
	if err != nil {
		return err
	}

	if !isMod {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file content for staging %s: %w", path, err)
	}

	var finalHash [sha1.Size]byte
	if len(calculatedHashBytes) == sha1.Size {
		copy(finalHash[:], calculatedHashBytes)
	} else {
		headerForHash := fmt.Sprintf("blob %d\000", len(content))
		dataForHash := append([]byte(headerForHash), content...)
		finalHash = sha1.Sum(dataForHash)
	}
	finalHashStr := hex.EncodeToString(finalHash[:])

	headerForStorage := fmt.Sprintf("blob %d\000", len(content))
	fullDataForStorage := append([]byte(headerForStorage), content...)

	err = WriteObject(finalHashStr, fullDataForStorage)
	if err != nil {
		return fmt.Errorf("writing object for file %s: %w", path, err)
	}

	mode := uint32(0100644)
	if stat.Mode()&0111 != 0 {
		mode = uint32(0100755)
	}

	index.AddOrUpdateEntry(path, finalHash, mode, stat)

	return nil
}

func isFileModified(path string, stat os.FileInfo, index *Index) (bool, []byte, error) {
	entry, exists := index.Entries[path]

	if !exists {
		return true, nil, nil
	}

	if entry.MTimeSeconds == stat.ModTime().Unix() &&
		entry.MTimeNanos == int64(stat.ModTime().Nanosecond()) &&
		entry.Size == uint64(stat.Size()) {
		return false, nil, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return false, nil, fmt.Errorf("reading file content %s: %w", path, err)
	}

	header := fmt.Sprintf("blob %d\000", len(content))
	fullData := append([]byte(header), content...)
	currentHash := sha1.Sum(fullData)

	if !bytes.Equal(entry.Hash[:], currentHash[:]) {
		return true, currentHash[:], nil
	}

	index.AddOrUpdateEntry(path, entry.Hash, entry.Mode, stat)
	return false, nil, nil
}

func checkDeletions(index *Index, touchedPaths map[string]bool) error {
	deletedPaths := []string{}
	for path := range index.Entries {
		if _, wasTouched := touchedPaths[path]; !wasTouched {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				deletedPaths = append(deletedPaths, path)
			}
		}
	}

	for _, path := range deletedPaths {
		index.RemoveEntry(path)
	}
	return nil
}
