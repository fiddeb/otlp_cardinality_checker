package analyzer

import (
	"testing"

	"github.com/fidde/otlp_cardinality_checker/pkg/models"
)

func TestSpanNameAnalyzer_ExtractPattern(t *testing.T) {
	analyzer := NewSpanNameAnalyzer()

	tests := []struct {
		name     string
		spanName string
		expected string
	}{
		{
			name:     "HTTP path with numeric ID",
			spanName: "GET /users/123",
			expected: "GET <URL>", // URL pattern matches paths
		},
		{
			name:     "HTTP path with multiple IDs",
			spanName: "POST /orders/456/items/789",
			expected: "POST <URL>", // URL pattern matches paths
		},
		{
			name:     "gRPC method",
			spanName: "grpc.UserService/GetUser",
			expected: "grpc.UserService/GetUser",
		},
		{
			name:     "operation with UUID",
			spanName: "process-550e8400-e29b-41d4-a716-446655440000",
			expected: "process-<UUID>",
		},
		{
			name:     "operation with hex string",
			spanName: "my-operation-deadbeef1234",
			expected: "my-operation-<HEX>",
		},
		{
			name:     "simple span name",
			spanName: "database-query",
			expected: "database-query",
		},
		{
			name:     "span with timestamp",
			spanName: "job-2024/01/22 10:30:45",
			expected: "job-<TIMESTAMP>", // Timestamp pattern is greedy
		},
		{
			name:     "span with IP address",
			spanName: "connect 192.168.1.100",
			expected: "connect <IP>",
		},
		{
			name:     "span with duration",
			spanName: "timeout-500ms",
			expected: "timeout-<DURATION>",
		},
		{
			name:     "kafka consumer topic partition",
			spanName: "kafka.consume/my-topic partition 12",
			expected: "kafka.consume/my-topic partition <NUM>",
		},
		{
			name:     "span with number suffix",
			spanName: "process-batch-42",
			expected: "process-batch-<NUM>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.ExtractPattern(tt.spanName)
			if result != tt.expected {
				t.Errorf("ExtractPattern(%q) = %q, want %q", tt.spanName, result, tt.expected)
			}
		})
	}
}

func TestSpanNameAnalyzer_AddSpanName(t *testing.T) {
	analyzer := NewSpanNameAnalyzer()

	// Add several similar span names (using patterns that don't get URL-replaced)
	analyzer.AddSpanName("process-batch-1")
	analyzer.AddSpanName("process-batch-2")
	analyzer.AddSpanName("process-batch-3")
	analyzer.AddSpanName("process-batch-4")
	analyzer.AddSpanName("send-notification-100")

	patterns := analyzer.GetPatterns()

	if len(patterns) != 2 {
		t.Fatalf("Expected 2 patterns, got %d", len(patterns))
	}

	// Find the process-batch pattern (should be first as it has higher count)
	var processBatchPattern *models.SpanNamePattern
	var sendNotificationPattern *models.SpanNamePattern
	for _, p := range patterns {
		if p.Template == "process-batch-<NUM>" {
			processBatchPattern = p
		}
		if p.Template == "send-notification-<NUM>" {
			sendNotificationPattern = p
		}
	}

	if processBatchPattern == nil {
		t.Fatal("Expected to find process-batch-<NUM> pattern")
	}
	if processBatchPattern.Count != 4 {
		t.Errorf("Expected count 4 for process-batch, got %d", processBatchPattern.Count)
	}
	if processBatchPattern.Percentage != 80.0 {
		t.Errorf("Expected percentage 80.0 for process-batch, got %.1f", processBatchPattern.Percentage)
	}
	if len(processBatchPattern.Examples) != 3 {
		t.Errorf("Expected 3 examples (max), got %d", len(processBatchPattern.Examples))
	}

	if sendNotificationPattern == nil {
		t.Fatal("Expected to find send-notification-<NUM> pattern")
	}
	if sendNotificationPattern.Count != 1 {
		t.Errorf("Expected count 1 for send-notification, got %d", sendNotificationPattern.Count)
	}
}

