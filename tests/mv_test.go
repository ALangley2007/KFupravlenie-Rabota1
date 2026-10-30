package tests

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vshell/src/shell"
	"vshell/src/vfs"
)

// mvTree is the VFS of the mv tests: two files, a directory with a file, an empty
// directory, and a directory holding entries with the same names as the others.
var mvTree = []zipEntry{
	{"a.txt", "a"},
	{"b.txt", "bb"},
	{"dir/inner.txt", ""},
	{"box/", ""},
	{"shelf/box/", ""},
	{"shelf/dir/x", ""},
	{"shelf/a.txt/", ""},
}

const mvTreePaths = "/a.txt(1) /b.txt(2) /box/ /dir/ /dir/inner.txt(0) /shelf/ /shelf/a.txt/ /shelf/box/ " +
	"/shelf/dir/ /shelf/dir/x(0)"

// treePaths lists every node below dir in depth-first order, directories with a trailing
// slash and files with their size.
func treePaths(dir *vfs.Node) []string {
	var paths []string
	for _, child := range dir.Children() {
		if !child.IsDir() {
			paths = append(paths, fmt.Sprintf("%s(%d)", child.Path(), child.Size()))
			continue
		}
		paths = append(paths, child.Path()+"/")
		paths = append(paths, treePaths(child)...)
	}
	return paths
}

var mvCases = []struct {
	name      string
	lines     []string
	wantPaths string
	wantOut   string
}{
	{"rename a file", []string{"mv a.txt c.txt"},
		"/b.txt(2) /box/ /c.txt(1) /dir/ /dir/inner.txt(0) /shelf/ /shelf/a.txt/ /shelf/box/ /shelf/dir/ " +
			"/shelf/dir/x(0)", ""},
	{"file into a directory", []string{"mv a.txt dir"},
		"/b.txt(2) /box/ /dir/ /dir/a.txt(1) /dir/inner.txt(0) /shelf/ /shelf/a.txt/ /shelf/box/ /shelf/dir/ " +
			"/shelf/dir/x(0)", ""},
	{"trailing slash on the directory", []string{"mv a.txt dir/"},
		"/b.txt(2) /box/ /dir/ /dir/a.txt(1) /dir/inner.txt(0) /shelf/ /shelf/a.txt/ /shelf/box/ /shelf/dir/ " +
			"/shelf/dir/x(0)", ""},
	{"rename a directory with a trailing slash", []string{"mv dir moved/"},
		"/a.txt(1) /b.txt(2) /box/ /moved/ /moved/inner.txt(0) /shelf/ /shelf/a.txt/ /shelf/box/ /shelf/dir/ " +
			"/shelf/dir/x(0)", ""},
	{"directory into a directory", []string{"mv dir box"},
		"/a.txt(1) /b.txt(2) /box/ /box/dir/ /box/dir/inner.txt(0) /shelf/ /shelf/a.txt/ /shelf/box/ /shelf/dir/ " +
			"/shelf/dir/x(0)", ""},
	{"several sources", []string{"mv a.txt b.txt box"},
		"/box/ /box/a.txt(1) /box/b.txt(2) /dir/ /dir/inner.txt(0) /shelf/ /shelf/a.txt/ /shelf/box/ /shelf/dir/ " +
			"/shelf/dir/x(0)", ""},
	{"replace a file", []string{"mv a.txt b.txt"},
		"/b.txt(1) /box/ /dir/ /dir/inner.txt(0) /shelf/ /shelf/a.txt/ /shelf/box/ /shelf/dir/ /shelf/dir/x(0)", ""},
	{"no clobber keeps the file", []string{"mv -n a.txt b.txt"}, mvTreePaths, ""},
	{"force undoes no clobber", []string{"mv -n -f a.txt b.txt"},
		"/b.txt(1) /box/ /dir/ /dir/inner.txt(0) /shelf/ /shelf/a.txt/ /shelf/box/ /shelf/dir/ /shelf/dir/x(0)", ""},
	{"replace an empty directory", []string{"mv box shelf"},
		"/a.txt(1) /b.txt(2) /dir/ /dir/inner.txt(0) /shelf/ /shelf/a.txt/ /shelf/box/ /shelf/dir/ " +
			"/shelf/dir/x(0)", ""},
	{"relative paths", []string{"cd dir", "mv ../a.txt ./../shelf/../box/."},
		"/b.txt(2) /box/ /box/a.txt(1) /dir/ /dir/inner.txt(0) /shelf/ /shelf/a.txt/ /shelf/box/ /shelf/dir/ " +
			"/shelf/dir/x(0)", ""},
	{"verbose", []string{"mv -v a.txt b.txt dir/", "mv -v dir/b.txt /c"},
		"/box/ /c(2) /dir/ /dir/a.txt(1) /dir/inner.txt(0) /shelf/ /shelf/a.txt/ /shelf/box/ /shelf/dir/ " +
			"/shelf/dir/x(0)",
		"renamed 'a.txt' -> 'dir/a.txt'\nrenamed 'b.txt' -> 'dir/b.txt'\nrenamed 'dir/b.txt' -> '/c'\n"},
}

