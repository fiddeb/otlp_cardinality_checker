// Package models defines the core data structures for metadata tracking.
package models

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"github.com/fidde/otlp_cardinality_checker/pkg/hyperloglog"
)

// Session naming validation
var sessionNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]*[a-z0-9]$|^[a-z0-9]$`)

// Session errors
var (
	ErrSessionNotFound     = errors.New("session not found")
	ErrSessionExists       = errors.New("session already exists")
	ErrInvalidSessionName  = errors.New("invalid session name: must be lowercase alphanumeric with hyphens")
	ErrSessionTooLarge     = errors.New("session exceeds size limit")
	ErrTooManySessions     = errors.New("maximum number of sessions reached")
)

// ValidateSessionName checks if a session name is valid.
// Names must be lowercase alphanumeric with hyphens, no spaces or special chars.
func ValidateSessionName(name string) error {
	if name == "" {
		return ErrInvalidSessionName
	}
	if len(name) > 128 {
		return ErrInvalidSessionName
	}
	if !sessionNameRegex.MatchString(name) {
		return ErrInvalidSessionName
	}
	return nil
}

// SessionMetadata contains information about a saved session without the full data.
type SessionMetadata struct {
	// ID is the unique session identifier (name)
	ID string `json:"id"`

	// Description is an optional user-provided description
	Description string `json:"description,omitempty"`

	// Created is when the session was saved
	Created time.Time `json:"created"`

	// Signals lists which signal types are included
	Signals []string `json:"signals"`

	// SizeBytes is the compressed file size
	SizeBytes int64 `json:"size_bytes"`

	// Stats contains summary counts
	Stats SessionStats `json:"stats"`
}

// SessionStats contains summary statistics for a session.
type SessionStats struct {
	MetricsCount    int      `json:"metrics_count"`
	SpansCount      int      `json:"spans_count"`
	LogsCount       int      `json:"logs_count"`
	AttributesCount int      `json:"attributes_count"`
	Services        []string `json:"services"`
}

// Session represents a complete snapshot of telemetry metadata.
type Session struct {
	// Version is the session format version for future compatibility
	Version int `json:"version"`

	// ID is the unique session identifier
	ID string `json:"id"`

	// Description is an optional user-provided description
	Description string `json:"description,omitempty"`

	// Created is when the session was saved
	Created time.Time `json:"created"`

	// Signals lists which signal types are included
	Signals []string `json:"signals"`

	// Data contains the actual telemetry metadata
	Data SessionData `json:"data"`

	// Stats contains summary counts
	Stats SessionStats `json:"stats"`
}

// SessionData contains the serializable telemetry metadata.
type SessionData struct {
	Metrics    []*SerializedMetric    `json:"metrics,omitempty"`
	Spans      []*SerializedSpan      `json:"spans,omitempty"`
	Logs       []*SerializedLog       `json:"logs,omitempty"`
	Attributes []*SerializedAttribute `json:"attributes,omitempty"`
}

// SerializedMetric is a JSON-serializable version of MetricMetadata.
type SerializedMetric struct {
	Name         string                       `json:"name"`
	Description  string                       `json:"description,omitempty"`
	Unit         string                       `json:"unit,omitempty"`
	Type         string                       `json:"type"`
	LabelKeys    map[string]*SerializedKey    `json:"label_keys"`
	ResourceKeys map[string]*SerializedKey    `json:"resource_keys"`
	SampleCount  int64                        `json:"sample_count"`
	Services     map[string]int64             `json:"services"`
	ActiveSeries int64                        `json:"active_series"`
	SeriesHLL    *SerializedHLL               `json:"series_hll,omitempty"`
}

// SerializedSpan is a JSON-serializable version of SpanMetadata.
type SerializedSpan struct {
	Name               string                                 `json:"name"`
	Kind               int32                                  `json:"kind"`
	KindName           string                                 `json:"kind_name,omitempty"`
	AttributeKeys      map[string]*SerializedKey              `json:"attribute_keys"`
	EventNames         []string                               `json:"event_names"`
	EventAttributeKeys map[string]map[string]*SerializedKey   `json:"event_attribute_keys,omitempty"`
	LinkAttributeKeys  map[string]*SerializedKey              `json:"link_attribute_keys,omitempty"`
	ResourceKeys       map[string]*SerializedKey              `json:"resource_keys"`
	StatusCodes        []string                               `json:"status_codes,omitempty"`
	NamePatterns       []*SpanNamePattern                     `json:"name_patterns,omitempty"`
	SampleCount        int64                                  `json:"sample_count"`
	Services           map[string]int64                       `json:"services"`
}

// SerializedLog is a JSON-serializable version of LogMetadata.
type SerializedLog struct {
	Severity       string                    `json:"severity"`
	SeverityNumber int32                     `json:"severity_number,omitempty"`
	AttributeKeys  map[string]*SerializedKey `json:"attribute_keys"`
	ResourceKeys   map[string]*SerializedKey `json:"resource_keys"`
	BodyTemplates  []*BodyTemplate           `json:"body_templates,omitempty"`
	EventNames     []string                  `json:"event_names,omitempty"`
	SampleCount    int64                     `json:"sample_count"`
	Services       map[string]int64          `json:"services"`
}

// SerializedAttribute is a JSON-serializable version of AttributeMetadata.
type SerializedAttribute struct {
	Key                  string         `json:"key"`
	Count                int64          `json:"count"`
	EstimatedCardinality int64          `json:"estimated_cardinality"`
	ValueSamples         []string       `json:"value_samples,omitempty"`
	SignalTypes          []string       `json:"signal_types"`
	Scope                string         `json:"scope"`
	FirstSeen            time.Time      `json:"first_seen"`
	LastSeen             time.Time      `json:"last_seen"`
	HLL                  *SerializedHLL `json:"hll,omitempty"`
}

// SerializedKey is a JSON-serializable version of KeyMetadata with HLL state.
type SerializedKey struct {
	Count                int64          `json:"count"`
	Percentage           float64        `json:"percentage"`
	EstimatedCardinality int64          `json:"estimated_cardinality"`
	ValueSamples         []string       `json:"value_samples,omitempty"`
	HLL                  *SerializedHLL `json:"hll,omitempty"`
}

// SerializedHLL contains HyperLogLog state for JSON serialization.
type SerializedHLL struct {
	Precision uint8  `json:"precision"`
	Registers string `json:"registers"` // base64-encoded
}

// MarshalHLL serializes an HLL to SerializedHLL.
func MarshalHLL(hll *hyperloglog.HyperLogLog) (*SerializedHLL, error) {
	if hll == nil {
		return nil, nil
	}

	data, err := hll.MarshalBinary()
	if err != nil {
		return nil, err
	}

	if len(data) < 1 {
		return nil, nil
	}

	return &SerializedHLL{
		Precision: data[0],
		Registers: base64.StdEncoding.EncodeToString(data[1:]),
	}, nil
}

// UnmarshalHLL deserializes a SerializedHLL to HyperLogLog.
func UnmarshalHLL(s *SerializedHLL) (*hyperloglog.HyperLogLog, error) {
	if s == nil {
		return nil, nil
	}

	registers, err := base64.StdEncoding.DecodeString(s.Registers)
	if err != nil {
		return nil, err
	}

	// Reconstruct binary format: [precision:1byte][registers:m bytes]
	data := make([]byte, 1+len(registers))
	data[0] = s.Precision
	copy(data[1:], registers)

	return hyperloglog.FromBytes(data)
}

// SerializeKeyMetadata converts KeyMetadata to SerializedKey.
func SerializeKeyMetadata(k *KeyMetadata) (*SerializedKey, error) {
	if k == nil {
		return nil, nil
	}

	sk := &SerializedKey{
		Count:                k.Count,
		Percentage:           k.Percentage,
		EstimatedCardinality: k.EstimatedCardinality,
		ValueSamples:         k.GetSortedSamples(),
	}

	// Serialize HLL if present
	hllBytes, err := k.MarshalHLL()
	if err != nil {
		return nil, err
	}
	if len(hllBytes) > 0 {
		sk.HLL = &SerializedHLL{
			Precision: hllBytes[0],
			Registers: base64.StdEncoding.EncodeToString(hllBytes[1:]),
		}
	}

	return sk, nil
}

// DeserializeKeyMetadata converts SerializedKey to KeyMetadata.
func DeserializeKeyMetadata(sk *SerializedKey) (*KeyMetadata, error) {
	if sk == nil {
		return nil, nil
	}

	k := NewKeyMetadata()
	k.Count = sk.Count
	k.Percentage = sk.Percentage
	k.EstimatedCardinality = sk.EstimatedCardinality
	k.ValueSamples = sk.ValueSamples

	// Deserialize HLL if present
	if sk.HLL != nil {
		hll, err := UnmarshalHLL(sk.HLL)
		if err != nil {
			return nil, err
		}
		if hll != nil {
			// Use UnmarshalHLL method to set internal HLL
			hllBytes, _ := hll.MarshalBinary()
			if err := k.UnmarshalHLL(hllBytes); err != nil {
				return nil, err
			}
		}
	}

	return k, nil
}

// SessionSaveOptions contains options for saving a session.
type SessionSaveOptions struct {
	// Name is the session identifier (required)
	Name string `json:"name"`

	// Description is an optional description
	Description string `json:"description,omitempty"`

	// Signals filters which signal types to include (empty = all)
	Signals []string `json:"signals,omitempty"`

	// Services filters which services to include (empty = all)
	Services []string `json:"services,omitempty"`
}

// SessionLoadOptions contains options for loading a session.
type SessionLoadOptions struct {
	// Signals filters which signal types to load (empty = all)
	Signals []string `json:"signals,omitempty"`

	// Services filters which services to load (empty = all)
	Services []string `json:"services,omitempty"`
}

// SessionLoadResult contains the result of loading a session.
type SessionLoadResult struct {
	Loaded        bool   `json:"loaded"`
	SessionID     string `json:"session_id"`
	MetricsLoaded int    `json:"metrics_loaded"`
	SpansLoaded   int    `json:"spans_loaded"`
	LogsLoaded    int    `json:"logs_loaded"`
}

// SessionMergeResult contains the result of merging a session.
type SessionMergeResult struct {
	Merged        bool   `json:"merged"`
	SessionID     string `json:"session_id"`
	MetricsAdded  int    `json:"metrics_added"`
	MetricsMerged int    `json:"metrics_merged"`
	SpansAdded    int    `json:"spans_added"`
	SpansMerged   int    `json:"spans_merged"`
	LogsAdded     int    `json:"logs_added"`
	LogsMerged    int    `json:"logs_merged"`
}

// Validate validates SessionSaveOptions.
func (o *SessionSaveOptions) Validate() error {
	return ValidateSessionName(o.Name)
}

// MarshalJSON for Session ensures proper JSON output.
func (s *Session) MarshalJSON() ([]byte, error) {
	type Alias Session
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	})
}

// containsSignal checks if a signal type is in the list.
func containsSignal(signals []string, signal string) bool {
	if len(signals) == 0 {
		return true // empty = all signals
	}
	for _, s := range signals {
		if s == signal {
			return true
		}
	}
	return false
}

// containsService checks if a service is in the list.
func containsService(services []string, service string) bool {
	if len(services) == 0 {
		return true // empty = all services
	}
	for _, s := range services {
		if s == service {
			return true
		}
	}
	return false
}

// FilterByService checks if metadata has any of the specified services.
func FilterByService(metadataServices map[string]int64, filterServices []string) bool {
	if len(filterServices) == 0 {
		return true // empty = all services
	}
	for _, svc := range filterServices {
		if _, ok := metadataServices[svc]; ok {
			return true
		}
	}
	return false
}
