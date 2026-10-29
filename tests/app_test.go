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

const rootPrompt = "alice@wonderland:/$ "

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
	})
	return result{status: status, out: out.String(), errOut: errOut.String()}
}

// writeFile saves content as the file name in a temporary directory and returns its path.
func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
	return path
}

// writeVFS saves the test tree as a base64 archive and returns its path.
func writeVFS(t *testing.T) string {
	t.Helper()
	return writeFile(t, "tree.b64", wrapBase64(makeZip(t, testTree...)))
}

// TestAppNoParameters checks that a launch without parameters prints the debug output
// and goes straight to the interactive session at the root of an empty VFS.
func TestAppNoParameters(t *testing.T) {
	got := runApp(nil, "tree\n")
	wantOut := rootPrompt + ".\n\n0 directories, 0 files\n" + rootPrompt + "exit\n"
	if want := (result{shell.StatusOK, wantOut, noParamsDebug}); got != want {
		t.Errorf("Run() = %+v, want %+v", got, want)
	}
}

// TestAppScript checks that the startup script runs after the debug output, against the
// loaded VFS and before the interactive session, and that its exit ends the run.
func TestAppScript(t *testing.T) {
	vfsPath := writeVFS(t)
	scriptPath := writeFile(t, "init.vsh", "// greet\ntree /empty\nexit 5\n")
	got := runApp([]string{"-vfs", vfsPath, "-script", scriptPath}, "ls typed\n")
	wantOut := rootPrompt + "// greet\n" + rootPrompt + "tree /empty\n" + "/empty\n\n0 directories, 0 files\n" +
		rootPrompt + "exit 5\n"
	wantErr := "[debug] vfs: \"" + vfsPath + "\"\n[debug] script: \"" + scriptPath + "\"\n"
	if want := (result{5, wantOut, wantErr}); got != want {
		t.Errorf("Run() = %+v, want %+v", got, want)
	}
}

// TestAppVFS checks that the interactive session works in the loaded VFS.
func TestAppVFS(t *testing.T) {
	got := runApp([]string{"-vfs", writeVFS(t)}, "tree /\n")
	if want := rootPrompt + "/\n" + fullTree + rootPrompt + "exit\n"; got.out != want {
		t.Errorf("Run() output = %q, want %q", got.out, want)
	}
}

// TestAppScriptError checks that a failing script command is reported with the script
// path and line number, and that the interactive session still follows.
func TestAppScriptError(t *testing.T) {
	path := writeFile(t, "init.vsh", "ls\ntree /nope\n")
	got := runApp([]string{"-script", path}, "")
	if got.status != shell.StatusError {
		t.Errorf("Run() status = %d, want %d", got.status, shell.StatusError)
	}
	want := "vshell: " + path + ": line 2: tree: /nope: no such file or directory\n"
	if !strings.HasSuffix(got.errOut, want) {
		t.Errorf("Run() errors = %q, want suffix %q", got.errOut, want)
	}
	if !strings.HasSuffix(got.out, rootPrompt+"exit\n") {
		t.Errorf("Run() output = %q, want the interactive session at the end", got.out)
	}
}

// TestAppUnloadable checks that a VFS or script that cannot be read is reported with its
// path and status, and that neither the script nor the session runs.
func TestAppUnloadable(t *testing.T) {
	dir := t.TempDir()
	script := writeFile(t, "init.vsh", "ls\n")
	cases := []struct {
		name, flag, path, message string
		status                    int
	}{
		{"missing VFS", "-vfs", filepath.Join(dir, "missing.b64"), "no such file or directory", shell.StatusError},
		{"VFS is a directory", "-vfs", dir, "is a directory", shell.StatusError},
		{"VFS is not an archive", "-vfs", script, "not a ZIP archive", shell.StatusError},
		{"missing script", "-script", filepath.Join(dir, "missing.vsh"), "no such file or directory",
			shell.StatusNotFound},
		{"script is a directory", "-script", dir, "is a directory", shell.StatusCannotExecute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runApp([]string{"-script", script, tc.flag, tc.path}, "ls\n")
			if got.status != tc.status {
				t.Errorf("Run() status = %d, want %d", got.status, tc.status)
			}
			if want := "vshell: " + tc.path + ": " + tc.message; !strings.Contains(got.errOut, want) {
				t.Errorf("Run() errors = %q, want %q", got.errOut, want)
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
