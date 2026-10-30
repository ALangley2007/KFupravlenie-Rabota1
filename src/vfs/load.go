package vfs

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"
)

// maxDataSize limits how many bytes all files of an archive may unpack into memory.
const maxDataSize = 64 << 20

// ErrNotArchive is returned for data that is neither a ZIP archive nor its base64 encoding.
var ErrNotArchive = errors.New("not a ZIP archive")

// ErrBadName is returned for an archive entry whose name is not a plain relative path.
var ErrBadName = errors.New("invalid name")

// ErrExist is returned for an archive entry that names an already loaded file.
var ErrExist = errors.New("file exists")

// ErrTooLarge is returned for an archive whose files would take more than 64 MiB of memory.
var ErrTooLarge = errors.New("archive is too large")

// zipSignatures are the bytes a ZIP archive can start with: a local file header,
// or the end-of-directory record of an archive with no entries.
var zipSignatures = [][]byte{[]byte("PK\x03\x04"), []byte("PK\x05\x06")}

// loader builds a file system from the entries of a ZIP archive.
type loader struct {
	fsys   *FS
	budget int64
}

// Load reads the ZIP archive at path, given as is or as base64 text, into a new file
// system in memory. The file on disk is only read, never unpacked or changed.
func Load(path string) (*FS, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Decode(data)
}

// Decode builds a file system from data, which is either a ZIP archive or its base64
// encoding; blanks and line breaks in base64 text are ignored. Entries keep the
// permissions and modification times stored in the archive. Directories missing from
// the archive but implied by file paths are created with the default mode and the time
// of the entry that implies them. All file contents are read into memory, at most
// 64 MiB in total.
func Decode(data []byte) (*FS, error) {
	archive, err := unwrapBase64(data)
	if err != nil {
		return nil, err
	}
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotArchive, err)
	}
	ld := &loader{fsys: New(), budget: maxDataSize}
	for _, file := range reader.File {
		if err := ld.add(file); err != nil {
			return nil, fmt.Errorf("entry %q: %w", file.Name, err)
		}
	}
	return ld.fsys, nil
}

// unwrapBase64 returns data when it already is a ZIP archive and the decoded data when
// it is base64 text of one.
func unwrapBase64(data []byte) ([]byte, error) {
	if isZip(data) {
		return data, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(string(data)), ""))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotArchive, err)
	}
	if !isZip(decoded) {
		return nil, ErrNotArchive
	}
	return decoded, nil
}

// isZip reports whether data starts with a ZIP archive signature.
func isZip(data []byte) bool {
	for _, signature := range zipSignatures {
		if bytes.HasPrefix(data, signature) {
			return true
		}
	}
	return false
}

// add puts the directory or file described by an archive entry into the file system.
// The entry name must be a relative slash-separated path without "." or ".." elements.
func (ld *loader) add(file *zip.File) error {
	name := strings.TrimSuffix(file.Name, separator)
	if name == currentDir {
		return nil
	}
	if !fs.ValidPath(name) {
		return ErrBadName
	}
	parts := strings.Split(name, separator)
	if file.Mode().IsDir() {
		return ld.addDir(file, parts)
	}
	modTime := localTime(file.Modified)
	dir, err := mkdirAll(ld.fsys.root, parts[:len(parts)-1], modTime)
	if err != nil {
		return err
	}
	base := parts[len(parts)-1]
	if _, exists := dir.children[base]; exists {
		return ErrExist
	}
	data, err := ld.read(file)
	if err != nil {
		return err
	}
	dir.children[base] = &Node{name: base, mode: file.Mode().Perm(), modTime: modTime, data: data, parent: dir}
	return nil
}

// addDir creates the directory entry file at the path in parts, or gives an already
// implied one the permissions and time the entry stores.
func (ld *loader) addDir(file *zip.File, parts []string) error {
	modTime := localTime(file.Modified)
	dir, err := mkdirAll(ld.fsys.root, parts, modTime)
	if err != nil {
		return err
	}
	dir.mode = fs.ModeDir | file.Mode().Perm()
	dir.modTime = modTime
	return nil
}

// localTime returns the modification time of an archive entry. An entry with only an
// MS-DOS timestamp stores the creator's wall clock without a time zone, which
// archive/zip returns as UTC; like unzip, it is taken as local time instead. Times with
// a zone, from the extended timestamp field, are kept.
func localTime(modified time.Time) time.Time {
	if modified.Location() != time.UTC {
		return modified
	}
	year, month, day := modified.Date()
	hour, minute, second := modified.Clock()
	return time.Date(year, month, day, hour, minute, second, 0, time.Local)
}

// read returns the unpacked content of file and takes its size from the loader's budget.
func (ld *loader) read(file *zip.File) ([]byte, error) {
	content, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer content.Close()
	data, err := io.ReadAll(io.LimitReader(content, ld.budget+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > ld.budget {
		return nil, ErrTooLarge
	}
	ld.budget -= int64(len(data))
	return data, nil
}

// mkdirAll returns the directory reached from dir through the names in parts, creating
// the missing ones with modTime. It fails with ErrNotDir when one of the names is a file.
func mkdirAll(dir *Node, parts []string, modTime time.Time) (*Node, error) {
	for _, part := range parts {
		child, ok := dir.children[part]
		if !ok {
			child = newDir(part, dir, modTime)
			dir.children[part] = child
		}
		if !child.IsDir() {
			return nil, ErrNotDir
		}
		dir = child
	}
	return dir, nil
}
