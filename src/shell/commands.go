package shell

import (
	"errors"
	"fmt"
	"strconv"
)

const (
	currentDirName = "."
	parentDirName  = ".."
	maxExitArgs    = 1
	exitStatusMask = 0xFF
)

// ErrTooManyArguments is returned by a command that got more arguments than it accepts.
var ErrTooManyArguments = errors.New("too many arguments")

// ErrNumericArgument is returned by exit when its argument is not a number.
var ErrNumericArgument = errors.New("numeric argument required")

// Command is a built-in command: it runs in session s with the arguments after its name.
type Command func(s *Shell, args []string) error

// builtins returns the table of built-in commands keyed by name.
func builtins() map[string]Command {
	return map[string]Command{
		"cd":    cdCommand,
		"clear": clearCommand,
		"exit":  exitCommand,
		"ls":    lsCommand,
		"mv":    mvCommand,
		"tree":  treeCommand,
		"uname": unameCommand,
	}
}

// exitCommand implements exit [n]: it ends the session with status n modulo 256, or with
// the last status when n is omitted. A non-numeric n ends the session with status 2,
// as in bash; more than one argument is an error that keeps the session running.
func exitCommand(s *Shell, args []string) error {
	if len(args) == 0 {
		s.Exit(s.status)
		return nil
	}
	code, err := strconv.Atoi(args[0])
	if err != nil {
		s.Exit(StatusMisuse)
		return &ExitError{Status: StatusMisuse, Err: fmt.Errorf("%s: %w", args[0], ErrNumericArgument)}
	}
	if len(args) > maxExitArgs {
		return ErrTooManyArguments
	}
	s.Exit(code & exitStatusMask)
	return nil
}
