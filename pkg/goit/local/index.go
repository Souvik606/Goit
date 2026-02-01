package local

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

const indexSignature = "GOITIDX"
const indexVersion = 1

type IndexHeader struct {
	Signature  [7]byte
	Version    uint32
	EntryCount uint32
}

type IndexEntry struct {
	Mode         uint32
	Hash         [sha1.Size]byte
	Path         string
	MTimeSeconds int64
	MTimeNanos   int64
	Size         uint64
}

type Index struct {
	Entries map[string]*IndexEntry
}

func NewIndex() *Index {
	return &Index{
		Entries: make(map[string]*IndexEntry),
	}
}

func getIndexPath() string {
	if IsValidBareRepo(".") {
		return "index"
	}
	return filepath.Join(goitDir, "index")
}

func (idx *Index) Load() error {
	indexPath := getIndexPath()
	if idx.Entries == nil {
		idx.Entries = make(map[string]*IndexEntry)
	}

	file, err := os.Open(indexPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("opening index file %s: %w", indexPath, err)
	}
	defer file.Close()

	headerSize := binary.Size(IndexHeader{})
	headerBytes := make([]byte, headerSize)
	_, err = io.ReadFull(file, headerBytes)
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		stat, _ := file.Stat()
		if stat != nil && stat.Size() == 0 {
			return nil
		}
		return fmt.Errorf("index file too small to contain header")
	}
	if err != nil {
		return fmt.Errorf("reading index header bytes: %w", err)
	}

	headerReader := bytes.NewReader(headerBytes)
	var header IndexHeader
	if err := binary.Read(headerReader, binary.BigEndian, &header); err != nil {
		return fmt.Errorf("parsing index header: %w", err)
	}

	if string(header.Signature[:]) != indexSignature {
		return fmt.Errorf("invalid index signature: expected %s, got %s", indexSignature, string(header.Signature[:]))
	}
	if header.Version != indexVersion {
		return fmt.Errorf("unsupported index version: expected %d, got %d", indexVersion, header.Version)
	}

	_, err = file.Seek(int64(headerSize), io.SeekStart)
	if err != nil {
		return fmt.Errorf("seeking past header in index file: %w", err)
	}

	remainingContent, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("reading index entries and checksum: %w", err)
	}

	if len(remainingContent) < sha1.Size {
		if header.EntryCount > 0 || (header.EntryCount == 0 && len(remainingContent) != sha1.Size) {
			return fmt.Errorf("index file is corrupted or incomplete after header")
		}
	}

	var contentToChecksum []byte
	var expectedChecksumBytes []byte
	if len(remainingContent) >= sha1.Size {
		contentToChecksum = append(headerBytes, remainingContent[:len(remainingContent)-sha1.Size]...)
		expectedChecksumBytes = remainingContent[len(remainingContent)-sha1.Size:]
	} else if len(remainingContent) == 0 && header.EntryCount == 0 {
		return nil
	} else {
		return fmt.Errorf("index file content too short for checksum validation")
	}

	actualChecksumBytes := sha1.Sum(contentToChecksum)

	if !bytes.Equal(expectedChecksumBytes, actualChecksumBytes[:]) {
		return fmt.Errorf("index checksum mismatch")
	}

	entriesReader := bytes.NewReader(remainingContent[:len(remainingContent)-sha1.Size])
	idx.Entries = make(map[string]*IndexEntry, header.EntryCount)

	for i := 0; i < int(header.EntryCount); i++ {
		entry := &IndexEntry{}

		if err := binary.Read(entriesReader, binary.BigEndian, &entry.Mode); err != nil {
			return fmt.Errorf("reading entry %d mode: %w", i, err)
		}
		if _, err := io.ReadFull(entriesReader, entry.Hash[:]); err != nil {
			return fmt.Errorf("reading entry %d hash: %w", i, err)
		}
		if err := binary.Read(entriesReader, binary.BigEndian, &entry.MTimeSeconds); err != nil {
			return fmt.Errorf("reading entry %d mtime_sec: %w", i, err)
		}
		if err := binary.Read(entriesReader, binary.BigEndian, &entry.MTimeNanos); err != nil {
			return fmt.Errorf("reading entry %d mtime_nano: %w", i, err)
		}
		if err := binary.Read(entriesReader, binary.BigEndian, &entry.Size); err != nil {
			return fmt.Errorf("reading entry %d size: %w", i, err)
		}

		pathBytes := []byte{}
		for {
			var b [1]byte
			if _, err := entriesReader.Read(b[:]); err != nil {
				if err == io.EOF {
					return fmt.Errorf("unexpected end of file while reading entry %d path", i)
				}
				return fmt.Errorf("reading entry %d path byte: %w", i, err)
			}
			if b[0] == 0 {
				break
			}
			pathBytes = append(pathBytes, b[0])
		}
		entry.Path = string(pathBytes)

		idx.Entries[entry.Path] = entry
	}

	if entriesReader.Len() > 0 {
		return fmt.Errorf("unexpected trailing data in index file before checksum")
	}

	return nil
}

