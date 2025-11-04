package models

import (
	"sync"
	"time"

	"github.com/fidde/otlp_cardinality_checker/pkg/hyperloglog"
)

// AttributeMetadata tracks metadata for a single attribute key across all signals.
// It uses HyperLogLog to estimate cardinality of unique values for the attribute.
type AttributeMetadata struct {
	mu sync.RWMutex

	// Key is the attribute key name (e.g., "user_id", "http.method")
	Key string `json:"key"`

	// hll is the HyperLogLog sketch for estimating unique value cardinality
	hll *hyperloglog.HyperLogLog

	// Count is the total number of times this attribute key has been observed
	Count int64 `json:"count"`

	// EstimatedCardinality is the HLL-estimated number of unique values
	EstimatedCardinality int64 `json:"estimated_cardinality"`

	// ValueSamples contains up to MaxSamples example values
	ValueSamples []string `json:"value_samples"`

	// SignalTypes tracks which signal types use this attribute (metric, span, log)
	SignalTypes []string `json:"signal_types"`

	// Scope tracks whether this is a resource or regular attribute
	// Values: "resource", "attribute", "both"
	Scope string `json:"scope"`

	// FirstSeen is when this attribute key was first observed
	FirstSeen time.Time `json:"first_seen"`

	// LastSeen is when this attribute key was last observed
	LastSeen time.Time `json:"last_seen"`
}

// NewAttributeMetadata creates a new AttributeMetadata with initialized HLL.
func NewAttributeMetadata(key string) *AttributeMetadata {
	const maxSamples = 10 // Keep first 10 unique values for sampling
	return &AttributeMetadata{
		Key:                  key,
		hll:                  hyperloglog.New(14), // Precision 14 = 16KB memory, 0.81% error
		Count:                0,
		EstimatedCardinality: 0,
		ValueSamples:         make([]string, 0, maxSamples),
		SignalTypes:          make([]string, 0, 3), // max 3: metric, span, log
		Scope:                "",
		FirstSeen:            time.Now(),
		LastSeen:             time.Now(),
	}
}

// AddValue adds a value observation to the attribute metadata.
// It updates HLL cardinality, samples, and timestamps.
func (a *AttributeMetadata) AddValue(value, signalType, scope string) {
	const maxSamples = 10
	a.mu.Lock()
	defer a.mu.Unlock()

	// Update HLL sketch
	a.hll.Add(value)
	a.EstimatedCardinality = int64(a.hll.Count())

	// Update count
	a.Count++

	// Add to samples if not already present and under limit
	if len(a.ValueSamples) < maxSamples {
		found := false
		for _, s := range a.ValueSamples {
			if s == value {
				found = true
				break
			}
		}
		if !found {
			a.ValueSamples = append(a.ValueSamples, value)
		}
	}

	// Add signal type if not already present
	found := false
	for _, st := range a.SignalTypes {
		if st == signalType {
			found = true
			break
		}
	}
	if !found {
		a.SignalTypes = append(a.SignalTypes, signalType)
	}

	// Update scope (resource, attribute, or both)
	if a.Scope == "" {
		a.Scope = scope
	} else if a.Scope != scope {
		a.Scope = "both"
	}

	// Update timestamp
	a.LastSeen = time.Now()
}

// MarshalHLL serializes the HLL sketch for persistence.
func (a *AttributeMetadata) MarshalHLL() ([]byte, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.hll == nil {
		return nil, nil
	}

	return a.hll.MarshalBinary()
}

// UnmarshalHLL deserializes the HLL sketch from storage.
func (a *AttributeMetadata) UnmarshalHLL(data []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(data) == 0 {
		return nil
	}

	hll, err := hyperloglog.FromBytes(data)
	if err != nil {
		return err
	}

	a.hll = hll
	a.EstimatedCardinality = int64(hll.Count())
	return nil
}

// MergeAttributeMetadata merges two attribute metadata objects.
// Used when combining data from multiple sources or during persistence.
func MergeAttributeMetadata(a1, a2 *AttributeMetadata) *AttributeMetadata {
	const maxSamples = 10
	if a1 == nil {
		return a2
	}
	if a2 == nil {
		return a1
	}

	a1.mu.Lock()
	defer a1.mu.Unlock()

	a2.mu.RLock()
	defer a2.mu.RUnlock()

	// Merge HLL sketches
	if a1.hll != nil && a2.hll != nil {
		a1.hll.Merge(a2.hll)
		a1.EstimatedCardinality = int64(a1.hll.Count())
	}

	// Merge counts
	a1.Count += a2.Count

	// Merge samples (deduplicate)
	sampleSet := make(map[string]bool)
	for _, s := range a1.ValueSamples {
		sampleSet[s] = true
	}
	for _, s := range a2.ValueSamples {
		if !sampleSet[s] && len(a1.ValueSamples) < maxSamples {
			a1.ValueSamples = append(a1.ValueSamples, s)
			sampleSet[s] = true
		}
	}

	// Merge signal types (deduplicate)
	signalSet := make(map[string]bool)
	for _, st := range a1.SignalTypes {
		signalSet[st] = true
	}
	for _, st := range a2.SignalTypes {
		if !signalSet[st] {
			a1.SignalTypes = append(a1.SignalTypes, st)
		}
	}

	// Merge scope
	if a1.Scope != a2.Scope && a1.Scope != "both" {
		a1.Scope = "both"
	}

	// Update timestamps
	if a2.FirstSeen.Before(a1.FirstSeen) {
		a1.FirstSeen = a2.FirstSeen
	}
	if a2.LastSeen.After(a1.LastSeen) {
		a1.LastSeen = a2.LastSeen
	}

	return a1
}

// AttributeFilter defines filtering options for listing attributes.
type AttributeFilter struct {
	// SignalType filters by signal type (metric, span, log)
	SignalType string

	// Scope filters by scope (resource, attribute, both)
	Scope string

	// MinCardinality filters attributes with cardinality >= this value
	MinCardinality int64

	// MaxCardinality filters attributes with cardinality <= this value
	MaxCardinality int64

	// SortBy specifies the sort field (cardinality, count, key, first_seen, last_seen)
	SortBy string

	// SortOrder specifies sort direction (asc, desc)
	SortOrder string

	// Limit specifies maximum number of results
	Limit int

	// Offset specifies pagination offset
	Offset int
}
