package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPatterns(t *testing.T) {
	// Create a temporary patterns file
	tmpDir := t.TempDir()
	patternsFile := filepath.Join(tmpDir, "patterns.yaml")
	
	yamlContent := `patterns:
  - name: test_number
    regex: '\b\d+\b'
    placeholder: '<NUM>'
    description: 'Test number pattern'
  - name: test_email
    regex: '\b[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b'
    placeholder: '<EMAIL>'
    description: 'Test email pattern'
`
	
	if err := os.WriteFile(patternsFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test patterns file: %v", err)
	}
	
	// Load patterns
	patterns, err := LoadPatterns(patternsFile)
	if err != nil {
		t.Fatalf("LoadPatterns failed: %v", err)
	}
	
	// Verify loaded patterns
	if len(patterns) != 2 {
		t.Errorf("Expected 2 patterns, got %d", len(patterns))
	}
	
	if patterns[0].Name != "test_number" {
		t.Errorf("Expected first pattern name 'test_number', got '%s'", patterns[0].Name)
	}
	
	if patterns[0].Placeholder != "<NUM>" {
		t.Errorf("Expected placeholder '<NUM>', got '%s'", patterns[0].Placeholder)
	}
	
	// Test that regex is compiled and works
	testString := "User 123 sent email to user@example.com"
	result := patterns[0].Regex.ReplaceAllString(testString, patterns[0].Placeholder)
	expected := "User <NUM> sent email to user@example.com"
	if result != expected {
		t.Errorf("Pattern replacement failed:\nExpected: %s\nGot: %s", expected, result)
	}
}

func TestDefaultPatterns(t *testing.T) {
	patterns := DefaultPatterns()
	
	if len(patterns) == 0 {
		t.Error("DefaultPatterns returned empty list")
	}
	
	// Verify we have expected patterns
	expectedNames := []string{"timestamp", "uuid", "email", "url", "duration", "size", "ip", "hex", "number"}
	if len(patterns) != len(expectedNames) {
		t.Errorf("Expected %d default patterns, got %d", len(expectedNames), len(patterns))
	}
	
	for i, expected := range expectedNames {
		if patterns[i].Name != expected {
			t.Errorf("Pattern %d: expected name '%s', got '%s'", i, expected, patterns[i].Name)
		}
	}
}
