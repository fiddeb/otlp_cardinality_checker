package analyzer

import (
	"testing"
)

func TestLogBodyAnalyzer_ExtractTemplate(t *testing.T) {
	analyzer := NewLogBodyAnalyzer()
	
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "HTTP request log",
			message:  `[fidde.local/czlriPsfX6-000023] "GET http://localhost:8080/api/v1/logs?limit=10000 HTTP/1.1" from [::1]:62171 - 200 4946B in 377.802µs`,
			expected: `[fidde.local/czlriPsfX6-<NUM>] "GET <URL> HTTP/<NUM>.<NUM>" from <IP>:<NUM> - <NUM> <SIZE> in <DURATION>`,
		},
		{
			name:     "Different limit value",
			message:  `[fidde.local/czlriPsfX6-000025] "GET http://localhost:8080/api/v1/logs?limit=1 HTTP/1.1" from [::1]:62173 - 200 1225B in 398.234µs`,
			expected: `[fidde.local/czlriPsfX6-<NUM>] "GET <URL> HTTP/<NUM>.<NUM>" from <IP>:<NUM> - <NUM> <SIZE> in <DURATION>`,
		},
		{
			name:     "URL in message",
			message:  "Application started successfully http://bilder.fberggren.se",
			expected: "Application started successfully <URL>",
		},
		{
			name:     "Path in message",
			message:  "Reading config from /etc/app/config.yml",
			expected: "Reading config from <URL>",
		},
		{
			name:     "UUID in message",
			message:  "Request 550e8400-e29b-41d4-a716-446655440000 failed",
			expected: "Request <UUID> failed",
		},
		{
			name:     "Timestamp in message",
			message:  "2025/10/24 23:56:44 Server started",
			expected: "<TIMESTAMP> Server started",
		},
		{
			name:     "Email in message",
			message:  "Email sent to user_59@example.com - delivery ID abc123def",
			expected: "Email sent to <EMAIL> - delivery ID <HEX>",
		},
		{
			name:     "Multiple emails",
			message:  "Forwarded from john.doe@company.com to jane_smith@example.org",
			expected: "Forwarded from <EMAIL> to <EMAIL>",
		},
		{
			name:     "Mixed patterns",
			message:  "User 12345 downloaded 2.5MB in 150ms from 192.168.1.1",
			expected: "User <NUM> downloaded <SIZE> in <DURATION> from <IP>",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.ExtractTemplate(tt.message)
			if result != tt.expected {
				t.Errorf("\nExpected: %s\nGot:      %s", tt.expected, result)
			}
		})
	}
}

func TestLogBodyAnalyzer_AddMessage(t *testing.T) {
	analyzer := NewLogBodyAnalyzer()
	
	// Add multiple messages with same pattern
	messages := []string{
		`"GET /api/v1/logs?limit=10000" - 200 4946B in 377µs`,
		`"GET /api/v1/logs?limit=1" - 200 1225B in 398µs`,
		`"GET /api/v1/logs?limit=5000" - 200 2500B in 250µs`,
		`"POST /api/v1/metrics" - 201 100B in 50µs`, // Different pattern
	}
	
	for _, msg := range messages {
		analyzer.AddMessage(msg)
	}
	
	templates := analyzer.GetTemplates()
	
	// Should have 2 unique templates
	if len(templates) != 2 {
		t.Errorf("Expected 2 templates, got %d", len(templates))
	}
	
	// First template (most common) should have count of 3
	if templates[0].Count != 3 {
		t.Errorf("Expected count 3 for first template, got %d", templates[0].Count)
	}
	
	// Check percentage calculation
	if templates[0].Percentage != 75.0 {
		t.Errorf("Expected percentage 75.0, got %.1f", templates[0].Percentage)
	}
	
	// Check stats
	stats := analyzer.GetStats()
	if stats["total_messages"].(int64) != 4 {
		t.Errorf("Expected total_messages 4, got %v", stats["total_messages"])
	}
	if stats["unique_templates"].(int) != 2 {
		t.Errorf("Expected unique_templates 2, got %v", stats["unique_templates"])
	}
}

func BenchmarkLogBodyAnalyzer_ExtractTemplate(b *testing.B) {
	analyzer := NewLogBodyAnalyzer()
	message := `[fidde.local/czlriPsfX6-000023] "GET http://localhost:8080/api/v1/logs?limit=10000 HTTP/1.1" from [::1]:62171 - 200 4946B in 377.802µs`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.ExtractTemplate(message)
	}
}

func BenchmarkLogBodyAnalyzer_AddMessage(b *testing.B) {
	analyzer := NewLogBodyAnalyzer()
	message := `[fidde.local/czlriPsfX6-000023] "GET http://localhost:8080/api/v1/logs?limit=10000 HTTP/1.1" from [::1]:62171 - 200 4946B in 377.802µs`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.AddMessage(message)
	}
}
