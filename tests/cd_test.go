package tests

import (
	"testing"

	"vshell/src/shell"
)

const rootPromptHome = "alice@wonderland:/$ "

var cdCases = []struct {
	name       string
	lines      []string
	wantPrompt string
	wantOut    string
}{
	{"relative path", []string{`cd "my dir"`}, "alice@wonderland:~/my dir$ ", ""},
	{"absolute path", []string{"cd /home"}, "alice@wonderland:/home$ ", ""},
	{"parent", []string{"cd .."}, "alice@wonderland:/home$ ", ""},
	{"dot-dot stops at the root", []string{"cd /../../.."}, rootPromptHome, ""},
	{"dot segments", []string{`cd ./my\ dir/../my\ dir/.`}, "alice@wonderland:~/my dir$ ", ""},
	{"trailing slash", []string{"cd /home/"}, "alice@wonderland:/home$ ", ""},
	{"no argument goes home", []string{"cd /", "cd"}, testPrompt, ""},
	{"tilde", []string{"cd /", "cd ~"}, testPrompt, ""},
	{"tilde with a path", []string{"cd /", `cd ~/my\ dir`}, "alice@wonderland:~/my dir$ ", ""},
	{"empty argument stays", []string{`cd ""`}, testPrompt, ""},
	{"dash returns and prints", []string{"cd /home", "cd -"}, testPrompt, "/home/alice\n"},
	{"dash twice", []string{"cd /", "cd -", "cd -"}, rootPromptHome, "/home/alice\n/\n"},
	{"failed cd keeps the previous directory", []string{"cd /home", "cd nope", "cd -"}, testPrompt,
		"/home/alice\n"},
}

// TestCd checks that cd changes the current directory, and the prompt with it, for
// relative and absolute paths, the home directory, a tilde and "cd -".
func TestCd(t *testing.T) {
	for _, tc := range cdCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newSession(t, "")
			for _, line := range tc.lines {
				s.shell.Execute(line)
			}
			if got := s.shell.Prompt(); got != tc.wantPrompt {
				t.Errorf("Prompt() = %q, want %q", got, tc.wantPrompt)
			}
			if got := s.out.String(); got != tc.wantOut {
				t.Errorf("output = %q, want %q", got, tc.wantOut)
			}
		})
	}
}

var cdErrorCases = []struct {
	name       string
	line       string
	wantStatus int
	wantErr    string
}{
	{"no previous directory", "cd -", shell.StatusError, "vshell: cd: OLDPWD not set\n"},
	{"missing directory", "cd nope", shell.StatusError, "vshell: cd: nope: no such file or directory\n"},
	{"file", "cd notes.txt", shell.StatusError, "vshell: cd: notes.txt: not a directory\n"},
	{"file with a trailing slash", "cd notes.txt/", shell.StatusError, "vshell: cd: notes.txt/: not a directory\n"},
	{"through a file", "cd notes.txt/x", shell.StatusError, "vshell: cd: notes.txt/x: not a directory\n"},
	{"too many arguments", "cd / /home", shell.StatusError, "vshell: cd: too many arguments\n"},
	{"option", "cd -P /", shell.StatusMisuse, "vshell: cd: -P: invalid option\n"},
}

// TestCdErrors checks that a failed cd reports why and keeps the current directory.
func TestCdErrors(t *testing.T) {
	for _, tc := range cdErrorCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newSession(t, "")
			if got := s.shell.Execute(tc.line); got != tc.wantStatus {
				t.Errorf("Execute(%q) status = %d, want %d", tc.line, got, tc.wantStatus)
			}
			if got := s.errOut.String(); got != tc.wantErr {
				t.Errorf("Execute(%q) errors = %q, want %q", tc.line, got, tc.wantErr)
			}
			if got := s.shell.Prompt(); got != testPrompt {
				t.Errorf("Prompt() = %q, want %q", got, testPrompt)
			}
		})
	}
}

// TestCdMissingHome checks that plain cd fails like bash when the home directory is not
// in the VFS.
func TestCdMissingHome(t *testing.T) {
	s := newFSSession("", nil)
	if got := s.shell.Execute("cd"); got != shell.StatusError {
		t.Errorf("cd status = %d, want %d", got, shell.StatusError)
	}
	if got, want := s.errOut.String(), "vshell: cd: /home/alice: no such file or directory\n"; got != want {
		t.Errorf("cd errors = %q, want %q", got, want)
	}
}
