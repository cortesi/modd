package conf

import (
	"fmt"
	"os"
	"path"
	"strings"
	"syscall"
)

// A Daemon is a persistent process that is kept running
type Daemon struct {
	Command       string
	RestartSignal os.Signal
}

// A Prep runs and terminates
type Prep struct {
	Command string
}

// Block is a match pattern and a set of specifications
type Block struct {
	Include        []string
	Exclude        []string
	NoCommonFilter bool

	Daemons []Daemon
	Preps   []Prep
}

func (b *Block) addDaemon(command string, options []string) error {
	if b.Daemons == nil {
		b.Daemons = []Daemon{}
	}
	d := Daemon{
		Command:       command,
		RestartSignal: syscall.SIGHUP,
	}
	for _, v := range options {
		switch v {
		case "+sighup":
			d.RestartSignal = syscall.SIGHUP
		case "+sigterm":
			d.RestartSignal = syscall.SIGTERM
		case "+sigint":
			d.RestartSignal = syscall.SIGINT
		case "+sigkill":
			d.RestartSignal = syscall.SIGKILL
		case "+sigquit":
			d.RestartSignal = syscall.SIGQUIT
		case "+sigusr1":
			d.RestartSignal = syscall.SIGUSR1
		case "+sigusr2":
			d.RestartSignal = syscall.SIGUSR2
		case "+sigwinch":
			d.RestartSignal = syscall.SIGWINCH
		default:
			return fmt.Errorf("unknown option: %s", v)
		}
	}
	b.Daemons = append(b.Daemons, d)
	return nil
}

func (b *Block) addPrep(command string, options []string) error {
	if b.Preps == nil {
		b.Preps = []Prep{}
	}
	prep := Prep{command}
	for _, v := range options {
		switch v {
		// No prep options for the moment
		default:
			return fmt.Errorf("unknown option: %s", v)
		}
	}
	b.Preps = append(b.Preps, prep)
	return nil
}

// Config represents a complete configuration
type Config struct {
	Blocks []Block
}

// Returns the base path for a match pattern
func basePath(pattern string) string {
	split := strings.IndexAny(pattern, "*{}?[]")
	if split >= 0 {
		pattern = pattern[:split]
	}
	dir, _ := path.Split(pattern)
	return dir
}

// WatchPaths retreives the set of watched paths (with patterns removed) from
// all blocks. The path set is de-duplicated.
// FIXME: return a consistent order here, so we can test without mysterious
// errors
func (c *Config) WatchPaths() []string {
	m := make(map[string]bool)
	for _, b := range c.Blocks {
		for _, p := range b.Include {
			m[basePath(p)] = true
		}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		if k == "" {
			keys = append(keys, ".")
		} else {
			keys = append(keys, k)
		}
	}
	return keys
}

func (c *Config) addBlock(b Block) {
	if c.Blocks == nil {
		c.Blocks = []Block{}
	}
	c.Blocks = append(c.Blocks, b)
}
