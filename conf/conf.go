package conf

// Block is a match pattern and a set of specifications
type Block struct {
	Patterns []string
	Daemons  []string
	Preps    []string
	Excludes []string
}

// Config represents a complete configuration
type Config struct {
	Blocks []Block
}
