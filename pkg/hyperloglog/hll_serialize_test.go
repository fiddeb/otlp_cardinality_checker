package hyperloglog

import (
	"bytes"
	"fmt"
	"testing"
)

func TestMarshalUnmarshal(t *testing.T) {
	hll := New(14)
	
	// Add some values
	for i := 0; i < 1000; i++ {
		hll.Add(fmt.Sprintf("value_%d", i))
	}
	
	originalCount := hll.Count()
	
	// Marshal
	data, err := hll.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}
	
	// Expected size: 1 byte precision + 16384 bytes registers
	expectedSize := 1 + 16384
	if len(data) != expectedSize {
		t.Errorf("MarshalBinary() size = %d, want %d", len(data), expectedSize)
	}
	
	// Unmarshal into new HLL
	hll2 := &HyperLogLog{}
	if err := hll2.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary() error = %v", err)
	}
	
	// Check that count is preserved
	newCount := hll2.Count()
	if newCount != originalCount {
		t.Errorf("After unmarshal, Count() = %d, want %d", newCount, originalCount)
	}
	
	// Check precision
	if hll2.precision != 14 {
		t.Errorf("After unmarshal, precision = %d, want 14", hll2.precision)
	}
	
	// Check that registers match
	if !bytes.Equal(hll.registers, hll2.registers) {
		t.Error("Registers don't match after unmarshal")
	}
}

func TestFromBytes(t *testing.T) {
	original := New(12)
	for i := 0; i < 500; i++ {
		original.Add(fmt.Sprintf("test_%d", i))
	}
	
	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}
	
	restored, err := FromBytes(data)
	if err != nil {
		t.Fatalf("FromBytes() error = %v", err)
	}
	
	if restored.Count() != original.Count() {
		t.Errorf("FromBytes() count = %d, want %d", restored.Count(), original.Count())
	}
}

func TestUnmarshalBinary_InvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"too short", []byte{14}},
		{"invalid precision", []byte{1, 0, 0, 0}},
		{"wrong size", []byte{14, 0, 0, 0}}, // precision 14 needs 16385 bytes total
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hll := &HyperLogLog{}
			err := hll.UnmarshalBinary(tt.data)
			if err == nil {
				t.Error("UnmarshalBinary() expected error, got nil")
			}
		})
	}
}

func TestSerializeEmpty(t *testing.T) {
	hll := New(10)
	
	data, err := hll.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}
	
	restored, err := FromBytes(data)
	if err != nil {
		t.Fatalf("FromBytes() error = %v", err)
	}
	
	if restored.Count() != 0 {
		t.Errorf("Empty HLL after restore, Count() = %d, want 0", restored.Count())
	}
}

func TestSerializeThenMerge(t *testing.T) {
	// Create two HLLs with different data
	hll1 := New(14)
	hll2 := New(14)
	
	for i := 0; i < 100; i++ {
		hll1.Add(fmt.Sprintf("set1_%d", i))
	}
	
	for i := 0; i < 100; i++ {
		hll2.Add(fmt.Sprintf("set2_%d", i))
	}
	
	// Serialize hll1
	data, err := hll1.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}
	
	// Restore hll1 from bytes
	restored, err := FromBytes(data)
	if err != nil {
		t.Fatalf("FromBytes() error = %v", err)
	}
	
	// Merge with hll2
	if err := restored.Merge(hll2); err != nil {
		t.Fatalf("Merge() error = %v", err)
	}
	
	// Result should be around 200
	count := restored.Count()
	if count < 190 || count > 210 {
		t.Errorf("After merge, Count() = %d, want ~200", count)
	}
}

func BenchmarkMarshalBinary(b *testing.B) {
	hll := New(14)
	for i := 0; i < 10000; i++ {
		hll.Add(fmt.Sprintf("value_%d", i))
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = hll.MarshalBinary()
	}
}

func BenchmarkUnmarshalBinary(b *testing.B) {
	hll := New(14)
	for i := 0; i < 10000; i++ {
		hll.Add(fmt.Sprintf("value_%d", i))
	}
	
	data, _ := hll.MarshalBinary()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h := &HyperLogLog{}
		_ = h.UnmarshalBinary(data)
	}
}
