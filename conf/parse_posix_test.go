// +build !windows

package conf

import (
	"syscall"
)

var parsePosixTests = []struct {
	input    string
	expected *Config
}{
	{
		"{\ndaemon +sigusr1: c\n}",
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", syscall.SIGUSR1, false}}}}},
	},
	{
		"{\ndaemon +sigusr2: c\n}",
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", syscall.SIGUSR2, false}}}}},
	},
	{
		"{\ndaemon +sigwinch: c\n}",
		&Config{Blocks: []Block{{Daemons: []Daemon{{"c", syscall.SIGWINCH, false}}}}},
	},
}

func init() {
	parseTests = append(parseTests, parsePosixTests...)
}
