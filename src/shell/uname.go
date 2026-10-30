package shell

import (
	"fmt"
	"io"
	"strings"
)

const (
	unameOptions  = "asnrvm"
	unameAll      = "a"
	unameAllShown = "snrvm"
	unameDefault  = "s"
)

// clear writes the cursor-home and erase-display sequences, then erases the scrollback,
// as the ncurses clear does for xterm-compatible terminals.
const (
	clearScreen     = "\x1b[H\x1b[2J"
	clearScrollback = "\x1b[3J"
	clearOptions    = "x"
	keepScrollback  = "x"
)

// unameCommand implements uname [-asnrvm] with the data of the real host: -s kernel name,
// -n network node name, -r kernel release, -v kernel version, -m machine type, and -a all
// of them. The values come in that order whatever the order of the options, separated by
// spaces; without options it prints the kernel name. An invalid option or an operand is
// an error.
func unameCommand(s *Shell, args []string) error {
	letters, operands, err := splitOptions(args, unameOptions)
	if err != nil {
		return err
	}
	if err := noOperands(operands); err != nil {
		return err
	}
	switch {
	case letters == "":
		letters = unameDefault
	case strings.Contains(letters, unameAll):
		letters = unameAllShown
	}
	fields := []struct {
		option string
		value  string
	}{
		{"s", s.system.Kernel},
		{"n", s.system.Node},
		{"r", s.system.Release},
		{"v", s.system.Version},
		{"m", s.system.Machine},
	}
	var values []string
	for _, field := range fields {
		if strings.Contains(letters, field.option) {
			values = append(values, field.value)
		}
	}
	_, err = fmt.Fprintln(s.out, strings.Join(values, " "))
	return err
}

// clearCommand implements clear [-x]: it clears the terminal screen and its scrollback
// with ANSI escape sequences; -x keeps the scrollback. An invalid option or an operand
// is an error.
func clearCommand(s *Shell, args []string) error {
	letters, operands, err := splitOptions(args, clearOptions)
	if err != nil {
		return err
	}
	if err := noOperands(operands); err != nil {
		return err
	}
	sequence := clearScreen + clearScrollback
	if strings.Contains(letters, keepScrollback) {
		sequence = clearScreen
	}
	_, err = io.WriteString(s.out, sequence)
	return err
}
