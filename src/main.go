package main

import (
	"os"
	"vshell/src/shell"
)

func main() {
	host := shell.DetectHost()
	session := shell.New(shell.Config{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
		Host:   host,
		Dir:    host.WorkingDir(),
	})
	os.Exit(session.Run())
}
