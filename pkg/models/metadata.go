// Package models defines the core data structures for metadata tracking.
//
// This package contains the domain models used throughout the application
// to represent metadata extracted from OTLP telemetry signals.
package models

import (
	"encoding/json"
	"errors"
	"sort"
	"sync"

	"github.com/fidde/otlp_cardinality_checker/pkg/hyperloglog"
)

// ErrNotFound is returned when a requested item is not found.
// Storage implementations wrap this error when an item doesn't exist.
var ErrNotFound = errors.New("not found")

// MetricMetadata contains metadata about an observed metric.
// It tracks all unique label keys and their cardinality statistics.
// Follows the structure of opentelemetry.proto.metrics.v1.Metric
type MetricMetadata struct {
	// Name is the metric name (required)
	Name string `json:"name"`

	// Description provides context about what the metric measures
	Description string `json:"description,omitempty"`

	// Unit specifies the unit of measurement (e.g., "By", "ms", "1")
	Unit string `json:"unit,omitempty"`

	// Data contains the type-specific metric data (Gauge, Sum, Histogram, etc.)
	// This replaces the old "Type" string field with a proper typed interface
	Data MetricData `json:"data"`

	// Metadata contains additional key-value metadata (optional, rarely used)
	Metadata map[string]string `json:"metadata,omitempty"`

	// LabelKeys maps label key names to their metadata and cardinality stats
	// These correspond to attributes on data points in OTLP
	LabelKeys map[string]*KeyMetadata `json:"label_keys"`

	// ResourceKeys maps resource attribute key names to their metadata
	ResourceKeys map[string]*KeyMetadata `json:"resource_keys"`

	// ScopeInfo contains instrumentation scope information
	ScopeInfo *ScopeMetadata `json:"scope_info,omitempty"`

	// SampleCount is the total number of data points observed for this metric
	SampleCount int64 `json:"sample_count"`

	// Services maps service names to the number of samples from that service
	Services map[string]int64 `json:"services"`

	// seriesHLL tracks unique combinations of label values to count active series
	// This uses HyperLogLog to efficiently estimate the number of unique time series
	// without storing all combinations in memory
	seriesHLL *hyperloglog.HyperLogLog `json:"-"`

	// ActiveSeries is the estimated number of unique time series (label combinations)
	// Updated from seriesHLL count
	ActiveSeries int64 `json:"active_series"`

	mu sync.RWMutex `json:"-"`
}

// SpanMetadata contains metadata about observed spans.
// Follows the structure of opentelemetry.proto.trace.v1.Span
type SpanMetadata struct {
	// Name is the span name (required)
	// Corresponds to Span.name
	Name string `json:"name"`

	// Kind is the span kind enum value
	// Corresponds to Span.kind (SpanKind enum)
	// Values: UNSPECIFIED=0, INTERNAL=1, SERVER=2, CLIENT=3, PRODUCER=4, CONSUMER=5
	Kind int32 `json:"kind"`

	// KindName is the human-readable span kind name for convenience
	KindName string `json:"kind_name,omitempty"`

	// AttributeKeys maps attribute key names to their metadata
	// Corresponds to Span.attributes
	AttributeKeys map[string]*KeyMetadata `json:"attribute_keys"`

	// EventNames tracks unique event names observed in spans
	// Corresponds to Span.Event.name
	EventNames []string `json:"event_names"`

	// EventAttributeKeys maps event names to their attribute keys
	// Corresponds to Span.Event.attributes
	EventAttributeKeys map[string]map[string]*KeyMetadata `json:"event_attribute_keys"`

	// LinkAttributeKeys tracks attribute keys found in span links
	// Corresponds to Span.Link.attributes
	LinkAttributeKeys map[string]*KeyMetadata `json:"link_attribute_keys"`

	// ResourceKeys maps resource attribute key names to their metadata
	ResourceKeys map[string]*KeyMetadata `json:"resource_keys"`

	// HasTraceState indicates if any spans had trace_state set
	// Corresponds to Span.trace_state
	HasTraceState bool `json:"has_trace_state"`

	// HasParentSpanId indicates if any spans had parent_span_id set (not root spans)
	// Corresponds to Span.parent_span_id
	HasParentSpanId bool `json:"has_parent_span_id"`

	// StatusCodes tracks which status codes have been observed
	// Corresponds to Span.Status.code enum (UNSET=0, OK=1, ERROR=2)
	StatusCodes []string `json:"status_codes,omitempty"`

	// DroppedAttributesStats tracks statistics about dropped attributes
	// Corresponds to Span.dropped_attributes_count
	DroppedAttributesStats *DroppedCountStats `json:"dropped_attributes_stats,omitempty"`

	// DroppedEventsStats tracks statistics about dropped events
	// Corresponds to Span.dropped_events_count
	DroppedEventsStats *DroppedCountStats `json:"dropped_events_stats,omitempty"`

	// DroppedLinksStats tracks statistics about dropped links
	// Corresponds to Span.dropped_links_count
	DroppedLinksStats *DroppedCountStats `json:"dropped_links_stats,omitempty"`

	// ScopeInfo contains instrumentation scope information
	ScopeInfo *ScopeMetadata `json:"scope_info,omitempty"`

	// NamePatterns contains extracted patterns from span names
	// Helps identify dynamic values in span names (e.g., IDs, timestamps)
	NamePatterns []*SpanNamePattern `json:"name_patterns,omitempty"`

	// SampleCount is the total number of spans observed with this name
	SampleCount int64 `json:"sample_count"`

	// Services maps service names to span counts
	Services map[string]int64 `json:"services"`

	mu sync.RWMutex `json:"-"`
}

