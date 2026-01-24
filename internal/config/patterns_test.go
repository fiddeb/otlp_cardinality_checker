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
	expectedNames := []string{"timestamp", "uuid", "email", "service_method", "url", "duration", "size", "ip", "hex", "number"}
	if len(patterns) != len(expectedNames) {
		t.Errorf("Expected %d default patterns, got %d", len(expectedNames), len(patterns))
	}
	
	for i, expected := range expectedNames {
		if patterns[i].Name != expected {
			t.Errorf("Pattern %d: expected name '%s', got '%s'", i, expected, patterns[i].Name)
		}
	}
}

func TestServiceMethodPattern(t *testing.T) {
	patterns := DefaultPatterns()
	
	// Find service_method pattern
	var pattern *CompiledPattern
	for i := range patterns {
		if patterns[i].Name == "service_method" {
			pattern = &patterns[i]
			break
		}
	}
	
	if pattern == nil {
		t.Fatal("service_method pattern not found")
	}
	
	tests := []struct {
		input    string
		expected string
	}{
		{"user-service/resetPassword", "user-service/<METHOD>"},
		{"user-service/getUserProfile", "user-service/<METHOD>"},
		{"order-service/createOrder", "order-service/<METHOD>"},
		{"product-service/getProductDetails", "product-service/<METHOD>"},
		{"cache/get", "cache/<METHOD>"},
		{"db/query", "db/<METHOD>"},
		// Should NOT match these
		{"GET /api/v1/users", "GET /api/v1/users"},               // HTTP method with URL
		{"POST /api/v1/orders/create", "POST /api/v1/orders/create"}, // HTTP method with URL
	}
	
	for _, tt := range tests {
		result := pattern.Regex.ReplaceAllString(tt.input, pattern.Placeholder)
		if result != tt.expected {
			t.Errorf("Input: %q\nExpected: %q\nGot: %q", tt.input, tt.expected, result)
		}
	}
}

// TestRealPatterns tests actual patterns from config/patterns.yaml
func TestRealPatterns(t *testing.T) {
	// Try to load real patterns file
	patterns, err := LoadPatterns("../../config/patterns.yaml")
	if err != nil {
		t.Skipf("Skipping real patterns test: %v", err)
		return
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Apache timestamp
		{
			name:     "Apache timestamp with notice",
			input:    "[Sun Dec 04 04:51:08 2005] [notice] jk2_init() Found child 6725",
			expected: "<TIMESTAMP> [notice] jk2_init() Found child <NUM>",
		},
		// Syslog timestamp
		{
			name:     "Syslog timestamp single digit",
			input:    "Jul  5 13:52:21 combo ftpd[6590]: connection",
			expected: "<TIMESTAMP> combo ftpd[<NUM>]: connection",
		},
		{
			name:     "Syslog timestamp double digit",
			input:    "Jun 22 13:16:30 combo sshd: auth failure",
			expected: "<TIMESTAMP> combo sshd: auth failure",
		},
		// ISO timestamp
		{
			name:     "ISO timestamp",
			input:    "2024/10/26 14:30:45 Server started",
			expected: "<TIMESTAMP> Server started",
		},
		// UUID
		{
			name:     "UUID in message",
			input:    "Request 550e8400-e29b-41d4-a716-446655440000 done",
			expected: "Request <UUID> done",
		},
		// Email
		{
			name:     "Email address",
			input:    "User john.doe@example.com logged in",
			expected: "User <EMAIL> logged in",
		},
		// URL
		{
			name:     "HTTPS URL",
			input:    "GET https://example.com/api/users completed",
			expected: "GET  <URL> completed", // Note: Two spaces before <URL> due to placeholder including leading space
		},
		// IP
		{
			name:     "IPv4 address",
			input:    "Connection from 192.168.1.100 accepted",
			expected: "Connection from <IP> accepted",
		},
		// Duration
		{
			name:     "Duration ms",
			input:    "Query took 150ms to complete",
			expected: "Query took <DURATION> to complete",
		},
		// Size
		{
			name:     "Size MB",
			input:    "Downloaded 25.3MB successfully",
			expected: "Downloaded <SIZE> successfully",
		},
		// Hex
		{
			name:     "Git commit hash",
			input:    "Commit 7f8a9b3c2d1e4f5a6b7c8d9e0f1a2b3c pushed",
			expected: "Commit <HEX> pushed",
		},
		// Numbers (applied last)
		{
			name:     "Port number",
			input:    "Listening on port 8080",
			expected: "Listening on port <NUM>",
		},
		// Combined
		{
			name:     "Multiple patterns",
			input:    "Jul 27 14:41:57 server sshd[1234]: Failed from 10.0.0.1",
			expected: "<TIMESTAMP> server sshd[<NUM>]: Failed from <IP>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input
			
			// Apply all patterns in order
			for _, pattern := range patterns {
				result = pattern.Regex.ReplaceAllString(result, pattern.Placeholder)
			}

			if result != tt.expected {
				t.Errorf("\n  Input:    %s\n  Expected: %s\n  Got:      %s", 
					tt.input, tt.expected, result)
			}
		})
	}
}
