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
)

const escapableInDoubleQuotes = "\"\\$`"

var ErrUnterminatedQuote = errors.New("unexpected EOF while looking for matching quote")

var ErrTrailingEscape = errors.New("unexpected EOF after escape character")

type lexer struct {
	input   []rune
	pos     int
	args    []string
	current strings.Builder
	inToken bool
}

func Parse(line string) ([]string, error) {
	lx := &lexer{input: []rune(line)}
	for !lx.done() {
		if err := lx.step(); err != nil {
			return nil, err
		}
	}
	lx.flush()
	return lx.args, nil
}

func (lx *lexer) done() bool {
	return lx.pos >= len(lx.input)
}

func (lx *lexer) next() rune {
	char := lx.input[lx.pos]
	lx.pos++
	return char
}

func (lx *lexer) write(char rune) {
	lx.current.WriteRune(char)
	lx.inToken = true
}

func (lx *lexer) flush() {
	if !lx.inToken {
		return
	}
	lx.args = append(lx.args, lx.current.String())
	lx.current.Reset()
	lx.inToken = false
}

func (lx *lexer) step() error {
	char := lx.next()
	switch {
	case unicode.IsSpace(char):
		lx.flush()
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

func (lx *lexer) readEscaped() error {
	if lx.done() {
		return ErrTrailingEscape
	}
	lx.write(lx.next())
	return nil
}

func unterminated(quote rune) error {
	return fmt.Errorf("%w `%c'", ErrUnterminatedQuote, quote)
}