// TestMv checks renaming and moving files and directories, several sources, replacing
// files and empty directories, -n, -f and -v.
func TestMv(t *testing.T) {
	for _, tc := range mvCases {
		t.Run(tc.name, func(t *testing.T) {
			fsys := decodeTree(t, mvTree...)
			s := newFSSession("", fsys)
			for _, line := range tc.lines {
				if got := s.shell.Execute(line); got != shell.StatusOK {
					t.Errorf("Execute(%q) status = %d, want %d: %s", line, got, shell.StatusOK, s.errOut)
				}
			}
			if got := strings.Join(treePaths(fsys.Root()), " "); got != tc.wantPaths {
				t.Errorf("tree after %q =\n%s\nwant\n%s", tc.lines, got, tc.wantPaths)
			}
			if got := s.out.String(); got != tc.wantOut {
				t.Errorf("output = %q, want %q", got, tc.wantOut)
			}
		})
	}
}

var mvErrorCases = []struct {
	name    string
	line    string
	wantErr string
}{
	{"no operands", "mv", "vshell: mv: missing file operand\n"},
	{"one operand", "mv a.txt", "vshell: mv: missing destination file operand after 'a.txt'\n"},
	{"invalid option", "mv -i a.txt c", "vshell: mv: invalid option -- 'i'\n"},
	{"long option", "mv --force a.txt c", "vshell: mv: unrecognized option '--force'\n"},
	{"missing source", "mv nope c", "vshell: mv: cannot stat 'nope': no such file or directory\n"},
	{"file source with a slash", "mv a.txt/ c", "vshell: mv: cannot stat 'a.txt/': not a directory\n"},
	{"missing destination parent", "mv a.txt nope/c",
		"vshell: mv: cannot move 'a.txt' to 'nope/c': no such file or directory\n"},
	{"file as destination parent", "mv a.txt b.txt/c",
		"vshell: mv: cannot move 'a.txt' to 'b.txt/c': not a directory\n"},
	{"file to a new directory name", "mv a.txt c/", "vshell: mv: cannot move 'a.txt' to 'c/': not a directory\n"},
	{"directory into itself", "mv dir dir", "vshell: mv: cannot move 'dir' to a subdirectory of itself, 'dir/dir'\n"},
	{"directory below itself", "mv shelf shelf/dir",
		"vshell: mv: cannot move 'shelf' to a subdirectory of itself, 'shelf/dir/shelf'\n"},
	{"root", "mv / dir", "vshell: mv: cannot move '/' to 'dir': device or resource busy\n"},
	{"dot", "mv . dir", "vshell: mv: cannot move '.' to 'dir': device or resource busy\n"},
	{"dot-dot", "mv dir/.. box", "vshell: mv: cannot move 'dir/..' to 'box': device or resource busy\n"},
	{"same file", "mv a.txt ./a.txt", "vshell: mv: 'a.txt' and './a.txt' are the same file\n"},
	{"same file in its directory", "mv a.txt .", "vshell: mv: 'a.txt' and './a.txt' are the same file\n"},
	{"non-empty directory", "mv dir shelf", "vshell: mv: cannot move 'dir' to 'shelf/dir': directory not empty\n"},
	{"file over a directory", "mv a.txt shelf",
		"vshell: mv: cannot overwrite directory 'shelf/a.txt' with non-directory\n"},
	{"directory over a file", "mv dir b.txt",
		"vshell: mv: cannot overwrite non-directory 'b.txt' with directory 'dir'\n"},
	{"several sources to a file", "mv a.txt dir b.txt", "vshell: mv: target 'b.txt' is not a directory\n"},
	{"several sources to nothing", "mv a.txt b.txt c", "vshell: mv: target 'c': no such file or directory\n"},
}

