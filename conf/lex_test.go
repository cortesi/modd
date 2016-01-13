package conf

import (
	"reflect"
	"testing"
)

type itm struct {
	typ itemType
	val string
}

func lextester(s string, expected []itm) []itm {
	l := lex("test", s)
	back := []itm{}
	for {
		nxt := l.nextItem()
		if nxt.typ == itemEOF {
			break
		}
		back = append(back, itm{nxt.typ, nxt.val})
	}
	return back
}

var lexTests = []struct {
	input    string
	expected []itm
}{
	{"   ", []itm{{itemSpace, "   "}}},
	// {"one two three", []string{"one", "two", "three"}},
	// {"one # two three", []string{"one", "# two three"}},
	// {"# one two three", []string{"# one two three"}},
	// {"{one}", []string{"{", "one", "}"}},
	// {"prep: one", []string{"prep:", "one"}},
	// {"prep : one", []string{"prep :", "one"}},
	// {"daemon: one", []string{"daemon:", "one"}},
	// {"daemon : one", []string{"daemon :", "one"}},
}

func TestLex(t *testing.T) {
	for i, tt := range lexTests {
		ret := lextester(tt.input, tt.expected)
		if !reflect.DeepEqual(ret, tt.expected) {
			t.Errorf("%d - expected %#v, got %#v", i, tt.expected, ret)
		}
	}
}
