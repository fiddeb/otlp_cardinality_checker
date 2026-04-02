package report

import (
	"fmt"
	"strings"
)

// FormatText formats a report as human-readable plain text.
func FormatText(r *Report) ([]byte, error) {
	var b strings.Builder

	b.WriteString("OCC Telemetry Report\n")
	b.WriteString("====================\n")
	fmt.Fprintf(&b, "Generated: %s\n", r.GeneratedAt.Format("2006-01-02 15:04:05 UTC"))
	if r.Duration != "" {
		fmt.Fprintf(&b, "Duration:  %s\n", r.Duration)
	}
	fmt.Fprintf(&b, "Version:   %s\n", r.OCCVersion)

	b.WriteString("\nSummary\n")
	b.WriteString("-------\n")
	fmt.Fprintf(&b, "Metrics:       %d\n", r.Summary.TotalMetrics)
	fmt.Fprintf(&b, "Spans:         %d\n", r.Summary.TotalSpanNames)
	fmt.Fprintf(&b, "Log patterns:  %d\n", r.Summary.TotalLogPatterns)
	fmt.Fprintf(&b, "Attributes:    %d\n", r.Summary.TotalAttributes)
	fmt.Fprintf(&b, "High cardinality: %d\n", r.Summary.HighCardinalityCount)

	if len(r.Metrics) > 0 {
		b.WriteString("\nMetrics (sorted by cardinality)\n")
		b.WriteString("-------------------------------\n")
		for _, m := range r.Metrics {
			tag := severityTag(m.Severity)
			typeSuffix := ""
			if m.Type != "" {
				typeSuffix = " (" + m.Type + ")"
			}
			fmt.Fprintf(&b, "%-9s %s%s\n", tag, m.Name, typeSuffix)
			fmt.Fprintf(&b, "          Labels: %s\n", strings.Join(m.LabelKeys, ", "))
			fmt.Fprintf(&b, "          Cardinality: %s | Samples: %s\n",
				formatNumber(m.EstimatedCardinality), formatNumber(m.SampleCount))
			b.WriteString("\n")
		}
	}

	if len(r.Spans) > 0 {
		b.WriteString("Spans (sorted by cardinality)\n")
		b.WriteString("-----------------------------\n")
		for _, s := range r.Spans {
			tag := severityTag(s.Severity)
			fmt.Fprintf(&b, "%-9s %s\n", tag, s.Name)
			fmt.Fprintf(&b, "          Attributes: %s\n", strings.Join(s.AttributeKeys, ", "))
			fmt.Fprintf(&b, "          Cardinality: %s | Spans: %s\n",
				formatNumber(s.EstimatedCardinality), formatNumber(s.SpanCount))
			b.WriteString("\n")
		}
	}

	if len(r.Logs) > 0 {
		b.WriteString("Logs (sorted by cardinality)\n")
		b.WriteString("----------------------------\n")
		for _, l := range r.Logs {
			tag := severityTag(l.SeverityLevel)
			fmt.Fprintf(&b, "%-9s %s\n", tag, l.Severity)
			fmt.Fprintf(&b, "          Attributes: %s\n", strings.Join(l.AttributeKeys, ", "))
			fmt.Fprintf(&b, "          Cardinality: %s | Logs: %s\n",
				formatNumber(l.EstimatedCardinality), formatNumber(l.LogCount))
			b.WriteString("\n")
		}
	}

	if len(r.Attributes) > 0 {
		b.WriteString("Attributes (cross-signal)\n")
		b.WriteString("-------------------------\n")
		for _, a := range r.Attributes {
			tag := severityTag(a.Severity)
			signals := strings.Join(a.SignalTypes, ", ")
			fmt.Fprintf(&b, "%-9s %s — %s — ~%s unique values\n",
				tag, a.Key, signals, formatNumber(a.EstimatedUniqueValues))
		}
	}

	return []byte(b.String()), nil
}

func severityTag(sev string) string {
	switch sev {
	case SeverityCritical:
		return "CRITICAL"
	case SeverityWarning:
		return "WARNING"
	default:
		return "OK"
	}
}

// formatNumber formats an int64 with thousand separators.
func formatNumber(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteByte(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}
