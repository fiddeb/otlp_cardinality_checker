package autotemplate

import "runtime"

// Config holds configuration for the template miner
type Config struct {
	// Number of shards for concurrent processing
	Shards int
	
	// Maximum depth of the parse tree
	MaxDepth int
	
	// Maximum children per internal node
	MaxChildren int
	
	// Maximum total clusters across all shards (LRU eviction when exceeded)
	MaxClusters int
	
	// Similarity threshold (0.0-1.0) for matching clusters
	SimThreshold float64
	
	// Extra delimiters beyond whitespace for tokenization
	ExtraDelimiters []rune
	
	// Training mode: if true, create new clusters; if false, match-only
	Training bool
}

// DefaultConfig returns sensible defaults for production use
func DefaultConfig() Config {
	nShards := runtime.NumCPU()
	if nShards < 4 {
		nShards = 4
	}
	return Config{
		Shards:          nShards,
		MaxDepth:        4,
		MaxChildren:     100,
		MaxClusters:     1000,
		SimThreshold:    0.5,
		ExtraDelimiters: []rune{':', '=', '/', '[', ']', '(', ')', ',', '"'},
		Training:        true,
	}
}
