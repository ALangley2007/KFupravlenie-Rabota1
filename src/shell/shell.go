// Package shell implements the command interpreter of the emulator: the parser,
// the prompt, the built-in commands and the read-eval-print loop.
package shell

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"vshell/src/vfs"
)

// Name is the shell name used as the prefix of every error message.
const Name = "vshell"

const lineDelimiter = '\n'

const lineTerminators = "\r\n"

// Exit statuses, matching the ones bash uses for the same situations.
const (
	StatusOK            = 0
	StatusError         = 1
	StatusMisuse        = 2
	StatusCannotExecute = 126
	StatusNotFound      = 127
)

// ErrCommandNotFound is reported when no command has the requested name.
var ErrCommandNotFound = errors.New("command not found")

// errReported marks a failure whose messages the command has already printed itself,
// so the shell only records its status.
var errReported = errors.New("errors already reported")

// ExitError is an error that carries the exit status it should set.
type ExitError struct {
	Status int
	Err    error
}

// Error returns the message of the wrapped error.
func (e *ExitError) Error() string {
	return e.Err.Error()
}

// Unwrap returns the wrapped error, so errors.Is and errors.As can look through it.
func (e *ExitError) Unwrap() error {
	return e.Err
}

// Config holds the streams and the environment a Shell runs with.
type Config struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
	Host   HostInfo
	// System describes the host OS for uname.
	System SystemInfo
	// FS is the virtual file system the session works in; nil means an empty one.
	FS *vfs.FS
}

// Shell is one interactive session of the emulator.
type Shell struct {
	in       *bufio.Reader
	out      io.Writer
	errOut   io.Writer
	host     HostInfo
	system   SystemInfo
	fsys     *vfs.FS
	cwd      *vfs.Node
	oldDir   *vfs.Node
	commands map[string]Command
	origin   string
	status   int
	exitCode int
	running  bool
}

// New returns a running session that reads commands from cfg.In, writes output to
// cfg.Out and errors to cfg.ErrOut, and works in the virtual file system cfg.FS, or in
// an empty one when it is nil. Like a login, the session starts in the user's home
// directory, or in the root when the VFS has no such directory.
func New(cfg Config) *Shell {
	s := &Shell{
		in:       bufio.NewReader(cfg.In),
		out:      cfg.Out,
		errOut:   cfg.ErrOut,
		host:     cfg.Host,
		system:   cfg.System,
		fsys:     cfg.FS,
		commands: builtins(),
		running:  true,
	}
	if s.fsys == nil {
		s.fsys = vfs.New()
	}
	if s.host.Home == "" {
		s.host.Home = s.host.DefaultHome()
	}
	if s.host.Group == "" {
		s.host.Group = s.host.User
	}
	s.cwd = s.fsys.Root()
	if home, err := s.fsys.Lookup(s.host.Home); err == nil && home.IsDir() {
		s.cwd = home
	}
	return s
}

// Prompt returns the input prompt for the current directory, e.g. "user@host:~$ ".
func (s *Shell) Prompt() string {
	return s.host.Prompt(s.cwd.Path())
}

// Run reads and executes commands from the input, printing the prompt before each one,
// until exit is called or the input ends. It returns the exit status of the session.
// When the session was already ended, for example by exit in the startup script,
// it returns at once without reading anything.
func (s *Shell) Run() int {
	for s.running {
		fmt.Fprint(s.out, s.Prompt())
		line, err := s.in.ReadString(lineDelimiter)
		if line != "" {
			s.Execute(strings.TrimRight(line, lineTerminators))
		}
		if err != nil {
			s.stopReading(err)
		}
	}
	return s.exitCode
}

// Execute parses and runs one command line, reports any error and returns the new
// status. Empty and comment-only lines keep the previous status.
func (s *Shell) Execute(line string) int {
	return s.executeFrom("", line)
}

// Exit ends the session with the given exit status once the current command returns.
func (s *Shell) Exit(code int) {
	s.exitCode = code
	s.running = false
}

// lookup returns the VFS node at name, taken relative to the current directory unless
// it starts with a slash.
func (s *Shell) lookup(name string) (*vfs.Node, error) {
	return s.fsys.LookupFrom(s.cwd, name)
}

// executeFrom runs line like Execute, adding origin, such as "script: line 3",
// to the errors it reports. An empty origin adds nothing.
func (s *Shell) executeFrom(origin, line string) int {
	s.origin = origin
	args, err := Parse(line, s.host.Home)
	if err != nil {
		return s.report(&ExitError{Status: StatusMisuse, Err: fmt.Errorf("syntax error: %w", err)})
	}
	if len(args) == 0 {
		return s.status
	}
	return s.report(s.dispatch(args[0], args[1:]))
}

// dispatch runs the command called name with args. It wraps the command's error with
// the command name and returns ErrCommandNotFound, with status 127, for unknown names.
func (s *Shell) dispatch(name string, args []string) error {
	command, ok := s.commands[name]
	if !ok {
		return &ExitError{Status: StatusNotFound, Err: fmt.Errorf("%s: %w", name, ErrCommandNotFound)}
	}
	if err := command(s, args); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// report stores the status that err stands for and returns it. A non-nil err is printed
// by warn unless the command has already reported it.
func (s *Shell) report(err error) int {
	s.status = statusOf(err)
	if err != nil && !errors.Is(err, errReported) {
		s.warn(err)
	}
	return s.status
}

// warn prints err to the error stream as "vshell: [origin: ]message". Commands that meet
// several errors call it for each one and go on.
func (s *Shell) warn(err error) {
	if s.origin != "" {
		err = fmt.Errorf("%s: %w", s.origin, err)
	}
	fmt.Fprintf(s.errOut, "%s: %v\n", Name, err)
}

// stopReading ends a session whose input failed with err. End of input acts like exit
// with the last status, as in bash; any other read error is reported first.
func (s *Shell) stopReading(err error) {
	if !s.running {
		return
	}
	s.origin = ""
	if !errors.Is(err, io.EOF) {
		s.report(fmt.Errorf("read error: %w", err))
	}
	fmt.Fprintln(s.out, "exit")
	s.Exit(s.status)
}

// statusOf returns the exit status for err: 0 for nil, the carried status for an
// ExitError and 1 for any other error.
func statusOf(err error) int {
	var exitErr *ExitError
	switch {
	case err == nil:
		return StatusOK
	case errors.As(err, &exitErr):
		return exitErr.Status
	default:
		return StatusError
	}
}