func TestSpanNameAnalyzer_MaxExamples(t *testing.T) {
	analyzer := NewSpanNameAnalyzer()

	// Add 5 different span names that map to same pattern
	for i := 1; i <= 5; i++ {
		analyzer.AddSpanName("GET /api/resource/" + string(rune('0'+i)))
	}

	patterns := analyzer.GetPatterns()

	if len(patterns) != 1 {
		t.Fatalf("Expected 1 pattern, got %d", len(patterns))
	}

	// Should have only 3 examples (MaxExamples default)
	if len(patterns[0].Examples) != 3 {
		t.Errorf("Expected 3 examples (max), got %d", len(patterns[0].Examples))
	}
}

func TestSpanNameAnalyzer_EmptySpanName(t *testing.T) {
	analyzer := NewSpanNameAnalyzer()

	analyzer.AddSpanName("")
	analyzer.AddSpanName("valid-span")

	patterns := analyzer.GetPatterns()

	if len(patterns) != 1 {
		t.Fatalf("Expected 1 pattern (empty ignored), got %d", len(patterns))
	}
	if patterns[0].Template != "valid-span" {
		t.Errorf("Expected template 'valid-span', got %q", patterns[0].Template)
	}
}

func TestSpanNameAnalyzer_GetTotal(t *testing.T) {
	analyzer := NewSpanNameAnalyzer()

	analyzer.AddSpanName("span-1")
	analyzer.AddSpanName("span-2")
	analyzer.AddSpanName("span-3")

	if analyzer.GetTotal() != 3 {
		t.Errorf("Expected total 3, got %d", analyzer.GetTotal())
	}
}

func TestSpanNameAnalyzer_Reset(t *testing.T) {
	analyzer := NewSpanNameAnalyzer()

	analyzer.AddSpanName("span-1")
	analyzer.AddSpanName("span-2")
	analyzer.Reset()

	if analyzer.GetTotal() != 0 {
		t.Errorf("Expected total 0 after reset, got %d", analyzer.GetTotal())
	}
	if len(analyzer.GetPatterns()) != 0 {
		t.Errorf("Expected 0 patterns after reset, got %d", len(analyzer.GetPatterns()))
	}
}

func TestSpanNameAnalyzer_DuplicateExamples(t *testing.T) {
	analyzer := NewSpanNameAnalyzer()

	// Add the exact same span name multiple times
	analyzer.AddSpanName("GET /users/123")
	analyzer.AddSpanName("GET /users/123")
	analyzer.AddSpanName("GET /users/123")

	patterns := analyzer.GetPatterns()

	if len(patterns) != 1 {
		t.Fatalf("Expected 1 pattern, got %d", len(patterns))
	}

	// Should only have 1 unique example
	if len(patterns[0].Examples) != 1 {
		t.Errorf("Expected 1 unique example, got %d", len(patterns[0].Examples))
	}
	if patterns[0].Count != 3 {
		t.Errorf("Expected count 3, got %d", patterns[0].Count)
	}
}

func TestSpanNameAnalyzer_SortedByCount(t *testing.T) {
	analyzer := NewSpanNameAnalyzer()

	// Add patterns with different counts
	analyzer.AddSpanName("rare-operation")
	for i := 0; i < 5; i++ {
		analyzer.AddSpanName("common-operation")
	}
	for i := 0; i < 10; i++ {
		analyzer.AddSpanName("very-common-operation")
	}

	patterns := analyzer.GetPatterns()

	if len(patterns) != 3 {
		t.Fatalf("Expected 3 patterns, got %d", len(patterns))
	}

	// Should be sorted by count descending
	if patterns[0].Template != "very-common-operation" {
		t.Errorf("Expected first pattern to be 'very-common-operation', got %q", patterns[0].Template)
	}
	if patterns[1].Template != "common-operation" {
		t.Errorf("Expected second pattern to be 'common-operation', got %q", patterns[1].Template)
	}
	if patterns[2].Template != "rare-operation" {
		t.Errorf("Expected third pattern to be 'rare-operation', got %q", patterns[2].Template)
	}
}
