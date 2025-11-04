package hyperloglog_test

import (
	"fmt"

	"github.com/fidde/otlp_cardinality_checker/pkg/hyperloglog"
)

// Example shows basic HyperLogLog usage
func Example() {
	hll := hyperloglog.New(14)

	// Add some values
	hll.Add("user_1")
	hll.Add("user_2")
	hll.Add("user_3")
	hll.Add("user_1") // Duplicate

	fmt.Printf("Unique users: ~%d\n", hll.Count())
	// Output: Unique users: ~3
}

// Example_merge shows how to merge multiple HLL sketches
func Example_merge() {
	// Track users from two different sources
	source1 := hyperloglog.New(14)
	source2 := hyperloglog.New(14)

	// Source 1: users A, B, C
	source1.Add("user_A")
	source1.Add("user_B")
	source1.Add("user_C")

	// Source 2: users C, D, E (C overlaps)
	source2.Add("user_C")
	source2.Add("user_D")
	source2.Add("user_E")

	// Merge to get union
	source1.Merge(source2)

	// Total unique users: A, B, C, D, E = 5
	fmt.Printf("Total unique users: ~%d\n", source1.Count())
	// Output: Total unique users: ~5
}

// Example_metricLabels shows integration with metric label tracking
func Example_metricLabels() {
	// Simulate tracking cardinality of a metric label
	type MetricInfo struct {
		Name          string
		LabelTrackers map[string]*hyperloglog.HyperLogLog
	}

	metric := MetricInfo{
		Name:          "http_requests_total",
		LabelTrackers: make(map[string]*hyperloglog.HyperLogLog),
	}

	// Initialize HLL for each label we want to track
	metric.LabelTrackers["user_id"] = hyperloglog.New(14)
	metric.LabelTrackers["endpoint"] = hyperloglog.New(14)

	// Simulate incoming metric data points
	dataPoints := []map[string]string{
		{"user_id": "user_123", "endpoint": "/api/users"},
		{"user_id": "user_456", "endpoint": "/api/posts"},
		{"user_id": "user_123", "endpoint": "/api/users"}, // Duplicate
		{"user_id": "user_789", "endpoint": "/api/posts"},
	}

	// Add each data point
	for _, labels := range dataPoints {
		for labelKey, labelValue := range labels {
			if hll, exists := metric.LabelTrackers[labelKey]; exists {
				hll.Add(labelValue)
			}
		}
	}

	// Report cardinality
	fmt.Printf("Metric: %s\n", metric.Name)
	fmt.Printf("  user_id cardinality: ~%d\n", metric.LabelTrackers["user_id"].Count())
	fmt.Printf("  endpoint cardinality: ~%d\n", metric.LabelTrackers["endpoint"].Count())
	// Output:
	// Metric: http_requests_total
	//   user_id cardinality: ~3
	//   endpoint cardinality: ~2
}

// Example_memoryComparison shows memory savings vs naive approach
func Example_memoryComparison() {
	const numUniqueValues = 100000

	// Naive approach: store all values
	naiveMap := make(map[string]struct{})
	for i := 0; i < numUniqueValues; i++ {
		naiveMap[fmt.Sprintf("value_%d", i)] = struct{}{}
	}

	// HyperLogLog approach: fixed memory
	hll := hyperloglog.New(14)
	for i := 0; i < numUniqueValues; i++ {
		hll.Add(fmt.Sprintf("value_%d", i))
	}

	naiveMemory := len(naiveMap) * 16 // Approximate: key pointer + overhead
	hllMemory := hll.MemorySize()

	fmt.Printf("Naive map memory: ~%d KB\n", naiveMemory/1024)
	fmt.Printf("HyperLogLog memory: ~%d KB\n", hllMemory/1024)
	fmt.Printf("Memory savings: %.1f%%\n", (1-float64(hllMemory)/float64(naiveMemory))*100)
	fmt.Printf("Cardinality (actual): %d\n", numUniqueValues)
	fmt.Printf("Cardinality (HLL estimate): ~%d\n", hll.Count())
	// Output will vary slightly due to HLL approximation, but format is:
	// Naive map memory: ~1562 KB
	// HyperLogLog memory: ~16 KB
	// Memory savings: 99.0%
	// Cardinality (actual): 100000
	// Cardinality (HLL estimate): ~102684
}
