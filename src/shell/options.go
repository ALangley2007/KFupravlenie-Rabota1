package shell

import (
	"errors"
	"fmt"
	"strings"
)

const (
	optionPrefix    = "-"
	endOfOptions    = "--"
	minOptionLength = 2
)

// ErrInvalidOption is returned for an option letter a command does not know.
var ErrInvalidOption = errors.New("invalid option")

// ErrUnrecognizedOption is returned for a long option such as --all, which no command takes.
var ErrUnrecognizedOption = errors.New("unrecognized option")

// ErrExtraOperand is returned by a command that takes no operands, or fewer than it got.
var ErrExtraOperand = errors.New("extra operand")

// splitOptions separates args into option letters and operands the way getopt does for
// commands with single-letter options: an option starts with "-" and may combine letters
// (-al), options may come between operands, "--" ends them, and a lone "-" is an operand.
// It fails with ErrInvalidOption for a letter missing from allowed and with
// ErrUnrecognizedOption for a long option.
func splitOptions(args []string, allowed string) (string, []string, error) {
	var letters strings.Builder
	var operands []string
	for i, arg := range args {
		switch {
		case arg == endOfOptions:
			return letters.String(), append(operands, args[i+1:]...), nil
		case strings.HasPrefix(arg, endOfOptions):
			return "", nil, fmt.Errorf("%w '%s'", ErrUnrecognizedOption, arg)
		case len(arg) < minOptionLength || !strings.HasPrefix(arg, optionPrefix):
			operands = append(operands, arg)
			continue
		}
		for _, letter := range arg[1:] {
			if !strings.ContainsRune(allowed, letter) {
				return "", nil, fmt.Errorf("%w -- '%c'", ErrInvalidOption, letter)
			}
			letters.WriteRune(letter)
		}
	}
	return letters.String(), operands, nil
}

// noOperands returns ErrExtraOperand naming the first operand, or nil when there is none.
func noOperands(operands []string) error {
	if len(operands) > 0 {
		return fmt.Errorf("%w '%s'", ErrExtraOperand, operands[0])
	}
	return nil
}
