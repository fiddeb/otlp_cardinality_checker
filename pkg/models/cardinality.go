package models

// CrossSignalCardinalityResponse represents high-cardinality keys across all signal types
type CrossSignalCardinalityResponse struct {
	HighCardinalityKeys []SignalKey `json:"high_cardinality_keys"`
	Total               int         `json:"total"`
	Threshold           int         `json:"threshold"`
}

// SignalKey represents a key with its signal type and metadata
type SignalKey struct {
	SignalType          string   `json:"signal_type"`          // "metric", "span", "log"
	SignalName          string   `json:"signal_name"`          // metric name, span name, or severity
	KeyScope            string   `json:"key_scope"`            // "label", "resource", "attribute", etc.
	KeyName             string   `json:"key_name"`             // The actual key name
	EventName           string   `json:"event_name,omitempty"` // For span events
	EstimatedCardinality int     `json:"estimated_cardinality"`
	KeyCount            int64    `json:"key_count"`
	ValueSamples        []string `json:"value_samples,omitempty"`
}
