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

	"vshell/src/vfs"
)

// zipEntry is one entry of a test archive; a name ending with a slash is a directory.
type zipEntry struct {
	name string
	body string
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
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for _, entry := range entries {
		file, err := writer.Create(entry.name)
		if err != nil {
			t.Fatalf("creating %q: %v", entry.name, err)
		}
		if _, err := file.Write([]byte(entry.body)); err != nil {
			t.Fatalf("writing %q: %v", entry.name, err)
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
	fsys, err := vfs.Decode(makeZip(t, entries...))
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
