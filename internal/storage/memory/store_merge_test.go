package memory

import (
	"context"
	"testing"

	"github.com/fidde/otlp_cardinality_checker/pkg/models"
)

// TestGetLog_AggregationDoesNotMutateStore verifies that GetLog returns cloned
// KeyMetadata objects, so repeated calls do not accumulate counts into the
// stored originals (pointer aliasing bug).
func TestGetLog_AggregationDoesNotMutateStore(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)

	// Store two logs with same severity but different services.
	// Both share the attribute key "http.method".
	log1 := &models.LogMetadata{
		Severity:      "ERROR",
		AttributeKeys: map[string]*models.KeyMetadata{"http.method": {Count: 10}},
		ResourceKeys:  map[string]*models.KeyMetadata{"host.name": {Count: 10}},
		Services:      map[string]int64{"svc-a": 5},
		SampleCount:   5,
	}
	log2 := &models.LogMetadata{
		Severity:      "ERROR",
		AttributeKeys: map[string]*models.KeyMetadata{"http.method": {Count: 20}},
		ResourceKeys:  map[string]*models.KeyMetadata{"host.name": {Count: 20}},
		Services:      map[string]int64{"svc-b": 3},
		SampleCount:   3,
	}

	if err := s.StoreLog(ctx, log1); err != nil {
		t.Fatalf("StoreLog(log1): %v", err)
	}
	if err := s.StoreLog(ctx, log2); err != nil {
		t.Fatalf("StoreLog(log2): %v", err)
	}

	// First call: should aggregate counts from both services.
	agg1, err := s.GetLog(ctx, "ERROR")
	if err != nil {
		t.Fatalf("GetLog(1): %v", err)
	}
	if agg1.AttributeKeys["http.method"].Count != 30 {
		t.Fatalf("first GetLog: expected http.method count 30, got %d", agg1.AttributeKeys["http.method"].Count)
	}
	if agg1.ResourceKeys["host.name"].Count != 30 {
		t.Fatalf("first GetLog: expected host.name count 30, got %d", agg1.ResourceKeys["host.name"].Count)
	}

	// Second call: counts must be identical — no accumulation from aliasing.
	agg2, err := s.GetLog(ctx, "ERROR")
	if err != nil {
		t.Fatalf("GetLog(2): %v", err)
	}
	if agg2.AttributeKeys["http.method"].Count != 30 {
		t.Errorf("second GetLog: expected http.method count 30, got %d (pointer aliasing bug)", agg2.AttributeKeys["http.method"].Count)
	}
	if agg2.ResourceKeys["host.name"].Count != 30 {
		t.Errorf("second GetLog: expected host.name count 30, got %d (pointer aliasing bug)", agg2.ResourceKeys["host.name"].Count)
	}
}

// TestStoreSpan_MergePreservesHLL verifies that merging spans with the same
// name properly merges HLL sketches and value samples, not just counts.
func TestStoreSpan_MergePreservesHLL(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)

	// Create first span with attribute that has value samples.
	km1 := models.NewKeyMetadata()
	km1.AddValue("GET")
	km1.AddValue("POST")

	span1 := &models.SpanMetadata{
		Name:          "HTTP request",
		Kind:          2,
		AttributeKeys: map[string]*models.KeyMetadata{"http.method": km1},
		ResourceKeys:  map[string]*models.KeyMetadata{},
		Services:      map[string]int64{"svc-a": 5},
		SampleCount:   5,
	}

	// Create second span with overlapping + new values.
	km2 := models.NewKeyMetadata()
	km2.AddValue("GET")
	km2.AddValue("DELETE")

	span2 := &models.SpanMetadata{
		Name:          "HTTP request",
		Kind:          2,
		AttributeKeys: map[string]*models.KeyMetadata{"http.method": km2},
		ResourceKeys:  map[string]*models.KeyMetadata{},
		Services:      map[string]int64{"svc-b": 3},
		SampleCount:   3,
	}

	if err := s.StoreSpan(ctx, span1); err != nil {
		t.Fatalf("StoreSpan(1): %v", err)
	}
	if err := s.StoreSpan(ctx, span2); err != nil {
		t.Fatalf("StoreSpan(2): %v", err)
	}

	got, err := s.GetSpan(ctx, "HTTP request")
	if err != nil {
		t.Fatalf("GetSpan: %v", err)
	}

	km := got.AttributeKeys["http.method"]
	if km == nil {
		t.Fatal("expected http.method key to exist")
	}

	// Count should be merged (2+2 = 4 AddValue calls).
	if km.Count != 4 {
		t.Errorf("expected count 4, got %d", km.Count)
	}

	// Cardinality should reflect 3 unique values (GET, POST, DELETE).
	card := km.Cardinality()
	if card != 3 {
		t.Errorf("expected cardinality 3, got %d", card)
	}

	// Value samples should contain all three.
	samples := km.GetSortedSamples()
	if len(samples) < 3 {
		t.Errorf("expected at least 3 value samples, got %v", samples)
	}
}

// TestStoreLog_MergePreservesHLL verifies that merging logs with the same
// key properly merges HLL sketches and value samples, not just counts.
func TestStoreLog_MergePreservesHLL(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(5)

	km1 := models.NewKeyMetadata()
	km1.AddValue("200")
	km1.AddValue("404")

	log1 := &models.LogMetadata{
		Severity:      "INFO",
		AttributeKeys: map[string]*models.KeyMetadata{"http.status": km1},
		ResourceKeys:  map[string]*models.KeyMetadata{},
		Services:      map[string]int64{"svc-a": 5},
		SampleCount:   5,
	}

	km2 := models.NewKeyMetadata()
	km2.AddValue("200")
	km2.AddValue("500")

	log2 := &models.LogMetadata{
		Severity:      "INFO",
		AttributeKeys: map[string]*models.KeyMetadata{"http.status": km2},
		ResourceKeys:  map[string]*models.KeyMetadata{},
		Services:      map[string]int64{"svc-a": 3},
		SampleCount:   3,
	}

	if err := s.StoreLog(ctx, log1); err != nil {
		t.Fatalf("StoreLog(1): %v", err)
	}
	if err := s.StoreLog(ctx, log2); err != nil {
		t.Fatalf("StoreLog(2): %v", err)
	}

	// Retrieve the stored log directly via the internal key.
	got, err := s.GetLogByServiceAndSeverity(ctx, "svc-a", "INFO")
	if err != nil {
		t.Fatalf("GetLogByServiceAndSeverity: %v", err)
	}

	km := got.AttributeKeys["http.status"]
	if km == nil {
		t.Fatal("expected http.status key to exist")
	}

	// Count should be merged (2+2 = 4 AddValue calls).
	if km.Count != 4 {
		t.Errorf("expected count 4, got %d", km.Count)
	}

	// Cardinality should reflect 3 unique values (200, 404, 500).
	card := km.Cardinality()
	if card != 3 {
		t.Errorf("expected cardinality 3, got %d", card)
	}
}
