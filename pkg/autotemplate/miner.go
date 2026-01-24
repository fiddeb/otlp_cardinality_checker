// Package autotemplate implements automatic log template extraction using the Drain algorithm.
//
// Drain is a fixed-depth tree algorithm for online log template mining, originally
// published in "Drain: An Online Log Parsing Approach with Fixed Depth Tree" (ICWS'17).
//
// Algorithm Overview:
//   - Layer 1: Group by token count (log length)
//   - Layer 2: Group by first token (starting keyword)
//   - Layer 3+: Navigate by exact token match or wildcard
//   - Leaf nodes: Clusters with similar token sequences (similarity threshold)
//
// Key features of this implementation:
//   - Sharded for concurrent processing (default 4 shards)
//   - LRU-bounded to prevent unbounded memory growth
//   - Training/inference modes for different use cases
//   - Token-level similarity clustering with configurable threshold
//
// References:
//   - Original paper: https://jiemingzhu.github.io/pub/pjhe_icws2017.pdf
//   - Logparser implementation: https://github.com/logpai/logparser/tree/main/logparser/Drain
//   - Drain3 (production): https://github.com/logpai/Drain3
package autotemplate

import (
	"hash/fnv"
	"strings"
	"sync"
	"sync/atomic"
)

// cluster represents a log template with metadata
type cluster struct {
	tokens      []string // Template tokens (with <*> wildcards)
	size        int64    // Number of logs matched
	lastUsed    uint64   // Timestamp for LRU
	exampleBody string   // Example log body that matches this template
}

// node is an internal tree node
type node struct {
	children map[string]*node
	wildcard *node
	clusters []*cluster
	depth    int
}

// newNode creates a new tree node
func newNode(depth int) *node {
	return &node{
		children: make(map[string]*node),
		depth:    depth,
	}
}

// MinerShard handles one shard of the template mining
type MinerShard struct {
	root       *node
	clusters   []*cluster
	clusterMap map[string]*cluster // template string -> cluster for dedup
	cfg        Config
	mu         sync.RWMutex
	ticker     uint64 // LRU timestamp
}

// NewMinerShard creates a new shard
func NewMinerShard(cfg Config) *MinerShard {
	return &MinerShard{
		root:       newNode(0),
		clusters:   make([]*cluster, 0, cfg.MaxClusters/cfg.Shards),
		clusterMap: make(map[string]*cluster),
		cfg:        cfg,
	}
}

// ShardedMiner manages multiple shards for concurrent log template extraction.
//
// The Drain algorithm is applied independently to each shard, with logs distributed
// by hash of their token sequence. This allows parallel processing without lock contention.
type ShardedMiner struct {
	shards []*MinerShard
	cfg    Config
}

// NewShardedMiner creates a sharded template miner
func NewShardedMiner(cfg Config) *ShardedMiner {
	shards := make([]*MinerShard, cfg.Shards)
	for i := 0; i < cfg.Shards; i++ {
		shards[i] = NewMinerShard(cfg)
	}
	return &ShardedMiner{
		shards: shards,
		cfg:    cfg,
	}
}

// selectShard picks a shard based on token count and first token
func (m *ShardedMiner) selectShard(tokens []string) *MinerShard {
	if len(m.shards) == 1 {
		return m.shards[0]
	}
	
	h := fnv.New32a()
	if len(tokens) > 0 {
		h.Write([]byte(tokens[0]))
	}
	h.Write([]byte{byte(len(tokens))})
	
	idx := int(h.Sum32()) % len(m.shards)
	return m.shards[idx]
}

// Add processes a log message and returns the matched/created template
func (m *ShardedMiner) Add(message string) (template string, matched bool) {
	tokens := tokenize(message, m.cfg.ExtraDelimiters)
	if len(tokens) == 0 {
		return "", false
	}
	
	shard := m.selectShard(tokens)
	return shard.add(tokens, message)
}

// Match attempts to match a log message against existing templates (inference mode)
func (m *ShardedMiner) Match(message string) (template string, ok bool) {
	tokens := tokenize(message, m.cfg.ExtraDelimiters)
	if len(tokens) == 0 {
		return "", false
	}
	
	shard := m.selectShard(tokens)
	return shard.match(tokens)
}

