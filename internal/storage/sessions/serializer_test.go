package sessions

import (
	"context"
	"testing"
	"time"

	"github.com/fidde/otlp_cardinality_checker/pkg/hyperloglog"
	"github.com/fidde/otlp_cardinality_checker/pkg/models"
)

func TestSerializer_MarshalUnmarshalMetrics_RoundTrip(t *testing.T) {
	serializer := NewSerializer()

	// Create a metric with HLL data
	metric := models.NewMetricMetadata("http_requests_total", nil)
	metric.Description = "Total HTTP requests"
	metric.Unit = "1"
	metric.SampleCount = 1000
	metric.ActiveSeries = 500
	metric.Services = map[string]int64{"api-server": 500, "web-server": 500}

	// Add label keys with HLL
	labelKey := models.NewKeyMetadata()
	labelKey.Count = 100
	for i := 0; i < 50; i++ {
		labelKey.AddValue(string(rune('0' + i%10)))
	}
	metric.LabelKeys["status_code"] = labelKey

	// Add resource keys
	resourceKey := models.NewKeyMetadata()
	resourceKey.Count = 50
	metric.ResourceKeys["host"] = resourceKey

	// Add series HLL
	hll := hyperloglog.New(14)
	for i := 0; i < 500; i++ {
		hll.Add(string([]byte{byte(i), byte(i >> 8)}))
	}
	metric.SetSeriesHLL(hll)

	// Marshal
	serialized, err := serializer.MarshalMetrics([]*models.MetricMetadata{metric})
	if err != nil {
		t.Fatalf("MarshalMetrics failed: %v", err)
	}
	if len(serialized) != 1 {
		t.Fatalf("Expected 1 serialized metric, got %d", len(serialized))
	}

	// Verify serialized data
	sm := serialized[0]
	if sm.Name != "http_requests_total" {
		t.Errorf("Expected name http_requests_total, got %s", sm.Name)
	}
	if sm.SeriesHLL == nil {
		t.Error("Expected SeriesHLL to be set")
	}
	if sm.LabelKeys["status_code"] == nil {
		t.Error("Expected status_code label key")
	}

	// Unmarshal
	restored, err := serializer.UnmarshalMetrics(serialized)
	if err != nil {
		t.Fatalf("UnmarshalMetrics failed: %v", err)
	}
	if len(restored) != 1 {
		t.Fatalf("Expected 1 restored metric, got %d", len(restored))
	}

	// Verify restored data
	rm := restored[0]
	if rm.Name != metric.Name {
		t.Errorf("Name mismatch: %s vs %s", rm.Name, metric.Name)
	}
	if rm.SampleCount != metric.SampleCount {
		t.Errorf("SampleCount mismatch: %d vs %d", rm.SampleCount, metric.SampleCount)
	}
	if rm.ActiveSeries != metric.ActiveSeries {
		t.Errorf("ActiveSeries mismatch: %d vs %d", rm.ActiveSeries, metric.ActiveSeries)
	}

	// Verify HLL cardinality is preserved (approximately)
	restoredHLL := rm.GetSeriesHLL()
	if restoredHLL == nil {
		t.Fatal("Expected restored SeriesHLL")
	}
	originalCount := hll.Count()
	restoredCount := restoredHLL.Count()
	if restoredCount != originalCount {
		t.Errorf("HLL count mismatch: %d vs %d", restoredCount, originalCount)
	}
}

