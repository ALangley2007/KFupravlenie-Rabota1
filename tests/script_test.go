package tests

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"vshell/src/shell"
)

const testScript = "init.vsh"

var scriptCases = []struct {
	name       string
	script     string
	wantStatus int
	wantErr    string
}{
	{"empty script", "", shell.StatusOK, ""},
	{"comments only", "// first\n  // second\n\n", shell.StatusOK, ""},
	{"failed command is located", "ls\nfoo\n", shell.StatusNotFound,
		"vshell: init.vsh: line 2: foo: command not found\n"},
	{"syntax error is located", "\n// note\nls \"x\n", shell.StatusMisuse,
		"vshell: init.vsh: line 3: syntax error: unexpected EOF while looking for matching quote `\"'\n"},
	{"script goes on after an error", "foo\nls\n", shell.StatusOK,
		"vshell: init.vsh: line 1: foo: command not found\n"},
	{"every error is reported", "foo\nexit 1 2\n", shell.StatusError,
		"vshell: init.vsh: line 1: foo: command not found\nvshell: init.vsh: line 2: exit: too many arguments\n"},
	{"CRLF line endings", "ls\r\nfoo\r\n", shell.StatusNotFound,
		"vshell: init.vsh: line 2: foo: command not found\n"},
	{"last line without newline", "ls\nfoo", shell.StatusNotFound,
		"vshell: init.vsh: line 2: foo: command not found\n"},
	{"exit stops the script", "exit 3\nfoo\n", shell.StatusOK, ""},
}

// TestRunScript checks the status a startup script leaves and the located errors it reports.
func TestRunScript(t *testing.T) {
	for _, tc := range scriptCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newSession(t, "")
			if got := s.shell.RunScript(testScript, strings.NewReader(tc.script)); got != tc.wantStatus {
				t.Errorf("RunScript(%q) = %d, want %d", tc.script, got, tc.wantStatus)
			}
			if got := s.errOut.String(); got != tc.wantErr {
				t.Errorf("RunScript(%q) errors = %q, want %q", tc.script, got, tc.wantErr)
			}
		})
	}
}

// TestRunScriptEcho checks that every script line, comments and blank lines included,
// is printed after the prompt before its output, as if the user typed it.
func TestRunScriptEcho(t *testing.T) {
	s := newSession(t, "")
	s.shell.RunScript(testScript, strings.NewReader("// setup\n\nls \"a b\" // list\n"))
	wantOut := testPrompt + "// setup\n" + testPrompt + "\n" +
		testPrompt + "ls \"a b\" // list\n" + "a b\n"
	if got := s.out.String(); got != wantOut {
		t.Errorf("output = %q, want %q", got, wantOut)
	}
}

// TestRunScriptExit checks that exit in the script ends the session: later script lines
// and the interactive input are never run, and Run returns the exit status at once.
func TestRunScriptExit(t *testing.T) {
	const wantStatus = 3
	s := newSession(t, "ls typed\n")
	s.shell.RunScript(testScript, strings.NewReader("exit 3\nls skipped\n"))
	if got := s.shell.Run(); got != wantStatus {
		t.Errorf("Run() = %d, want %d", got, wantStatus)
	}
	if got, want := s.out.String(), testPrompt+"exit 3\n"; got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

// TestRunScriptThenInteractive checks that the interactive session follows the script in
// the directory the script left, keeps its last status and reports its own errors
// without a script location.
func TestRunScriptThenInteractive(t *testing.T) {
	s := newSession(t, "foo\nexit\n")
	s.shell.RunScript(testScript, strings.NewReader("cd \"my dir\"\n"))
	if got := s.shell.Run(); got != shell.StatusNotFound {
		t.Errorf("Run() = %d, want %d", got, shell.StatusNotFound)
	}
	dirPrompt := "alice@wonderland:~/my dir$ "
	wantOut := testPrompt + "cd \"my dir\"\n" + dirPrompt + dirPrompt
	if got := s.out.String(); got != wantOut {
		t.Errorf("output = %q, want %q", got, wantOut)
	}
	if got, want := s.errOut.String(), "vshell: foo: command not found\n"; got != want {
		t.Errorf("errors = %q, want %q", got, want)
	}
}

// TestRunScriptReadError checks that a failing script reader is reported under the script
// name after the lines read before the failure have run.
func TestRunScriptReadError(t *testing.T) {
	s := newSession(t, "")
	script := io.MultiReader(strings.NewReader("ls notes.txt\n"), iotest.ErrReader(errors.New("disk failure")))
	if got := s.shell.RunScript(testScript, script); got != shell.StatusError {
		t.Errorf("RunScript() = %d, want %d", got, shell.StatusError)
	}
	if got, want := s.errOut.String(), "vshell: init.vsh: read error: disk failure\n"; got != want {
		t.Errorf("errors = %q, want %q", got, want)
	}
	if got := s.out.String(); !strings.HasSuffix(got, "ls notes.txt\nnotes.txt\n") {
		t.Errorf("output = %q, want the ls output", got)
	}
}
