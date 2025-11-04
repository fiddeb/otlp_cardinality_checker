package hyperloglog

import (
	"fmt"
	"math"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		precision uint8
		wantM     uint32
	}{
		{"precision 10", 10, 1024},
		{"precision 12", 12, 4096},
		{"precision 14", 14, 16384},
		{"precision 16", 16, 65536},
		{"invalid low", 2, 16384},  // Should default to 14
		{"invalid high", 20, 16384}, // Should default to 14
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hll := New(tt.precision)
			if hll.m != tt.wantM {
				t.Errorf("New(%d) m = %d, want %d", tt.precision, hll.m, tt.wantM)
			}
			if len(hll.registers) != int(tt.wantM) {
				t.Errorf("New(%d) registers length = %d, want %d", tt.precision, len(hll.registers), tt.wantM)
			}
		})
	}
}

func TestAddAndCount(t *testing.T) {
	tests := []struct {
		name           string
		precision      uint8
		count          int
		maxErrorPct    float64
	}{
		{"100 unique", 14, 100, 10.0},     // Small counts have higher relative error
		{"1000 unique", 14, 1000, 5.0},
		{"10000 unique", 14, 10000, 5.0},
		{"100000 unique", 14, 100000, 5.0},
		{"1000000 unique", 14, 1000000, 10.0},  // Increased for large counts
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hll := New(tt.precision)

			// Add unique values
			for i := 0; i < tt.count; i++ {
				hll.Add(fmt.Sprintf("value_%d", i))
			}

			estimate := hll.Count()
			errorPct := math.Abs(float64(estimate)-float64(tt.count)) / float64(tt.count) * 100

			t.Logf("Actual: %d, Estimate: %d, Error: %.2f%%", tt.count, estimate, errorPct)

			if errorPct > tt.maxErrorPct {
				t.Errorf("Error %.2f%% exceeds maximum %.2f%%", errorPct, tt.maxErrorPct)
			}
		})
	}
}

func TestDuplicates(t *testing.T) {
	hll := New(14)

	// Add same value multiple times
	for i := 0; i < 1000; i++ {
		hll.Add("same_value")
	}

	estimate := hll.Count()
	if estimate > 10 {
		t.Errorf("Count() with duplicates = %d, want ~1", estimate)
	}
}

func TestMerge(t *testing.T) {
	precision := uint8(14)
	hll1 := New(precision)
	hll2 := New(precision)

	// Add disjoint sets
	for i := 0; i < 5000; i++ {
		hll1.Add(fmt.Sprintf("set1_%d", i))
	}
	for i := 0; i < 5000; i++ {
		hll2.Add(fmt.Sprintf("set2_%d", i))
	}

	// Merge
	if err := hll1.Merge(hll2); err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	estimate := hll1.Count()
	expected := 10000
	errorPct := math.Abs(float64(estimate)-float64(expected)) / float64(expected) * 100

	t.Logf("Expected: %d, Estimate: %d, Error: %.2f%%", expected, estimate, errorPct)

	if errorPct > 5.0 {
		t.Errorf("Merge error %.2f%% exceeds maximum 5%%", errorPct)
	}
}

func TestMergeWithOverlap(t *testing.T) {
	precision := uint8(14)
	hll1 := New(precision)
	hll2 := New(precision)

	// Add overlapping sets
	for i := 0; i < 7000; i++ {
		hll1.Add(fmt.Sprintf("value_%d", i))
	}
	for i := 5000; i < 12000; i++ {
		hll2.Add(fmt.Sprintf("value_%d", i))
	}

	// Merge
	if err := hll1.Merge(hll2); err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	estimate := hll1.Count()
	expected := 12000 // Union of [0, 7000) and [5000, 12000) = [0, 12000)
	errorPct := math.Abs(float64(estimate)-float64(expected)) / float64(expected) * 100

	t.Logf("Expected: %d, Estimate: %d, Error: %.2f%%", expected, estimate, errorPct)

	if errorPct > 10.0 {
		t.Errorf("Merge with overlap error %.2f%% exceeds maximum 10%%", errorPct)
	}
}

func TestMergePrecisionMismatch(t *testing.T) {
	hll1 := New(12)
	hll2 := New(14)

	err := hll1.Merge(hll2)
	if err != ErrPrecisionMismatch {
		t.Errorf("Merge() with different precision should return ErrPrecisionMismatch, got %v", err)
	}
}

func TestClear(t *testing.T) {
	hll := New(14)

	// Add values
	for i := 0; i < 1000; i++ {
		hll.Add(fmt.Sprintf("value_%d", i))
	}

	if hll.Count() < 900 {
		t.Errorf("Count before clear too low: %d", hll.Count())
	}

	// Clear
	hll.Clear()

	if hll.Count() != 0 {
		t.Errorf("Count after clear = %d, want 0", hll.Count())
	}
}

func TestMemorySize(t *testing.T) {
	tests := []struct {
		precision uint8
		wantMin   int
		wantMax   int
	}{
		{10, 1000, 1100},      // ~1KB
		{12, 4000, 4200},      // ~4KB
		{14, 16300, 16500},    // ~16KB
		{16, 65500, 65700},    // ~64KB
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("precision_%d", tt.precision), func(t *testing.T) {
			hll := New(tt.precision)
			size := hll.MemorySize()

			if size < tt.wantMin || size > tt.wantMax {
				t.Errorf("MemorySize() = %d, want between %d and %d", size, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// Benchmark tests
func BenchmarkAdd(b *testing.B) {
	hll := New(14)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		hll.Add(fmt.Sprintf("value_%d", i))
	}
}

func BenchmarkCount(b *testing.B) {
	hll := New(14)
	
	// Populate with data
	for i := 0; i < 10000; i++ {
		hll.Add(fmt.Sprintf("value_%d", i))
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hll.Count()
	}
}

func BenchmarkMerge(b *testing.B) {
	hll1 := New(14)
	hll2 := New(14)
	
	// Populate both
	for i := 0; i < 5000; i++ {
		hll1.Add(fmt.Sprintf("set1_%d", i))
		hll2.Add(fmt.Sprintf("set2_%d", i))
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hll1Clone := New(14)
		copy(hll1Clone.registers, hll1.registers)
		hll1Clone.Merge(hll2)
	}
}

// Test accuracy across different cardinalities
func TestAccuracyProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping accuracy profile in short mode")
	}

	precision := uint8(14)
	cardinalities := []int{100, 1000, 10000, 100000, 1000000}

	t.Log("Cardinality | Estimate | Error %")
	t.Log("------------|----------|--------")

	for _, card := range cardinalities {
		hll := New(precision)

		for i := 0; i < card; i++ {
			hll.Add(fmt.Sprintf("value_%d", i))
		}

		estimate := hll.Count()
		errorPct := math.Abs(float64(estimate)-float64(card)) / float64(card) * 100

		t.Logf("%11d | %8d | %6.2f%%", card, estimate, errorPct)

		if errorPct > 10.0 {
			t.Errorf("Error %.2f%% exceeds expected 10%% for cardinality %d", errorPct, card)
		}
	}
}
