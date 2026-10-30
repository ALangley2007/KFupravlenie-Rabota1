//go:build !(darwin || dragonfly || freebsd || netbsd || openbsd || linux)

package shell

// readKernelInfo keeps the fallback values, since this system offers no portable way to
// read the kernel release and version.
func readKernelInfo(*SystemInfo) {}
