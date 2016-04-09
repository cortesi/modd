package conf

import (
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
		&Config{
			Blocks: []Block{
				{},
			},
		},
	},
	{
		"foo {}",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
				},
			},
		},
	},
	{
		"foo bar {}",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo", "bar"},
				},
			},
		},
	},
	{
		"!foo {}",
		&Config{
			Blocks: []Block{
				{
					Exclude: []string{"foo"},
				},
			},
		},
	},
	{
		`!"foo" {}`,
		&Config{
			Blocks: []Block{
				{
					Exclude: []string{"foo"},
				},
			},
		},
	},
	{
		`!"foo" !'bar' !voing {}`,
		&Config{
			Blocks: []Block{
				{Exclude: []string{"foo", "bar", "voing"}},
			},
		},
	},
	{
		`foo +noignore {}`,
		&Config{
			Blocks: []Block{
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
			Blocks: []Block{
				{
					Include: []string{"foo bar", "voing"},
				},
			},
		},
	},
	{
		"foo {\ndaemon: command\n}",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
					Daemons: []Daemon{{"command", syscall.SIGHUP}},
				},
			},
		},
	},
	{
		"{\ndaemon +sighup: c\n}",
		&Config{
			Blocks: []Block{
				{Daemons: []Daemon{{"c", syscall.SIGHUP}}},
			},
		},
	},
	{
		"{\ndaemon +sigterm: c\n}",
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", syscall.SIGTERM}}}}},
	},
	{
		"{\ndaemon +sigint: c\n}",
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", syscall.SIGINT}}}}},
	},
	{
		"{\ndaemon +sigkill: c\n}",
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", syscall.SIGKILL}}}}},
	},
	{
		"{\ndaemon +sigquit: c\n}",
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", syscall.SIGQUIT}}}}},
	},
	{
		"foo {\nprep: command\n}",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
					Preps:   []Prep{Prep{Command: "command"}},
				},
			},
		},
	},
	{
		"foo {\nprep +onchange: command\n}",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
					Preps:   []Prep{Prep{Command: "command", Onchange: true}},
				},
			},
		},
	},
	{
		"foo {\nprep: 'command\n-one\n-two'}",
		&Config{
			Blocks: []Block{
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
			Blocks: []Block{
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
			Blocks: []Block{
				{
					Include: []string{"foo", "bar"},
					Preps:   []Prep{{"command", false}},
				},
			},
		},
	},
	{
		"@var=bar\nfoo {}",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
				},
			},
			variables: map[string]string{
				"@var": "bar",
			},
		},
	},
	{
		"@var='bar\nvoing'\nfoo {}",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
				},
			},
			variables: map[string]string{
				"@var": "bar\nvoing",
			},
		},
	},
	{
		"foo {}\n@var=bar\n",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
				},
			},
			variables: map[string]string{
				"@var": "bar",
			},
		},
	},
	{
		"@oink=foo\nfoo {}\n@var=bar\n",
		&Config{
			Blocks: []Block{
				{
					Include: []string{"foo"},
				},
			},
			variables: map[string]string{
				"@var":  "bar",
				"@oink": "foo",
			},
		},
	},
	{
		"{ indir: foo\n }",
		&Config{
			Blocks: []Block{
				{InDir: "foo"},
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
		if !ret.Equals(tt.expected) {
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
	{"@foo bar {}", "test:1: Expected ="},
	{"@foo =", "test:1: unterminated variable assignment"},
	{"@foo=bar\n@foo=bar {}", "test:2: variable @foo shadows previous declaration"},
	{"{indir +foo: bar\n}", "test:1: indir takes no options"},
	{"{indir: bar\nindir: voing\n}", "test:2: indir can only be used once per block"},
}

func TestErrorsParse(t *testing.T) {
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
