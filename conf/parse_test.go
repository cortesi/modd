package conf

import (
	"reflect"
	"syscall"
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
					Watch: []string{"foo"},
				},
			},
		},
	},
	{
		"foo bar {}",
		&Config{
			[]Block{
				{
					Watch: []string{"foo", "bar"},
				},
			},
		},
	},
	{
		"!foo {}",
		&Config{
			[]Block{
				{
					Exclude: []string{"foo"},
				},
			},
		},
	},
	{
		`!"foo" {}`,
		&Config{
			[]Block{
				{
					Exclude: []string{"foo"},
				},
			},
		},
	},
	{
		`!"foo" !'bar' !voing {}`,
		&Config{
			[]Block{
				{Exclude: []string{"foo", "bar", "voing"}},
			},
		},
	},
	{
		`foo +common {}`,
		&Config{
			[]Block{
				{
					Watch:          []string{"foo"},
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
					Watch: []string{"foo bar", "voing"},
				},
			},
		},
	},
	{
		"foo {\ndaemon: command\n}",
		&Config{
			[]Block{
				{
					Watch:   []string{"foo"},
					Daemons: []Daemon{{"command\n", syscall.SIGHUP}},
				},
			},
		},
	},
	{
		"{\ndaemon +sighup: c\n}",
		&Config{[]Block{{Daemons: []Daemon{{"c\n", syscall.SIGHUP}}}}},
	},
	{
		"{\ndaemon +sigterm: c\n}",
		&Config{[]Block{{Daemons: []Daemon{{"c\n", syscall.SIGTERM}}}}},
	},
	{
		"{\ndaemon +sigint: c\n}",
		&Config{[]Block{{Daemons: []Daemon{{"c\n", syscall.SIGINT}}}}},
	},
	{
		"{\ndaemon +sigkill: c\n}",
		&Config{[]Block{{Daemons: []Daemon{{"c\n", syscall.SIGKILL}}}}},
	},
	{
		"{\ndaemon +sigquit: c\n}",
		&Config{[]Block{{Daemons: []Daemon{{"c\n", syscall.SIGQUIT}}}}},
	},
	{
		"{\ndaemon +sigusr1: c\n}",
		&Config{[]Block{{Daemons: []Daemon{{"c\n", syscall.SIGUSR1}}}}},
	},
	{
		"{\ndaemon +sigusr2: c\n}",
		&Config{[]Block{{Daemons: []Daemon{{"c\n", syscall.SIGUSR2}}}}},
	},
	{
		"{\ndaemon +sigwinch: c\n}",
		&Config{[]Block{{Daemons: []Daemon{{"c\n", syscall.SIGWINCH}}}}},
	},
	{
		"foo {\nprep: command\n}",
		&Config{
			[]Block{
				{
					Watch: []string{"foo"},
					Preps: []Prep{Prep{Command: "command\n"}},
				},
			},
		},
	},
	{
		"foo #comment\nbar\n#comment\n{\n#comment\nprep: command\n}",
		&Config{
			[]Block{
				{
					Watch: []string{"foo", "bar"},
					Preps: []Prep{Prep{Command: "command\n"}},
				},
			},
		},
	},
	{
		"foo #comment\n#comment\nbar { #comment \nprep: command\n}",
		&Config{
			[]Block{
				{
					Watch: []string{"foo", "bar"},
					Preps: []Prep{{"command\n"}},
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
	{`foo { "bar": "bar" }`, "test:1: invalid input"},
	{"foo { daemon: \n }", "test:1: empty command specification"},
	{"foo { daemon: \" }", "test:1: unterminated quoted string"},
}

func TestParseErrors(t *testing.T) {
	for i, tt := range parseErrorTests {
		v, err := Parse("test", tt.input)
		if err == nil {
			t.Fatalf("%d: Expected error, got %#v", i, v)
		}
		if err.Error() != tt.err {
			t.Errorf("Expected\n%q\ngot\n%q", tt.err, err.Error())
		}
	}
}
