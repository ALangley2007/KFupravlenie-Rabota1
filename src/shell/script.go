package shell

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// RunScript executes the startup script read from script as if the user typed it:
// each line is printed after the prompt and then run, so the screen shows a regular
// dialogue. Errors are reported with the script name and the line number, and the
// script goes on after a failed command, like a non-interactive UNIX shell. Running
// exit ends both the script and the session. It returns the status of the last command.
func (s *Shell) RunScript(name string, script io.Reader) int {
	reader := bufio.NewReader(script)
	for number := 1; s.running; number++ {
		line, err := reader.ReadString(lineDelimiter)
		if line != "" {
			s.runScriptLine(fmt.Sprintf("%s: line %d", name, number), strings.TrimRight(line, lineTerminators))
		}
		if err != nil {
			s.stopScript(name, err)
			break
		}
	}
	return s.status
}

// runScriptLine echoes line after the prompt and executes it, reporting errors at origin.
func (s *Shell) runScriptLine(origin, line string) {
	fmt.Fprintf(s.out, "%s%s\n", s.Prompt(), line)
	s.executeFrom(origin, line)
}

// stopScript handles the error that ended reading the script called name. The end of
// the file is the normal way for a script to finish; other errors are reported.
func (s *Shell) stopScript(name string, err error) {
	if !errors.Is(err, io.EOF) {
		s.origin = name
		s.report(fmt.Errorf("read error: %w", err))
	}
}