func TestSerializer_MarshalUnmarshalSpans_RoundTrip(t *testing.T) {
	serializer := NewSerializer()

	span := &models.SpanMetadata{
		Name:          "GET /api/users/{id}",
		Kind:          1, // Server
		KindName:      "Server",
		AttributeKeys: map[string]*models.KeyMetadata{},
		EventNames:    []string{"exception"},
		ResourceKeys:  map[string]*models.KeyMetadata{},
		StatusCodes:   []string{"OK", "ERROR"},
		SampleCount:   1000,
		Services:      map[string]int64{"user-service": 1000},
	}

	attrKey := models.NewKeyMetadata()
	attrKey.Count = 1000
	span.AttributeKeys["http.method"] = attrKey

	// Marshal and unmarshal
	serialized, err := serializer.MarshalSpans([]*models.SpanMetadata{span})
	if err != nil {
		t.Fatalf("MarshalSpans failed: %v", err)
	}

	restored, err := serializer.UnmarshalSpans(serialized)
	if err != nil {
		t.Fatalf("UnmarshalSpans failed: %v", err)
	}

	if len(restored) != 1 {
		t.Fatalf("Expected 1 restored span, got %d", len(restored))
	}

	rs := restored[0]
	if rs.Name != span.Name {
		t.Errorf("Name mismatch: %s vs %s", rs.Name, span.Name)
	}
	if rs.Kind != span.Kind {
		t.Errorf("Kind mismatch: %d vs %d", rs.Kind, span.Kind)
	}
	if len(rs.StatusCodes) != 2 {
		t.Errorf("StatusCodes length mismatch: %d vs 2", len(rs.StatusCodes))
	}
}

func TestSerializer_MarshalUnmarshalLogs_RoundTrip(t *testing.T) {
	serializer := NewSerializer()

	log := &models.LogMetadata{
		Severity:       "ERROR",
		SeverityNumber: 17,
		AttributeKeys:  map[string]*models.KeyMetadata{},
		ResourceKeys:   map[string]*models.KeyMetadata{},
		BodyTemplates:  []*models.BodyTemplate{{Template: "Connection failed to {host}", Count: 500}},
		EventNames:     []string{},
		SampleCount:    500,
		Services:       map[string]int64{"db-service": 500},
	}

	attrKey := models.NewKeyMetadata()
	attrKey.Count = 500
	log.AttributeKeys["error.type"] = attrKey

	// Marshal and unmarshal
	serialized, err := serializer.MarshalLogs([]*models.LogMetadata{log})
	if err != nil {
		t.Fatalf("MarshalLogs failed: %v", err)
	}

	restored, err := serializer.UnmarshalLogs(serialized)
	if err != nil {
		t.Fatalf("UnmarshalLogs failed: %v", err)
	}

	if len(restored) != 1 {
		t.Fatalf("Expected 1 restored log, got %d", len(restored))
	}

	rl := restored[0]
	if rl.Severity != log.Severity {
		t.Errorf("Severity mismatch: %s vs %s", rl.Severity, log.Severity)
	}
	if rl.SeverityNumber != log.SeverityNumber {
		t.Errorf("SeverityNumber mismatch: %d vs %d", rl.SeverityNumber, log.SeverityNumber)
	}
}

func TestSerializer_MarshalUnmarshalAttributes_RoundTrip(t *testing.T) {
	serializer := NewSerializer()

	attr := models.NewAttributeMetadata("user.id")
	attr.Count = 10000
	attr.Scope = "span"
	attr.SignalTypes = []string{"span", "log"}
	attr.FirstSeen = time.Now().Add(-1 * time.Hour)
	attr.LastSeen = time.Now()
	
	// Add some values to build cardinality
	for i := 0; i < 1000; i++ {
		attr.AddValue(string(rune(i)), "span", "span")
	}

	// Marshal and unmarshal
	serialized, err := serializer.MarshalAttributes([]*models.AttributeMetadata{attr})
	if err != nil {
		t.Fatalf("MarshalAttributes failed: %v", err)
	}

	restored, err := serializer.UnmarshalAttributes(serialized)
	if err != nil {
		t.Fatalf("UnmarshalAttributes failed: %v", err)
	}

	if len(restored) != 1 {
		t.Fatalf("Expected 1 restored attribute, got %d", len(restored))
	}

	ra := restored[0]
	if ra.Key != attr.Key {
		t.Errorf("Key mismatch: %s vs %s", ra.Key, attr.Key)
	}
	if ra.Count != attr.Count {
		t.Errorf("Count mismatch: %d vs %d", ra.Count, attr.Count)
	}

	// Cardinality should be approximately preserved
	if ra.EstimatedCardinality < 900 || ra.EstimatedCardinality > 1100 {
		t.Errorf("Cardinality not in expected range: %d (expected ~1000)", ra.EstimatedCardinality)
	}
}

