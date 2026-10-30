package tests

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vshell/src/vfs"
)

// zipEntry is one entry of a test archive; a name ending with a slash is a directory.
type zipEntry struct {
	name string
	body string
}

// zipFile is an archive entry with the mode and modification time it stores; zero values
// leave them out of the archive.
type zipFile struct {
	zipEntry
	mode     fs.FileMode
	modified time.Time
}

// testTree is a small archive with nested directories, an empty one, a binary file
// and directories that are only implied by file paths.
var testTree = []zipEntry{
	{"docs/", ""},
	{"docs/readme.txt", "hello\n"},
	{"docs/guides/deep/note.md", "deep"},
	{"bin.dat", "\x00\x01\xff"},
	{"empty/", ""},
}

// makeZip returns a ZIP archive holding entries in order.
func makeZip(t *testing.T, entries ...zipEntry) []byte {
	t.Helper()
	files := make([]zipFile, len(entries))
	for i, entry := range entries {
		files[i] = zipFile{zipEntry: entry}
	}
	return makeZipFiles(t, files...)
}

// makeZipFiles returns a ZIP archive holding files in order, with their modes and times.
func makeZipFiles(t *testing.T, files ...zipFile) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for _, file := range files {
		header := &zip.FileHeader{Name: file.name, Method: zip.Deflate, Modified: file.modified}
		if file.mode != 0 {
			header.SetMode(file.mode)
		}
		content, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatalf("creating %q: %v", file.name, err)
		}
		if _, err := content.Write([]byte(file.body)); err != nil {
			t.Fatalf("writing %q: %v", file.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing the archive: %v", err)
	}
	return buf.Bytes()
}

// decodeTree returns the file system built from a ZIP archive holding entries.
func decodeTree(t *testing.T, entries ...zipEntry) *vfs.FS {
	t.Helper()
	return decodeArchive(t, makeZip(t, entries...))
}

// decodeArchive returns the file system built from archive.
func decodeArchive(t *testing.T, archive []byte) *vfs.FS {
	t.Helper()
	fsys, err := vfs.Decode(archive)
	if err != nil {
		t.Fatalf("Decode() returned error: %v", err)
	}
	return fsys
}

// wrapBase64 encodes data as base64 text broken into lines, as base64 tools print it.
func wrapBase64(data []byte) string {
	const lineLength = 76
	text := base64.StdEncoding.EncodeToString(data)
	var lines []string
	for len(text) > lineLength {
		lines = append(lines, text[:lineLength])
		text = text[lineLength:]
	}
	return strings.Join(append(lines, text), "\n") + "\n"
}

var lookupCases = []struct {
	name    string
	path    string
	wantDir bool
	want    string
	size    int64
}{
	{"root", "/", true, "/", 0},
	{"empty path is the root", "", true, "/", 0},
	{"explicit directory", "/docs", true, "docs", 0},
	{"implied directory", "/docs/guides/deep", true, "deep", 0},
	{"path without a leading slash", "docs/readme.txt", false, "readme.txt", 6},
	{"binary file", "/bin.dat", false, "bin.dat", 3},
	{"empty directory", "/empty/", true, "empty", 0},
	{"dot and dot-dot", "/docs/./guides/../readme.txt", false, "readme.txt", 6},
	{"dot-dot at the root", "/../../docs", true, "docs", 0},
	{"repeated slashes", "//docs///guides", true, "guides", 0},
}

var lookupErrorCases = []struct {
	name string
	path string
	want error
}{
	{"missing entry", "/nope", vfs.ErrNotExist},
	{"missing nested entry", "/docs/guides/nope/x", vfs.ErrNotExist},
	{"file as a directory", "/bin.dat/x", vfs.ErrNotDir},
	{"file with a trailing slash", "/bin.dat/", vfs.ErrNotDir},
	{"dot-dot after a file", "/docs/readme.txt/..", vfs.ErrNotDir},
	{"dot after a file", "/docs/readme.txt/.", vfs.ErrNotDir},
}

// TestLookup checks that paths resolve to the right node, kind and size.
func TestLookup(t *testing.T) {
	fsys := decodeTree(t, testTree...)
	for _, tc := range lookupCases {
		t.Run(tc.name, func(t *testing.T) {
			node, err := fsys.Lookup(tc.path)
			if err != nil {
				t.Fatalf("Lookup(%q) returned error: %v", tc.path, err)
			}
			if node.Name() != tc.want || node.IsDir() != tc.wantDir || node.Size() != tc.size {
				t.Errorf("Lookup(%q) = %q (dir %v, size %d), want %q (dir %v, size %d)",
					tc.path, node.Name(), node.IsDir(), node.Size(), tc.want, tc.wantDir, tc.size)
			}
		})
	}
}

