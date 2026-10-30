package tests

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"time"

	"vshell/src/shell"
)

// lsTime is the modification time of every entry of lsTree; it is old enough for ls -l
// to show the year.
var lsTime = time.Date(2021, time.March, 4, 5, 6, 7, 0, time.UTC)

// crowdSize is the number of files in the crowded directory of lsTree.
const crowdSize = 15

// lsTree returns the VFS of the ls tests: a directory with files of different modes and
// sizes, a hidden file, a name with a blank and an empty subdirectory, and a crowded
// directory for the column layout.
func lsTree(t *testing.T) []zipFile {
	t.Helper()
	files := []zipFile{
		{zipEntry{"docs/", ""}, fs.ModeDir | 0o755, lsTime},
		{zipEntry{"docs/a.txt", "hello\n"}, 0o644, lsTime},
		{zipEntry{"docs/big.bin", strings.Repeat("x", 5000)}, 0o600, lsTime},
		{zipEntry{"docs/run.sh", "#!/bin/sh\n"}, 0o755, lsTime},
		{zipEntry{"docs/my file", "x"}, 0o644, lsTime},
		{zipEntry{"docs/.hidden", ""}, 0o644, lsTime},
		{zipEntry{"docs/sub/", ""}, fs.ModeDir | 0o750, lsTime},
	}
	for i := 1; i <= crowdSize; i++ {
		files = append(files, zipFile{zipEntry{fmt.Sprintf("crowd/entry-%02d", i), ""}, 0o644, lsTime})
	}
	return files
}

// newLsSession returns a session at the root of lsTree.
func newLsSession(t *testing.T) *session {
	t.Helper()
	fsys := decodeArchive(t, makeZipFiles(t, lsTree(t)...))
	return newFSSession("", fsys)
}

const docsLong = `total 24
-rw-r--r-- 1 alice staff    6 Mar  4  2021 a.txt
-rw------- 1 alice staff 5000 Mar  4  2021 big.bin
-rw-r--r-- 1 alice staff    1 Mar  4  2021 my file
-rwxr-xr-x 1 alice staff   10 Mar  4  2021 run.sh
drwxr-x--- 2 alice staff 4096 Mar  4  2021 sub
`

const crowdColumns = "entry-01  entry-03  entry-05  entry-07  entry-09  entry-11  entry-13  entry-15\n" +
	"entry-02  entry-04  entry-06  entry-08  entry-10  entry-12  entry-14\n"

var lsCases = []struct {
	name string
	line string
	want string
}{
	{"current directory", "ls", "crowd  docs\n"},
	{"directory", "ls docs", "a.txt  big.bin  my file  run.sh  sub\n"},
	{"relative path with dot-dot", "ls docs/sub/../..", "crowd  docs\n"},
	{"hidden entries", "ls -a docs", ".  ..  .hidden  a.txt  big.bin  my file  run.sh  sub\n"},
	{"file", "ls docs/a.txt", "docs/a.txt\n"},
	{"empty directory", "ls docs/sub", ""},
	{"columns", "ls crowd", crowdColumns},
	{"long format", "ls -l docs", docsLong},
	{"options after the operand", "ls docs -l", docsLong},
	{"end of options", "ls -l -- docs", docsLong},
	{"long format with hidden entries", "ls -la docs/sub",
		"total 8\ndrwxr-x--- 2 alice staff 4096 Mar  4  2021 .\ndrwxr-xr-x 3 alice staff 4096 Mar  4  2021 ..\n"},
	{"separate options", "ls -l -a docs/sub",
		"total 8\ndrwxr-x--- 2 alice staff 4096 Mar  4  2021 .\ndrwxr-xr-x 3 alice staff 4096 Mar  4  2021 ..\n"},
	{"long format of files", "ls -l docs/run.sh docs/a.txt",
		"-rw-r--r-- 1 alice staff  6 Mar  4  2021 docs/a.txt\n-rwxr-xr-x 1 alice staff 10 Mar  4  2021 docs/run.sh\n"},
	{"files before directories", "ls docs/sub docs/a.txt docs",
		"docs/a.txt\n\ndocs:\na.txt  big.bin  my file  run.sh  sub\n\ndocs/sub:\n"},
	{"quoted name", `ls "docs/my file"`, "docs/my file\n"},
}