func TestSerializer_CreateSession(t *testing.T) {
	serializer := NewSerializer()
	ctx := context.Background()

	// Create test data
	metrics := []*models.MetricMetadata{
		{
			Name:        "test_metric",
			SampleCount: 100,
			Services:    map[string]int64{"svc-a": 100},
			LabelKeys:   map[string]*models.KeyMetadata{},
			ResourceKeys: map[string]*models.KeyMetadata{},
		},
	}

	spans := []*models.SpanMetadata{
		{
			Name:          "test_span",
			SampleCount:   50,
			Services:      map[string]int64{"svc-b": 50},
			AttributeKeys: map[string]*models.KeyMetadata{},
			ResourceKeys:  map[string]*models.KeyMetadata{},
		},
	}

	logs := []*models.LogMetadata{
		{
			Severity:      "INFO",
			SampleCount:   25,
			Services:      map[string]int64{"svc-a": 25},
			AttributeKeys: map[string]*models.KeyMetadata{},
			ResourceKeys:  map[string]*models.KeyMetadata{},
		},
	}

	attributes := []*models.AttributeMetadata{
		{Key: "test.attr", Count: 100},
	}

	services := []string{"svc-a", "svc-b"}

	// Create session with all signals
	session, err := serializer.CreateSession(ctx, CreateSessionOptions{
		Name:        "test-session",
		Description: "Test session",
	}, metrics, spans, logs, attributes, services)

	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if session.ID != "test-session" {
		t.Errorf("ID mismatch: %s", session.ID)
	}
	// Stats should reflect total sample counts, not number of unique items
	if session.Stats.MetricsCount != 100 {
		t.Errorf("MetricsCount mismatch: expected 100, got %d", session.Stats.MetricsCount)
	}
	if session.Stats.SpansCount != 50 {
		t.Errorf("SpansCount mismatch: expected 50, got %d", session.Stats.SpansCount)
	}
	if session.Stats.LogsCount != 25 {
		t.Errorf("LogsCount mismatch: expected 25, got %d", session.Stats.LogsCount)
	}
}

func TestSerializer_CreateSession_FilterBySignal(t *testing.T) {
	serializer := NewSerializer()
	ctx := context.Background()

	metrics := []*models.MetricMetadata{
		{Name: "m1", SampleCount: 10, LabelKeys: map[string]*models.KeyMetadata{}, ResourceKeys: map[string]*models.KeyMetadata{}},
	}
	spans := []*models.SpanMetadata{
		{Name: "s1", SampleCount: 5, AttributeKeys: map[string]*models.KeyMetadata{}, ResourceKeys: map[string]*models.KeyMetadata{}},
	}

	// Create session with only metrics
	session, err := serializer.CreateSession(ctx, CreateSessionOptions{
		Name:    "metrics-only",
		Signals: []string{"metrics"},
	}, metrics, spans, nil, nil, nil)

	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// MetricsCount should be sum of SampleCount (10), not number of metrics (1)
	if session.Stats.MetricsCount != 10 {
		t.Errorf("Expected 10 metric samples, got %d", session.Stats.MetricsCount)
	}
	if session.Stats.SpansCount != 0 {
		t.Errorf("Expected 0 spans (filtered out), got %d", session.Stats.SpansCount)
	}
}

