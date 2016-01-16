package conf

// Block is a match pattern and a set of specifications
type Block struct {
	Patterns []string
	Daemons  []string
	Preps    []string
	Excludes []string
}

func (b *Block) addDaemon(s string) {
	if b.Daemons == nil {
		b.Daemons = []string{}
	}
	b.Daemons = append(b.Daemons, s)
}

func (b *Block) addPrep(s string) {
	if b.Preps == nil {
		b.Preps = []string{}
	}
	b.Preps = append(b.Preps, s)
}

// Config represents a complete configuration
type Config struct {
	Blocks []Block
}

func (c *Config) addBlock(b Block) {
	if c.Blocks == nil {
		c.Blocks = []Block{}
	}
	c.Blocks = append(c.Blocks, b)
}
