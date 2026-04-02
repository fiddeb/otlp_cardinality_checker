package report

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestCardinalitySeverity(t *testing.T) {
	tests := []struct {
		cardinality int64
		want        string
	}{
		{0, SeverityOK},
		{999, SeverityOK},
		{1000, SeverityWarning},
		{5000, SeverityWarning},
		{9999, SeverityWarning},
		{10000, SeverityCritical},
		{50000, SeverityCritical},
	}
	for _, tt := range tests {
		got := CardinalitySeverity(tt.cardinality)
		if got != tt.want {
			t.Errorf("CardinalitySeverity(%d) = %q, want %q", tt.cardinality, got, tt.want)
		}
	}
}

func TestMaxExitCode(t *testing.T) {
	tests := []struct {
		name string
		rpt  Report
		want int
	}{
		{
			name: "empty report",
			rpt:  Report{},
			want: 0,
		},
		{
			name: "all ok",
			rpt: Report{
				Metrics: []MetricItem{{Severity: SeverityOK}},
				Spans:   []SpanItem{{Severity: SeverityOK}},
			},
			want: 0,
		},
		{
			name: "warning in metrics",
			rpt: Report{
				Metrics: []MetricItem{{Severity: SeverityWarning}},
			},
			want: 1,
		},
		{
			name: "critical in spans",
			rpt: Report{
				Spans: []SpanItem{{Severity: SeverityCritical}},
			},
			want: 2,
		},
		{
			name: "warning and critical mixed",
			rpt: Report{
				Metrics:    []MetricItem{{Severity: SeverityWarning}},
				Attributes: []AttrItem{{Severity: SeverityCritical}},
			},
			want: 2,
		},
		{
			name: "critical in logs",
			rpt: Report{
				Logs: []LogItem{{SeverityLevel: SeverityCritical}},
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rpt.MaxExitCode()
			if got != tt.want {
				t.Errorf("MaxExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFormatJSON_Roundtrip(t *testing.T) {
	rpt := sampleReport()

	data, err := FormatJSON(rpt)
	if err != nil {
		t.Fatalf("FormatJSON: %v", err)
	}

	var decoded Report
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if decoded.Version != rpt.Version {
		t.Errorf("version = %q, want %q", decoded.Version, rpt.Version)
	}
	if decoded.Summary.TotalMetrics != rpt.Summary.TotalMetrics {
		t.Errorf("total_metrics = %d, want %d", decoded.Summary.TotalMetrics, rpt.Summary.TotalMetrics)
	}
	if len(decoded.Metrics) != len(rpt.Metrics) {
		t.Errorf("metrics count = %d, want %d", len(decoded.Metrics), len(rpt.Metrics))
	}
}

func TestFormatText_ContainsExpected(t *testing.T) {
	rpt := sampleReport()
	data, err := FormatText(rpt)
	if err != nil {
		t.Fatalf("FormatText: %v", err)
	}
	text := string(data)

	expects := []string{
		"OCC Telemetry Report",
		"CRITICAL",
		"user_events",
		"OK",
		"http_requests_total",
	}
	for _, want := range expects {
		if !strings.Contains(text, want) {
			t.Errorf("text report missing %q", want)
		}
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{12345, "12,345"},
		{1234567, "1,234,567"},
	}
	for _, tt := range tests {
		got := formatNumber(tt.n)
		if got != tt.want {
			t.Errorf("formatNumber(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func sampleReport() *Report {
	return &Report{
		Version:     "1.0",
		GeneratedAt: time.Date(2026, 1, 25, 10, 30, 0, 0, time.UTC),
		Duration:    "5m0s",
		OCCVersion:  "dev",
		Summary: Summary{
			TotalMetrics:         2,
			TotalSpanNames:       1,
			HighCardinalityCount: 1,
		},
		Metrics: []MetricItem{
			{Name: "user_events", Type: "Sum", LabelKeys: []string{"user_id"}, SampleCount: 50000, EstimatedCardinality: 12000, Severity: SeverityCritical},
			{Name: "http_requests_total", Type: "Sum", LabelKeys: []string{"method", "status"}, SampleCount: 5000, EstimatedCardinality: 120, Severity: SeverityOK},
		},
		Spans: []SpanItem{
			{Name: "GET /api", AttributeKeys: []string{"http.method"}, SpanCount: 1000, EstimatedCardinality: 50, Severity: SeverityOK},
		},
		Logs: []LogItem{
			{Severity: "INFO", AttributeKeys: []string{"user.id"}, LogCount: 2000, EstimatedCardinality: 500, SeverityLevel: SeverityOK},
		},
		Attributes: []AttrItem{
			{Key: "user_id", SignalTypes: []string{"metric", "span"}, EstimatedUniqueValues: 5000, Severity: SeverityWarning},
		},
	}
}
