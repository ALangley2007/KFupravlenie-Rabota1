package shell

import (
	"errors"
	"fmt"
	"strconv"
)

const (
	maxExitArgs    = 1
	exitStatusMask = 0xFF
)

var ErrTooManyArguments = errors.New("too many arguments")

var ErrNumericArgument = errors.New("numeric argument required")

type Command func(s *Shell, args []string) error

func builtins() map[string]Command {
	return map[string]Command{
		"ls":   stub("ls"),
		"cd":   stub("cd"),
		"exit": exitCommand,
	}
}

func stub(name string) Command {
	return func(s *Shell, args []string) error {
		_, err := fmt.Fprintf(s.out, "%s args: %q\n", name, args)
		return err
	}
}

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
