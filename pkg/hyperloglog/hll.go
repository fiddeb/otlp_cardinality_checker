package hyperloglog

import (
	"hash/fnv"
	"math"
	"math/bits"
)

// HyperLogLog implements the HyperLogLog cardinality estimation algorithm.
// It provides memory-efficient approximate counting of unique elements.
//
// Memory usage: 2^precision bytes (e.g., precision=14 uses 16KB)
// Standard error: ~1.04 / sqrt(2^precision)
// For precision=14: error ~0.81%, memory ~16KB
type HyperLogLog struct {
	precision uint8    // Number of bits for register index (4-18)
	m         uint32   // Number of registers (2^precision)
	registers []uint8  // Register array
	alpha     float64  // Bias correction constant
}

// New creates a new HyperLogLog with the given precision.
// Precision must be between 4 and 18.
// Higher precision = more accuracy but more memory.
//
// Recommended values:
//   - 10: ~1KB, 1.6% error
//   - 12: ~4KB, 1.04% error
//   - 14: ~16KB, 0.81% error (recommended)
//   - 16: ~64KB, 0.65% error
func New(precision uint8) *HyperLogLog {
	if precision < 4 || precision > 18 {
		precision = 14 // Default to 14 for good balance
	}

	m := uint32(1 << precision)
	
	// Calculate alpha constant for bias correction
	var alpha float64
	switch m {
	case 16:
		alpha = 0.673
	case 32:
		alpha = 0.697
	case 64:
		alpha = 0.709
	default:
		alpha = 0.7213 / (1 + 1.079/float64(m))
	}

	return &HyperLogLog{
		precision: precision,
		m:         m,
		registers: make([]uint8, m),
		alpha:     alpha,
	}
}

// Add adds an element to the HyperLogLog.
// The element is hashed and used to update the appropriate register.
func (h *HyperLogLog) Add(value string) {
	hash := h.hash(value)
	h.AddHash(hash)
}

// AddHash adds a pre-computed hash to the HyperLogLog.
// This is useful when you already have a hash value.
func (h *HyperLogLog) AddHash(hash uint64) {
	// Split hash into register index (first p bits) and remaining bits
	registerIndex := hash & ((1 << h.precision) - 1)
	w := hash >> h.precision
	
	// Count leading zeros in remaining bits + 1
	// If w is 0, we've used all bits, so set to max possible
	var leadingZeros uint8
	if w == 0 {
		leadingZeros = uint8(64 - h.precision + 1)
	} else {
		leadingZeros = uint8(bits.LeadingZeros64(w) - int(h.precision) + 1)
	}
	
	// Update register with maximum value
	if leadingZeros > h.registers[registerIndex] {
		h.registers[registerIndex] = leadingZeros
	}
}

// Count returns the estimated cardinality.
func (h *HyperLogLog) Count() uint64 {
	// Calculate raw estimate using harmonic mean
	sum := 0.0
	zeros := 0
	
	for _, val := range h.registers {
		sum += 1.0 / float64(uint32(1)<<val)
		if val == 0 {
			zeros++
		}
	}
	
	m := float64(h.m)
	estimate := h.alpha * m * m / sum
	
	// Apply bias corrections for small and large ranges
	if estimate <= 2.5*m {
		// Small range correction
		if zeros != 0 {
			estimate = m * math.Log(m/float64(zeros))
		}
	} else if estimate > (1.0/30.0)*math.Pow(2, 32) {
		// Large range correction
		estimate = -math.Pow(2, 32) * math.Log(1-estimate/math.Pow(2, 32))
	}
	
	return uint64(estimate)
}

// Merge merges another HyperLogLog into this one.
// Both HLLs must have the same precision.
// The result is the union of both sets.
func (h *HyperLogLog) Merge(other *HyperLogLog) error {
	if h.precision != other.precision {
		return ErrPrecisionMismatch
	}
	
	for i := uint32(0); i < h.m; i++ {
		if other.registers[i] > h.registers[i] {
			h.registers[i] = other.registers[i]
		}
	}
	
	return nil
}

// Clear resets all registers to zero.
func (h *HyperLogLog) Clear() {
	for i := range h.registers {
		h.registers[i] = 0
	}
}

// MemorySize returns the approximate memory usage in bytes.
func (h *HyperLogLog) MemorySize() int {
	return int(h.m) + 32 // registers + struct overhead
}

// hash computes a 64-bit hash of the input string.
func (h *HyperLogLog) hash(s string) uint64 {
	hasher := fnv.New64a()
	hasher.Write([]byte(s))
	return hasher.Sum64()
}

var (
	// ErrPrecisionMismatch is returned when trying to merge HLLs with different precisions.
	ErrPrecisionMismatch = &HLLError{"precision mismatch"}
	
	// ErrInvalidData is returned when trying to deserialize invalid HLL data.
	ErrInvalidData = &HLLError{"invalid serialized data"}
)

// HLLError represents an error in HyperLogLog operations.
type HLLError struct {
	message string
}

func (e *HLLError) Error() string {
	return "hyperloglog: " + e.message
}

// MarshalBinary encodes the HLL into a binary format.
// Format: [precision:1byte][registers:m bytes]
func (h *HyperLogLog) MarshalBinary() ([]byte, error) {
	data := make([]byte, 1+len(h.registers))
	data[0] = h.precision
	copy(data[1:], h.registers)
	return data, nil
}

// UnmarshalBinary decodes an HLL from binary format.
func (h *HyperLogLog) UnmarshalBinary(data []byte) error {
	if len(data) < 2 {
		return ErrInvalidData
	}
	
	precision := data[0]
	if precision < 4 || precision > 18 {
		return ErrInvalidData
	}
	
	expectedSize := 1 + (1 << precision)
	if len(data) != expectedSize {
		return ErrInvalidData
	}
	
	// Reinitialize HLL with correct precision
	*h = *New(precision)
	copy(h.registers, data[1:])
	
	return nil
}

// FromBytes creates a new HLL from serialized bytes.
func FromBytes(data []byte) (*HyperLogLog, error) {
	h := &HyperLogLog{}
	if err := h.UnmarshalBinary(data); err != nil {
		return nil, err
	}
	return h, nil
}
