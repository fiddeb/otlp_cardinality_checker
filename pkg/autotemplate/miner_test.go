package autotemplate

import (
	"strings"
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		extraDelimiters []rune
		want            []string
	}{
		{
			name:  "whitespace only",
			input: "user john logged in",
			want:  []string{"user", "john", "logged", "in"},
		},
		{
			name:            "with delimiters",
			input:           "user:john logged=in",
			extraDelimiters: []rune{':', '='},
			want:            []string{"user", "john", "logged", "in"},
		},
		{
			name:  "multiple spaces",
			input: "user  john   logged",
			want:  []string{"user", "john", "logged"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenize(tt.input, tt.extraDelimiters)
			if len(got) != len(tt.want) {
				t.Errorf("tokenize() len = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("tokenize()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want float64
	}{
		{
			name: "exact match",
			a:    []string{"user", "john", "logged", "in"},
			b:    []string{"user", "john", "logged", "in"},
			want: 1.0,
		},
		{
			name: "partial match",
			a:    []string{"user", "<*>", "logged", "in"},
			b:    []string{"user", "jane", "logged", "in"},
			want: 1.0, // wildcard matches
		},
		{
			name: "different length",
			a:    []string{"user", "john"},
			b:    []string{"user", "john", "logged"},
			want: 0.0,
		},
		{
			name: "50% match",
			a:    []string{"user", "john", "logged", "out"},
			b:    []string{"user", "jane", "logged", "in"},
			want: 0.5,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := similarity(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("similarity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShardedMiner(t *testing.T) {
	cfg := Config{
		Shards:       2,
		MaxDepth:     4,
		MaxChildren:  100,
		MaxClusters:  1000,
		SimThreshold: 0.5, // 50% threshold
		Training:     true,
	}
	
	miner := NewShardedMiner(cfg)
	
	// Add similar messages
	template1, matched1 := miner.Add("user john logged in")
	if matched1 {
		t.Error("first message should not match existing cluster")
	}
	if template1 != "user john logged in" {
		t.Errorf("template1 = %v, want 'user john logged in'", template1)
	}
	
	// Second similar message should match (3/4 tokens = 75% > 50%)
	template2, matched2 := miner.Add("user jane logged in")
	if !matched2 {
		t.Error("second message should match existing cluster (75% similarity)")
	}
	// Should have generalized to include wildcard
	if !strings.Contains(template2, "<*>") {
		t.Errorf("template2 = %v, should contain wildcard", template2)
	}
	if !strings.Contains(template2, "user") || !strings.Contains(template2, "logged") {
		t.Errorf("template2 = %v, should contain 'user' and 'logged'", template2)
	}
	
	// Add different message
	template3, matched3 := miner.Add("error connecting to database")
	if matched3 {
		t.Error("different message should not match existing cluster")
	}
	if template3 != "error connecting to database" {
		t.Errorf("template3 = %v, want 'error connecting to database'", template3)
	}
}

func TestInferenceMode(t *testing.T) {
	// Train first
	trainCfg := Config{
		Shards:       1,
		MaxDepth:     4,
		MaxChildren:  100,
		MaxClusters:  1000,
		SimThreshold: 0.5, // Lower threshold for testing
		Training:     true,
	}
	
	miner := NewShardedMiner(trainCfg)
	miner.Add("user john logged in")
	miner.Add("user jane logged in") // This should generalize to "user <*> logged in"
	miner.Add("error connecting to database")
	
	// Switch to inference mode
	miner.cfg.Training = false
	for _, shard := range miner.shards {
		shard.cfg.Training = false
	}
	
	// Should match known patterns
	template, ok := miner.Match("user bob logged in")
	if !ok {
		t.Error("should match known pattern")
	}
	t.Logf("Matched template: %s", template)
	if !strings.Contains(template, "user") || !strings.Contains(template, "logged") {
		t.Errorf("template = %v, should contain 'user' and 'logged'", template)
	}
	
	// Should not match unknown pattern
	_, ok = miner.Match("totally new pattern here")
	if ok {
		t.Error("should not match unknown pattern in inference mode")
	}
}

func BenchmarkMinerAdd(b *testing.B) {
	cfg := DefaultConfig()
	cfg.Shards = 4
	miner := NewShardedMiner(cfg)
	
	messages := []string{
		"user john logged in from 192.168.1.1",
		"user jane logged out at 12:34:56",
		"error connecting to database server",
		"request GET /api/users/123 completed in 45ms",
		"cache hit for key user:456",
		"starting background job worker-5",
		"received message on queue orders",
		"authentication failed for user@example.com",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := messages[i%len(messages)]
		miner.Add(msg)
	}
}

func BenchmarkMinerMatch(b *testing.B) {
	cfg := DefaultConfig()
	cfg.Shards = 4
	miner := NewShardedMiner(cfg)
	
	// Pre-train
	messages := []string{
		"user john logged in from 192.168.1.1",
		"user jane logged out at 12:34:56",
		"error connecting to database server",
		"request GET /api/users/123 completed in 45ms",
	}
	
	for _, msg := range messages {
		miner.Add(msg)
	}
	
	// Switch to inference
	miner.cfg.Training = false
	for _, shard := range miner.shards {
		shard.cfg.Training = false
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := messages[i%len(messages)]
		miner.Match(msg)
	}
}