// TestLookupErrors checks that missing paths and paths through files fail with the right error.
func TestLookupErrors(t *testing.T) {
	fsys := decodeTree(t, testTree...)
	for _, tc := range lookupErrorCases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := fsys.Lookup(tc.path); !errors.Is(err, tc.want) {
				t.Errorf("Lookup(%q) error = %v, want %v", tc.path, err, tc.want)
			}
		})
	}
}

// TestChildren checks that directory entries come sorted by name and files have none.
func TestChildren(t *testing.T) {
	fsys := decodeTree(t, zipEntry{"b", ""}, zipEntry{"c/", ""}, zipEntry{"a", ""}, zipEntry{"B", ""})
	root, _ := fsys.Lookup("/")
	var names []string
	for _, child := range root.Children() {
		names = append(names, child.Name())
	}
	if got, want := strings.Join(names, " "), "B a b c"; got != want {
		t.Errorf("Children() = %q, want %q", got, want)
	}
	if file, _ := fsys.Lookup("/a"); file.Children() != nil {
		t.Errorf("Children() of a file = %v, want nil", file.Children())
	}
}

// TestNewIsEmpty checks that a new file system holds only an empty root.
func TestNewIsEmpty(t *testing.T) {
	root, err := vfs.New().Lookup(vfs.Root)
	if err != nil || !root.IsDir() || len(root.Children()) != 0 {
		t.Errorf("New() root = %v, %v, want an empty directory", root, err)
	}
}

// TestDecodeFormats checks that an archive is accepted raw, as wrapped base64 text and
// as base64 on one line with surrounding blanks, and that an archive with no entries
// gives an empty file system.
func TestDecodeFormats(t *testing.T) {
	archive := makeZip(t, testTree...)
	formats := map[string][]byte{
		"raw ZIP":         archive,
		"wrapped base64":  []byte(wrapBase64(archive)),
		"one-line base64": []byte("  " + base64.StdEncoding.EncodeToString(archive) + " \r\n"),
	}
	for name, data := range formats {
		t.Run(name, func(t *testing.T) {
			fsys, err := vfs.Decode(data)
			if err != nil {
				t.Fatalf("Decode() returned error: %v", err)
			}
			if _, err := fsys.Lookup("/docs/guides/deep/note.md"); err != nil {
				t.Errorf("Lookup() after Decode() returned error: %v", err)
			}
		})
	}
	fsys, err := vfs.Decode([]byte(wrapBase64(makeZip(t))))
	if root, _ := fsys.Lookup("/"); err != nil || len(root.Children()) != 0 {
		t.Errorf("Decode(empty archive) = %v, want an empty file system", err)
	}
}

// TestDecodeErrors checks that data which is no archive, and archives with bad entries,
// are rejected with the matching error.
func TestDecodeErrors(t *testing.T) {
	archive := makeZip(t, testTree...)
	cases := []struct {
		name string
		data []byte
		want error
	}{
		{"empty data", nil, vfs.ErrNotArchive},
		{"invalid base64", []byte("not base64!"), vfs.ErrNotArchive},
		{"base64 of plain text", []byte(wrapBase64([]byte("plain text"))), vfs.ErrNotArchive},
		{"truncated archive", archive[:len(archive)/2], vfs.ErrNotArchive},
		{"parent reference", makeZip(t, zipEntry{"../escape.txt", ""}), vfs.ErrBadName},
		{"absolute name", makeZip(t, zipEntry{"/etc/passwd", ""}), vfs.ErrBadName},
		{"empty element", makeZip(t, zipEntry{"a//b", ""}), vfs.ErrBadName},
		{"duplicate file", makeZip(t, zipEntry{"a", "1"}, zipEntry{"a", "2"}), vfs.ErrExist},
		{"file over a directory", makeZip(t, zipEntry{"a/", ""}, zipEntry{"a", ""}), vfs.ErrExist},
		{"entry inside a file", makeZip(t, zipEntry{"a", ""}, zipEntry{"a/b", ""}), vfs.ErrNotDir},
		{"directory over a file", makeZip(t, zipEntry{"a", ""}, zipEntry{"a/", ""}), vfs.ErrNotDir},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if fsys, err := vfs.Decode(tc.data); !errors.Is(err, tc.want) || fsys != nil {
				t.Errorf("Decode() = %v, %v, want error %v", fsys, err, tc.want)
			}
		})
	}
}

// TestDecodeTooLarge checks that an archive unpacking to more than 64 MiB is rejected.
func TestDecodeTooLarge(t *testing.T) {
	const limit = 64 << 20
	huge := zipEntry{"huge.bin", strings.Repeat("\x00", limit+1)}
	if _, err := vfs.Decode(makeZip(t, huge)); !errors.Is(err, vfs.ErrTooLarge) {
		t.Errorf("Decode() error = %v, want %v", err, vfs.ErrTooLarge)
	}
}

