package models

// MetadataComplexityResponse represents signals with high metadata complexity
type MetadataComplexityResponse struct {
	Signals   []SignalComplexity `json:"signals"`
	Total     int                `json:"total"`
	Threshold int                `json:"threshold"` // Minimum total key count to be considered complex
}

// SignalComplexity represents a signal with its metadata complexity metrics
type SignalComplexity struct {
	SignalType           string `json:"signal_type"`             // "metric", "span", "log"
	SignalName           string `json:"signal_name"`             // metric name, span name, or severity
	TotalKeys            int    `json:"total_keys"`              // Total unique keys across all scopes
	AttributeKeyCount    int    `json:"attribute_key_count"`     // Number of attribute/label keys
	ResourceKeyCount     int    `json:"resource_key_count"`      // Number of resource keys
	EventKeyCount        int    `json:"event_key_count"`         // Number of event keys (spans only)
	LinkKeyCount         int    `json:"link_key_count"`          // Number of link keys (spans only)
	MaxCardinality       int    `json:"max_cardinality"`         // Highest cardinality among all keys
	HighCardinalityCount int    `json:"high_cardinality_count"`  // Number of keys with cardinality > 100
	ComplexityScore      int    `json:"complexity_score"`        // Total keys × max cardinality
}
