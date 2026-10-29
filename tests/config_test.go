package tests

import (
	"bytes"
	"errors"
	"flag"
	"strings"
	"testing"

	"vshell/src/config"
)

const usageLine = "Usage: vshell [-vfs path] [-script path]"

var optionCases = []struct {
	name string
	args []string
	want config.Options
}{
	{"no arguments", nil, config.Options{}},
	{"vfs only", []string{"-vfs", "/tmp/vfs"}, config.Options{VFSPath: "/tmp/vfs"}},
	{"script only", []string{"-script", "init.vsh"}, config.Options{ScriptPath: "init.vsh"}},
	{"both", []string{"-vfs", "vfs", "-script", "init.vsh"}, config.Options{VFSPath: "vfs", ScriptPath: "init.vsh"}},
	{"opposite order", []string{"-script", "init.vsh", "-vfs", "vfs"},
		config.Options{VFSPath: "vfs", ScriptPath: "init.vsh"}},
	{"double dash and equals", []string{"--vfs=my vfs", "--script=init.vsh"},
		config.Options{VFSPath: "my vfs", ScriptPath: "init.vsh"}},
	{"last value wins", []string{"-vfs", "a", "-vfs", "b"}, config.Options{VFSPath: "b"}},
	{"end of flags", []string{"-vfs", "a", "--"}, config.Options{VFSPath: "a"}},
}

var optionErrorCases = []struct {
	name     string
	args     []string
	wantErr  error
	wantText string
}{
	{"help", []string{"-h"}, flag.ErrHelp, usageLine},
	{"long help", []string{"--help"}, flag.ErrHelp, usageLine},
	{"unknown flag", []string{"-verbose"}, nil, "flag provided but not defined: -verbose"},
	{"missing value", []string{"-vfs", "a", "-script"}, nil, "flag needs an argument: -script"},
	{"positional argument", []string{"-vfs", "a", "extra"}, config.ErrUnexpectedArgument,
		"unexpected argument: extra"},
}

// TestParseOptions checks that every spelling and order of the flags yields the right Options.
func TestParseOptions(t *testing.T) {
	for _, tc := range optionCases {
		t.Run(tc.name, func(t *testing.T) {
			var errOut bytes.Buffer
			got, err := config.Parse("vshell", tc.args, &errOut)
			if err != nil {
				t.Fatalf("Parse(%q) returned error: %v", tc.args, err)
			}
			if got != tc.want {
				t.Errorf("Parse(%q) = %+v, want %+v", tc.args, got, tc.want)
			}
			if errOut.Len() != 0 {
				t.Errorf("Parse(%q) wrote %q, want nothing", tc.args, errOut.String())
			}
		})
	}
}

// TestParseOptionsErrors checks that invalid arguments and help requests fail with the
// expected error and print the reason followed by the usage text. A nil wantErr accepts
// any error.
func TestParseOptionsErrors(t *testing.T) {
	for _, tc := range optionErrorCases {
		t.Run(tc.name, func(t *testing.T) {
			var errOut bytes.Buffer
			_, err := config.Parse("vshell", tc.args, &errOut)
			if err == nil || tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("Parse(%q) error = %v, want %v", tc.args, err, tc.wantErr)
			}
			if got := errOut.String(); !strings.Contains(got, tc.wantText) || !strings.Contains(got, usageLine) {
				t.Errorf("Parse(%q) wrote %q, want %q and the usage", tc.args, got, tc.wantText)
			}
		})
	}
}

// TestWriteDebug checks that the debug output lists every parameter, quoting given
// values and marking missing ones.
func TestWriteDebug(t *testing.T) {
	var out bytes.Buffer
	config.Options{VFSPath: "/my vfs"}.WriteDebug(&out)
	want := "[debug] vfs: \"/my vfs\"\n[debug] script: (not set)\n"
	if got := out.String(); got != want {
		t.Errorf("WriteDebug() wrote %q, want %q", got, want)
	}
}
