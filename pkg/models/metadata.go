// Package models defines the core data structures for metadata tracking.
//
// This package contains the domain models used throughout the application
// to represent metadata extracted from OTLP telemetry signals.
package models

import (
	"encoding/json"
	"sort"
	"sync"
	"time"
)

// MetricMetadata contains metadata about an observed metric.
// It tracks all unique label keys and their cardinality statistics.
type MetricMetadata struct {
	Name        string `json:"name"`
	Type        string `json:"type"`                   // Gauge, Sum, Histogram, etc.
	Unit        string `json:"unit,omitempty"`         // Optional unit
	Description string `json:"description,omitempty"`  // Metric description

	// LabelKeys maps label key names to their metadata and cardinality stats
	LabelKeys map[string]*KeyMetadata `json:"label_keys"`

	// ResourceKeys maps resource attribute key names to their metadata
	ResourceKeys map[string]*KeyMetadata `json:"resource_keys"`

	// ScopeInfo contains instrumentation scope information
	ScopeInfo *ScopeMetadata `json:"scope_info,omitempty"`

	// Timestamps
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`

	// SampleCount is the total number of data points observed for this metric
	SampleCount int64 `json:"sample_count"`

	// Services maps service names to the number of samples from that service
	Services map[string]int64 `json:"services"`

	mu sync.RWMutex `json:"-"`
}

// SpanMetadata contains metadata about observed spans.
type SpanMetadata struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // Client, Server, Internal, Producer, Consumer

	// AttributeKeys maps attribute key names to their metadata
	AttributeKeys map[string]*KeyMetadata `json:"attribute_keys"`

	// EventNames tracks unique event names observed in spans
	EventNames []string `json:"event_names"`

	// EventAttributeKeys maps event names to their attribute keys
	EventAttributeKeys map[string]map[string]*KeyMetadata `json:"event_attribute_keys"`

	// LinkAttributeKeys tracks attribute keys found in span links
	LinkAttributeKeys map[string]*KeyMetadata `json:"link_attribute_keys"`

	// ResourceKeys maps resource attribute key names to their metadata
	ResourceKeys map[string]*KeyMetadata `json:"resource_keys"`

	// ScopeInfo contains instrumentation scope information
	ScopeInfo *ScopeMetadata `json:"scope_info,omitempty"`

	// Timestamps
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`

	// SpanCount is the total number of spans observed with this name
	SpanCount int64 `json:"span_count"`

	// Services maps service names to span counts
	Services map[string]int64 `json:"services"`

	mu sync.RWMutex `json:"-"`
}

// LogMetadata contains metadata about observed log records.
type LogMetadata struct {
	SeverityText string `json:"severity_text"` // INFO, WARN, ERROR, etc.

	// AttributeKeys maps attribute key names to their metadata
	AttributeKeys map[string]*KeyMetadata `json:"attribute_keys"`

	// ResourceKeys maps resource attribute key names to their metadata
	ResourceKeys map[string]*KeyMetadata `json:"resource_keys"`

	// ScopeInfo contains instrumentation scope information
	ScopeInfo *ScopeMetadata `json:"scope_info,omitempty"`

	// Timestamps
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`

	// RecordCount is the total number of log records observed
	RecordCount int64 `json:"record_count"`

	// Services maps service names to record counts
	Services map[string]int64 `json:"services"`

	mu sync.RWMutex `json:"-"`
}

// KeyMetadata tracks statistics about a specific attribute/label key.
type KeyMetadata struct {
	// Count is the number of times this key has been observed
	Count int64 `json:"count"`

	// Percentage is the percentage of samples that include this key
	Percentage float64 `json:"percentage"`

	// EstimatedCardinality is an approximate count of unique values
	// Uses HyperLogLog for memory-efficient estimation
	EstimatedCardinality int64 `json:"estimated_cardinality"`

	// ValueSamples contains a sample of observed values (first N unique)
	// Limited to MaxSamples to prevent memory issues
	ValueSamples []string `json:"value_samples,omitempty"`

	// valueSampleSet is used internally to track unique samples
	valueSampleSet map[string]struct{} `json:"-"`

	// MaxSamples is the maximum number of value samples to keep
	MaxSamples int `json:"-"`

	// Timestamps
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`

	mu sync.RWMutex `json:"-"`
}

// ScopeMetadata contains information about the instrumentation scope.
type ScopeMetadata struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// NewMetricMetadata creates a new MetricMetadata instance.
func NewMetricMetadata(name, metricType string) *MetricMetadata {
	return &MetricMetadata{
		Name:         name,
		Type:         metricType,
		LabelKeys:    make(map[string]*KeyMetadata),
		ResourceKeys: make(map[string]*KeyMetadata),
		Services:     make(map[string]int64),
		FirstSeen:    time.Now(),
		LastSeen:     time.Now(),
	}
}

// NewSpanMetadata creates a new SpanMetadata instance.
func NewSpanMetadata(name, kind string) *SpanMetadata {
	return &SpanMetadata{
		Name:               name,
		Kind:               kind,
		AttributeKeys:      make(map[string]*KeyMetadata),
		EventNames:         []string{},
		EventAttributeKeys: make(map[string]map[string]*KeyMetadata),
		LinkAttributeKeys:  make(map[string]*KeyMetadata),
		ResourceKeys:       make(map[string]*KeyMetadata),
		Services:           make(map[string]int64),
		FirstSeen:          time.Now(),
		LastSeen:           time.Now(),
	}
}

