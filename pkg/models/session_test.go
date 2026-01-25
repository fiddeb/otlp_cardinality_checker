package models

import (
	"testing"

	"github.com/fidde/otlp_cardinality_checker/pkg/hyperloglog"
)

func TestValidateSessionName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "my-session", false},
		{"valid single char", "a", false},
		{"valid with numbers", "test123", false},
		{"valid with hyphens", "pre-deploy-2024", false},
		{"empty", "", true},
		{"too long", string(make([]byte, 129)), true},
		{"uppercase", "MySession", true},
		{"spaces", "my session", true},
		{"underscore", "my_session", true},
		{"starts with hyphen", "-session", true},
		{"ends with hyphen", "session-", true},
		{"special chars", "my@session", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSessionName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSessionName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestSerializedHLL_RoundTrip(t *testing.T) {
	// Create HLL and add some values
	hll := hyperloglog.New(14)
	values := []string{"user1", "user2", "user3", "user1", "user4", "user5"}
	for _, v := range values {
		hll.Add(v)
	}

	// Get original estimate
	originalEstimate := hll.Count()

	// Marshal
	serialized, err := MarshalHLL(hll)
	if err != nil {
		t.Fatalf("MarshalHLL failed: %v", err)
	}

	if serialized == nil {
		t.Fatal("MarshalHLL returned nil")
	}

	if serialized.Precision != 14 {
		t.Errorf("Expected precision 14, got %d", serialized.Precision)
	}

	if serialized.Registers == "" {
		t.Error("Expected non-empty registers")
	}

	// Unmarshal
	restored, err := UnmarshalHLL(serialized)
	if err != nil {
		t.Fatalf("UnmarshalHLL failed: %v", err)
	}

	if restored == nil {
		t.Fatal("UnmarshalHLL returned nil")
	}

	// Compare estimates
	restoredEstimate := restored.Count()
	if originalEstimate != restoredEstimate {
		t.Errorf("Estimate mismatch: original %d, restored %d", originalEstimate, restoredEstimate)
	}
}

func TestSerializedHLL_NilHandling(t *testing.T) {
	// Test nil HLL
	serialized, err := MarshalHLL(nil)
	if err != nil {
		t.Fatalf("MarshalHLL(nil) should not error: %v", err)
	}
	if serialized != nil {
		t.Error("MarshalHLL(nil) should return nil")
	}

	// Test nil SerializedHLL
	restored, err := UnmarshalHLL(nil)
	if err != nil {
		t.Fatalf("UnmarshalHLL(nil) should not error: %v", err)
	}
	if restored != nil {
		t.Error("UnmarshalHLL(nil) should return nil")
	}
}

func TestSerializedHLL_LargeDataset(t *testing.T) {
	hll := hyperloglog.New(14)

	// Add 10000 unique values
	for i := 0; i < 10000; i++ {
		hll.Add("value-" + string(rune(i)))
	}

	originalEstimate := hll.Count()

	// Round trip
	serialized, err := MarshalHLL(hll)
	if err != nil {
		t.Fatalf("MarshalHLL failed: %v", err)
	}

	restored, err := UnmarshalHLL(serialized)
	if err != nil {
		t.Fatalf("UnmarshalHLL failed: %v", err)
	}

	restoredEstimate := restored.Count()

	// Estimates should be identical after round-trip
	if originalEstimate != restoredEstimate {
		t.Errorf("Large dataset estimate mismatch: original %d, restored %d", 
			originalEstimate, restoredEstimate)
	}
}

func TestSerializeKeyMetadata_RoundTrip(t *testing.T) {
	// Create KeyMetadata with HLL
	km := NewKeyMetadata()
	values := []string{"val1", "val2", "val3", "val4", "val5"}
	for _, v := range values {
		km.AddValue(v)
	}
	km.Count = 100
	km.Percentage = 50.5

	// Serialize
	serialized, err := SerializeKeyMetadata(km)
	if err != nil {
		t.Fatalf("SerializeKeyMetadata failed: %v", err)
	}

	if serialized.Count != 100 {
		t.Errorf("Expected count 100, got %d", serialized.Count)
	}

	if serialized.Percentage != 50.5 {
		t.Errorf("Expected percentage 50.5, got %f", serialized.Percentage)
	}

	if serialized.HLL == nil {
		t.Error("Expected HLL to be serialized")
	}

	// Deserialize
	restored, err := DeserializeKeyMetadata(serialized)
	if err != nil {
		t.Fatalf("DeserializeKeyMetadata failed: %v", err)
	}

	if restored.Count != km.Count {
		t.Errorf("Count mismatch: original %d, restored %d", km.Count, restored.Count)
	}

	if restored.EstimatedCardinality != km.EstimatedCardinality {
		t.Errorf("Cardinality mismatch: original %d, restored %d", 
			km.EstimatedCardinality, restored.EstimatedCardinality)
	}
}

func TestSerializeKeyMetadata_NilHandling(t *testing.T) {
	serialized, err := SerializeKeyMetadata(nil)
	if err != nil {
		t.Fatalf("SerializeKeyMetadata(nil) should not error: %v", err)
	}
	if serialized != nil {
		t.Error("SerializeKeyMetadata(nil) should return nil")
	}

	restored, err := DeserializeKeyMetadata(nil)
	if err != nil {
		t.Fatalf("DeserializeKeyMetadata(nil) should not error: %v", err)
	}
	if restored != nil {
		t.Error("DeserializeKeyMetadata(nil) should return nil")
	}
}

func TestCalculateSeverity(t *testing.T) {
	tests := []struct {
		name     string
		from     int64
		to       int64
		expected string
	}{
		{"10x increase", 100, 1000, SeverityCritical},
		{"exactly 10x", 100, 1000, SeverityCritical},
		{"5x increase", 100, 500, SeverityWarning},
		{"2x increase", 100, 200, SeverityWarning},
		{"1.5x increase", 100, 150, SeverityInfo},
		{"no change", 100, 100, SeverityInfo},
		{"decrease", 100, 50, SeverityInfo},
		{"from zero high", 0, 1000, SeverityWarning},
		{"from zero low", 0, 50, SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateSeverity(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("CalculateSeverity(%d, %d) = %s, want %s", 
					tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

func TestMaxSeverity(t *testing.T) {
	tests := []struct {
		a, b     string
		expected string
	}{
		{SeverityInfo, SeverityInfo, SeverityInfo},
		{SeverityInfo, SeverityWarning, SeverityWarning},
		{SeverityWarning, SeverityInfo, SeverityWarning},
		{SeverityWarning, SeverityCritical, SeverityCritical},
		{SeverityCritical, SeverityWarning, SeverityCritical},
		{SeverityCritical, SeverityCritical, SeverityCritical},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			result := MaxSeverity(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("MaxSeverity(%s, %s) = %s, want %s", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestFilterBySeverity(t *testing.T) {
	changes := []Change{
		{Name: "info1", Severity: SeverityInfo},
		{Name: "warn1", Severity: SeverityWarning},
		{Name: "crit1", Severity: SeverityCritical},
		{Name: "info2", Severity: SeverityInfo},
		{Name: "warn2", Severity: SeverityWarning},
	}

	tests := []struct {
		minSeverity string
		expected    int
	}{
		{"", 5},
		{SeverityInfo, 5},
		{SeverityWarning, 3},
		{SeverityCritical, 1},
	}

	for _, tt := range tests {
		t.Run(tt.minSeverity, func(t *testing.T) {
			filtered := FilterBySeverity(changes, tt.minSeverity)
			if len(filtered) != tt.expected {
				t.Errorf("FilterBySeverity(%s) returned %d, want %d", 
					tt.minSeverity, len(filtered), tt.expected)
			}
		})
	}
}

func TestDiffResult_AddChange(t *testing.T) {
	diff := NewDiffResult("session-a", "session-b")

	// Add various changes
	diff.AddChange(Change{Type: ChangeTypeAdded, SignalType: SignalTypeMetric, Name: "new_metric", Severity: SeverityInfo})
	diff.AddChange(Change{Type: ChangeTypeRemoved, SignalType: SignalTypeMetric, Name: "old_metric", Severity: SeverityInfo})
	diff.AddChange(Change{Type: ChangeTypeChanged, SignalType: SignalTypeMetric, Name: "changed_metric", Severity: SeverityCritical})
	diff.AddChange(Change{Type: ChangeTypeAdded, SignalType: SignalTypeSpan, Name: "new_span", Severity: SeverityWarning})

	// Check summary
	if diff.Summary.Metrics.Added != 1 {
		t.Errorf("Expected 1 added metric, got %d", diff.Summary.Metrics.Added)
	}
	if diff.Summary.Metrics.Removed != 1 {
		t.Errorf("Expected 1 removed metric, got %d", diff.Summary.Metrics.Removed)
	}
	if diff.Summary.Metrics.Changed != 1 {
		t.Errorf("Expected 1 changed metric, got %d", diff.Summary.Metrics.Changed)
	}
	if diff.Summary.Spans.Added != 1 {
		t.Errorf("Expected 1 added span, got %d", diff.Summary.Spans.Added)
	}

	// Check critical changes tracked
	if len(diff.CriticalChanges) != 1 {
		t.Errorf("Expected 1 critical change, got %d", len(diff.CriticalChanges))
	}
	if diff.CriticalChanges[0].Name != "changed_metric" {
		t.Errorf("Expected critical change name 'changed_metric', got %s", diff.CriticalChanges[0].Name)
	}
}
