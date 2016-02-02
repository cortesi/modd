package conf

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"syscall"

	"github.com/cortesi/modd/filter"
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
	Blocks    []Block
	variables map[string]string
}

// Equals checks if this Config equals another
func (c *Config) Equals(other *Config) bool {
	if (c.Blocks != nil || len(c.Blocks) != 0) || (other.Blocks != nil || len(other.Blocks) != 0) {
		if !reflect.DeepEqual(c.Blocks, other.Blocks) {
			return false
		}
	}
	if (c.variables != nil || len(c.variables) != 0) || (other.variables != nil || len(other.variables) != 0) {
		if !reflect.DeepEqual(c.variables, other.variables) {
			return false
		}
	}
	return true
}

// WatchPaths retreives the set of watched paths (with patterns removed) from
// all blocks. The path set is de-duplicated.
func (c *Config) WatchPaths() []string {
	paths := []string{}
	for _, b := range c.Blocks {
		paths = filter.GetBasePaths(paths, b.Include)
	}
	sort.Strings(paths)
	return paths
}

func (c *Config) addBlock(b Block) {
	if c.Blocks == nil {
		c.Blocks = []Block{}
	}
	c.Blocks = append(c.Blocks, b)
}

func (c *Config) addVariable(key string, value string) error {
	if c.variables == nil {
		c.variables = map[string]string{}
	}
	c.variables[key] = value
	return nil
}

// GetVariables returns a copy of the Variables map
func (c *Config) GetVariables() map[string]string {
	n := map[string]string{}
	for k, v := range c.variables {
		n[k] = v
	}
	return n
}
