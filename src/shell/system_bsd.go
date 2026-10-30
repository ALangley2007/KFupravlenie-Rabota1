//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package shell

import (
	"strings"
	"syscall"
)

// readKernelInfo fills in the kernel name, release and version and the machine type
// from sysctl, the source uname itself uses on BSD systems and macOS.
func readKernelInfo(info *SystemInfo) {
	setIfKnown(&info.Kernel, sysctl("kern.ostype"))
	setIfKnown(&info.Release, sysctl("kern.osrelease"))
	setIfKnown(&info.Version, sysctl("kern.version"))
	setIfKnown(&info.Machine, sysctl("hw.machine"))
}

// sysctl returns the string value of the kernel variable name without surrounding
// blanks, or "" when it cannot be read.
func sysctl(name string) string {
	value, err := syscall.Sysctl(name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}
