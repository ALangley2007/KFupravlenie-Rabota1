package tests

import (
	"bytes"
	"strings"
	"testing"

	"vshell/src/shell"
	"vshell/src/vfs"
)

var testHost = shell.HostInfo{User: "alice", Group: "staff", Host: "wonderland", Home: "/home/alice"}

var testSystem = shell.SystemInfo{
	Kernel:  "Linux",
	Node:    "wonderland.local",
	Release: "6.1.0",
	Version: "#1 SMP",
	Machine: "x86_64",
}

const testPrompt = "alice@wonderland:~$ "

// homeTree is the VFS of the shell tests: the test user's home directory with a file,
// a directory and a file whose names contain blanks.
var homeTree = []zipEntry{
	{"home/alice/notes.txt", "hi"},
	{"home/alice/my dir/inner.txt", ""},
	{"home/alice/a b", ""},
}

// session is a shell under test together with the buffers that capture its output.
type session struct {
	shell  *shell.Shell
	out    *bytes.Buffer
	errOut *bytes.Buffer
}

// newSession returns a session for the test host over homeTree that reads input as the
// user's typing and starts in the user's home directory.
func newSession(t *testing.T, input string) *session {
	t.Helper()
	return newFSSession(input, decodeTree(t, homeTree...))
}

// newFSSession returns a session for the test host over fsys, an empty VFS when nil,
// that reads input as the user's typing.
func newFSSession(input string, fsys *vfs.FS) *session {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	return &session{
		shell: shell.New(shell.Config{
			In:     strings.NewReader(input),
			Out:    out,
			ErrOut: errOut,
			Host:   testHost,
			System: testSystem,
			FS:     fsys,
		}),
		out:    out,
		errOut: errOut,
	}
}

var executeCases = []struct {
	name       string
	line       string
	wantStatus int
	wantOut    string
	wantErr    string
}{
	{"empty line", "   ", shell.StatusOK, "", ""},
	{"ls without arguments", "ls", shell.StatusOK, "a b  my dir  notes.txt\n", ""},
	{"double-quoted argument", `ls "my dir"`, shell.StatusOK, "inner.txt\n", ""},
	{"single-quoted argument", `ls 'a b'`, shell.StatusOK, "a b\n", ""},
	{"escaped blank", `ls my\ dir`, shell.StatusOK, "inner.txt\n", ""},
	{"quoted command name", `"ls" notes.txt`, shell.StatusOK, "notes.txt\n", ""},
	{"comment line", "// ls", shell.StatusOK, "", ""},
	{"trailing comment", "ls notes.txt // b", shell.StatusOK, "notes.txt\n", ""},
	{"unknown command", "foo bar", shell.StatusNotFound, "", "vshell: foo: command not found\n"},
	{"unterminated quote", `ls "abc`, shell.StatusMisuse, "",
		"vshell: syntax error: unexpected EOF while looking for matching quote `\"'\n"},
	{"trailing escape", `ls abc\`, shell.StatusMisuse, "",
		"vshell: syntax error: unexpected EOF after escape character\n"},
	{"exit with too many arguments", "exit 1 2", shell.StatusError, "", "vshell: exit: too many arguments\n"},
}

// TestExecute checks the status, output and errors of single command lines.
func TestExecute(t *testing.T) {
	for _, tc := range executeCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newSession(t, "")
			if got := s.shell.Execute(tc.line); got != tc.wantStatus {
				t.Errorf("Execute(%q) status = %d, want %d", tc.line, got, tc.wantStatus)
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

var runCases = []struct {
	name       string
	input      string
	wantStatus int
}{
	{"end of input", "", 0},
	{"exit", "exit\n", 0},
	{"exit with status", "exit 3\n", 3},
	{"exit status wraps around", "exit 258\n", 2},
	{"negative exit status", "exit -1\n", 255},
	{"exit keeps the last status", "foo\nexit\n", shell.StatusNotFound},
	{"end of input keeps the last status", "foo\n", shell.StatusNotFound},
	{"success resets the status", "foo\nls\nexit\n", shell.StatusOK},
	{"comment keeps the status", "foo\n// note\nexit\n", shell.StatusNotFound},
	{"exit with a non-numeric argument", "exit abc\nexit 5\n", shell.StatusMisuse},
	{"exit with too many arguments goes on", "exit 1 2\nexit 7\n", 7},
	{"last line without newline", "exit 4", 4},
}

// TestRun checks the exit status a whole session ends with.
func TestRun(t *testing.T) {
	for _, tc := range runCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := newSession(t, tc.input).shell.Run(); got != tc.wantStatus {
				t.Errorf("Run(%q) = %d, want %d", tc.input, got, tc.wantStatus)
			}
		})
	}
}

// TestRunDialogue checks that prompts, output and errors interleave like a real dialogue
// and that the prompt follows the current directory.
func TestRunDialogue(t *testing.T) {
	s := newSession(t, "ls \"a b\"\nfoo\ncd ..\nexit\nls ignored\n")
	s.shell.Run()
	wantOut := testPrompt + "a b\n" + testPrompt + testPrompt + "alice@wonderland:/home$ "
	if got := s.out.String(); got != wantOut {
		t.Errorf("output = %q, want %q", got, wantOut)
	}
	if got, want := s.errOut.String(), "vshell: foo: command not found\n"; got != want {
		t.Errorf("errors = %q, want %q", got, want)
	}
}

// TestRunEndOfInput checks that the end of input prints exit, as bash does.
func TestRunEndOfInput(t *testing.T) {
	s := newSession(t, "ls notes.txt\n")
	s.shell.Run()
	if got, want := s.out.String(), testPrompt+"notes.txt\n"+testPrompt+"exit\n"; got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

// TestExitNonNumeric checks the error exit reports for a non-numeric status.
func TestExitNonNumeric(t *testing.T) {
	s := newSession(t, "exit abc\n")
	s.shell.Run()
	if got, want := s.errOut.String(), "vshell: exit: abc: numeric argument required\n"; got != want {
		t.Errorf("errors = %q, want %q", got, want)
	}
}

// TestRunTrailingEscape checks that a backslash before CRLF is a syntax error, not an escaped CR.
func TestRunTrailingEscape(t *testing.T) {
	s := newSession(t, "ls notes.txt\\\r\nexit\n")
	s.shell.Run()
	if got, want := s.errOut.String(), "vshell: syntax error: unexpected EOF after escape character\n"; got != want {
		t.Errorf("errors = %q, want %q", got, want)
	}
	if got := s.out.String(); strings.Contains(got, "notes.txt") {
		t.Errorf("output = %q, want no command output", got)
	}
}

// TestStartDirectory checks that a session starts in the user's home directory when the
// VFS has it, and in the root otherwise.
func TestStartDirectory(t *testing.T) {
	if got := newSession(t, "").shell.Prompt(); got != testPrompt {
		t.Errorf("Prompt() with a home directory = %q, want %q", got, testPrompt)
	}
	if got, want := newFSSession("", nil).shell.Prompt(), "alice@wonderland:/$ "; got != want {
		t.Errorf("Prompt() without a home directory = %q, want %q", got, want)
	}
}
