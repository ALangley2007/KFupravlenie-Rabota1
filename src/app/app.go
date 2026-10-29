// Package app wires the command-line parameters, the startup script and the shell
// session into one emulator run.
package app

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"

	"vshell/src/config"
	"vshell/src/shell"
)

// Run starts the emulator with args, the command-line arguments without the program
// name, and the streams and host data in cfg. It prints the parameters for debugging,
// runs the startup script when one is given, then the interactive session. It returns
// the process exit status: 0 for help, 2 for invalid arguments, 126 or 127 for an
// unreadable script, otherwise the status the session ended with.
func Run(args []string, cfg shell.Config) int {
	opts, err := config.Parse(shell.Name, args, cfg.ErrOut)
	if errors.Is(err, flag.ErrHelp) {
		return shell.StatusOK
	}
	if err != nil {
		return shell.StatusMisuse
	}
	opts.WriteDebug(cfg.ErrOut)
	session := shell.New(cfg)
	if opts.ScriptPath != "" {
		script, err := os.ReadFile(opts.ScriptPath)
		if err != nil {
			return reportUnreadable(cfg.ErrOut, opts.ScriptPath, err)
		}
		session.RunScript(opts.ScriptPath, bytes.NewReader(script))
	}
	return session.Run()
}

// reportUnreadable tells that the startup script at path could not be read and returns
// the status bash gives for the same failure: 127 when the file does not exist and 126
// when it exists but cannot be read, for example a directory or a file without access.
func reportUnreadable(errOut io.Writer, path string, err error) int {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}
	fmt.Fprintf(errOut, "%s: %s: %v\n", shell.Name, path, err)
	if errors.Is(err, fs.ErrNotExist) {
		return shell.StatusNotFound
	}
	return shell.StatusCannotExecute
}