// NewLogMetadata creates a new LogMetadata instance.
func NewLogMetadata(severityText string) *LogMetadata {
	return &LogMetadata{
		SeverityText:  severityText,
		AttributeKeys: make(map[string]*KeyMetadata),
		ResourceKeys:  make(map[string]*KeyMetadata),
		Services:      make(map[string]int64),
		FirstSeen:     time.Now(),
		LastSeen:      time.Now(),
	}
}

// NewKeyMetadata creates a new KeyMetadata instance with default max samples.
func NewKeyMetadata() *KeyMetadata {
	return &KeyMetadata{
		ValueSamples:   []string{},
		valueSampleSet: make(map[string]struct{}),
		MaxSamples:     100, // Default: keep first 100 unique values
		FirstSeen:      time.Now(),
		LastSeen:       time.Now(),
	}
}

// AddValue adds a value observation to the key metadata.
// It updates cardinality estimation and value samples.
func (k *KeyMetadata) AddValue(value string) {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.Count++
	k.LastSeen = time.Now()

	// Add to sample set if not full
	if _, exists := k.valueSampleSet[value]; !exists {
		if len(k.valueSampleSet) < k.MaxSamples {
			k.valueSampleSet[value] = struct{}{}
			k.ValueSamples = append(k.ValueSamples, value)
		}
		// Update estimated cardinality (includes values beyond MaxSamples)
		k.EstimatedCardinality++
	}
}

// GetSortedSamples returns the value samples in sorted order.
// This is only called when serializing to JSON, not on every insert.
func (k *KeyMetadata) GetSortedSamples() []string {
	k.mu.RLock()
	defer k.mu.RUnlock()

	samples := make([]string, len(k.ValueSamples))
	copy(samples, k.ValueSamples)
	sort.Strings(samples)
	return samples
}

// MarshalJSON implements custom JSON marshaling for KeyMetadata.
// This ensures value_samples are sorted in the output without affecting internal state.
func (k *KeyMetadata) MarshalJSON() ([]byte, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	// Create a temporary struct with sorted samples
	type Alias KeyMetadata
	return json.Marshal(&struct {
		ValueSamples []string `json:"value_samples,omitempty"`
		*Alias
	}{
		ValueSamples: k.GetSortedSamples(),
		Alias:        (*Alias)(k),
	})
}

// UpdatePercentage updates the percentage field based on total samples.
func (k *KeyMetadata) UpdatePercentage(totalSamples int64) {
	k.mu.Lock()
	defer k.mu.Unlock()

	if totalSamples > 0 {
		k.Percentage = (float64(k.Count) / float64(totalSamples)) * 100.0
	}
}

// MergeMetricMetadata merges another MetricMetadata into this one.
// This is used when observing the same metric with different label sets.
func (m *MetricMetadata) MergeMetricMetadata(other *MetricMetadata) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update timestamps
	if other.LastSeen.After(m.LastSeen) {
		m.LastSeen = other.LastSeen
	}
	if other.FirstSeen.Before(m.FirstSeen) {
		m.FirstSeen = other.FirstSeen
	}

	// Update sample count
	m.SampleCount += other.SampleCount

	// Merge label keys
	for key, otherKeyMeta := range other.LabelKeys {
		if existing, exists := m.LabelKeys[key]; exists {
			existing.mu.Lock()
			existing.Count += otherKeyMeta.Count
			existing.LastSeen = otherKeyMeta.LastSeen
			
			// Merge value samples
			for _, sample := range otherKeyMeta.ValueSamples {
				if _, exists := existing.valueSampleSet[sample]; !exists {
					if len(existing.valueSampleSet) < existing.MaxSamples {
						existing.valueSampleSet[sample] = struct{}{}
						existing.ValueSamples = append(existing.ValueSamples, sample)
					}
					// Always update cardinality count (even beyond MaxSamples)
					existing.EstimatedCardinality++
				}
			}
			existing.mu.Unlock()
		} else {
			m.LabelKeys[key] = otherKeyMeta
		}
	}

	// Merge resource keys
	for key, otherKeyMeta := range other.ResourceKeys {
		if existing, exists := m.ResourceKeys[key]; exists {
			existing.Count += otherKeyMeta.Count
			existing.LastSeen = otherKeyMeta.LastSeen
		} else {
			m.ResourceKeys[key] = otherKeyMeta
		}
	}

	// Merge services
	for service, count := range other.Services {
		m.Services[service] += count
	}

	// Update percentages
	for _, keyMeta := range m.LabelKeys {
		keyMeta.UpdatePercentage(m.SampleCount)
	}
	for _, keyMeta := range m.ResourceKeys {
		keyMeta.UpdatePercentage(m.SampleCount)
	}
}

// GetLabelKeysSorted returns label keys sorted alphabetically.
func (m *MetricMetadata) GetLabelKeysSorted() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.LabelKeys))
	for k := range m.LabelKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// GetResourceKeysSorted returns resource keys sorted alphabetically.
func (m *MetricMetadata) GetResourceKeysSorted() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.ResourceKeys))
	for k := range m.ResourceKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// HasHighCardinalityLabel checks if any label has estimated cardinality above threshold.
func (m *MetricMetadata) HasHighCardinalityLabel(threshold int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, keyMeta := range m.LabelKeys {
		if keyMeta.EstimatedCardinality > threshold {
			return true
		}
	}
	return false
}

// GetHighCardinalityLabels returns label keys with cardinality above threshold.
func (m *MetricMetadata) GetHighCardinalityLabels(threshold int64) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	highCard := []string{}
	for key, keyMeta := range m.LabelKeys {
		if keyMeta.EstimatedCardinality > threshold {
			highCard = append(highCard, key)
		}
	}
	sort.Strings(highCard)
	return highCard
}
