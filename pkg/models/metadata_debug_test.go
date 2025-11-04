package models

import (
	"testing"
)

func TestKeyMetadata_AddValue_Integration(t *testing.T) {
	km := NewKeyMetadata()
	
	// Add same value multiple times
	for i := 0; i < 5; i++ {
		km.AddValue("value_A")
	}
	
	if km.Count != 5 {
		t.Errorf("Count = %d, want 5", km.Count)
	}
	if km.EstimatedCardinality != 1 {
		t.Errorf("EstimatedCardinality = %d, want 1 (same value)", km.EstimatedCardinality)
	}
	
	// Add different values
	km.AddValue("value_B")
	km.AddValue("value_C")
	km.AddValue("value_D")
	
	if km.Count != 8 {
		t.Errorf("Count = %d, want 8", km.Count)
	}
	if km.EstimatedCardinality < 3 || km.EstimatedCardinality > 5 {
		t.Errorf("EstimatedCardinality = %d, want 4 (±1)", km.EstimatedCardinality)
	}
	
	t.Logf("After 8 AddValue calls (4 unique): Count=%d, Card=%d, Samples=%v",
		km.Count, km.EstimatedCardinality, km.ValueSamples)
}

func TestKeyMetadata_Merge_HLL(t *testing.T) {
	km1 := NewKeyMetadata()
	km2 := NewKeyMetadata()
	
	// Add different values to each
	for i := 0; i < 100; i++ {
		km1.AddValue("A")
		km1.AddValue("B")
	}
	
	for i := 0; i < 100; i++ {
		km2.AddValue("C")
		km2.AddValue("D")
	}
	
	t.Logf("Before merge: km1.Card=%d, km2.Card=%d", km1.EstimatedCardinality, km2.EstimatedCardinality)
	
	// Simulate merge logic from MergeMetricMetadata
	km1.mu.Lock()
	km1.Count += km2.Count
	km1.hll.Merge(km2.hll)
	km1.EstimatedCardinality = int64(km1.hll.Count())
	km1.mu.Unlock()
	
	t.Logf("After merge: km1.Card=%d (should be ~4)", km1.EstimatedCardinality)
	
	if km1.EstimatedCardinality < 3 || km1.EstimatedCardinality > 5 {
		t.Errorf("After merge, EstimatedCardinality = %d, want 4 (±1)", km1.EstimatedCardinality)
	}
}
