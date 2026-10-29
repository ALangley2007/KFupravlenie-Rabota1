package tests

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vshell/src/app"
	"vshell/src/shell"
)

const noParamsDebug = "[debug] vfs: (not set)\n[debug] script: (not set)\n"

// result is what one emulator run produced.
type result struct {
	status int
	out    string
	errOut string
}

// runApp runs the emulator for the test host with args, feeding input as the user's
// typing, and returns its status and both output streams.
func runApp(args []string, input string) result {
	var out, errOut bytes.Buffer
	status := app.Run(args, shell.Config{
		In:     strings.NewReader(input),
		Out:    &out,
		ErrOut: &errOut,
		Host:   testHost,
		Dir:    testHost.Home,
	})
	return result{status: status, out: out.String(), errOut: errOut.String()}
}

// writeScript saves content as a startup script in a temporary directory and returns its path.
func writeScript(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "init.vsh")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing the script: %v", err)
	}
	return path
}

// TestAppNoParameters checks that a launch without parameters prints the debug output
// and goes straight to the interactive session.
func TestAppNoParameters(t *testing.T) {
	got := runApp(nil, "ls\n")
	want := result{shell.StatusOK, testPrompt + "ls args: []\n" + testPrompt + "exit\n", noParamsDebug}
	if got != want {
		t.Errorf("Run() = %+v, want %+v", got, want)
	}
}

// TestAppScript checks that the startup script runs after the debug output, before the
// interactive session, and that its exit ends the run.
func TestAppScript(t *testing.T) {
	path := writeScript(t, "// greet\nls a\nexit 5\n")
	got := runApp([]string{"-vfs", "vfs", "-script", path}, "ls typed\n")
	wantOut := testPrompt + "// greet\n" + testPrompt + "ls a\n" + "ls args: [\"a\"]\n" + testPrompt + "exit 5\n"
	wantErr := "[debug] vfs: \"vfs\"\n[debug] script: \"" + path + "\"\n"
	if want := (result{5, wantOut, wantErr}); got != want {
		t.Errorf("Run() = %+v, want %+v", got, want)
	}
}

// TestAppScriptError checks that a failing script command is reported with the script
// path and line number, and that the interactive session still follows.
func TestAppScriptError(t *testing.T) {
	path := writeScript(t, "ls\nfoo\n")
	got := runApp([]string{"-script", path}, "")
	if got.status != shell.StatusNotFound {
		t.Errorf("Run() status = %d, want %d", got.status, shell.StatusNotFound)
	}
	if want := "vshell: " + path + ": line 2: foo: command not found\n"; !strings.HasSuffix(got.errOut, want) {
		t.Errorf("Run() errors = %q, want suffix %q", got.errOut, want)
	}
	if !strings.HasSuffix(got.out, testPrompt+"exit\n") {
		t.Errorf("Run() output = %q, want the interactive session at the end", got.out)
	}
}

// TestAppUnreadableScript checks that a missing script or a directory given as the
// script is reported with the bash exit statuses and that no session starts.
func TestAppUnreadableScript(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name, path, message string
		status              int
	}{
		{"missing file", filepath.Join(dir, "missing.vsh"), "no such file or directory", shell.StatusNotFound},
		{"directory", dir, "is a directory", shell.StatusCannotExecute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runApp([]string{"-script", tc.path}, "ls\n")
			if got.status != tc.status {
				t.Errorf("Run() status = %d, want %d", got.status, tc.status)
			}
			if want := "vshell: " + tc.path + ": " + tc.message + "\n"; !strings.HasSuffix(got.errOut, want) {
				t.Errorf("Run() errors = %q, want suffix %q", got.errOut, want)
			}
			if got.out != "" {
				t.Errorf("Run() output = %q, want no session", got.out)
			}
		})
	}
}

// TestAppInvalidArguments checks that bad arguments and help print the usage, skip the
// debug output and the session, and exit with 2 and 0 respectively.
func TestAppInvalidArguments(t *testing.T) {
	cases := []struct {
		name   string
		args   []string
		status int
	}{
		{"help", []string{"-h"}, shell.StatusOK},
		{"unknown flag", []string{"-x"}, shell.StatusMisuse},
		{"positional argument", []string{"extra"}, shell.StatusMisuse},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runApp(tc.args, "ls\n")
			if got.status != tc.status {
				t.Errorf("Run(%q) status = %d, want %d", tc.args, got.status, tc.status)
			}
			if !strings.Contains(got.errOut, usageLine) || strings.Contains(got.errOut, "[debug]") {
				t.Errorf("Run(%q) errors = %q, want only the usage", tc.args, got.errOut)
			}
			if got.out != "" {
				t.Errorf("Run(%q) output = %q, want no session", tc.args, got.out)
			}
		})
	}
}
