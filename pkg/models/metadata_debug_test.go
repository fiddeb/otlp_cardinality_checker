package models

import (
	"fmt"
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
	if km.Cardinality() != 1 {
		t.Errorf("Cardinality() = %d, want 1 (same value)", km.Cardinality())
	}
	
	// Add different values
	km.AddValue("value_B")
	km.AddValue("value_C")
	km.AddValue("value_D")
	
	if km.Count != 8 {
		t.Errorf("Count = %d, want 8", km.Count)
	}
	if got := km.Cardinality(); got < 3 || got > 5 {
		t.Errorf("Cardinality() = %d, want 4 (±1)", got)
	}
	
	t.Logf("After 8 AddValue calls (4 unique): Count=%d, Card=%d, Samples=%v",
		km.Count, km.Cardinality(), km.ValueSamples)
}

func TestKeyMetadata_Merge_HLL(t *testing.T) {
	km1 := NewKeyMetadata()
	km2 := NewKeyMetadata()

	// Add >MaxSamples unique values to each so HLL is initialized lazily.
	for i := 0; i < 15; i++ {
		km1.AddValue(fmt.Sprintf("km1-val-%d", i))
	}

	for i := 0; i < 15; i++ {
		km2.AddValue(fmt.Sprintf("km2-val-%d", i))
	}

	t.Logf("Before merge: km1.Card=%d, km2.Card=%d", km1.Cardinality(), km2.Cardinality())

	// Use MergeMetricMetadata via a MetricMetadata wrapper to test merge logic
	// without accessing internal hll fields directly.
	m1 := &MetricMetadata{LabelKeys: map[string]*KeyMetadata{"key": km1}}
	m2 := &MetricMetadata{LabelKeys: map[string]*KeyMetadata{"key": km2}}
	m1.MergeMetricMetadata(m2)

	merged := m1.LabelKeys["key"]
	got := merged.Cardinality()
	t.Logf("After merge: Card=%d (should be ~30)", got)

	if got < 25 || got > 35 {
		t.Errorf("After merge, Cardinality() = %d, want ~30 (±5)", got)
	}
}
