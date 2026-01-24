package analyzer

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"sync"

	"github.com/fidde/otlp_cardinality_checker/internal/patterns"
)

// LogTemplate represents a pattern extracted from log messages
type LogTemplate struct {
	Template     string            `json:"template"`
	Hash         uint64            `json:"hash"`
	Count        int64             `json:"count"`
	Percentage   float64           `json:"percentage"`
	ExampleBody  string            `json:"example_body"`            // Example log message matching this template
	SampleValues map[string]string `json:"sample_values,omitempty"` // First occurrence of each placeholder
}

// LogBodyAnalyzer extracts templates from log body text
type LogBodyAnalyzer struct {
	mu        sync.RWMutex
	templates map[uint64]*LogTemplate
	total     int64
	
	// Compiled patterns from config
	patterns []patterns.CompiledPattern
}

// NewLogBodyAnalyzer creates a new log body analyzer
func NewLogBodyAnalyzer() *LogBodyAnalyzer {
	return NewLogBodyAnalyzerWithPatterns(nil)
}

// NewLogBodyAnalyzerWithPatterns creates a new analyzer with custom patterns
func NewLogBodyAnalyzerWithPatterns(pats []patterns.CompiledPattern) *LogBodyAnalyzer {
	if pats == nil {
		// Use default patterns if none provided
		pats = patterns.DefaultPatterns()
	}
	
	return &LogBodyAnalyzer{
		templates: make(map[uint64]*LogTemplate),
		patterns:  pats,
	}
}

// ExtractTemplate converts a log message into a template
func (a *LogBodyAnalyzer) ExtractTemplate(message string) string {
	template := message
	
	// Apply patterns in order
	for _, pattern := range a.patterns {
		template = pattern.Regex.ReplaceAllString(template, pattern.Placeholder)
	}
	
	// Normalize whitespace
	template = strings.Join(strings.Fields(template), " ")
	
	return template
}

// hashString creates a hash for a template string
func hashString(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

// AddMessage processes a log message and updates templates
func (a *LogBodyAnalyzer) AddMessage(message string) {
	if message == "" {
		return
	}
	
	template := a.ExtractTemplate(message)
	hash := hashString(template)
	
	a.mu.Lock()
	defer a.mu.Unlock()
	
	a.total++
	
	if existing, ok := a.templates[hash]; ok {
		existing.Count++
	} else {
		a.templates[hash] = &LogTemplate{
			Template:     template,
			Hash:         hash,
			Count:        1,
			SampleValues: map[string]string{"original": message[:min(len(message), 200)]},
		}
	}
}

// GetTemplates returns all templates sorted by count
func (a *LogBodyAnalyzer) GetTemplates() []*LogTemplate {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	templates := make([]*LogTemplate, 0, len(a.templates))
	for _, tmpl := range a.templates {
		// Calculate percentage
		if a.total > 0 {
			tmpl.Percentage = float64(tmpl.Count) / float64(a.total) * 100
		}
		templates = append(templates, tmpl)
	}
	
	// Sort by count descending
	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Count > templates[j].Count
	})
	
	return templates
}

// GetStats returns summary statistics
func (a *LogBodyAnalyzer) GetStats() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	return map[string]interface{}{
		"total_messages":      a.total,
		"unique_templates":    len(a.templates),
		"template_efficiency": fmt.Sprintf("%.1f:1", float64(a.total)/float64(max(len(a.templates), 1))),
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
