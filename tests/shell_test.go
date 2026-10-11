package tests

import (
	"bytes"
	"strings"
	"testing"

	"vshell/src/shell"
)

var testHost = shell.HostInfo{User: "alice", Host: "wonderland", Home: "/home/alice"}

const testPrompt = "alice@wonderland:~$ "

type session struct {
	shell  *shell.Shell
	out    *bytes.Buffer
	errOut *bytes.Buffer
}

func newSession(input string) *session {
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	return &session{
		shell: shell.New(shell.Config{
			In:     strings.NewReader(input),
			Out:    out,
			ErrOut: errOut,
			Host:   testHost,
			Dir:    testHost.Home,
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
	{"ls without arguments", "ls", shell.StatusOK, "ls args: []\n", ""},
	{"ls with quoted arguments", `ls -l "my dir" 'a b'`, shell.StatusOK, "ls args: [\"-l\" \"my dir\" \"a b\"]\n", ""},
	{"cd with arguments", `cd /usr/local`, shell.StatusOK, "cd args: [\"/usr/local\"]\n", ""},
	{"quoted command name", `"ls" x`, shell.StatusOK, "ls args: [\"x\"]\n", ""},
	{"unknown command", "foo bar", shell.StatusNotFound, "", "vshell: foo: command not found\n"},
	{"unterminated quote", `ls "abc`, shell.StatusMisuse, "",
		"vshell: syntax error: unexpected EOF while looking for matching quote `\"'\n"},
	{"trailing escape", `ls abc\`, shell.StatusMisuse, "",
		"vshell: syntax error: unexpected EOF after escape character\n"},
	{"exit with too many arguments", "exit 1 2", shell.StatusError, "", "vshell: exit: too many arguments\n"},
}

func TestExecute(t *testing.T) {
	for _, tc := range executeCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newSession("")
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
	{"exit with a non-numeric argument", "exit abc\nexit 5\n", shell.StatusMisuse},
	{"exit with too many arguments goes on", "exit 1 2\nexit 7\n", 7},
	{"last line without newline", "exit 4", 4},
}

func TestRun(t *testing.T) {
	for _, tc := range runCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := newSession(tc.input).shell.Run(); got != tc.wantStatus {
				t.Errorf("Run(%q) = %d, want %d", tc.input, got, tc.wantStatus)
			}
		})
	}
}

func TestRunDialogue(t *testing.T) {
	s := newSession("ls \"a b\"\nfoo\ncd ..\nexit\nls ignored\n")
	s.shell.Run()
	wantOut := testPrompt + "ls args: [\"a b\"]\n" + testPrompt + testPrompt + "cd args: [\"..\"]\n" + testPrompt
	if got := s.out.String(); got != wantOut {
		t.Errorf("output = %q, want %q", got, wantOut)
	}
	if got, want := s.errOut.String(), "vshell: foo: command not found\n"; got != want {
		t.Errorf("errors = %q, want %q", got, want)
	}
}

func TestRunEndOfInput(t *testing.T) {
	s := newSession("ls\n")
	s.shell.Run()
	if got, want := s.out.String(), testPrompt+"ls args: []\n"+testPrompt+"exit\n"; got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestExitNonNumeric(t *testing.T) {
	s := newSession("exit abc\n")
	s.shell.Run()
	if got, want := s.errOut.String(), "vshell: exit: abc: numeric argument required\n"; got != want {
		t.Errorf("errors = %q, want %q", got, want)
	}
}

func TestRunTrailingEscape(t *testing.T) {
	s := newSession("ls abc\\\r\nexit\n")
	s.shell.Run()
	if got, want := s.errOut.String(), "vshell: syntax error: unexpected EOF after escape character\n"; got != want {
		t.Errorf("errors = %q, want %q", got, want)
	}
	if got := s.out.String(); strings.Contains(got, "ls args") {
		t.Errorf("output = %q, want no command output", got)
	}
}
