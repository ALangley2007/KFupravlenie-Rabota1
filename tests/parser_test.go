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
	{"comment line", "// just a comment", nil},
	{"indented comment", " \t// comment", nil},
	{"trailing comment", `ls -l // list "files"`, []string{"ls", "-l"}},
	{"comment right after a tab", "ls\t//x", []string{"ls"}},
	{"comment without text", "ls //", []string{"ls"}},
	{"comment hides quotes", `ls // "unterminated`, []string{"ls"}},
	{"double slash inside a word", "ls a//b /usr//lib", []string{"ls", "a//b", "/usr//lib"}},
	{"double slash after quotes", `ls "a"//b`, []string{"ls", "a//b"}},
	{"quoted double slash", `ls "// x" '//y'`, []string{"ls", "// x", "//y"}},
	{"escaped double slash", `ls \//x`, []string{"ls", "//x"}},
	{"single slashes", "ls / /a / b", []string{"ls", "/", "/a", "/", "b"}},
	{"tilde alone", "cd ~", []string{"cd", "/home/alice"}},
	{"tilde with a path", "cd ~/docs/x", []string{"cd", "/home/alice/docs/x"}},
	{"tilde before a comment", "ls ~ // home", []string{"ls", "/home/alice"}},
	{"quoted and escaped tildes", `ls "~" '~/x' \~ ~"/x"`, []string{"ls", "~", "~/x", "~", "~/x"}},
	{"tilde inside a word", "ls a~ a/~", []string{"ls", "a~", "a/~"}},
	{"tilde with a user name", "ls ~bob ~+", []string{"ls", "~bob", "~+"}},
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

// TestParse checks that valid lines, including ones with comments, split into the expected arguments.
func TestParse(t *testing.T) {
	for _, tc := range parseCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := shell.Parse(tc.line, testHost.Home)
			if err != nil {
				t.Fatalf("Parse(%q) returned error: %v", tc.line, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Parse(%q) = %q, want %q", tc.line, got, tc.want)
			}
		})
	}
}

// TestParseErrors checks that malformed lines fail with the matching error and no arguments.
func TestParseErrors(t *testing.T) {
	for _, tc := range parseErrorCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := shell.Parse(tc.line, testHost.Home)
			if !errors.Is(err, tc.want) {
				t.Errorf("Parse(%q) error = %v, want %v", tc.line, err, tc.want)
			}
			if got != nil {
				t.Errorf("Parse(%q) = %q, want no arguments", tc.line, got)
			}
		})
	}
}

// TestParseWithoutHome checks that a tilde stays literal when no home directory is known.
func TestParseWithoutHome(t *testing.T) {
	got, err := shell.Parse("cd ~ ~/x", "")
	if want := []string{"cd", "~", "~/x"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Parse() = %q, %v, want %q", got, err, want)
	}
}
