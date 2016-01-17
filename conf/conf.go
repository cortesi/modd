package conf

import (
	"fmt"
	"os"
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
	Watch          []string
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

// WatchPaths retreives the set of watched paths (with patterns removed) from
// all blocks. The path set is de-duplicated.
func (c *Config) WatchPaths() []string {
	return nil
}

func (c *Config) addBlock(b Block) {
	if c.Blocks == nil {
		c.Blocks = []Block{}
	}
	c.Blocks = append(c.Blocks, b)
}