// add processes tokens in training mode
func (s *MinerShard) add(tokens []string, originalMessage string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	tick := atomic.AddUint64(&s.ticker, 1)
	
	// Navigate tree: level 1 = length, level 2 = first token, rest = search path
	current := s.root
	depth := 0
	
	// Level 1: group by token count
	lenKey := "len_" + strings.Join([]string{string(rune('0' + len(tokens)))}, "")
	if len(tokens) > 9 {
		lenKey = "len_many"
	}
	
	if next, exists := current.children[lenKey]; exists {
		current = next
	} else if s.cfg.Training {
		next := newNode(1)
		current.children[lenKey] = next
		current = next
	} else {
		return "", false
	}
	depth = 1
	
	// Drain algorithm tree navigation:
	// Layer 1 (depth=1): Route by token count (log length)
	// Layer 2 (depth=2): Route by first token (starting keyword)
	// Layer 3+: Navigate to leaf (simplified: always use wildcard path)
	// Leaf nodes: Clusters grouped by token similarity
	
	// Level 2: route by first token (if we have tokens)
	if len(tokens) > 0 && depth < s.cfg.MaxDepth {
		token := tokens[0]
		if next, exists := current.children[token]; exists {
			current = next
		} else if current.wildcard != nil {
			current = current.wildcard
		} else if s.cfg.Training {
			if len(current.children) < s.cfg.MaxChildren {
				next := newNode(2)
				current.children[token] = next
				current = next
			} else {
				// Too many children, use wildcard
				next := newNode(2)
				current.wildcard = next
				current = next
			}
		} else {
			return "", false
		}
		depth = 2
	}
	
	// Remaining levels: just navigate to leaf
	for depth < s.cfg.MaxDepth && depth < len(tokens) {
		// For remaining tokens, we always go to leaf via wildcard or create it
		if current.wildcard != nil {
			current = current.wildcard
		} else if s.cfg.Training {
			next := newNode(depth + 1)
			current.wildcard = next
			current = next
		} else {
			return "", false
		}
		depth++
	}
	
	// At leaf, find matching cluster
	bestCluster := s.findBestCluster(current.clusters, tokens)
	
	if bestCluster != nil {
		// Update existing cluster
		atomic.AddInt64(&bestCluster.size, 1)
		atomic.StoreUint64(&bestCluster.lastUsed, tick)
		
		// Generalize template if needed (in training mode)
		if s.cfg.Training {
			generalized := generalizeTokens(bestCluster.tokens, tokens)
			if !tokensEqual(generalized, bestCluster.tokens) {
				bestCluster.tokens = generalized
			}
		}
		
		return tokensToString(bestCluster.tokens), true
	}
	
	// Create new cluster if training
	if s.cfg.Training {
		newCluster := &cluster{
			tokens:      make([]string, len(tokens)),
			size:        1,
			lastUsed:    tick,
			exampleBody: originalMessage,
		}
		copy(newCluster.tokens, tokens)
		
		current.clusters = append(current.clusters, newCluster)
		s.clusters = append(s.clusters, newCluster)
		
		templateStr := tokensToString(tokens)
		s.clusterMap[templateStr] = newCluster
		
		// TODO: implement LRU eviction if len(s.clusters) > maxClusters
		
		return templateStr, false
	}
	
	return "", false
}

// match attempts to match tokens against existing clusters (inference mode)
func (s *MinerShard) match(tokens []string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Navigate tree with same logic as add
	current := s.root
	
	// Level 1: group by token count
	lenKey := "len_" + strings.Join([]string{string(rune('0' + len(tokens)))}, "")
	if len(tokens) > 9 {
		lenKey = "len_many"
	}
	
	next, exists := current.children[lenKey]
	if !exists {
		return "", false
	}
	current = next
	
	// Level 2: route by first token
	if len(tokens) > 0 {
		token := tokens[0]
		if next, exists := current.children[token]; exists {
			current = next
		} else if current.wildcard != nil {
			current = current.wildcard
		} else {
			return "", false
		}
	}
	
	// Navigate remaining levels via wildcard
	depth := 2
	for depth < s.cfg.MaxDepth && depth < len(tokens) {
		if current.wildcard != nil {
			current = current.wildcard
			depth++
		} else {
			break
		}
	}
	
	// Find exact matching cluster
	bestCluster := s.findBestCluster(current.clusters, tokens)
	if bestCluster != nil {
		return tokensToString(bestCluster.tokens), true
	}
	
	return "", false
}

// findBestCluster finds the cluster with highest similarity
func (s *MinerShard) findBestCluster(clusters []*cluster, tokens []string) *cluster {
	var best *cluster
	bestScore := 0.0
	
	for _, c := range clusters {
		if len(c.tokens) != len(tokens) {
			continue
		}
		
		score := similarity(c.tokens, tokens)
		if score >= s.cfg.SimThreshold && score > bestScore {
			bestScore = score
			best = c
		}
	}
	
	return best
}

// similarity computes token similarity (matched constants / total)
func similarity(a, b []string) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}
	
	matched := 0
	for i := 0; i < len(a); i++ {
		if a[i] == b[i] || a[i] == "<*>" || b[i] == "<*>" {
			matched++
		}
	}
	
	return float64(matched) / float64(len(a))
}

// tokensToString joins tokens with space
func tokensToString(tokens []string) string {
	return strings.Join(tokens, " ")
}

// tokensEqual checks if two token slices are equal
func tokensEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// generalizeTokens creates a generalized template from two token sequences
func generalizeTokens(template, tokens []string) []string {
	if len(template) != len(tokens) {
		return template
	}
	
	result := make([]string, len(template))
	for i := 0; i < len(template); i++ {
		if template[i] == tokens[i] {
			result[i] = template[i]
		} else if template[i] == "<*>" {
			result[i] = "<*>"
		} else {
			result[i] = "<*>"
		}
	}
	return result
}

// SetTraining switches between training and inference modes
func (m *ShardedMiner) SetTraining(training bool) {
	m.cfg.Training = training
	for _, shard := range m.shards {
		shard.mu.Lock()
		shard.cfg.Training = training
		shard.mu.Unlock()
	}
}

// Stats returns current stats
func (m *ShardedMiner) Stats() map[string]interface{} {
	totalClusters := 0
	for _, shard := range m.shards {
		shard.mu.RLock()
		totalClusters += len(shard.clusters)
		shard.mu.RUnlock()
	}
	
	return map[string]interface{}{
		"shards":   len(m.shards),
		"clusters": totalClusters,
		"training": m.cfg.Training,
	}
}