// TestLoad checks that Load reads a base64 archive from disk and reports missing files.
func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tree.b64")
	if err := os.WriteFile(path, []byte(wrapBase64(makeZip(t, testTree...))), 0o600); err != nil {
		t.Fatalf("writing the archive: %v", err)
	}
	fsys, err := vfs.Load(path)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if _, err := fsys.Lookup("/docs/readme.txt"); err != nil {
		t.Errorf("Lookup() after Load() returned error: %v", err)
	}
	if _, err := vfs.Load(path + ".missing"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Load(missing) error = %v, want %v", err, fs.ErrNotExist)
	}
}

// TestNodePaths checks the absolute path and parent of nodes, and lookups relative to a
// directory.
func TestNodePaths(t *testing.T) {
	fsys := decodeTree(t, testTree...)
	deep, _ := fsys.Lookup("docs/guides/deep")
	if got, want := deep.Path(), "/docs/guides/deep"; got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
	if got := deep.Parent().Parent().Path(); got != "/docs" {
		t.Errorf("Parent().Parent().Path() = %q, want /docs", got)
	}
	if root := fsys.Root(); root.Path() != vfs.Root || root.Parent() != root {
		t.Errorf("Root() = %q with parent %q, want / as its own parent", root.Path(), root.Parent().Path())
	}
	cases := map[string]string{"": "/docs/guides/deep", "note.md": "/docs/guides/deep/note.md",
		"../../readme.txt": "/docs/readme.txt", "/bin.dat": "/bin.dat", "../../../../..": "/"}
	for name, want := range cases {
		if node, err := fsys.LookupFrom(deep, name); err != nil || node.Path() != want {
			t.Errorf("LookupFrom(deep, %q) = %v, %v, want %q", name, node, err, want)
		}
	}
}

// TestLinks checks the link counts: one for files, two plus subdirectories for directories.
func TestLinks(t *testing.T) {
	fsys := decodeTree(t, testTree...)
	cases := map[string]int{"/": 4, "/docs": 3, "/empty": 2, "/bin.dat": 1}
	for name, want := range cases {
		if node, _ := fsys.Lookup(name); node.Links() != want {
			t.Errorf("Lookup(%q).Links() = %d, want %d", name, node.Links(), want)
		}
	}
}

// TestModesAndTimes checks that entries keep their stored permissions and times, and that
// implied directories get the default mode and the time of the entry that implies them.
func TestModesAndTimes(t *testing.T) {
	early := time.Date(2020, time.January, 2, 3, 4, 5, 0, time.UTC)
	late := time.Date(2021, time.June, 7, 8, 9, 10, 0, time.UTC)
	fsys, err := vfs.Decode(makeZipFiles(t,
		zipFile{zipEntry{"bin/", ""}, fs.ModeDir | 0o700, early},
		zipFile{zipEntry{"bin/run", "x"}, 0o755, late},
		zipFile{zipEntry{"implied/dir/file", ""}, 0o600, late},
	))
	if err != nil {
		t.Fatalf("Decode() returned error: %v", err)
	}
	cases := []struct {
		path string
		mode fs.FileMode
		time time.Time
	}{
		{"/bin", fs.ModeDir | 0o700, early},
		{"/bin/run", 0o755, late},
		{"/implied", vfs.DefaultDirMode, late},
		{"/implied/dir/file", 0o600, late},
	}
	for _, tc := range cases {
		node, _ := fsys.Lookup(tc.path)
		if node.Mode() != tc.mode || !node.ModTime().Equal(tc.time) {
			t.Errorf("Lookup(%q) = %v %v, want %v %v", tc.path, node.Mode(), node.ModTime(), tc.mode, tc.time)
		}
	}
}

// TestDOSTimeIsLocal checks that an entry with only an MS-DOS timestamp, which has no
// time zone, is read as local time, as unzip does.
func TestDOSTimeIsLocal(t *testing.T) {
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	header := &zip.FileHeader{Name: "old.txt"}
	header.ModifiedDate = 42<<9 | 7<<5 | 8
	header.ModifiedTime = 9<<11 | 10<<5 | 6
	if _, err := writer.CreateHeader(header); err != nil || writer.Close() != nil {
		t.Fatalf("writing the archive: %v", err)
	}
	fsys, err := vfs.Decode(buf.Bytes())
	if err != nil {
		t.Fatalf("Decode() returned error: %v", err)
	}
	node, _ := fsys.Lookup("old.txt")
	if want := time.Date(2022, time.July, 8, 9, 10, 12, 0, time.Local); !node.ModTime().Equal(want) {
		t.Errorf("ModTime() = %v, want %v", node.ModTime(), want)
	}
}