func (idx *Index) Save() error {
	indexPath := getIndexPath()
	buffer := new(bytes.Buffer)

	paths := make([]string, 0, len(idx.Entries))
	for path := range idx.Entries {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	header := IndexHeader{
		Version:    indexVersion,
		EntryCount: uint32(len(paths)),
	}
	copy(header.Signature[:], indexSignature)
	if err := binary.Write(buffer, binary.BigEndian, &header); err != nil {
		return fmt.Errorf("writing index header: %w", err)
	}

	for _, path := range paths {
		entry := idx.Entries[path]

		if err := binary.Write(buffer, binary.BigEndian, entry.Mode); err != nil {
			return fmt.Errorf("writing entry mode for %s: %w", path, err)
		}
		if _, err := buffer.Write(entry.Hash[:]); err != nil {
			return fmt.Errorf("writing entry hash for %s: %w", path, err)
		}
		if err := binary.Write(buffer, binary.BigEndian, entry.MTimeSeconds); err != nil {
			return fmt.Errorf("writing entry mtime_sec for %s: %w", path, err)
		}
		if err := binary.Write(buffer, binary.BigEndian, entry.MTimeNanos); err != nil {
			return fmt.Errorf("writing entry mtime_nano for %s: %w", path, err)
		}
		if err := binary.Write(buffer, binary.BigEndian, entry.Size); err != nil {
			return fmt.Errorf("writing entry size for %s: %w", path, err)
		}

		if _, err := buffer.WriteString(entry.Path); err != nil {
			return fmt.Errorf("writing entry path for %s: %w", path, err)
		}
		if err := buffer.WriteByte(0); err != nil {
			return fmt.Errorf("writing entry path null terminator for %s: %w", path, err)
		}
	}

	contentBytes := buffer.Bytes()
	checksum := sha1.Sum(contentBytes)

	if _, err := buffer.Write(checksum[:]); err != nil {
		return fmt.Errorf("writing index checksum: %w", err)
	}

	tempFile, err := os.CreateTemp(filepath.Dir(indexPath), filepath.Base(indexPath)+".tmp")
	if err != nil {
		return fmt.Errorf("creating temporary index file: %w", err)
	}
	tempName := tempFile.Name()
	renameSuccessful := false
	defer func() {
		if !renameSuccessful {
			os.Remove(tempName)
		}
	}()

	if _, err := tempFile.Write(buffer.Bytes()); err != nil {
		tempFile.Close()
		return fmt.Errorf("writing to temporary index file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("closing temporary index file: %w", err)
	}

	if err := os.Rename(tempName, indexPath); err != nil {
		return fmt.Errorf("renaming temporary index file to %s: %w", indexPath, err)
	}
	renameSuccessful = true

	return nil
}

func (idx *Index) AddOrUpdateEntry(path string, hash [sha1.Size]byte, mode uint32, stat os.FileInfo) {
	entry := &IndexEntry{
		Mode:         mode,
		Hash:         hash,
		Path:         path,
		MTimeSeconds: stat.ModTime().Unix(),
		MTimeNanos:   int64(stat.ModTime().Nanosecond()),
		Size:         uint64(stat.Size()),
	}
	if idx.Entries == nil {
		idx.Entries = make(map[string]*IndexEntry)
	}
	idx.Entries[path] = entry
}

func (idx *Index) RemoveEntry(path string) {
	if idx.Entries != nil {
		delete(idx.Entries, path)
	}
}

func (idx *Index) LoadAndGetEntries() (map[string]*IndexEntry, error) {
	if err := idx.Load(); err != nil {
		return nil, err
	}
	return idx.Entries, nil
}
