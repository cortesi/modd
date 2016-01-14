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
					Patterns: []string{"foo"},
				},
			},
		},
	},
	{
		"foo bar {}",
		&Config{
			[]Block{
				{
					Patterns: []string{"foo", "bar"},
				},
			},
		},
	},
	{
		"'foo bar' voing {}",
		&Config{
			[]Block{
				{
					Patterns: []string{"foo bar", "voing"},
				},
			},
		},
	},
}

func TestParse(t *testing.T) {
	for i, tt := range parseTests {
		ret, err := Parse("test", tt.input)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(ret, tt.expected) {
			t.Errorf("%d - expected\n%#v\ngot\n%#v", i, tt.expected, ret)
		}
	}
}

var parseErrorTests = []struct {
	input string
	err   string
}{
	{"{", "test:1: unterminated block"},
	{"a", "test:1: expected block open parentheses"},
}

func TestParseErrors(t *testing.T) {
	for i, tt := range parseErrorTests {
		_, err := Parse("test", tt.input)
		if err == nil {
			t.Fatalf("%d: Expected error", i)
		}
		if err.Error() != tt.err {
			t.Errorf("Expected %q, got %q", err.Error(), tt.err)
		}
	}
}
