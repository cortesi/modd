package conf

import (
	"reflect"
	"testing"
)

var parseTests = []struct {
	input    string
	expected *Config
}{
	{
		"",
		&Config{},
	},
	{
		"{}",
		&Config{[]Block{{}}},
	},
	{
		"foo {}",
		&Config{
			[]Block{
				{
					Patterns: []Pattern{{Spec: "foo"}},
				},
			},
		},
	},
	{
		"foo bar {}",
		&Config{
			[]Block{
				{
					Patterns: []Pattern{{Spec: "foo"}, {Spec: "bar"}},
				},
			},
		},
	},
	{
		"!foo {}",
		&Config{
			[]Block{
				{
					Patterns: []Pattern{{Spec: "foo", Filter: true}},
				},
			},
		},
	},
	{
		`!"foo" {}`,
		&Config{
			[]Block{
				{
					Patterns: []Pattern{{Spec: "foo", Filter: true}},
				},
			},
		},
	},
	{
		`!"foo" !'bar' !voing {}`,
		&Config{
			[]Block{
				{
					Patterns: []Pattern{
						{Spec: "foo", Filter: true},
						{Spec: "bar", Filter: true},
						{Spec: "voing", Filter: true},
					},
				},
			},
		},
	},
	{
		`foo +common {}`,
		&Config{
			[]Block{
				{
					Patterns:       []Pattern{{Spec: "foo"}},
					NoCommonFilter: true,
				},
			},
		},
	},
	{
		"'foo bar' voing {}",
		&Config{
			[]Block{
				{
					Patterns: []Pattern{{Spec: "foo bar"}, {Spec: "voing"}},
				},
			},
		},
	},
	{
		"foo {\ndaemon: command\n}",
		&Config{
			[]Block{
				{
					Patterns: []Pattern{{Spec: "foo"}},
					Daemons:  []string{"command\n"},
				},
			},
		},
	},
	{
		"foo {\nprep: command\n}",
		&Config{
			[]Block{
				{
					Patterns: []Pattern{{Spec: "foo"}},
					Preps:    []string{"command\n"},
				},
			},
		},
	},
}

func TestParse(t *testing.T) {
	for i, tt := range parseTests {
		ret, err := Parse("test", tt.input)
		if err != nil {
			t.Fatalf("%q - %s", tt.input, err)
		}
		if !reflect.DeepEqual(ret, tt.expected) {
			t.Errorf("%d %q\nexpected:\n\t%#v\ngot\n\t%#v", i, tt.input, tt.expected, ret)
		}
	}
}

var parseErrorTests = []struct {
	input string
	err   string
}{
	{"{", "test:1: unterminated block"},
	{"a", "test:1: expected block open parentheses, got \"\""},
}

func TestParseErrors(t *testing.T) {
	for i, tt := range parseErrorTests {
		v, err := Parse("test", tt.input)
		if err == nil {
			t.Fatalf("%d: Expected error, got %#v", i, v)
		}
		if err.Error() != tt.err {
			t.Errorf("Expected %q, got %q", err.Error(), tt.err)
		}
	}
}
