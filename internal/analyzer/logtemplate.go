package analyzer

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// LogTemplate represents a pattern extracted from log messages
type LogTemplate struct {
	Template     string            `json:"template"`
	Hash         uint64            `json:"hash"`
	Count        int64             `json:"count"`
	Percentage   float64           `json:"percentage"`
	SampleValues map[string]string `json:"sample_values,omitempty"` // First occurrence of each placeholder
}

// LogBodyAnalyzer extracts templates from log body text
type LogBodyAnalyzer struct {
	mu        sync.RWMutex
	templates map[uint64]*LogTemplate
	total     int64
	
	// Regex patterns for common variable parts
	numberPattern   *regexp.Regexp
	uuidPattern     *regexp.Regexp
	ipPattern       *regexp.Regexp
	timestampPattern *regexp.Regexp
	durationPattern *regexp.Regexp
	sizePattern     *regexp.Regexp
	hexPattern      *regexp.Regexp
	urlPattern      *regexp.Regexp
}

// NewLogBodyAnalyzer creates a new log body analyzer
func NewLogBodyAnalyzer() *LogBodyAnalyzer {
	return &LogBodyAnalyzer{
		templates: make(map[uint64]*LogTemplate),
		
		// Simple regex patterns - optimize these if they're too slow
		numberPattern:    regexp.MustCompile(`\b\d+\b`),
		uuidPattern:      regexp.MustCompile(`\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`),
		ipPattern:        regexp.MustCompile(`\[::1\]|\b(?:\d{1,3}\.){3}\d{1,3}\b`),
		timestampPattern: regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`),
		durationPattern:  regexp.MustCompile(`\d+(?:\.\d+)?(?:µs|ms|s|m|h)\b`),
		sizePattern:      regexp.MustCompile(`\d+(?:\.\d+)?(?:B|KB|MB|GB)\b`),
		hexPattern:       regexp.MustCompile(`\b[0-9a-f]{8,}\b`),
		// URL pattern: matches full URLs (http/https) and absolute paths starting with /
		urlPattern:       regexp.MustCompile(`https?://[^\s]+|\s(/[a-zA-Z0-9/_.-]+)`),
	}
}

// ExtractTemplate converts a log message into a template
func (a *LogBodyAnalyzer) ExtractTemplate(message string) string {
	template := message
	
	// Order matters - do most specific patterns first
	template = a.timestampPattern.ReplaceAllString(template, "<TIMESTAMP>")
	template = a.uuidPattern.ReplaceAllString(template, "<UUID>")
	template = a.urlPattern.ReplaceAllString(template, " <URL>") // Add space before to preserve whitespace
	template = a.durationPattern.ReplaceAllString(template, "<DURATION>")
	template = a.sizePattern.ReplaceAllString(template, "<SIZE>")
	template = a.ipPattern.ReplaceAllString(template, "<IP>")
	template = a.hexPattern.ReplaceAllString(template, "<HEX>")
	template = a.numberPattern.ReplaceAllString(template, "<NUM>")
	
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
