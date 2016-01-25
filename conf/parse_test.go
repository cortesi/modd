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
					Include: []string{"foo"},
				},
			},
		},
	},
	{
		"foo bar {}",
		&Config{
			[]Block{
				{
					Include: []string{"foo", "bar"},
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
		`foo +noignore {}`,
		&Config{
			[]Block{
				{
					Include:        []string{"foo"},
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
					Include: []string{"foo bar", "voing"},
				},
			},
		},
	},
	{
		"foo {\ndaemon: command\n}",
		&Config{
			[]Block{
				{
					Include: []string{"foo"},
					Daemons: []Daemon{{"command", syscall.SIGHUP}},
				},
			},
		},
	},
	{
		"{\ndaemon +sighup: c\n}",
		&Config{[]Block{{Daemons: []Daemon{{"c", syscall.SIGHUP}}}}},
	},
	{
		"{\ndaemon +sigterm: c\n}",
		&Config{[]Block{{Daemons: []Daemon{{"c", syscall.SIGTERM}}}}},
	},
	{
		"{\ndaemon +sigint: c\n}",
		&Config{[]Block{{Daemons: []Daemon{{"c", syscall.SIGINT}}}}},
	},
	{
		"{\ndaemon +sigkill: c\n}",
		&Config{[]Block{{Daemons: []Daemon{{"c", syscall.SIGKILL}}}}},
	},
	{
		"{\ndaemon +sigquit: c\n}",
		&Config{[]Block{{Daemons: []Daemon{{"c", syscall.SIGQUIT}}}}},
	},
	{
		"{\ndaemon +sigusr1: c\n}",
		&Config{[]Block{{Daemons: []Daemon{{"c", syscall.SIGUSR1}}}}},
	},
	{
		"{\ndaemon +sigusr2: c\n}",
		&Config{[]Block{{Daemons: []Daemon{{"c", syscall.SIGUSR2}}}}},
	},
	{
		"{\ndaemon +sigwinch: c\n}",
		&Config{[]Block{{Daemons: []Daemon{{"c", syscall.SIGWINCH}}}}},
	},
	{
		"foo {\nprep: command\n}",
		&Config{
			[]Block{
				{
					Include: []string{"foo"},
					Preps:   []Prep{Prep{Command: "command"}},
				},
			},
		},
	},
	{
		"foo {\nprep: 'command\n-one\n-two'}",
		&Config{
			[]Block{
				{
					Include: []string{"foo"},
					Preps:   []Prep{Prep{Command: "command\n-one\n-two"}},
				},
			},
		},
	},
	{
		"foo #comment\nbar\n#comment\n{\n#comment\nprep: command\n}",
		&Config{
			[]Block{
				{
					Include: []string{"foo", "bar"},
					Preps:   []Prep{Prep{Command: "command"}},
				},
			},
		},
	},
	{
		"foo #comment\n#comment\nbar { #comment \nprep: command\n}",
		&Config{
			[]Block{
				{
					Include: []string{"foo", "bar"},
					Preps:   []Prep{{"command"}},
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
	{"foo { daemon *: foo }", "test:1: invalid syntax"},
	{"foo { daemon +invalid: foo }", "test:1: unknown option: +invalid"},
	{"foo { prep +invalid: foo }", "test:1: unknown option: +invalid"},
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
