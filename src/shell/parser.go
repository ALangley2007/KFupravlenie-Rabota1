package shell

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

const (
	singleQuote = '\''
	doubleQuote = '"'
	escapeChar  = '\\'
	commentChar = '/'
	tildeChar   = '~'
	slashChar   = '/'
)

const escapableInDoubleQuotes = "\"\\$`"

// ErrUnterminatedQuote is returned by Parse when a quoted part is not closed.
var ErrUnterminatedQuote = errors.New("unexpected EOF while looking for matching quote")

// ErrTrailingEscape is returned by Parse when the line ends with a backslash.
var ErrTrailingEscape = errors.New("unexpected EOF after escape character")

// lexer splits one command line into arguments, keeping the state between characters.
type lexer struct {
	input   []rune
	pos     int
	home    string
	args    []string
	current strings.Builder
	inToken bool
}

// Parse splits line into arguments following the POSIX shell quoting rules: blanks
// separate arguments, single quotes keep text literally, double quotes keep blanks and
// allow escaping ", \, $ and `, and a backslash outside quotes escapes the next character.
// A "//" at the start of a word begins a Go-style comment that runs to the end of the
// line. An unquoted "~" that forms a word or starts one followed by "/" is replaced by
// home, unless home is empty. Parse returns nil for a line with no arguments.
func Parse(line, home string) ([]string, error) {
	lx := &lexer{input: []rune(line), home: home}
	for !lx.done() {
		if err := lx.step(); err != nil {
			return nil, err
		}
	}
	lx.flush()
	return lx.args, nil
}

// done reports whether the whole input has been read.
func (lx *lexer) done() bool {
	return lx.pos >= len(lx.input)
}

// next returns the character at the current position and moves past it.
func (lx *lexer) next() rune {
	char := lx.input[lx.pos]
	lx.pos++
	return char
}

// write appends char to the argument being built and marks that argument as started.
func (lx *lexer) write(char rune) {
	lx.current.WriteRune(char)
	lx.inToken = true
}

// flush finishes the argument being built, if any, and adds it to the result.
func (lx *lexer) flush() {
	if !lx.inToken {
		return
	}
	lx.args = append(lx.args, lx.current.String())
	lx.current.Reset()
	lx.inToken = false
}

// step reads one character and acts on it: ends the argument on a blank, skips a comment,
// expands a tilde, reads a quoted or escaped part, or adds a plain character to the argument.
func (lx *lexer) step() error {
	char := lx.next()
	switch {
	case unicode.IsSpace(char):
		lx.flush()
	case lx.opensComment(char):
		lx.pos = len(lx.input)
	case lx.expandsTilde(char):
		lx.current.WriteString(lx.home)
		lx.inToken = true
	case char == singleQuote:
		return lx.readSingleQuoted()
	case char == doubleQuote:
		return lx.readDoubleQuoted()
	case char == escapeChar:
		return lx.readEscaped()
	default:
		lx.write(char)
	}
	return nil
}

// opensComment reports whether char, just read, starts a "//" comment. Like "#" in
// a UNIX shell, it does so only at the start of a word, so paths such as a//b and
// quoted or escaped slashes stay intact.
func (lx *lexer) opensComment(char rune) bool {
	return char == commentChar && !lx.inToken && !lx.done() && lx.input[lx.pos] == commentChar
}

// expandsTilde reports whether char, just read, is a tilde to replace by the home
// directory: unquoted, at the start of a word and followed by "/", a blank or nothing,
// as in bash. Other forms such as ~user or a quoted "~" stay literal.
func (lx *lexer) expandsTilde(char rune) bool {
	if char != tildeChar || lx.inToken || lx.home == "" {
		return false
	}
	return lx.done() || lx.input[lx.pos] == slashChar || unicode.IsSpace(lx.input[lx.pos])
}

// readSingleQuoted reads up to the closing single quote, keeping every character as is.
func (lx *lexer) readSingleQuoted() error {
	lx.inToken = true
	for !lx.done() {
		char := lx.next()
		if char == singleQuote {
			return nil
		}
		lx.current.WriteRune(char)
	}
	return unterminated(singleQuote)
}

// readDoubleQuoted reads up to the closing double quote, handling escapes inside it.
func (lx *lexer) readDoubleQuoted() error {
	lx.inToken = true
	for !lx.done() {
		char := lx.next()
		switch char {
		case doubleQuote:
			return nil
		case escapeChar:
			if err := lx.readEscapedInQuotes(); err != nil {
				return err
			}
		default:
			lx.current.WriteRune(char)
		}
	}
	return unterminated(doubleQuote)
}

// readEscapedInQuotes reads the character after a backslash inside double quotes.
// Only ", \, $ and ` are escaped; before any other character the backslash is kept.
func (lx *lexer) readEscapedInQuotes() error {
	if lx.done() {
		return unterminated(doubleQuote)
	}
	char := lx.next()
	if !strings.ContainsRune(escapableInDoubleQuotes, char) {
		lx.current.WriteRune(escapeChar)
	}
	lx.current.WriteRune(char)
	return nil
}

// readEscaped reads the character after a backslash outside quotes and keeps it literally.
func (lx *lexer) readEscaped() error {
	if lx.done() {
		return ErrTrailingEscape
	}
	lx.write(lx.next())
	return nil
}

// unterminated returns ErrUnterminatedQuote naming the quote that was left open.
func unterminated(quote rune) error {
	return fmt.Errorf("%w `%c'", ErrUnterminatedQuote, quote)
}
