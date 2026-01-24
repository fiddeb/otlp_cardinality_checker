// Package analyzer provides analysis of OTLP telemetry data.
//
// This file implements span name pattern extraction, which identifies dynamic
// values (IDs, UUIDs, timestamps, etc.) in span names and replaces them with
// placeholders. This helps identify instrumentation patterns and detect
// high-cardinality span naming issues.
//
// Pattern matching uses regex patterns from config.CompiledPattern, applied
// in this order (first match wins):
//  1. Timestamps: 2024/01/22 -> <TIMESTAMP>
//  2. UUIDs: 550e8400-e29b-41d4-... -> <UUID>
//  3. Emails: user@example.com -> <EMAIL>
//  4. URLs/Paths: /api/users/123 -> <URL>
//  5. Durations: 500ms, 1.5s -> <DURATION>
//  6. Sizes: 100MB, 1.5GB -> <SIZE>
//  7. IPs: 192.168.1.1 -> <IP>
//  8. Hex strings: deadbeef1234 -> <HEX>
//  9. Numbers: 123, 456.78 -> <NUM>
package analyzer

import (
	"hash/fnv"
	"sort"
	"strings"
	"sync"

	"github.com/fidde/otlp_cardinality_checker/internal/config"
	"github.com/fidde/otlp_cardinality_checker/pkg/models"
)

// spanNameEntry tracks a pattern and its examples
type spanNameEntry struct {
	Template string
	Count    int64
	Examples []string
}

// SpanNameAnalyzer extracts patterns from span names
type SpanNameAnalyzer struct {
	mu       sync.RWMutex
	patterns map[uint64]*spanNameEntry // hash -> entry
	total    int64

	// Compiled patterns from config (reused from log templates)
	compiledPatterns []config.CompiledPattern

	// MaxExamples is the maximum number of example span names to keep per pattern
	MaxExamples int
}

// NewSpanNameAnalyzer creates a new span name analyzer with default patterns
func NewSpanNameAnalyzer() *SpanNameAnalyzer {
	return NewSpanNameAnalyzerWithPatterns(nil)
}

// NewSpanNameAnalyzerWithPatterns creates a new analyzer with custom patterns
func NewSpanNameAnalyzerWithPatterns(patterns []config.CompiledPattern) *SpanNameAnalyzer {
	if patterns == nil {
		patterns = config.DefaultPatterns()
	}

	return &SpanNameAnalyzer{
		patterns:         make(map[uint64]*spanNameEntry),
		compiledPatterns: patterns,
		MaxExamples:      3, // Default: keep first 3 unique examples
	}
}

// ExtractPattern converts a span name into a template by replacing dynamic values
// with placeholders. For example: "GET /users/123" -> "GET /users/<NUM>"
func (a *SpanNameAnalyzer) ExtractPattern(spanName string) string {
	template := spanName

	// Apply patterns in order (same as log template extraction)
	for _, pattern := range a.compiledPatterns {
		template = pattern.Regex.ReplaceAllString(template, pattern.Placeholder)
	}

	// Normalize whitespace
	template = strings.Join(strings.Fields(template), " ")

	return template
}

// hashString creates a hash for a template string
func hashSpanTemplate(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

// AddSpanName processes a span name and updates pattern tracking.
// It extracts a template pattern and stores up to MaxExamples original names.
func (a *SpanNameAnalyzer) AddSpanName(name string) {
	if name == "" {
		return
	}

	template := a.ExtractPattern(name)
	hash := hashSpanTemplate(template)

	a.mu.Lock()
	defer a.mu.Unlock()

	a.total++

	if existing, ok := a.patterns[hash]; ok {
		existing.Count++
		// Add example if not at max and not duplicate
		if len(existing.Examples) < a.MaxExamples {
			found := false
			for _, ex := range existing.Examples {
				if ex == name {
					found = true
					break
				}
			}
			if !found {
				existing.Examples = append(existing.Examples, name)
			}
		}
	} else {
		a.patterns[hash] = &spanNameEntry{
			Template: template,
			Count:    1,
			Examples: []string{name},
		}
	}
}

// GetPatterns returns all patterns sorted by count (descending).
// Percentages are calculated based on total span count.
func (a *SpanNameAnalyzer) GetPatterns() []*models.SpanNamePattern {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.patterns) == 0 {
		return nil
	}

	patterns := make([]*models.SpanNamePattern, 0, len(a.patterns))
	for _, entry := range a.patterns {
		percentage := 0.0
		if a.total > 0 {
			percentage = float64(entry.Count) / float64(a.total) * 100
		}

		// Copy examples slice
		examples := make([]string, len(entry.Examples))
		copy(examples, entry.Examples)

		patterns = append(patterns, &models.SpanNamePattern{
			Template:   entry.Template,
			Count:      entry.Count,
			Percentage: percentage,
			Examples:   examples,
		})
	}

	// Sort by count descending
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Count > patterns[j].Count
	})

	return patterns
}

// GetTotal returns the total number of span names processed
func (a *SpanNameAnalyzer) GetTotal() int64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.total
}

// Reset clears all tracked patterns
func (a *SpanNameAnalyzer) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.patterns = make(map[uint64]*spanNameEntry)
	a.total = 0
}
