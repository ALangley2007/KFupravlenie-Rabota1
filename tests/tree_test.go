package tests

import (
	"testing"

	"vshell/src/shell"
)

const fullTree = `├── bin.dat
├── docs
│   ├── guides
│   │   └── deep
│   │       └── note.md
│   └── readme.txt
└── empty

4 directories, 3 files
`

var treeCases = []struct {
	name string
	dir  string
	line string
	want string
}{
	{"current directory by default", "/", "tree", ".\n" + fullTree},
	{"absolute path", "/docs", "tree /", "/\n" + fullTree},
	{"relative path", "/docs", "tree guides",
		"guides\n└── deep\n    └── note.md\n\n1 directory, 1 file\n"},
	{"dot-dot", "/docs/guides", "tree ../..", "../..\n" + fullTree},
	{"trailing slash", "/", "tree docs/guides/deep/",
		"docs/guides/deep/\n└── note.md\n\n0 directories, 1 file\n"},
	{"file", "/", "tree docs/readme.txt", "docs/readme.txt\n\n0 directories, 1 file\n"},
	{"empty directory", "/", "tree empty", "empty\n\n0 directories, 0 files\n"},
}

var treeErrorCases = []struct {
	name    string
	line    string
	wantErr string
}{
	{"missing path", "tree nope", "vshell: tree: nope: no such file or directory\n"},
	{"file as a directory", "tree bin.dat/x", "vshell: tree: bin.dat/x: not a directory\n"},
	{"too many arguments", "tree docs empty", "vshell: tree: too many arguments\n"},
}

// newTreeSession returns a session over the test tree that starts in dir.
func newTreeSession(t *testing.T, dir string) *session {
	t.Helper()
	return newFSSession("", decodeTree(t, testTree...), dir)
}

// TestTree checks the tree listing for default, absolute and relative paths, files and
// empty directories.
func TestTree(t *testing.T) {
	for _, tc := range treeCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newTreeSession(t, tc.dir)
			if got := s.shell.Execute(tc.line); got != shell.StatusOK {
				t.Errorf("Execute(%q) status = %d, want %d: %s", tc.line, got, shell.StatusOK, s.errOut)
			}
			if got := s.out.String(); got != tc.want {
				t.Errorf("Execute(%q) output =\n%s\nwant\n%s", tc.line, got, tc.want)
			}
		})
	}
}

// TestTreeErrors checks that tree reports bad paths and extra arguments with status 1
// and prints nothing.
func TestTreeErrors(t *testing.T) {
	for _, tc := range treeErrorCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newTreeSession(t, "/")
			if got := s.shell.Execute(tc.line); got != shell.StatusError {
				t.Errorf("Execute(%q) status = %d, want %d", tc.line, got, shell.StatusError)
			}
			if got := s.errOut.String(); got != tc.wantErr {
				t.Errorf("Execute(%q) errors = %q, want %q", tc.line, got, tc.wantErr)
			}
			if got := s.out.String(); got != "" {
				t.Errorf("Execute(%q) output = %q, want nothing", tc.line, got)
			}
		})
	}
}

// TestDefaultFileSystem checks that a session without a file system works in an empty
// one, starting at its root.
func TestDefaultFileSystem(t *testing.T) {
	s := newFSSession("", nil, "")
	if got, want := s.shell.Prompt(), "alice@wonderland:/$ "; got != want {
		t.Errorf("Prompt() = %q, want %q", got, want)
	}
	s.shell.Execute("tree")
	if got, want := s.out.String(), ".\n\n0 directories, 0 files\n"; got != want {
		t.Errorf("tree output = %q, want %q", got, want)
	}
}
