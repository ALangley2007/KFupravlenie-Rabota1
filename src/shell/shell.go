// Package shell implements the command interpreter of the emulator: the parser,
// the prompt, the built-in commands and the read-eval-print loop.
package shell

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
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
	Dir    string
}

// Shell is one interactive session of the emulator.
type Shell struct {
	in       *bufio.Reader
	out      io.Writer
	errOut   io.Writer
	host     HostInfo
	dir      string
	commands map[string]Command
	status   int
	exitCode int
	running  bool
}

// New returns a running session that reads commands from cfg.In, writes output to
// cfg.Out and errors to cfg.ErrOut, and starts in the directory cfg.Dir.
func New(cfg Config) *Shell {
	return &Shell{
		in:       bufio.NewReader(cfg.In),
		out:      cfg.Out,
		errOut:   cfg.ErrOut,
		host:     cfg.Host,
		dir:      cfg.Dir,
		commands: builtins(),
		running:  true,
	}
}

// Prompt returns the input prompt for the current directory, e.g. "user@host:~$ ".
func (s *Shell) Prompt() string {
	return s.host.Prompt(s.dir)
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

// executeFrom runs line like Execute, adding origin, such as "script: line 3",
// to the error it reports. An empty origin adds nothing.
func (s *Shell) executeFrom(origin, line string) int {
	args, err := Parse(line)
	if err != nil {
		return s.report(origin, &ExitError{Status: StatusMisuse, Err: fmt.Errorf("syntax error: %w", err)})
	}
	if len(args) == 0 {
		return s.status
	}
	return s.report(origin, s.dispatch(args[0], args[1:]))
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

// report stores the status that err stands for and returns it. A non-nil err is
// printed to the error stream as "vshell: [origin: ]message".
func (s *Shell) report(origin string, err error) int {
	s.status = statusOf(err)
	if err == nil {
		return s.status
	}
	if origin != "" {
		err = fmt.Errorf("%s: %w", origin, err)
	}
	fmt.Fprintf(s.errOut, "%s: %v\n", Name, err)
	return s.status
}

// stopReading ends a session whose input failed with err. End of input acts like exit
// with the last status, as in bash; any other read error is reported first.
func (s *Shell) stopReading(err error) {
	if !s.running {
		return
	}
	if !errors.Is(err, io.EOF) {
		s.report("", fmt.Errorf("read error: %w", err))
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