func TestSerializer_CreateSession_FilterByService(t *testing.T) {
	serializer := NewSerializer()
	ctx := context.Background()

	metrics := []*models.MetricMetadata{
		{Name: "m1", SampleCount: 100, Services: map[string]int64{"svc-a": 100}, LabelKeys: map[string]*models.KeyMetadata{}, ResourceKeys: map[string]*models.KeyMetadata{}},
		{Name: "m2", SampleCount: 100, Services: map[string]int64{"svc-b": 100}, LabelKeys: map[string]*models.KeyMetadata{}, ResourceKeys: map[string]*models.KeyMetadata{}},
		{Name: "m3", SampleCount: 50, Services: map[string]int64{"svc-a": 50, "svc-c": 50}, LabelKeys: map[string]*models.KeyMetadata{}, ResourceKeys: map[string]*models.KeyMetadata{}},
	}

	// Create session filtered by service
	session, err := serializer.CreateSession(ctx, CreateSessionOptions{
		Name:     "svc-a-only",
		Services: []string{"svc-a"},
	}, metrics, nil, nil, nil, nil)

	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Should include m1 (100 samples) and m3 (50 samples) - both have svc-a
	// Total = 100 + 50 = 150
	if session.Stats.MetricsCount != 150 {
		t.Errorf("Expected 150 metric samples (m1=100 + m3=50), got %d", session.Stats.MetricsCount)
	}
}

func TestSerializer_LargeSession(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large session test in short mode")
	}

	serializer := NewSerializer()
	ctx := context.Background()

	// Create 10k metrics with HLL data (simulating high cardinality scenario)
	const numMetrics = 10000
	metrics := make([]*models.MetricMetadata, numMetrics)

	for i := 0; i < numMetrics; i++ {
		m := models.NewMetricMetadata("metric_"+string(rune('a'+i%26))+string(rune('0'+i%10)), nil)
		m.SampleCount = int64(i * 10)
		m.Services = map[string]int64{"service-" + string(rune('a'+i%5)): int64(i)}

		// Add label keys with HLL
		for j := 0; j < 5; j++ {
			key := models.NewKeyMetadata()
			key.Count = int64(i + j)
			// Add some values to build cardinality
			for k := 0; k < 10; k++ {
				key.AddValue(string(rune(i + j + k)))
			}
			m.LabelKeys["label_"+string(rune('a'+j))] = key
		}

		// Add series HLL
		hll := hyperloglog.New(14)
		for j := 0; j < 100; j++ {
			hll.Add(string([]byte{byte(i), byte(j)}))
		}
		m.SetSeriesHLL(hll)

		metrics[i] = m
	}

	// Create session
	start := time.Now()
	session, err := serializer.CreateSession(ctx, CreateSessionOptions{
		Name:        "large-session",
		Description: "Performance test session",
	}, metrics, nil, nil, nil, nil)
	createDuration := time.Since(start)

	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// MetricsCount is sum of all SampleCount values
	// SampleCount = i * 10 for i=0..9999, so sum = 10 * (0+1+2+...+9999) = 10 * 9999*10000/2 = 499950000
	expectedSampleCount := 0
	for i := 0; i < numMetrics; i++ {
		expectedSampleCount += i * 10
	}
	if session.Stats.MetricsCount != expectedSampleCount {
		t.Errorf("Expected %d metric samples, got %d", expectedSampleCount, session.Stats.MetricsCount)
	}

	t.Logf("Created session with %d metrics (%d samples) in %v", numMetrics, session.Stats.MetricsCount, createDuration)

	// Verify round-trip
	restored, err := serializer.UnmarshalMetrics(session.Data.Metrics)
	if err != nil {
		t.Fatalf("UnmarshalMetrics failed: %v", err)
	}

	if len(restored) != numMetrics {
		t.Errorf("Expected %d restored metrics, got %d", numMetrics, len(restored))
	}

	// Spot check HLL preservation
	if restored[0].GetSeriesHLL() == nil {
		t.Error("Expected SeriesHLL on first restored metric")
	}
}
