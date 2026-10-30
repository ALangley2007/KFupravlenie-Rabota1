package tests

import (
	"runtime"
	"testing"

	"vshell/src/shell"
)

var unameCases = []struct {
	name string
	line string
	want string
}{
	{"kernel name by default", "uname", "Linux\n"},
	{"kernel name", "uname -s", "Linux\n"},
	{"node name", "uname -n", "wonderland.local\n"},
	{"kernel release", "uname -r", "6.1.0\n"},
	{"kernel version", "uname -v", "#1 SMP\n"},
	{"machine", "uname -m", "x86_64\n"},
	{"everything", "uname -a", "Linux wonderland.local 6.1.0 #1 SMP x86_64\n"},
	{"combined options in a fixed order", "uname -mn", "wonderland.local x86_64\n"},
	{"separate options in a fixed order", "uname -m -s", "Linux x86_64\n"},
	{"all wins over others", "uname -sa", "Linux wonderland.local 6.1.0 #1 SMP x86_64\n"},
	{"end of options", "uname --", "Linux\n"},
}

var clearCases = []struct {
	name string
	line string
	want string
}{
	{"screen and scrollback", "clear", "\x1b[H\x1b[2J\x1b[3J"},
	{"screen only", "clear -x", "\x1b[H\x1b[2J"},
}

var utilityErrorCases = []struct {
	name    string
	line    string
	wantErr string
}{
	{"uname invalid option", "uname -x", "vshell: uname: invalid option -- 'x'\n"},
	{"uname long option", "uname --all", "vshell: uname: unrecognized option '--all'\n"},
	{"uname operand", "uname -a now", "vshell: uname: extra operand 'now'\n"},
	{"clear invalid option", "clear -z", "vshell: clear: invalid option -- 'z'\n"},
	{"clear operand", "clear now", "vshell: clear: extra operand 'now'\n"},
}

// TestUname checks every option of uname against fixed host data.
func TestUname(t *testing.T) {
	for _, tc := range append(unameCases, clearCases...) {
		t.Run(tc.name, func(t *testing.T) {
			s := newSession(t, "")
			if got := s.shell.Execute(tc.line); got != shell.StatusOK {
				t.Errorf("Execute(%q) status = %d, want %d: %s", tc.line, got, shell.StatusOK, s.errOut)
			}
			if got := s.out.String(); got != tc.want {
				t.Errorf("Execute(%q) output = %q, want %q", tc.line, got, tc.want)
			}
		})
	}
}

// TestUtilityErrors checks that uname and clear reject invalid options and operands with
// status 1 and print nothing.
func TestUtilityErrors(t *testing.T) {
	for _, tc := range utilityErrorCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newSession(t, "")
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

// TestDetectSystem checks that the real host data has every field, and the expected
// kernel name on the systems uname is usually run on.
func TestDetectSystem(t *testing.T) {
	info := shell.DetectSystem()
	for name, value := range map[string]string{"kernel": info.Kernel, "node": info.Node,
		"release": info.Release, "version": info.Version, "machine": info.Machine} {
		if value == "" {
			t.Errorf("DetectSystem() returned an empty %s", name)
		}
	}
	kernels := map[string]string{"darwin": "Darwin", "linux": "Linux"}
	if want, ok := kernels[runtime.GOOS]; ok && info.Kernel != want {
		t.Errorf("DetectSystem().Kernel = %q, want %q", info.Kernel, want)
	}
}
