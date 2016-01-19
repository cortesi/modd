package modd

import "testing"

var shortCommandTests = []struct {
	command  string
	expected string
}{
	{"one", "one"},
	{"one\ntwo", "one"},
	{"one\\\ntwo", "one"},
	{"\n\none\\\ntwo", "one"},
	{"\n   \none\\\ntwo", "one"},

	{"# one", "one"},
	{"\n\n# one\\\ntwo", "one"},
	{"\n   \n# one\\\ntwo", "one"},
}

func TestShortCommand(t *testing.T) {
	for i, tst := range shortCommandTests {
		result := shortCommand(tst.command)
		if result != tst.expected {
			t.Errorf("Test %d: expected\n%q\ngot\n%q", i, tst.expected, result)
		}
	}
}
