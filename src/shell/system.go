package shell

import (
	"os"
	"runtime"
)

const unknownValue = "unknown"

// kernelNames maps Go operating system names to the kernel names uname prints.
var kernelNames = map[string]string{
	"darwin":    "Darwin",
	"dragonfly": "DragonFly",
	"freebsd":   "FreeBSD",
	"linux":     "Linux",
	"netbsd":    "NetBSD",
	"openbsd":   "OpenBSD",
	"windows":   "Windows_NT",
}

// machineNames maps Go architecture names to the machine types uname prints on Linux.
var machineNames = map[string]string{
	"386":   "i686",
	"amd64": "x86_64",
	"arm":   "armv7l",
	"arm64": "aarch64",
}

// SystemInfo describes the host operating system the way uname reports it.
type SystemInfo struct {
	// Kernel is the kernel name, uname -s.
	Kernel string
	// Node is the network node host name, uname -n.
	Node string
	// Release is the kernel release, uname -r.
	Release string
	// Version is the kernel version, uname -v.
	Version string
	// Machine is the hardware type, uname -m.
	Machine string
}

// DetectSystem reads the kernel name, release and version, the full host name and the
// machine type of the real host. Kernel data comes from sysctl on BSD systems and macOS
// and from /proc on Linux; elsewhere, and for values that cannot be read, it falls back
// to names derived from the Go runtime and "unknown".
func DetectSystem() SystemInfo {
	info := SystemInfo{
		Kernel:  lookupOr(kernelNames, runtime.GOOS),
		Node:    fallbackHost,
		Release: unknownValue,
		Version: unknownValue,
		Machine: lookupOr(machineNames, runtime.GOARCH),
	}
	if host, err := os.Hostname(); err == nil && host != "" {
		info.Node = host
	}
	readKernelInfo(&info)
	return info
}

// lookupOr returns the value names holds for key, or key itself when there is none.
func lookupOr(names map[string]string, key string) string {
	if name, ok := names[key]; ok {
		return name
	}
	return key
}

// setIfKnown stores value in dst unless it is empty, which means it could not be read.
func setIfKnown(dst *string, value string) {
	if value != "" {
		*dst = value
	}
}
