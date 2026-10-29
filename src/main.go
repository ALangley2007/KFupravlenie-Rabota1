// Command vshell is an emulator of a UNIX-like OS shell.
package main

import (
	"os"

	"vshell/src/app"
	"vshell/src/shell"
)

// main runs the emulator with the process arguments, the standard streams and the data
// of the real host, then exits with the status the emulator returned.
func main() {
	host := shell.DetectHost()
	os.Exit(app.Run(os.Args[1:], shell.Config{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
		Host:   host,
		Dir:    host.WorkingDir(),
	}))
}
