//go:build linux

package shell

import (
	"os"
	"strings"
)

const kernelInfoDir = "/proc/sys/kernel/"

// readKernelInfo fills in the kernel name, release and version from /proc, which holds
// the same values the uname system call returns.
func readKernelInfo(info *SystemInfo) {
	setIfKnown(&info.Kernel, readKernelFile("ostype"))
	setIfKnown(&info.Release, readKernelFile("osrelease"))
	setIfKnown(&info.Version, readKernelFile("version"))
}

// readKernelFile returns the content of the /proc/sys/kernel file name without
// surrounding blanks, or "" when it cannot be read.
func readKernelFile(name string) string {
	data, err := os.ReadFile(kernelInfoDir + name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
