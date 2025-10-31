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

func CalculateHash(data []byte) string {
	hashBytes := sha1.Sum(data)
	return hex.EncodeToString(hashBytes[:])
}

func FormatObject(objType string, content []byte) []byte {
	header := fmt.Sprintf("%s %d\000", objType, len(content))
	return append([]byte(header), content...)
}

func HashObject(filePath string, write bool, objType string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("reading file %s: %w", filePath, err)
	}

	if objType == "" {
		objType = "blob"
	}

	fullData := FormatObject(objType, content)
	hashStr := CalculateHash(fullData)

	if write {
		if err := WriteObject(hashStr, fullData); err != nil {
			return "", err
		}
	}

	return hashStr, nil
}

func WriteObject(hash string, data []byte) error {
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

	fileClosed := false
	defer func() {
		if !fileClosed {
			file.Close()
		}
	}()

	zlibWriter := zlib.NewWriter(file)
	_, writeErr := zlibWriter.Write(data)
	closeErr := zlibWriter.Close()

	if writeErr != nil {
		return fmt.Errorf("compressing and writing object data: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("closing zlib writer: %w", closeErr)
	}

	fileClosed = true
	if err := file.Close(); err != nil {
		return fmt.Errorf("closing object file: %w", err)
	}

	return nil
}

func ReadObject(hash string) ([]byte, error) {
	if len(hash) != 40 {
		return nil, fmt.Errorf("invalid hash length: %d", len(hash))
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

	return decompressedData, nil
}

func ParseObject(rawData []byte) (objType string, content []byte, err error) {
	nullByteIndex := bytes.IndexByte(rawData, 0)
	if nullByteIndex == -1 {
		err = fmt.Errorf("invalid object format: missing null byte separator")
		return "", nil, err
	}

	header := rawData[:nullByteIndex]
	content = rawData[nullByteIndex+1:]

	headerParts := bytes.SplitN(header, []byte(" "), 2)
	if len(headerParts) != 2 {
		err = fmt.Errorf("invalid object format: malformed header '%s'", string(header))
		return "", nil, err
	}

	objType = string(headerParts[0])
	sizeStr := string(headerParts[1])
	expectedSize, parseErr := strconv.Atoi(sizeStr)
	if parseErr != nil {
		err = fmt.Errorf("invalid object format: non-integer size '%s' in header", sizeStr)
		return "", nil, err
	}

	if len(content) != expectedSize {
		err = fmt.Errorf("invalid object format: actual size (%d) does not match header size (%d)", len(content), expectedSize)
		return "", nil, err
	}

	return objType, content, nil
}

func CatFile(hash string) (objType string, content []byte, err error) {
	rawData, err := ReadObject(hash)
	if err != nil {
		return "", nil, err
	}

	objType, content, err = ParseObject(rawData)
	if err != nil {
		return "", nil, err
	}

	return objType, content, nil
}
