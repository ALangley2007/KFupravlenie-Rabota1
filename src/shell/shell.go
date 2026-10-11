package shell

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

const Name = "vshell"

const lineDelimiter = '\n'

const lineTerminators = "\r\n"

const (
	StatusOK       = 0
	StatusError    = 1
	StatusMisuse   = 2
	StatusNotFound = 127
)

var ErrCommandNotFound = errors.New("command not found")

type ExitError struct {
	Status int
	Err    error
}

func (e *ExitError) Error() string {
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

type Config struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
	Host   HostInfo
	Dir    string
}

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

func New(cfg Config) *Shell {
	return &Shell{
		in:       bufio.NewReader(cfg.In),
		out:      cfg.Out,
		errOut:   cfg.ErrOut,
		host:     cfg.Host,
		dir:      cfg.Dir,
		commands: builtins(),
	}
}

func (s *Shell) Prompt() string {
	return s.host.Prompt(s.dir)
}

func (s *Shell) Run() int {
	s.running = true
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

func (s *Shell) Execute(line string) int {
	args, err := Parse(line)
	if err != nil {
		return s.report(&ExitError{Status: StatusMisuse, Err: fmt.Errorf("syntax error: %w", err)})
	}
	if len(args) == 0 {
		return s.status
	}
	return s.report(s.dispatch(args[0], args[1:]))
}

func (s *Shell) Exit(code int) {
	s.exitCode = code
	s.running = false
}

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

func (s *Shell) report(err error) int {
	s.status = statusOf(err)
	if err != nil {
		fmt.Fprintf(s.errOut, "%s: %v\n", Name, err)
	}
	return s.status
}

func (s *Shell) stopReading(err error) {
	if !s.running {
		return
	}
	if !errors.Is(err, io.EOF) {
		s.report(fmt.Errorf("read error: %w", err))
	}
	fmt.Fprintln(s.out, "exit")
	s.Exit(s.status)
}

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