// LogMetadata contains metadata about observed log records.
// Follows the structure of opentelemetry.proto.logs.v1.LogRecord
type LogMetadata struct {
	// Severity is the severity text (INFO, WARN, ERROR, etc.)
	// Corresponds to LogRecord.severity_text
	Severity string `json:"severity"`

	// SeverityNumber is the numerical severity value (1-24)
	// Corresponds to LogRecord.severity_number enum
	SeverityNumber int32 `json:"severity_number,omitempty"`

	// AttributeKeys maps attribute key names to their metadata
	// These are from LogRecord.attributes
	AttributeKeys map[string]*KeyMetadata `json:"attribute_keys"`

	// ResourceKeys maps resource attribute key names to their metadata
	ResourceKeys map[string]*KeyMetadata `json:"resource_keys"`

	// BodyTemplates contains extracted templates from log body text
	// This is our custom feature for analyzing LogRecord.body patterns
	BodyTemplates []*BodyTemplate `json:"body_templates,omitempty"`

	// EventNames tracks unique event_name values observed
	// Corresponds to LogRecord.event_name
	EventNames []string `json:"event_names,omitempty"`

	// HasTraceContext indicates if any log records had trace_id set
	HasTraceContext bool `json:"has_trace_context"`

	// HasSpanContext indicates if any log records had span_id set
	HasSpanContext bool `json:"has_span_context"`

	// DroppedAttributesCount tracks statistics about dropped attributes
	DroppedAttributesStats *DroppedAttributesStats `json:"dropped_attributes_stats,omitempty"`

	// ScopeInfo contains instrumentation scope information
	ScopeInfo *ScopeMetadata `json:"scope_info,omitempty"`

	// SampleCount is the total number of log records observed
	SampleCount int64 `json:"sample_count"`

	// Services maps service names to record counts
	Services map[string]int64 `json:"services"`

	mu sync.RWMutex `json:"-"`
}

// DroppedAttributesStats tracks statistics about dropped attributes in log records
type DroppedAttributesStats struct {
	// TotalDropped is the sum of all dropped_attributes_count values
	TotalDropped uint32 `json:"total_dropped"`

	// RecordsWithDropped is the count of log records that had dropped attributes
	RecordsWithDropped int64 `json:"records_with_dropped"`

	// MaxDropped is the maximum dropped_attributes_count seen in a single record
	MaxDropped uint32 `json:"max_dropped"`
}

// DroppedCountStats tracks statistics about dropped items (attributes, events, or links)
// Used for Span.dropped_attributes_count, dropped_events_count, dropped_links_count
type DroppedCountStats struct {
	// TotalDropped is the sum of all dropped counts
	TotalDropped uint32 `json:"total_dropped"`

	// ItemsWithDropped is the count of items (spans) that had dropped data
	ItemsWithDropped int64 `json:"items_with_dropped"`

	// MaxDropped is the maximum dropped count seen in a single item
	MaxDropped uint32 `json:"max_dropped"`
}

