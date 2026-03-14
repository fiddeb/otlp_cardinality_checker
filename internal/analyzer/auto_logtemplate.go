package analyzer

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/fidde/otlp_cardinality_checker/internal/patterns"
	"github.com/fidde/otlp_cardinality_checker/pkg/autotemplate"
)

// AutoLogBodyAnalyzer uses the autotemplate miner for log template extraction.
// Pre-masking with regex patterns was removed: Drain generalises variable
// tokens (IPs, numbers, UUIDs, etc.) to <*> naturally, making the regex
// pass redundant on the log hot path.
type AutoLogBodyAnalyzer struct {
	mu        sync.RWMutex
	miner     *autotemplate.ShardedMiner
	templates map[string]*LogTemplate // template string -> metadata
	total     int64
}

// NewAutoLogBodyAnalyzer creates a new auto-template analyzer
func NewAutoLogBodyAnalyzer(minerCfg autotemplate.Config) *AutoLogBodyAnalyzer {
	return NewAutoLogBodyAnalyzerWithPatterns(minerCfg, nil)
}

// NewAutoLogBodyAnalyzerWithPatterns creates analyzer.
// The pats parameter is accepted for API compatibility but ignored;
// Drain generalises variable tokens without regex pre-masking.
func NewAutoLogBodyAnalyzerWithPatterns(minerCfg autotemplate.Config, _ []patterns.CompiledPattern) *AutoLogBodyAnalyzer {
	miner := autotemplate.NewShardedMiner(minerCfg)

	return &AutoLogBodyAnalyzer{
		miner:     miner,
		templates: make(map[string]*LogTemplate),
	}
}

// ProcessMessage processes a single log body and extracts/updates template
func (a *AutoLogBodyAnalyzer) ProcessMessage(body string) string {
	// Extract template using miner (shard-level locking inside)
	template, _ := a.miner.Add(body)
	
	atomic.AddInt64(&a.total, 1)

	// Fast path: template already tracked — use atomic increment, no map lock.
	a.mu.RLock()
	tmpl, exists := a.templates[template]
	a.mu.RUnlock()
	if exists {
		atomic.AddInt64(&tmpl.Count, 1)
		return template
	}

	// Slow path: first time seeing this template — write lock, re-check.
	a.mu.Lock()
	tmpl, exists = a.templates[template]
	if exists {
		a.mu.Unlock()
		atomic.AddInt64(&tmpl.Count, 1)
		return template
	}
	hash := hashString(template)
	a.templates[template] = &LogTemplate{
		Template:    template,
		Hash:        hash,
		Count:       1,
		ExampleBody: body,
	}
	a.mu.Unlock()
	
	return template
}

// AddMessage is an alias for ProcessMessage to match the interface
func (a *AutoLogBodyAnalyzer) AddMessage(body string) {
	a.ProcessMessage(body)
}

// GetTemplates returns all templates sorted by count.
// Templates and counts are read directly from drain's cluster state so that
// generalized templates (e.g. "Received <*>" from multiple "Received X" variants)
// are reported with the correct aggregated count, not split across stale entries.
func (a *AutoLogBodyAnalyzer) GetTemplates() []*LogTemplate {
	clusters := a.miner.GetClusters()

	// Sum total across all clusters for percentage calculation
	var total int64
	for _, c := range clusters {
		total += c.Count
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make([]*LogTemplate, 0, len(clusters))
	for _, c := range clusters {
		pct := 0.0
		if total > 0 {
			pct = float64(c.Count) / float64(total) * 100.0
		}
		// Prefer original (unmasked) example body from our cache when available.
		// After generalization the key in a.templates may not match c.Template,
		// so fall back to drain's stored (masked) example body.
		exampleBody := c.ExampleBody
		if tmpl, exists := a.templates[c.Template]; exists && tmpl.ExampleBody != "" {
			exampleBody = tmpl.ExampleBody
		}
		result = append(result, &LogTemplate{
			Template:    c.Template,
			Hash:        hashString(c.Template),
			Count:       c.Count,
			Percentage:  pct,
			ExampleBody: exampleBody,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	return result
}

// GetStats returns statistics about the analyzer
func (a *AutoLogBodyAnalyzer) GetStats() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	minerStats := a.miner.Stats()
	
	return map[string]interface{}{
		"total_messages":    atomic.LoadInt64(&a.total),
		"template_count":    len(a.templates),
		"miner_shards":      minerStats["shards"],
		"miner_clusters":    minerStats["clusters"],
		"miner_training":    minerStats["training"],
	}
}

// SetTrainingMode switches between training and inference modes
func (a *AutoLogBodyAnalyzer) SetTrainingMode(training bool) {
	// This requires updating the miner config
	// For now, we'll add this capability to the miner itself
	a.miner.SetTraining(training)
}

// Merge is a placeholder for future snapshot/restore functionality
func (a *AutoLogBodyAnalyzer) Merge(other map[string]*LogTemplate) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	for template, otherTmpl := range other {
		if existing, exists := a.templates[template]; exists {
			existing.Count += otherTmpl.Count
		} else {
			a.templates[template] = &LogTemplate{
				Template: otherTmpl.Template,
				Hash:     otherTmpl.Hash,
				Count:    otherTmpl.Count,
			}
		}
		a.total += otherTmpl.Count
	}
	
	return nil
}

// Clear resets the analyzer state
func (a *AutoLogBodyAnalyzer) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	a.templates = make(map[string]*LogTemplate)
	a.total = 0
	// Note: miner state is not cleared - it retains learned clusters
}

// ToJSON exports templates for persistence
func (a *AutoLogBodyAnalyzer) ToJSON() ([]byte, error) {
	templates := a.GetTemplates()
	return fmt.Appendf(nil, "%+v", templates), nil
}
