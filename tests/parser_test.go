package tests

import (
	"errors"
	"reflect"
	"testing"

	"vshell/src/shell"
)

var parseCases = []struct {
	name string
	line string
	want []string
}{
	{"empty line", "", nil},
	{"only whitespace", " \t \n", nil},
	{"plain words", "ls -l /tmp", []string{"ls", "-l", "/tmp"}},
	{"extra whitespace", "  cd \t dir  \n", []string{"cd", "dir"}},
	{"double quotes", `ls "my dir"`, []string{"ls", "my dir"}},
	{"single quotes", `ls 'my dir'`, []string{"ls", "my dir"}},
	{"quotes inside a word", `a"b c"d'e f'`, []string{"ab cde f"}},
	{"empty quoted arguments", `ls "" ''`, []string{"ls", "", ""}},
	{"single quote inside double", `"it's"`, []string{"it's"}},
	{"double quote inside single", `'say "hi"'`, []string{`say "hi"`}},
	{"escaped space", `cd my\ dir`, []string{"cd", "my dir"}},
	{"escaped quote", `ls \"a\'`, []string{"ls", `"a'`}},
	{"escaped quote inside double", `"a \"b\" \\"`, []string{`a "b" \`}},
	{"backslash kept inside double", `"a\nb"`, []string{`a\nb`}},
	{"backslash literal inside single", `'a\\b'`, []string{`a\\b`}},
	{"unicode", `ls "папка один" 文件`, []string{"ls", "папка один", "文件"}},
}

var parseErrorCases = []struct {
	name string
	line string
	want error
}{
	{"unterminated double quote", `ls "abc`, shell.ErrUnterminatedQuote},
	{"unterminated single quote", `ls 'abc`, shell.ErrUnterminatedQuote},
	{"escape at the end of double quotes", `ls "abc\`, shell.ErrUnterminatedQuote},
	{"escaped closing quote", `ls "abc\"`, shell.ErrUnterminatedQuote},
	{"trailing escape", `ls abc\`, shell.ErrTrailingEscape},
}

func TestParse(t *testing.T) {
	for _, tc := range parseCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := shell.Parse(tc.line)
			if err != nil {
				t.Fatalf("Parse(%q) returned error: %v", tc.line, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Parse(%q) = %q, want %q", tc.line, got, tc.want)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	for _, tc := range parseErrorCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := shell.Parse(tc.line)
			if !errors.Is(err, tc.want) {
				t.Errorf("Parse(%q) error = %v, want %v", tc.line, err, tc.want)
			}
			if got != nil {
				t.Errorf("Parse(%q) = %q, want no arguments", tc.line, got)
			}
		})
	}
}
