package goit

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

const goitDir = ".goit"
const objectsDir = "objects"

func HashObject(filePath string, write bool) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("reading file %s: %w", filePath, err)
	}

	header := fmt.Sprintf("blob %d\000", len(content))
	fullData := append([]byte(header), content...)

	hash := sha1.Sum(fullData)
	hashStr := hex.EncodeToString(hash[:])

	if write {
		if err := writeObject(hashStr, fullData); err != nil {
			return "", err
		}
	}

	return hashStr, nil
}

func writeObject(hash string, data []byte) error {
	objectDir := filepath.Join(goitDir, objectsDir, hash[:2])
	objectPath := filepath.Join(objectDir, hash[2:])

	if _, err := os.Stat(objectPath); err == nil {
		return nil
	}

	if err := os.MkdirAll(objectDir, 0755); err != nil {
		return fmt.Errorf("creating object directory %s: %w", objectDir, err)
	}

	file, err := os.Create(objectPath)
	if err != nil {
		return fmt.Errorf("creating object file %s: %w", objectPath, err)
	}
	defer file.Close()

	zlibWriter := zlib.NewWriter(file)
	defer zlibWriter.Close()

	if _, err := zlibWriter.Write(data); err != nil {
		return fmt.Errorf("compressing and writing object data: %w", err)
	}

	return nil
}

func CatFile(hash string) ([]byte, error) {
	if len(hash) != 40 {
		return nil, fmt.Errorf("invalid hash length")
	}

	objectPath := filepath.Join(goitDir, objectsDir, hash[:2], hash[2:])

	file, err := os.Open(objectPath)
	if err != nil {
		return nil, fmt.Errorf("opening object file %s: %w", objectPath, err)
	}
	defer file.Close()

	zlibReader, err := zlib.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("creating zlib reader: %w", err)
	}
	defer zlibReader.Close()

	decompressedData, err := io.ReadAll(zlibReader)
	if err != nil {
		return nil, fmt.Errorf("decompressing object data: %w", err)
	}

	nullByteIndex := bytes.IndexByte(decompressedData, 0)
	if nullByteIndex == -1 {
		return nil, fmt.Errorf("invalid object format: missing null byte separator")
	}

	headerParts := bytes.SplitN(decompressedData[:nullByteIndex], []byte(" "), 2)
	if len(headerParts) != 2 {
		return nil, fmt.Errorf("invalid object format: malformed header")
	}

	objType := string(headerParts[0])
	sizeStr := string(headerParts[1])
	expectedSize, err := strconv.Atoi(sizeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid object format: non-integer size in header")
	}

	content := decompressedData[nullByteIndex+1:]
	if len(content) != expectedSize {
		return nil, fmt.Errorf("invalid object format: actual size (%d) does not match header size (%d)", len(content), expectedSize)
	}

	if objType != "blob" {
		return nil, fmt.Errorf("unsupported object type: %s", objType)
	}

	return content, nil
}