// BodyTemplate represents a pattern extracted from log message bodies
type BodyTemplate struct {
	Template   string  `json:"template"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
	Example    string  `json:"example"` // First sample that matched this template
}

// SpanNamePattern represents a pattern extracted from span names
type SpanNamePattern struct {
	Template   string   `json:"template"`   // Pattern: "GET /users/<NUM>"
	Count      int64    `json:"count"`      // How many spans matched
	Percentage float64  `json:"percentage"` // % of total spans
	Examples   []string `json:"examples"`   // First 3 unique examples
}

// SpanPatternGroup represents an aggregated pattern across multiple span names.
type SpanPatternGroup struct {
	Pattern       string             `json:"pattern"`
	MatchingSpans []SpanPatternMatch `json:"matching_spans"`
	TotalSamples  int64              `json:"total_samples"`
	SpanCount     int                `json:"span_count"`
}

// SpanPatternMatch represents a span that matches a pattern.
type SpanPatternMatch struct {
	SpanName    string   `json:"span_name"`
	SampleCount int64    `json:"sample_count"`
	Services    []string `json:"services"`
	Kind        string   `json:"kind"`
}

// SpanPatternResponse is the API response for span patterns.
type SpanPatternResponse struct {
	Patterns []SpanPatternGroup `json:"patterns"`
	Total    int                `json:"total"`
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

	// hll is the HyperLogLog sketch for cardinality estimation
	// Uses fixed ~16KB memory regardless of cardinality
	hll *hyperloglog.HyperLogLog `json:"-"`

	// MaxSamples is the maximum number of value samples to keep
	MaxSamples int `json:"-"`

	mu sync.RWMutex `json:"-"`
}

// MarshalHLL serializes the HLL sketch to bytes for storage.
func (k *KeyMetadata) MarshalHLL() ([]byte, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	if k.hll == nil {
		return nil, nil
	}
	return k.hll.MarshalBinary()
}

// UnmarshalHLL deserializes an HLL sketch from bytes.
func (k *KeyMetadata) UnmarshalHLL(data []byte) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if len(data) == 0 {
		return nil
	}

	hll, err := hyperloglog.FromBytes(data)
	if err != nil {
		return err
	}

	k.hll = hll
	k.EstimatedCardinality = int64(hll.Count())
	return nil
}

// ScopeMetadata contains information about the instrumentation scope.
type ScopeMetadata struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// NewMetricMetadata creates a new MetricMetadata instance with the specified metric data type.
// The data parameter should be one of: *GaugeMetric, *SumMetric, *HistogramMetric, 
// *ExponentialHistogramMetric, or *SummaryMetric
func NewMetricMetadata(name string, data MetricData) *MetricMetadata {
	return &MetricMetadata{
		Name:         name,
		Data:         data,
		LabelKeys:    make(map[string]*KeyMetadata),
		ResourceKeys: make(map[string]*KeyMetadata),
		Services:     make(map[string]int64),
		Metadata:     make(map[string]string),
		seriesHLL:    hyperloglog.New(14), // Precision 14 = ~16KB memory, 0.81% error
		ActiveSeries: 0,
	}
}

// GetType returns the metric type as a string (for backward compatibility)
func (m *MetricMetadata) GetType() string {
	if m.Data == nil {
		return "Unknown"
	}
	return m.Data.GetType()
}

// NewSpanMetadata creates a new SpanMetadata instance.
func NewSpanMetadata(name string, kind int32, kindName string) *SpanMetadata {
	return &SpanMetadata{
		Name:               name,
		Kind:               kind,
		KindName:           kindName,
		AttributeKeys:      make(map[string]*KeyMetadata),
		EventNames:         []string{},
		EventAttributeKeys: make(map[string]map[string]*KeyMetadata),
		LinkAttributeKeys:  make(map[string]*KeyMetadata),
		ResourceKeys:       make(map[string]*KeyMetadata),
		Services:           make(map[string]int64),
		StatusCodes:        []string{},
	}
}

// NewLogMetadata creates a new LogMetadata instance.
func NewLogMetadata(severity string) *LogMetadata {
	return &LogMetadata{
		Severity:      severity,
		AttributeKeys: make(map[string]*KeyMetadata),
		ResourceKeys:  make(map[string]*KeyMetadata),
		Services:      make(map[string]int64),
		EventNames:    []string{},
	}
}

// NewKeyMetadata creates a new KeyMetadata instance with default max samples.
func NewKeyMetadata() *KeyMetadata {
	return &KeyMetadata{
		ValueSamples: []string{},
		hll:          hyperloglog.New(14), // Precision 14 = ~0.81% standard error
		MaxSamples:   10,                  // Default: keep first 10 unique values (enough for sampling)
	}
}

// AddValue adds a value observation to the key metadata.
// It updates cardinality estimation using HyperLogLog and value samples.
func (k *KeyMetadata) AddValue(value string) {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.Count++

	// Add to HyperLogLog for cardinality estimation
	k.hll.Add(value)

	// Add to sample list if not full and value is new
	if len(k.ValueSamples) < k.MaxSamples {
		// Check if value already exists in samples
		exists := false
		for _, sample := range k.ValueSamples {
			if sample == value {
				exists = true
				break
			}
		}
		if !exists {
			k.ValueSamples = append(k.ValueSamples, value)
		}
	}

	// Update estimated cardinality from HLL
	k.EstimatedCardinality = int64(k.hll.Count())
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

	// Update sample count
	m.SampleCount += other.SampleCount

	// Merge label keys
	for key, otherKeyMeta := range other.LabelKeys {
		if existing, exists := m.LabelKeys[key]; exists {
			existing.mu.Lock()
			existing.Count += otherKeyMeta.Count

			// Merge HLL sketches for accurate cardinality
			existing.hll.Merge(otherKeyMeta.hll)
			existing.EstimatedCardinality = int64(existing.hll.Count())

			// Merge value samples (keep first N unique)
			for _, sample := range otherKeyMeta.ValueSamples {
				if len(existing.ValueSamples) >= existing.MaxSamples {
					break
				}
				// Check if sample already exists
				found := false
				for _, existingSample := range existing.ValueSamples {
					if existingSample == sample {
						found = true
						break
					}
				}
				if !found {
					existing.ValueSamples = append(existing.ValueSamples, sample)
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
			existing.mu.Lock()
			existing.Count += otherKeyMeta.Count

			// Merge HLL sketches for accurate cardinality
			existing.hll.Merge(otherKeyMeta.hll)
			existing.EstimatedCardinality = int64(existing.hll.Count())

			// Merge value samples (keep first N unique)
			for _, sample := range otherKeyMeta.ValueSamples {
				if len(existing.ValueSamples) >= existing.MaxSamples {
					break
				}
				// Check if sample already exists
				found := false
				for _, existingSample := range existing.ValueSamples {
					if existingSample == sample {
						found = true
						break
					}
				}
				if !found {
					existing.ValueSamples = append(existing.ValueSamples, sample)
				}
			}
			existing.mu.Unlock()
		} else {
			m.ResourceKeys[key] = otherKeyMeta
		}
	}

	// Merge services
	for service, count := range other.Services {
		m.Services[service] += count
	}

	// Merge active series HLL sketch
	if other.seriesHLL != nil {
		if m.seriesHLL == nil {
			m.seriesHLL = hyperloglog.New(14)
		}
		m.seriesHLL.Merge(other.seriesHLL)
		m.ActiveSeries = int64(m.seriesHLL.Count())
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

// AddSeriesFingerprint adds a unique series fingerprint to the HLL tracker.
// The fingerprint should be a hash of all label key-value pairs for a data point.
func (m *MetricMetadata) AddSeriesFingerprint(fingerprint string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.seriesHLL == nil {
		m.seriesHLL = hyperloglog.New(14)
	}
	
	m.seriesHLL.Add(fingerprint)
	m.ActiveSeries = int64(m.seriesHLL.Count())
}

// GetActiveSeries returns the current count of active series (unique label combinations).
func (m *MetricMetadata) GetActiveSeries() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.seriesHLL == nil {
		return 1 // No series tracker = constant metric
	}
	
	count := int64(m.seriesHLL.Count())
	if count == 0 {
		return 1 // No series seen yet = treat as constant
	}
	return count
}

// CalculateActiveSeries returns the estimated number of active time series.
// This uses the HLL-based series tracker for accurate counting of unique combinations.
func (m *MetricMetadata) CalculateActiveSeries() int64 {
	return m.GetActiveSeries()
}

// MarshalSeriesHLL serializes the series HLL to bytes for session storage.
func (m *MetricMetadata) MarshalSeriesHLL() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.seriesHLL == nil {
		return nil, nil
	}
	return m.seriesHLL.MarshalBinary()
}

// UnmarshalSeriesHLL deserializes series HLL from bytes.
func (m *MetricMetadata) UnmarshalSeriesHLL(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(data) == 0 {
		return nil
	}

	hll, err := hyperloglog.FromBytes(data)
	if err != nil {
		return err
	}

	m.seriesHLL = hll
	m.ActiveSeries = int64(hll.Count())
	return nil
}

// GetSeriesHLL returns the series HLL for session serialization.
func (m *MetricMetadata) GetSeriesHLL() *hyperloglog.HyperLogLog {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.seriesHLL
}

// SetSeriesHLL sets the series HLL from session deserialization.
func (m *MetricMetadata) SetSeriesHLL(hll *hyperloglog.HyperLogLog) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seriesHLL = hll
	if hll != nil {
		m.ActiveSeries = int64(hll.Count())
	}
}

// ServiceOverview contains a summary of all telemetry for a service.
// This is used by the API to provide aggregated views across all signal types.
type ServiceOverview struct {
	ServiceName string            `json:"service_name"`
	MetricCount int               `json:"metric_count"`
	SpanCount   int               `json:"span_count"`
	LogCount    int               `json:"log_count"`
	Metrics     []*MetricMetadata `json:"metrics"`
	Spans       []*SpanMetadata   `json:"spans"`
	Logs        []*LogMetadata    `json:"logs"`
}
