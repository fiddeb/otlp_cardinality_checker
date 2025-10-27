package analyzer

import (
	"fmt"
	"sort"
	"sync"

	"github.com/fidde/otlp_cardinality_checker/internal/analyzer/autotemplate"
	"github.com/fidde/otlp_cardinality_checker/internal/config"
)

// AutoLogBodyAnalyzer uses the autotemplate miner for log template extraction
type AutoLogBodyAnalyzer struct {
	mu        sync.RWMutex
	miner     *autotemplate.ShardedMiner
	templates map[string]*LogTemplate // template string -> metadata
	total     int64
	
	// Optional pre-masking patterns (applied before miner)
	patterns []config.CompiledPattern
}

// NewAutoLogBodyAnalyzer creates a new auto-template analyzer
func NewAutoLogBodyAnalyzer(minerCfg autotemplate.Config) *AutoLogBodyAnalyzer {
	return NewAutoLogBodyAnalyzerWithPatterns(minerCfg, nil)
}

// NewAutoLogBodyAnalyzerWithPatterns creates analyzer with pre-masking patterns
func NewAutoLogBodyAnalyzerWithPatterns(minerCfg autotemplate.Config, patterns []config.CompiledPattern) *AutoLogBodyAnalyzer {
	if patterns == nil {
		// Use default patterns for pre-masking
		patterns = config.DefaultPatterns()
	}
	
	miner := autotemplate.NewShardedMiner(minerCfg)
	
	return &AutoLogBodyAnalyzer{
		miner:     miner,
		templates: make(map[string]*LogTemplate),
		patterns:  patterns,
	}
}

// ProcessMessage processes a single log body and extracts/updates template
func (a *AutoLogBodyAnalyzer) ProcessMessage(body string) string {
	return a.ProcessMessageWithKeys(body, nil, nil)
}

// ProcessMessageWithKeys processes a log body with its attribute and resource keys
func (a *AutoLogBodyAnalyzer) ProcessMessageWithKeys(body string, attributeKeys, resourceKeys []string) string {
	// Pre-mask with regex patterns
	masked := a.preMask(body)
	
	// Extract template using miner
	template, _ := a.miner.Add(masked)
	
	// Update metadata
	a.mu.Lock()
	
	a.total++
	
	existing, ok := a.templates[template]
	if !ok {
		hash := hashString(template)
		existing = &LogTemplate{
			Template:      template,
			Hash:          hash,
			Count:         1,
			ExampleBody:   body, // Store original unmasked body as example
			AttributeKeys: make(map[string]struct{}),
			ResourceKeys:  make(map[string]struct{}),
		}
		a.templates[template] = existing
	}
	
	a.mu.Unlock()
	
	// Now update the template with proper locking
	existing.mu.Lock()
	existing.Count++
	for _, key := range attributeKeys {
		existing.AttributeKeys[key] = struct{}{}
	}
	for _, key := range resourceKeys {
		existing.ResourceKeys[key] = struct{}{}
	}
	existing.mu.Unlock()
	
	return template
}

// AddMessage is an alias for ProcessMessage to match the interface
func (a *AutoLogBodyAnalyzer) AddMessage(body string) {
	a.ProcessMessage(body)
}

// AddMessageWithKeys processes a log message with keys (interface method)
func (a *AutoLogBodyAnalyzer) AddMessageWithKeys(body string, attributeKeys, resourceKeys []string) {
	a.ProcessMessageWithKeys(body, attributeKeys, resourceKeys)
}

// preMask applies regex-based masking before template extraction
func (a *AutoLogBodyAnalyzer) preMask(body string) string {
	result := body
	for _, pattern := range a.patterns {
		result = pattern.Regex.ReplaceAllString(result, pattern.Placeholder)
	}
	return result
}

// GetTemplates returns all templates sorted by count
func (a *AutoLogBodyAnalyzer) GetTemplates() []*LogTemplate {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	templates := make([]*LogTemplate, 0, len(a.templates))
	for _, tmpl := range a.templates {
		// Calculate percentage
		if a.total > 0 {
			tmpl.Percentage = float64(tmpl.Count) / float64(a.total) * 100.0
		}
		templates = append(templates, tmpl)
	}
	
	// Sort by count descending
	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Count > templates[j].Count
	})
	
	return templates
}

// GetStats returns statistics about the analyzer
func (a *AutoLogBodyAnalyzer) GetStats() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	minerStats := a.miner.Stats()
	
	return map[string]interface{}{
		"total_messages":    a.total,
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
