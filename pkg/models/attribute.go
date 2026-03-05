package models

import (
	"strings"
	"sync"
	"time"

	"github.com/fidde/otlp_cardinality_checker/pkg/hyperloglog"
)

// AttributeMetadata tracks metadata for a single attribute key across all telemetry signals.
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

	// HasInvalidUTF8 is true when at least one observed value for this key
	// contained invalid UTF-8 bytes that were replaced with U+FFFD by the
	// receiver's sanitizeUTF8 helper. Sticky: once set it stays set.
	HasInvalidUTF8 bool `json:"has_invalid_utf8,omitempty"`
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

	// Flag if the value contains the Unicode replacement character, which our
	// sanitizeUTF8 receiver helper inserts in place of invalid UTF-8 bytes.
	if strings.ContainsRune(value, '\uFFFD') {
		a.HasInvalidUTF8 = true
	}
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

	// Propagate invalid-UTF-8 flag (sticky)
	if a2.HasInvalidUTF8 {
		a1.HasInvalidUTF8 = true
	}

	return a1
}

// WatchedAttribute holds full value-frequency data for a deep-watched attribute key.
// It is separate from AttributeMetadata to avoid touching the hot path for all attributes.
type WatchedAttribute struct {
	mu sync.RWMutex

	// Key is the attribute key being watched.
	Key string `json:"key"`

	// Values maps unique observed values to their occurrence count.
	Values map[string]int64 `json:"values"`

	// UniqueCount is len(Values), cached to avoid lock on read.
	UniqueCount int64 `json:"unique_count"`

	// TotalObservations is the total number of AddValue calls since watching started.
	TotalObservations int64 `json:"total_observations"`

	// Active is true when the watch is collecting new values.
	// False when restored from a session (read-only view of historical data).
	Active bool `json:"active"`

	// Overflow is true when the unique value cap was reached; new unique values are ignored.
	Overflow bool `json:"overflow"`

	// WatchingSince is when deep watch was first activated for this key.
	WatchingSince time.Time `json:"watching_since"`

	// MaxValues is the unique-value cap (default 10,000).
	MaxValues int `json:"-"`
}

// NewWatchedAttribute creates a new WatchedAttribute ready for collection.
func NewWatchedAttribute(key string, maxValues int) *WatchedAttribute {
	if maxValues <= 0 {
		maxValues = 10000
	}
	return &WatchedAttribute{
		Key:           key,
		Values:        make(map[string]int64),
		Active:        true,
		WatchingSince: time.Now(),
		MaxValues:     maxValues,
	}
}

// AddValue records a value observation. It is safe for concurrent use.
// When the watch is not Active (restored session) observations are ignored.
func (w *WatchedAttribute) AddValue(value string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.Active {
		return
	}

	w.TotalObservations++

	if existing, ok := w.Values[value]; ok {
		// Value already tracked: increment its count.
		w.Values[value] = existing + 1
		return
	}

	// New unique value.
	if w.Overflow || int(w.UniqueCount) >= w.MaxValues {
		w.Overflow = true
		return
	}

	w.Values[value] = 1
	w.UniqueCount++
}

// Snapshot returns a consistent read of the watched attribute's current state.
// The returned Values map is a copy safe to read without holding the lock.
func (w *WatchedAttribute) Snapshot() (key string, values map[string]int64, uniqueCount, totalObs int64, active, overflow bool, since time.Time) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	valCopy := make(map[string]int64, len(w.Values))
	for k, v := range w.Values {
		valCopy[k] = v
	}
	return w.Key, valCopy, w.UniqueCount, w.TotalObservations, w.Active, w.Overflow, w.WatchingSince
}

// SetActive safely sets the Active flag under the lock.
func (w *WatchedAttribute) SetActive(active bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.Active = active
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
