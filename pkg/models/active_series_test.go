package models

import (
	"testing"
)

func TestCreateSeriesFingerprint(t *testing.T) {
	tests := []struct {
		name        string
		labels      map[string]string
		expectConst bool
	}{
		{
			name:        "empty labels",
			labels:      map[string]string{},
			expectConst: true,
		},
		{
			name: "single label",
			labels: map[string]string{
				"method": "GET",
			},
			expectConst: false,
		},
		{
			name: "multiple labels",
			labels: map[string]string{
				"method": "GET",
				"status": "200",
				"path":   "/api",
			},
			expectConst: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := CreateSeriesFingerprintFast(tt.labels)
			
			if tt.expectConst && fp != "constant" {
				t.Errorf("Expected 'constant' for empty labels, got %s", fp)
			}
			
			if !tt.expectConst && fp == "constant" {
				t.Errorf("Expected non-constant fingerprint, got 'constant'")
			}
			
			// Test consistency: same labels should produce same fingerprint
			fp2 := CreateSeriesFingerprintFast(tt.labels)
			if fp != fp2 {
				t.Errorf("Fingerprint not consistent: %s != %s", fp, fp2)
			}
		})
	}
}

func TestCreateSeriesFingerprintOrdering(t *testing.T) {
	// Test that order of insertion doesn't matter
	labels1 := map[string]string{
		"a": "1",
		"b": "2",
		"c": "3",
	}
	
	labels2 := map[string]string{
		"c": "3",
		"a": "1",
		"b": "2",
	}
	
	fp1 := CreateSeriesFingerprintFast(labels1)
	fp2 := CreateSeriesFingerprintFast(labels2)
	
	if fp1 != fp2 {
		t.Errorf("Fingerprints should be equal regardless of insertion order:\n%s\n%s", fp1, fp2)
	}
}

func TestGetActiveSeries(t *testing.T) {
	tests := []struct {
		name              string
		seriesFingerprints []string
		expectedCount     int64
	}{
		{
			name:              "no series tracked",
			seriesFingerprints: []string{},
			expectedCount:     1, // constant metric
		},
		{
			name: "single series",
			seriesFingerprints: []string{
				"method=GET,status=200",
			},
			expectedCount: 1,
		},
		{
			name: "multiple unique series",
			seriesFingerprints: []string{
				"method=GET,status=200",
				"method=GET,status=404",
				"method=POST,status=200",
			},
			expectedCount: 3,
		},
		{
			name: "duplicate series",
			seriesFingerprints: []string{
				"method=GET,status=200",
				"method=GET,status=200",
				"method=GET,status=200",
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric := NewMetricMetadata("test_metric", &GaugeMetric{})
			
			for _, fp := range tt.seriesFingerprints {
				metric.AddSeriesFingerprint(fp)
			}
			
			count := metric.GetActiveSeries()
			
			if count != tt.expectedCount {
				t.Errorf("Expected %d active series, got %d", tt.expectedCount, count)
			}
		})
	}
}

func BenchmarkCreateSeriesFingerprint(b *testing.B) {
	labels := map[string]string{
		"service.name":     "api",
		"http.method":      "GET",
		"http.status_code": "200",
		"endpoint":         "/users",
		"region":           "us-east-1",
		"environment":      "production",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CreateSeriesFingerprintFast(labels)
	}
}

func TestMergeActiveSeries(t *testing.T) {
	// Test that active series are correctly merged when combining metrics
	metric1 := NewMetricMetadata("test_metric", &SumMetric{})
	metric2 := NewMetricMetadata("test_metric", &SumMetric{})
	
	// Batch 1: Add some series
	batch1Series := []string{
		"button=btn1,screen=screen1",
		"button=btn2,screen=screen1",
		"button=btn3,screen=screen2",
		"button=btn4,screen=screen2",
		"button=btn5,screen=screen3",
	}
	for _, fp := range batch1Series {
		metric1.AddSeriesFingerprint(fp)
		metric1.SampleCount++
	}
	
	// Batch 2: Add different series with one overlap
	batch2Series := []string{
		"button=btn5,screen=screen3", // Duplicate from batch1
		"button=btn6,screen=screen3",
		"button=btn7,screen=screen4",
		"button=btn8,screen=screen4",
		"button=btn9,screen=screen5",
	}
	for _, fp := range batch2Series {
		metric2.AddSeriesFingerprint(fp)
		metric2.SampleCount++
	}
	
	series1 := metric1.GetActiveSeries()
	series2 := metric2.GetActiveSeries()
	
	if series1 != 5 {
		t.Errorf("Batch 1 should have 5 active series, got %d", series1)
	}
	if series2 != 5 {
		t.Errorf("Batch 2 should have 5 active series, got %d", series2)
	}
	
	// Merge batch2 into batch1
	metric1.MergeMetricMetadata(metric2)
	
	mergedSeries := metric1.GetActiveSeries()
	expectedSeries := int64(9) // 9 unique combinations (btn1-btn9)
	
	// HLL is approximate, allow for small error
	if mergedSeries < expectedSeries-1 || mergedSeries > expectedSeries+1 {
		t.Errorf("After merge, expected ~%d active series, got %d", expectedSeries, mergedSeries)
	}
	
	if metric1.SampleCount != 10 {
		t.Errorf("After merge, expected 10 samples, got %d", metric1.SampleCount)
	}
}

func TestMergeActiveSeriesWithNilHLL(t *testing.T) {
	// Test that merge handles nil HLL gracefully
	metric1 := NewMetricMetadata("test_metric", &GaugeMetric{})
	metric2 := NewMetricMetadata("test_metric", &GaugeMetric{})
	
	metric1.AddSeriesFingerprint("series1")
	metric2.AddSeriesFingerprint("series2")
	
	// This should not panic
	metric1.MergeMetricMetadata(metric2)
	
	mergedSeries := metric1.GetActiveSeries()
	if mergedSeries != 2 {
		t.Errorf("Expected 2 active series after merge, got %d", mergedSeries)
	}
}

func BenchmarkAddSeriesFingerprint(b *testing.B) {
	metric := NewMetricMetadata("benchmark_metric", &GaugeMetric{})
	
	fingerprints := []string{
		"method=GET,status=200",
		"method=GET,status=404",
		"method=POST,status=200",
		"method=POST,status=404",
		"method=PUT,status=200",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metric.AddSeriesFingerprint(fingerprints[i%len(fingerprints)])
	}
}
