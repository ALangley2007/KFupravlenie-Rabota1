package shell

import (
	"errors"
	"fmt"
	"strings"

	"vshell/src/vfs"
)

const (
	maxCdArgs      = 1
	previousDirArg = "-"
)

// ErrOldDirNotSet is returned by "cd -" before any directory change.
var ErrOldDirNotSet = errors.New("OLDPWD not set")

// cdCommand implements cd [dir | -] like the bash builtin: it changes the current VFS
// directory to dir, taken relative to the current one unless it starts with a slash,
// or to the home directory without an argument. "cd -" returns to the previous
// directory and prints its path. A missing path or a file is an error that keeps the
// current directory, and an option is invalid, with status 2.
func cdCommand(s *Shell, args []string) error {
	target, announce, err := s.cdTarget(args)
	if err != nil {
		return err
	}
	dir, err := s.lookup(target)
	if err == nil && !dir.IsDir() {
		err = vfs.ErrNotDir
	}
	if err != nil {
		return fmt.Errorf("%s: %w", target, err)
	}
	s.oldDir, s.cwd = s.cwd, dir
	if announce {
		_, err = fmt.Fprintln(s.out, dir.Path())
	}
	return err
}

// cdTarget returns the path cd should change to for args and whether cd must print the
// directory it reaches, which it does only for "cd -".
func (s *Shell) cdTarget(args []string) (string, bool, error) {
	switch {
	case len(args) == 0:
		return s.host.Home, false, nil
	case args[0] != previousDirArg && strings.HasPrefix(args[0], optionPrefix):
		return "", false, &ExitError{Status: StatusMisuse, Err: fmt.Errorf("%s: %w", args[0], ErrInvalidOption)}
	case len(args) > maxCdArgs:
		return "", false, ErrTooManyArguments
	case args[0] == previousDirArg && s.oldDir == nil:
		return "", false, ErrOldDirNotSet
	case args[0] == previousDirArg:
		return s.oldDir.Path(), true, nil
	default:
		return args[0], false, nil
	}
}