// TestMvErrors checks that mv reports every failure the way GNU mv does, with status 1,
// and leaves the VFS as it was.
func TestMvErrors(t *testing.T) {
	for _, tc := range mvErrorCases {
		t.Run(tc.name, func(t *testing.T) {
			fsys := decodeTree(t, mvTree...)
			s := newFSSession("", fsys)
			if got := s.shell.Execute(tc.line); got != shell.StatusError {
				t.Errorf("Execute(%q) status = %d, want %d", tc.line, got, shell.StatusError)
			}
			if got := s.errOut.String(); got != tc.wantErr {
				t.Errorf("Execute(%q) errors = %q, want %q", tc.line, got, tc.wantErr)
			}
			if got := strings.Join(treePaths(fsys.Root()), " "); got != mvTreePaths {
				t.Errorf("tree after %q = %s, want it unchanged", tc.line, got)
			}
		})
	}
}

// TestMvPartialFailure checks that a failed source is reported while the others still move.
func TestMvPartialFailure(t *testing.T) {
	fsys := decodeTree(t, mvTree...)
	s := newFSSession("", fsys)
	if got := s.shell.Execute("mv nope a.txt box"); got != shell.StatusError {
		t.Errorf("status = %d, want %d", got, shell.StatusError)
	}
	if got, want := s.errOut.String(), "vshell: mv: cannot stat 'nope': no such file or directory\n"; got != want {
		t.Errorf("errors = %q, want %q", got, want)
	}
	if _, err := fsys.Lookup("/box/a.txt"); err != nil {
		t.Errorf("a.txt was not moved: %v", err)
	}
}

// TestMvCurrentDirectory checks that the prompt follows the current directory when one
// of its parents is moved.
func TestMvCurrentDirectory(t *testing.T) {
	s := newFSSession("", decodeTree(t, mvTree...))
	s.shell.Execute("cd /dir")
	s.shell.Execute("mv /dir /box/moved")
	if got, want := s.shell.Prompt(), "alice@wonderland:/box/moved$ "; got != want {
		t.Errorf("Prompt() = %q, want %q", got, want)
	}
}

// TestRenameErrors checks the errors of vfs.Rename the mv command never reaches: bad
// names, the root and a file as the target directory.
func TestRenameErrors(t *testing.T) {
	fsys := decodeTree(t, mvTree...)
	file, _ := fsys.Lookup("/a.txt")
	dir, _ := fsys.Lookup("/dir")
	for _, name := range []string{"", ".", "..", "x/y"} {
		if err := fsys.Rename(file, dir, name); !errors.Is(err, vfs.ErrBadName) {
			t.Errorf("Rename(file, dir, %q) error = %v, want %v", name, err, vfs.ErrBadName)
		}
	}
	if err := fsys.Rename(fsys.Root(), dir, "root"); !errors.Is(err, vfs.ErrBusy) {
		t.Errorf("Rename(root) error = %v, want %v", err, vfs.ErrBusy)
	}
	if err := fsys.Rename(dir, file, "x"); !errors.Is(err, vfs.ErrNotDir) {
		t.Errorf("Rename into a file error = %v, want %v", err, vfs.ErrNotDir)
	}
}

// TestRename checks that vfs.Rename leaves a node renamed onto itself alone, and that a
// move updates the node's name, parent and path and the times of both directories.
func TestRename(t *testing.T) {
	fsys := decodeTree(t, mvTree...)
	root := fsys.Root()
	file, _ := fsys.Lookup("/a.txt")
	dir, _ := fsys.Lookup("/dir")
	if err := fsys.Rename(file, root, "a.txt"); err != nil {
		t.Errorf("Rename onto itself error = %v, want nil", err)
	}
	before := time.Now()
	if err := fsys.Rename(file, dir, "moved"); err != nil {
		t.Fatalf("Rename() returned error: %v", err)
	}
	if file.Path() != "/dir/moved" || file.Parent() != dir || file.Name() != "moved" {
		t.Errorf("after Rename the file is %q in %q", file.Name(), file.Parent().Path())
	}
	if root.ModTime().Before(before) || dir.ModTime().Before(before) {
		t.Errorf("Rename() kept the old times of the directories: %v, %v", root.ModTime(), dir.ModTime())
	}
}
