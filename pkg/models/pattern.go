package models

// PatternExplorerResponse represents patterns with service breakdown
type PatternExplorerResponse struct {
	Patterns []PatternGroup `json:"patterns"`
	Total    int            `json:"total"`
}

// PatternGroup represents a log pattern with all its metadata
type PatternGroup struct {
	Template         string                   `json:"template"`
	ExampleBody      string                   `json:"example_body"`
	TotalCount       int64                    `json:"total_count"`
	SeverityBreakdown map[string]int64        `json:"severity_breakdown"` // severity -> count
	Services         []ServicePatternInfo     `json:"services"`
}

// ServicePatternInfo represents pattern data for a specific service
type ServicePatternInfo struct {
	ServiceName    string              `json:"service_name"`
	SampleCount    int64               `json:"sample_count"`
	Severities     []string            `json:"severities"`      // Severities where this pattern appears for this service
	ResourceKeys   []KeyInfo           `json:"resource_keys"`   // Unique resource keys
	AttributeKeys  []KeyInfo           `json:"attribute_keys"`  // Unique log attribute keys
}

// KeyInfo represents a key with cardinality and sample values
type KeyInfo struct {
	Name                string   `json:"name"`
	Cardinality         int      `json:"cardinality"`
	SampleValues        []string `json:"sample_values,omitempty"`
}