// TestLs checks the listing of directories, files and several operands in short and
// long format, with and without hidden entries.
func TestLs(t *testing.T) {
	for _, tc := range lsCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newLsSession(t)
			if got := s.shell.Execute(tc.line); got != shell.StatusOK {
				t.Errorf("Execute(%q) status = %d, want %d: %s", tc.line, got, shell.StatusOK, s.errOut)
			}
			if got := s.out.String(); got != tc.want {
				t.Errorf("Execute(%q) output =\n%s\nwant\n%s", tc.line, got, tc.want)
			}
		})
	}
}

var lsErrorCases = []struct {
	name    string
	line    string
	wantOut string
	wantErr string
}{
	{"missing operands are reported in order", "ls nope docs/a.txt missing", "docs/a.txt\n",
		"vshell: ls: cannot access 'nope': no such file or directory\n" +
			"vshell: ls: cannot access 'missing': no such file or directory\n"},
	{"file as a directory", "ls docs/a.txt/", "", "vshell: ls: cannot access 'docs/a.txt/': not a directory\n"},
	{"operand after the end of options", "ls -- -a", "", "vshell: ls: cannot access '-a': no such file or directory\n"},
	{"invalid option", "ls -lz docs", "", "vshell: ls: invalid option -- 'z'\n"},
	{"long option", "ls --all", "", "vshell: ls: unrecognized option '--all'\n"},
}

// TestLsErrors checks that ls reports every bad operand and option with status 2 and
// still lists the operands it can access.
func TestLsErrors(t *testing.T) {
	for _, tc := range lsErrorCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newLsSession(t)
			if got := s.shell.Execute(tc.line); got != shell.StatusMisuse {
				t.Errorf("Execute(%q) status = %d, want %d", tc.line, got, shell.StatusMisuse)
			}
			if got := s.out.String(); got != tc.wantOut {
				t.Errorf("Execute(%q) output = %q, want %q", tc.line, got, tc.wantOut)
			}
			if got := s.errOut.String(); got != tc.wantErr {
				t.Errorf("Execute(%q) errors = %q, want %q", tc.line, got, tc.wantErr)
			}
		})
	}
}

// TestLsTimes checks that ls -l shows the clock for times within the last six months and
// the year for older and future ones.
func TestLsTimes(t *testing.T) {
	recent := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	future := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	fsys := decodeArchive(t, makeZipFiles(t,
		zipFile{zipEntry{"recent", ""}, 0o644, recent},
		zipFile{zipEntry{"future", ""}, 0o644, future},
	))
	s := newFSSession("", fsys)
	s.shell.Execute("ls -l recent future")
	want := fmt.Sprintf("-rw-r--r-- 1 alice staff 0 %s future\n-rw-r--r-- 1 alice staff 0 %s recent\n",
		future.Format("Jan _2  2006"), recent.Format("Jan _2 15:04"))
	if got := s.out.String(); got != want {
		t.Errorf("ls -l output = %q, want %q", got, want)
	}
}

// TestLsWideNames checks that names too wide for the terminal are listed one per line.
func TestLsWideNames(t *testing.T) {
	wide := strings.Repeat("w", 90)
	fsys := decodeTree(t, zipEntry{wide, ""}, zipEntry{"z", ""})
	s := newFSSession("", fsys)
	s.shell.Execute("ls")
	if got, want := s.out.String(), wide+"\nz\n"; got != want {
		t.Errorf("ls output = %q, want %q", got, want)
	}
}
