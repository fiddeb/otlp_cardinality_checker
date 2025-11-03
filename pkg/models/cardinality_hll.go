package models

import (
	"github.com/fidde/otlp_cardinality_checker/pkg/hyperloglog"
)

// CardinalityInfoHLL is a proof-of-concept implementation using HyperLogLog
// for memory-efficient cardinality estimation.
//
// Memory comparison (100k unique values):
//   - Old approach (map): ~3.2 MB per label
//   - HLL approach: ~16 KB per label (99.5% reduction)
//
// Trade-off: 1-5% estimation error vs unbounded memory growth
type CardinalityInfoHLL struct {
	EstimatedCardinality uint64
	SampleValues         []string                       // Keep small sample for debugging
	MaxSamples           int                            // Limit sample size
	HLL                  *hyperloglog.HyperLogLog      // Fixed-size cardinality tracker
}

// NewCardinalityInfoHLL creates a new HLL-based cardinality tracker.
// precision: HyperLogLog precision (10-18, default 14 = ~16KB, 0.81% error)
// maxSamples: Number of sample values to keep for debugging (default 5)
func NewCardinalityInfoHLL(precision uint8, maxSamples int) *CardinalityInfoHLL {
	if precision == 0 {
		precision = 14 // Default: ~16KB, 0.81% error
	}
	if maxSamples == 0 {
		maxSamples = 5
	}

	return &CardinalityInfoHLL{
		EstimatedCardinality: 0,
		SampleValues:         make([]string, 0, maxSamples),
		MaxSamples:           maxSamples,
		HLL:                  hyperloglog.New(precision),
	}
}

// Add adds a value to the cardinality tracker.
func (c *CardinalityInfoHLL) Add(value string) {
	// Add to HLL for cardinality tracking
	c.HLL.Add(value)
	
	// Keep a small sample for debugging (bounded memory)
	if len(c.SampleValues) < c.MaxSamples {
		// Check if already in samples to avoid duplicates
		found := false
		for _, s := range c.SampleValues {
			if s == value {
				found = true
				break
			}
		}
		if !found {
			c.SampleValues = append(c.SampleValues, value)
		}
	}
	
	// Update estimated cardinality
	c.EstimatedCardinality = c.HLL.Count()
}

// Count returns the estimated cardinality.
func (c *CardinalityInfoHLL) Count() uint64 {
	if c.HLL == nil {
		return 0
	}
	return c.HLL.Count()
}

// MemorySize returns approximate memory usage in bytes.
func (c *CardinalityInfoHLL) MemorySize() int {
	size := 0
	
	// HLL memory
	if c.HLL != nil {
		size += c.HLL.MemorySize()
	}
	
	// Sample values (approximate)
	for _, s := range c.SampleValues {
		size += len(s) + 16 // string + overhead
	}
	
	// Struct overhead
	size += 32
	
	return size
}

// Merge combines another CardinalityInfoHLL into this one.
func (c *CardinalityInfoHLL) Merge(other *CardinalityInfoHLL) error {
	if c.HLL == nil || other.HLL == nil {
		return nil
	}
	
	// Merge HLL sketches
	if err := c.HLL.Merge(other.HLL); err != nil {
		return err
	}
	
	// Merge samples (keep first MaxSamples unique)
	for _, val := range other.SampleValues {
		if len(c.SampleValues) >= c.MaxSamples {
			break
		}
		
		// Check if already exists
		found := false
		for _, existing := range c.SampleValues {
			if existing == val {
				found = true
				break
			}
		}
		
		if !found {
			c.SampleValues = append(c.SampleValues, val)
		}
	}
	
	// Update cardinality
	c.EstimatedCardinality = c.HLL.Count()
	
	return nil
}

// Clear resets the tracker.
func (c *CardinalityInfoHLL) Clear() {
	c.EstimatedCardinality = 0
	c.SampleValues = c.SampleValues[:0]
	if c.HLL != nil {
		c.HLL.Clear()
	}
}
