package models

import (
	"fmt"
	"testing"
)

func TestNewCardinalityInfoHLL(t *testing.T) {
	c := NewCardinalityInfoHLL(14, 5)

	if c.HLL == nil {
		t.Error("HLL should be initialized")
	}
	if c.MaxSamples != 5 {
		t.Errorf("MaxSamples = %d, want 5", c.MaxSamples)
	}
	if len(c.SampleValues) != 0 {
		t.Errorf("SampleValues length = %d, want 0", len(c.SampleValues))
	}
}

func TestCardinalityInfoHLL_Add(t *testing.T) {
	c := NewCardinalityInfoHLL(14, 3)

	// Add unique values
	c.Add("value1")
	c.Add("value2")
	c.Add("value3")
	c.Add("value1") // Duplicate

	count := c.Count()
	if count != 3 {
		t.Errorf("Count() = %d, want 3", count)
	}

	// Check samples (should have max 3)
	if len(c.SampleValues) > 3 {
		t.Errorf("SampleValues length = %d, want <= 3", len(c.SampleValues))
	}
}

func TestCardinalityInfoHLL_LargeCardinality(t *testing.T) {
	c := NewCardinalityInfoHLL(14, 5)

	// Add many unique values
	const numValues = 10000
	for i := 0; i < numValues; i++ {
		c.Add(fmt.Sprintf("value_%d", i))
	}

	count := c.Count()
	errorPct := float64(int(count)-numValues) / float64(numValues) * 100
	if errorPct < 0 {
		errorPct = -errorPct
	}

	t.Logf("Actual: %d, Estimated: %d, Error: %.2f%%", numValues, count, errorPct)

	if errorPct > 10.0 {
		t.Errorf("Error %.2f%% exceeds 10%%", errorPct)
	}

	// Samples should be capped
	if len(c.SampleValues) > 5 {
		t.Errorf("SampleValues length = %d, want <= 5", len(c.SampleValues))
	}
}

func TestCardinalityInfoHLL_Merge(t *testing.T) {
	c1 := NewCardinalityInfoHLL(14, 5)
	c2 := NewCardinalityInfoHLL(14, 5)

	// Add different values to each
	for i := 0; i < 5000; i++ {
		c1.Add(fmt.Sprintf("set1_%d", i))
	}
	for i := 0; i < 5000; i++ {
		c2.Add(fmt.Sprintf("set2_%d", i))
	}

	// Merge
	if err := c1.Merge(c2); err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	count := c1.Count()
	expected := 10000
	errorPct := float64(int(count)-expected) / float64(expected) * 100
	if errorPct < 0 {
		errorPct = -errorPct
	}

	t.Logf("Expected: %d, Estimated: %d, Error: %.2f%%", expected, count, errorPct)

	if errorPct > 10.0 {
		t.Errorf("Merge error %.2f%% exceeds 10%%", errorPct)
	}
}

func TestCardinalityInfoHLL_MemorySize(t *testing.T) {
	c := NewCardinalityInfoHLL(14, 5)

	// Add some values
	for i := 0; i < 1000; i++ {
		c.Add(fmt.Sprintf("value_%d", i))
	}

	size := c.MemorySize()

	// Should be ~16KB + samples (small)
	if size < 15000 || size > 20000 {
		t.Errorf("MemorySize() = %d, want ~16KB", size)
	}

	t.Logf("Memory usage: %d bytes (~%.1f KB)", size, float64(size)/1024)
}

func TestCardinalityInfoHLL_Clear(t *testing.T) {
	c := NewCardinalityInfoHLL(14, 5)

	// Add values
	for i := 0; i < 100; i++ {
		c.Add(fmt.Sprintf("value_%d", i))
	}

	if c.Count() < 90 {
		t.Errorf("Count before clear too low: %d", c.Count())
	}

	// Clear
	c.Clear()

	if c.Count() != 0 {
		t.Errorf("Count after clear = %d, want 0", c.Count())
	}
	if len(c.SampleValues) != 0 {
		t.Errorf("SampleValues after clear = %d, want 0", len(c.SampleValues))
	}
}

// Benchmark comparison: naive map vs HLL
func BenchmarkNaiveMap(b *testing.B) {
	m := make(map[string]struct{})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m[fmt.Sprintf("value_%d", i)] = struct{}{}
	}

	b.Logf("Map size: %d, memory: ~%d KB", len(m), len(m)*16/1024)
}

func BenchmarkHLL(b *testing.B) {
	c := NewCardinalityInfoHLL(14, 5)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Add(fmt.Sprintf("value_%d", i))
	}

	b.Logf("HLL count: %d, memory: ~%d KB", c.Count(), c.MemorySize()/1024)
}

// Test memory comparison
func TestMemoryComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory comparison in short mode")
	}

	const numValues = 100000

	// Naive approach
	naiveMap := make(map[string]struct{})
	for i := 0; i < numValues; i++ {
		naiveMap[fmt.Sprintf("value_%d", i)] = struct{}{}
	}
	naiveMemory := len(naiveMap) * 16 // Approximate

	// HLL approach
	hll := NewCardinalityInfoHLL(14, 5)
	for i := 0; i < numValues; i++ {
		hll.Add(fmt.Sprintf("value_%d", i))
	}
	hllMemory := hll.MemorySize()

	savings := (1 - float64(hllMemory)/float64(naiveMemory)) * 100

	t.Logf("Comparison for %d unique values:", numValues)
	t.Logf("  Naive map: ~%d KB", naiveMemory/1024)
	t.Logf("  HLL:       ~%d KB", hllMemory/1024)
	t.Logf("  Savings:   %.1f%%", savings)
	t.Logf("  Actual count: %d", numValues)
	t.Logf("  HLL estimate: %d", hll.Count())

	if savings < 95.0 {
		t.Errorf("Expected >95%% memory savings, got %.1f%%", savings)
	}
}
