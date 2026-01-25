// Package models defines the core data structures for metadata tracking.
package models

// Severity levels for changes
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

// Change types
const (
	ChangeTypeAdded   = "added"
	ChangeTypeRemoved = "removed"
	ChangeTypeChanged = "changed"
)

// Signal types
const (
	SignalTypeMetric    = "metric"
	SignalTypeSpan      = "span"
	SignalTypeLog       = "log"
	SignalTypeAttribute = "attribute"
)

// DiffRequest contains parameters for comparing two sessions.
type DiffRequest struct {
	// From is the baseline session name
	From string `json:"from"`

	// To is the comparison session name
	To string `json:"to"`

	// SignalType filters to a specific signal type (optional)
	SignalType string `json:"signal_type,omitempty"`

	// Service filters to a specific service (optional)
	Service string `json:"service,omitempty"`

	// MinSeverity filters to changes at or above this severity (optional)
	MinSeverity string `json:"min_severity,omitempty"`
}

// DiffResult contains the result of comparing two sessions.
type DiffResult struct {
	// From is the baseline session name
	From string `json:"from"`

	// To is the comparison session name
	To string `json:"to"`

	// Summary contains aggregate counts
	Summary DiffSummary `json:"summary"`

	// Changes contains detailed change information
	Changes DiffChanges `json:"changes"`

	// CriticalChanges lists the most important changes
	CriticalChanges []Change `json:"critical_changes,omitempty"`
}

// DiffSummary contains aggregate change counts per signal type.
type DiffSummary struct {
	Metrics    SignalDiffSummary `json:"metrics"`
	Spans      SignalDiffSummary `json:"spans"`
	Logs       SignalDiffSummary `json:"logs"`
	Attributes SignalDiffSummary `json:"attributes"`
}

// SignalDiffSummary contains counts for a single signal type.
type SignalDiffSummary struct {
	Added   int `json:"added"`
	Removed int `json:"removed"`
	Changed int `json:"changed"`
}

// DiffChanges contains detailed changes grouped by signal type.
type DiffChanges struct {
	Metrics    SignalChanges `json:"metrics,omitempty"`
	Spans      SignalChanges `json:"spans,omitempty"`
	Logs       SignalChanges `json:"logs,omitempty"`
	Attributes SignalChanges `json:"attributes,omitempty"`
}

// SignalChanges contains added/removed/changed items for a signal type.
type SignalChanges struct {
	Added   []Change `json:"added,omitempty"`
	Removed []Change `json:"removed,omitempty"`
	Changed []Change `json:"changed,omitempty"`
}

// Change represents a single difference between two sessions.
type Change struct {
	// Type is "added", "removed", or "changed"
	Type string `json:"type"`

	// SignalType is "metric", "span", "log", or "attribute"
	SignalType string `json:"signal_type"`

	// Name is the signal name (metric name, span name, severity, attribute key)
	Name string `json:"name"`

	// Severity is "info", "warning", or "critical"
	Severity string `json:"severity"`

	// Details contains specific field changes (for "changed" type)
	Details []FieldChange `json:"details,omitempty"`

	// Metadata contains additional context
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// FieldChange represents a change to a specific field.
type FieldChange struct {
	// Field is the path to the changed field (e.g., "labels.user_id.cardinality")
	Field string `json:"field"`

	// From is the original value
	From interface{} `json:"from"`

	// To is the new value
	To interface{} `json:"to"`

	// ChangePct is the percentage change (for numeric values)
	ChangePct float64 `json:"change_pct,omitempty"`

	// Severity is the severity of this specific field change
	Severity string `json:"severity"`

	// Message is a human-readable description of the change
	Message string `json:"message,omitempty"`
}

// CalculateSeverity determines the severity of a cardinality change.
func CalculateSeverity(from, to int64) string {
	if from == 0 {
		if to >= 1000 {
			return SeverityWarning // New high-cardinality attribute
		}
		return SeverityInfo
	}

	ratio := float64(to) / float64(from)

	if ratio >= 10.0 {
		return SeverityCritical // 10x increase
	}
	if ratio >= 2.0 {
		return SeverityWarning // 2x increase
	}
	return SeverityInfo
}

// CalculateSampleRateSeverity determines the severity of a sample rate change.
func CalculateSampleRateSeverity(from, to int64) string {
	if from == 0 {
		return SeverityInfo
	}

	ratio := float64(to) / float64(from)

	if ratio >= 5.0 {
		return SeverityWarning // 5x increase in sample rate
	}
	return SeverityInfo
}

// MaxSeverity returns the higher severity between two.
func MaxSeverity(a, b string) string {
	severityRank := map[string]int{
		SeverityInfo:     0,
		SeverityWarning:  1,
		SeverityCritical: 2,
	}

	if severityRank[a] >= severityRank[b] {
		return a
	}
	return b
}

// FilterBySeverity filters changes to only include those at or above minSeverity.
func FilterBySeverity(changes []Change, minSeverity string) []Change {
	if minSeverity == "" || minSeverity == SeverityInfo {
		return changes
	}

	severityRank := map[string]int{
		SeverityInfo:     0,
		SeverityWarning:  1,
		SeverityCritical: 2,
	}

	minRank := severityRank[minSeverity]
	filtered := make([]Change, 0)

	for _, c := range changes {
		if severityRank[c.Severity] >= minRank {
			filtered = append(filtered, c)
		}
	}

	return filtered
}

// NewDiffResult creates an empty DiffResult.
func NewDiffResult(from, to string) *DiffResult {
	return &DiffResult{
		From:    from,
		To:      to,
		Summary: DiffSummary{},
		Changes: DiffChanges{},
	}
}

// AddChange adds a change to the appropriate category in DiffResult.
func (d *DiffResult) AddChange(change Change) {
	// Update summary
	switch change.SignalType {
	case SignalTypeMetric:
		d.addToSignalChanges(&d.Changes.Metrics, &d.Summary.Metrics, change)
	case SignalTypeSpan:
		d.addToSignalChanges(&d.Changes.Spans, &d.Summary.Spans, change)
	case SignalTypeLog:
		d.addToSignalChanges(&d.Changes.Logs, &d.Summary.Logs, change)
	case SignalTypeAttribute:
		d.addToSignalChanges(&d.Changes.Attributes, &d.Summary.Attributes, change)
	}

	// Track critical changes
	if change.Severity == SeverityCritical {
		d.CriticalChanges = append(d.CriticalChanges, change)
	}
}

func (d *DiffResult) addToSignalChanges(changes *SignalChanges, summary *SignalDiffSummary, change Change) {
	switch change.Type {
	case ChangeTypeAdded:
		changes.Added = append(changes.Added, change)
		summary.Added++
	case ChangeTypeRemoved:
		changes.Removed = append(changes.Removed, change)
		summary.Removed++
	case ChangeTypeChanged:
		changes.Changed = append(changes.Changed, change)
		summary.Changed++
	}
}
