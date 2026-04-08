package report

import (
	"context"
	"testing"
	"time"

	"github.com/fidde/otlp_cardinality_checker/pkg/models"
)

func TestGenerator_Generate(t *testing.T) {
	store := &mockStorage{
		metrics: []*models.MetricMetadata{
			newTestMetric("http_requests_total", 500, "method", "status"),
			newTestMetric("user_events", 50000, "user_id"),
		},
		spans: []*models.SpanMetadata{
			newTestSpan("GET /api", 1000, "http.method"),
		},
		logs: []*models.LogMetadata{
			newTestLog("INFO", 2000, "source"),
		},
		attrs: []*models.AttributeMetadata{
			{Key: "user_id", EstimatedCardinality: 5000, SignalTypes: []string{"metric", "span"}},
		},
	}

	gen := NewGenerator(store)
	rpt, err := gen.Generate(context.Background(), 5*time.Minute)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if rpt.Version != "1.0" {
		t.Errorf("version = %q, want %q", rpt.Version, "1.0")
	}
	if rpt.Duration != "5m0s" {
		t.Errorf("duration = %q, want %q", rpt.Duration, "5m0s")
	}
	if rpt.Summary.TotalMetrics != 2 {
		t.Errorf("total_metrics = %d, want 2", rpt.Summary.TotalMetrics)
	}
	if rpt.Summary.TotalSpanNames != 1 {
		t.Errorf("total_spans = %d, want 1", rpt.Summary.TotalSpanNames)
	}
	if rpt.Summary.TotalLogPatterns != 1 {
		t.Errorf("total_logs = %d, want 1", rpt.Summary.TotalLogPatterns)
	}
	if rpt.Summary.TotalAttributes != 1 {
		t.Errorf("total_attrs = %d, want 1", rpt.Summary.TotalAttributes)
	}

	// Metrics should be sorted by cardinality descending.
	if len(rpt.Metrics) < 2 {
		t.Fatalf("metrics count = %d, want >= 2", len(rpt.Metrics))
	}

	// The first metric should have higher cardinality than the second.
	if rpt.Metrics[0].EstimatedCardinality < rpt.Metrics[1].EstimatedCardinality {
		t.Errorf("metrics not sorted by cardinality descending")
	}

	// Check severity assignment for attributes.
	if rpt.Attributes[0].Severity != SeverityWarning {
		t.Errorf("attribute severity = %q, want %q", rpt.Attributes[0].Severity, SeverityWarning)
	}
}

func TestGenerator_EmptyStorage(t *testing.T) {
	store := &mockStorage{}
	gen := NewGenerator(store)
	rpt, err := gen.Generate(context.Background(), 0)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if rpt.Summary.TotalMetrics != 0 {
		t.Errorf("total_metrics = %d, want 0", rpt.Summary.TotalMetrics)
	}
	if rpt.Duration != "" {
		t.Errorf("duration = %q, want empty", rpt.Duration)
	}
}

func newTestMetric(name string, sampleCount int64, labelKeys ...string) *models.MetricMetadata {
	m := models.NewMetricMetadata(name, nil)
	m.SampleCount = sampleCount
	for _, k := range labelKeys {
		km := models.NewKeyMetadata()
		for i := 0; i < 10; i++ {
			km.AddValue(k + string(rune('a'+i)))
		}
		m.LabelKeys[k] = km
	}
	return m
}

func newTestSpan(name string, sampleCount int64, attrKeys ...string) *models.SpanMetadata {
	s := models.NewSpanMetadata(name, 2, "SERVER")
	s.SampleCount = sampleCount
	for _, k := range attrKeys {
		km := models.NewKeyMetadata()
		for i := 0; i < 5; i++ {
			km.AddValue(k + string(rune('a'+i)))
		}
		s.AttributeKeys[k] = km
	}
	return s
}

func newTestLog(severity string, sampleCount int64, attrKeys ...string) *models.LogMetadata {
	l := models.NewLogMetadata(severity)
	l.SampleCount = sampleCount
	for _, k := range attrKeys {
		km := models.NewKeyMetadata()
		for i := 0; i < 3; i++ {
			km.AddValue(k + string(rune('a'+i)))
		}
		l.AttributeKeys[k] = km
	}
	return l
}
