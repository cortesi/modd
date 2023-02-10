package conf

import (
	"fmt"
	"os"
	"sort"
)

// A Daemon is a persistent process that is kept running
type Daemon struct {
	Command       string
	RestartSignal os.Signal
}

// A Prep runs and terminates
type Prep struct {
	Command  string
	Onchange bool // Should prep skip initial run
}

// Block is a match pattern and a set of specifications
type Block struct {
	Include        []string
	Exclude        []string
	NoCommonFilter bool
	InDir          string
	// todo: cumulative env with envEndIdx
	// envEndIdx int
	Env []string

	Daemons []Daemon
	Preps   []Prep
}

func (b *Block) addPrep(command string, options []string) error {
	if b.Preps == nil {
		b.Preps = []Prep{}
	}

	onchange := false
	for _, v := range options {
		switch v {
		case "+onchange":
			onchange = true
		default:
			return fmt.Errorf("unknown option: %s", v)
		}
	}

	prep := Prep{os.ExpandEnv(command), onchange}

	b.Preps = append(b.Preps, prep)
	return nil
}

// Config represents a complete configuration
type Config struct {
	Blocks    []Block
	variables map[string]string
}

// IncludePatterns retrieves all include patterns from all blocks.
func (c *Config) IncludePatterns() []string {
	pmap := map[string]bool{}
	for _, b := range c.Blocks {
		for _, p := range b.Include {
			pmap[p] = true
		}
	}
	paths := make([]string, len(pmap))
	i := 0
	for k := range pmap {
		paths[i] = k
		i++
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
	if _, ok := c.variables[key]; ok {
		return fmt.Errorf("variable %s shadows previous declaration", key)
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

// CommonExcludes extends all blocks that require it with a common exclusion
// set
func (c *Config) CommonExcludes(excludes []string) {
	for i, b := range c.Blocks {
		if !b.NoCommonFilter {
			b.Exclude = append(b.Exclude, excludes...)
		}
		c.Blocks[i] = b
	}
}
