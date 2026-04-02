// Package report provides cardinality report generation for CI/CD mode.
package report

import "time"

// Report is the top-level cardinality report.
type Report struct {
	Version     string       `json:"version"`
	GeneratedAt time.Time   `json:"generated_at"`
	Duration    string       `json:"duration,omitempty"`
	OCCVersion  string       `json:"occ_version"`
	Summary     Summary      `json:"summary"`
	Metrics     []MetricItem `json:"metrics"`
	Spans       []SpanItem   `json:"spans"`
	Logs        []LogItem    `json:"logs"`
	Attributes  []AttrItem   `json:"attributes"`
}

// Summary provides aggregate counts.
type Summary struct {
	TotalMetrics         int          `json:"total_metrics"`
	TotalSpanNames       int          `json:"total_span_names"`
	TotalLogPatterns     int          `json:"total_log_patterns"`
	TotalAttributes      int          `json:"total_attributes"`
	HighCardinalityCount int          `json:"high_cardinality_count"`
	Samples              SampleCounts `json:"samples"`
}

// SampleCounts tracks total samples per signal type.
type SampleCounts struct {
	Metrics int64 `json:"metrics"`
	Spans   int64 `json:"spans"`
	Logs    int64 `json:"logs"`
}

// MetricItem represents one metric in the report.
type MetricItem struct {
	Name                 string   `json:"name"`
	Type                 string   `json:"type"`
	LabelKeys            []string `json:"label_keys"`
	SampleCount          int64    `json:"sample_count"`
	EstimatedCardinality int64    `json:"estimated_cardinality"`
	Severity             string   `json:"severity"`
}

// SpanItem represents one span name in the report.
type SpanItem struct {
	Name                 string   `json:"name"`
	AttributeKeys        []string `json:"attribute_keys"`
	SpanCount            int64    `json:"span_count"`
	EstimatedCardinality int64    `json:"estimated_cardinality"`
	Severity             string   `json:"severity"`
}

// LogItem represents one log pattern in the report.
type LogItem struct {
	Severity             string   `json:"severity_text"`
	AttributeKeys        []string `json:"attribute_keys"`
	LogCount             int64    `json:"log_count"`
	EstimatedCardinality int64    `json:"estimated_cardinality"`
	SeverityLevel        string   `json:"severity"`
}

// AttrItem represents one attribute in the cross-signal report.
type AttrItem struct {
	Key                   string   `json:"key"`
	SignalTypes           []string `json:"signal_types"`
	EstimatedUniqueValues int64    `json:"estimated_unique_values"`
	Severity              string   `json:"severity"`
}

// Severity thresholds.
const (
	SeverityOK       = "ok"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"

	ThresholdWarning  = 1000
	ThresholdCritical = 10000
)

// CardinalitySeverity returns the severity level for a given cardinality.
func CardinalitySeverity(cardinality int64) string {
	switch {
	case cardinality >= ThresholdCritical:
		return SeverityCritical
	case cardinality >= ThresholdWarning:
		return SeverityWarning
	default:
		return SeverityOK
	}
}

// MaxExitCode returns the highest exit code based on all items' severities.
// 0 = ok, 1 = warning, 2 = critical.
func (r *Report) MaxExitCode() int {
	code := 0
	check := func(sev string) {
		switch sev {
		case SeverityCritical:
			code = 2
		case SeverityWarning:
			if code < 1 {
				code = 1
			}
		}
	}
	for _, m := range r.Metrics {
		check(m.Severity)
	}
	for _, s := range r.Spans {
		check(s.Severity)
	}
	for _, l := range r.Logs {
		check(l.SeverityLevel)
	}
	for _, a := range r.Attributes {
		check(a.Severity)
	}
	return code
}
