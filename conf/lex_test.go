package conf

import (
	"fmt"
	"reflect"
	"testing"
)

type itm struct {
	typ itemType
	val string
}

func (i itm) String() string {
	return fmt.Sprintf("(%s %q)", i.typ, i.val)
}

func lexcollect(l *lexer) []itm {
	back := []itm{}
	for {
		nxt := l.nextItem()
		if nxt.typ == itemEOF {
			break
		}
		if nxt.typ != itemSpace {
			back = append(back, itm{nxt.typ, nxt.val})
		}
		if nxt.typ == itemError {
			break
		}
	}
	return back
}

var lexTests = []struct {
	input    string
	expected []itm
}{
	{"one", []itm{{itemBareString, "one"}}},
	{
		" one ", []itm{
			{itemBareString, "one"},
		},
	},
	{
		"# two three", []itm{
			{itemComment, "# two three"},
		},
	},
	{
		"# one two\n# three four", []itm{
			{itemComment, "# one two\n"},
			{itemComment, "# three four"},
		},
	},
	{
		"one # two three", []itm{
			{itemBareString, "one"},
			{itemComment, "# two three"},
		},
	},
	{
		"'foo'", []itm{
			{itemQuotedString, "'foo'"},
		},
	},
	{
		`'foo\bar'`, []itm{
			{itemQuotedString, `'foo\bar'`},
		},
	},
	{
		`'foo\'bar'`, []itm{
			{itemQuotedString, `'foo\'bar'`},
		},
	},
	{
		`'foo''bar'`, []itm{
			{itemQuotedString, `'foo'`},
			{itemQuotedString, `'bar'`},
		},
	},
	{
		"one {\ndaemon: foo\n}", []itm{
			{itemBareString, "one"},
			{itemLeftParen, "{"},
			{itemDaemon, "daemon:"},
			{itemBareString, "foo\n"},
			{itemRightParen, "}"},
		},
	},
	{
		"one { daemon: command\nprep: command\nexclude: command\n}", []itm{
			{itemBareString, "one"},
			{itemLeftParen, "{"},
			{itemDaemon, "daemon:"},
			{itemBareString, "command\n"},

			{itemPrep, "prep:"},
			{itemBareString, "command\n"},

			{itemExclude, "exclude:"},
			{itemBareString, "command"},

			{itemRightParen, "}"},
		},
	},
	{
		`"one{" { daemon: "two}"}`, []itm{
			{itemQuotedString, "\"one{\""},
			{itemLeftParen, "{"},
			{itemDaemon, "daemon:"},
			{itemQuotedString, "\"two}\""},
			{itemRightParen, "}"},
		},
	},
	{
		"# comment\none two # comment2\n\tthree{ daemon: foo\n   }", []itm{
			{itemComment, "# comment\n"},
			{itemBareString, "one"},
			{itemBareString, "two"},
			{itemComment, "# comment2\n"},
			{itemBareString, "three"},
			{itemLeftParen, "{"},
			{itemDaemon, "daemon:"},
			{itemBareString, "foo\n"},
			{itemRightParen, "}"},
		},
	},
}

func TestLex(t *testing.T) {
	for i, tt := range lexTests {
		ret := lexcollect(lex("test", tt.input))
		if !reflect.DeepEqual(ret, tt.expected) {
			t.Errorf("%d %q - expected\n%v\ngot\n%v", i, tt.input, tt.expected, ret)
		}
	}
}

var lexErrorTests = []struct {
	input string
	error string
	pos   Pos
}{
	{"'", "unterminated quoted string", 1},
	{"'\\", "unterminated quoted string", 2},
	{"  '\nfoo", "unterminated quoted string", 7},
	{"foo }", "invalid input", 5},
	{"{", "unterminated block", 1},
	{"{{}", "invalid input", 2},
	{"{daemon: '}", "unterminated quoted string", 11},
	{"{#}", "unterminated block", 3},
}

func TestLexErrors(t *testing.T) {
	for i, tt := range lexErrorTests {
		l := lex("test", tt.input)
		ret := lexcollect(l)
		itm := ret[len(ret)-1]
		if itm.typ != itemError {
			t.Errorf("%d: %q - Expected error, got %s", i, tt.input, itm)
		}
		if itm.val != tt.error {
			t.Errorf("%d: %q - Expected error value %s, got %s", i, tt.input, tt.error, itm.val)
		}
		if tt.pos != l.pos {
			t.Errorf("%d: %q - Expected position %s, got %s", i, tt.input, tt.pos, l.pos)
		}
	}
}
