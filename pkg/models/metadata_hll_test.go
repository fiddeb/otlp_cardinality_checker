package models

import (
	"fmt"
	"testing"
)

func TestKeyMetadata_HyperLogLog(t *testing.T) {
	tests := []struct {
		name           string
		values         []string
		wantCardinality int64
		wantSamples    int
	}{
		{
			name:           "unique values",
			values:         []string{"a", "b", "c", "d", "e"},
			wantCardinality: 5,
			wantSamples:    5,
		},
		{
			name:           "duplicate values",
			values:         []string{"a", "b", "a", "c", "b", "a"},
			wantCardinality: 3,
			wantSamples:    3,
		},
		{
			name: "many values beyond MaxSamples",
			values: func() []string {
				vals := make([]string, 100)
				for i := 0; i < 100; i++ {
					vals[i] = fmt.Sprintf("value_%d", i)
				}
				return vals
			}(),
			wantCardinality: 100,
			wantSamples:    10, // MaxSamples is 10
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			km := NewKeyMetadata()

			for _, v := range tt.values {
				km.AddValue(v)
			}

			// Check count
			if km.Count != int64(len(tt.values)) {
				t.Errorf("Count = %d, want %d", km.Count, len(tt.values))
			}

			// Check cardinality (allow small HLL error)
			if km.EstimatedCardinality < tt.wantCardinality-5 || km.EstimatedCardinality > tt.wantCardinality+5 {
				t.Errorf("EstimatedCardinality = %d, want %d (±5)", km.EstimatedCardinality, tt.wantCardinality)
			}

			// Check samples
			if len(km.ValueSamples) != tt.wantSamples {
				t.Errorf("len(ValueSamples) = %d, want %d", len(km.ValueSamples), tt.wantSamples)
			}
		})
	}
}

func TestKeyMetadata_HyperLogLog_HighCardinality(t *testing.T) {
	// Test with high cardinality to validate HLL accuracy
	km := NewKeyMetadata()

	const numValues = 10000
	for i := 0; i < numValues; i++ {
		km.AddValue(fmt.Sprintf("value_%d", i))
	}

	// Check count
	if km.Count != numValues {
		t.Errorf("Count = %d, want %d", km.Count, numValues)
	}

	// Check cardinality (HLL standard error is ~0.81% at precision 14, allow 5% margin)
	errorMargin := float64(numValues) * 0.05 // Allow 5% error
	if float64(km.EstimatedCardinality) < float64(numValues)-errorMargin ||
		float64(km.EstimatedCardinality) > float64(numValues)+errorMargin {
		t.Errorf("EstimatedCardinality = %d, want %d (±%.0f)", km.EstimatedCardinality, numValues, errorMargin)
	}

	// Check that samples are capped at MaxSamples
	if len(km.ValueSamples) != km.MaxSamples {
		t.Errorf("len(ValueSamples) = %d, want %d", len(km.ValueSamples), km.MaxSamples)
	}

	t.Logf("High cardinality test: Count=%d, EstimatedCardinality=%d (%.2f%% error), Samples=%d",
		km.Count, km.EstimatedCardinality,
		100*float64(numValues-int(km.EstimatedCardinality))/float64(numValues),
		len(km.ValueSamples))
}

func BenchmarkKeyMetadata_AddValue(b *testing.B) {
	km := NewKeyMetadata()
	values := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		values[i] = fmt.Sprintf("value_%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		km.AddValue(values[i%1000])
	}
}
