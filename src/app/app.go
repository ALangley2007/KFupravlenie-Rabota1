// Package app wires the command-line parameters, the virtual file system, the startup
// script and the shell session into one emulator run.
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
	"vshell/src/vfs"
)

// Run starts the emulator with args, the command-line arguments without the program
// name, and the streams and host data in cfg. It prints the parameters for debugging,
// loads the VFS into memory, runs the startup script when one is given, then the
// interactive session. It returns the process exit status: 0 for help, 2 for invalid
// arguments, 1 for a VFS that cannot be loaded, 126 or 127 for an unreadable script,
// otherwise the status the session ended with.
func Run(args []string, cfg shell.Config) int {
	opts, err := config.Parse(shell.Name, args, cfg.ErrOut)
	if errors.Is(err, flag.ErrHelp) {
		return shell.StatusOK
	}
	if err != nil {
		return shell.StatusMisuse
	}
	opts.WriteDebug(cfg.ErrOut)
	if cfg.FS, err = loadVFS(opts.VFSPath); err != nil {
		reportFailure(cfg.ErrOut, opts.VFSPath, err)
		return shell.StatusError
	}
	session := shell.New(cfg)
	if opts.ScriptPath != "" {
		script, err := os.ReadFile(opts.ScriptPath)
		if err != nil {
			reportFailure(cfg.ErrOut, opts.ScriptPath, err)
			return scriptStatus(err)
		}
		session.RunScript(opts.ScriptPath, bytes.NewReader(script))
	}
	return session.Run()
}

// loadVFS loads the VFS archive at path, or returns an empty file system when no path
// is given.
func loadVFS(path string) (*vfs.FS, error) {
	if path == "" {
		return vfs.New(), nil
	}
	return vfs.Load(path)
}

// reportFailure tells that the file at path could not be used. The operation and path
// a *fs.PathError would repeat are dropped, since the message already names the path.
func reportFailure(errOut io.Writer, path string, err error) {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}
	fmt.Fprintf(errOut, "%s: %s: %v\n", shell.Name, path, err)
}

// scriptStatus returns the status bash gives for a script it cannot read: 127 when the
// file does not exist and 126 when it exists but cannot be read, such as a directory.
func scriptStatus(err error) int {
	if errors.Is(err, fs.ErrNotExist) {
		return shell.StatusNotFound
	}
	return shell.StatusCannotExecute
}
