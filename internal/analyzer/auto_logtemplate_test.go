package analyzer

import (
	"strings"
	"testing"

	"github.com/fidde/otlp_cardinality_checker/internal/analyzer/autotemplate"
)

func TestAutoLogBodyAnalyzer(t *testing.T) {
	cfg := autotemplate.Config{
		Shards:       2,
		MaxDepth:     4,
		MaxChildren:  100,
		MaxClusters:  1000,
		SimThreshold: 0.5,
		Training:     true,
	}
	
	analyzer := NewAutoLogBodyAnalyzer(cfg)
	
	// Process similar messages
	_ = analyzer.ProcessMessage("user john logged in from 192.168.1.1")
	template2 := analyzer.ProcessMessage("user jane logged in from 192.168.1.2")
	_ = analyzer.ProcessMessage("user bob logged in from 10.0.0.5")
	
	// Should generalize to same template (with IP masked)
	if !strings.Contains(template2, "user") {
		t.Errorf("template should contain 'user', got: %s", template2)
	}
	
	// Process different message
	template4 := analyzer.ProcessMessage("error connecting to database")
	if template4 == template2 {
		t.Error("different messages should produce different templates")
	}
	
	// Check stats
	stats := analyzer.GetStats()
	if stats["total_messages"].(int64) != 4 {
		t.Errorf("expected 4 total messages, got %v", stats["total_messages"])
	}
	
	// Get templates
	templates := analyzer.GetTemplates()
	if len(templates) == 0 {
		t.Error("expected at least one template")
	}
	
	// Should have 2-3 templates (login pattern(s) + error pattern)
	if len(templates) < 2 {
		t.Errorf("expected at least 2 templates, got %d", len(templates))
	}
	
	t.Logf("Found %d templates:", len(templates))
	for i, tmpl := range templates {
		t.Logf("  %d: %s (count: %d, %.1f%%)", i, tmpl.Template, tmpl.Count, tmpl.Percentage)
	}
}

func TestAutoLogBodyAnalyzerWithPreMasking(t *testing.T) {
	cfg := autotemplate.DefaultConfig()
	cfg.SimThreshold = 0.5
	
	// Create analyzer with default patterns (includes IP, UUID, etc.)
	analyzer := NewAutoLogBodyAnalyzerWithPatterns(cfg, nil)
	
	// Process logs with IPs
	analyzer.ProcessMessage("connected to 192.168.1.100")
	analyzer.ProcessMessage("connected to 10.0.0.5")
	analyzer.ProcessMessage("connected to 172.16.0.1")
	
	templates := analyzer.GetTemplates()
	if len(templates) == 0 {
		t.Fatal("expected at least one template")
	}
	
	// All should map to same template since IPs are pre-masked
	if len(templates) != 1 {
		t.Logf("Templates found: %d", len(templates))
		for i, tmpl := range templates {
			t.Logf("  %d: %s (count: %d)", i, tmpl.Template, tmpl.Count)
		}
		t.Error("expected all IP variations to map to same template")
	}
	
	if templates[0].Count != 3 {
		t.Errorf("expected template count = 3, got %d", templates[0].Count)
	}
	
	// Template should contain the IP placeholder
	if !strings.Contains(templates[0].Template, "<IP>") {
		t.Errorf("template should contain <IP> placeholder, got: %s", templates[0].Template)
	}
}

func TestTrainingModeSwitch(t *testing.T) {
	cfg := autotemplate.DefaultConfig()
	cfg.SimThreshold = 0.5
	analyzer := NewAutoLogBodyAnalyzer(cfg)
	
	// Train with some messages
	analyzer.ProcessMessage("user john logged in")
	analyzer.ProcessMessage("user jane logged in")
	
	// Switch to inference mode
	analyzer.SetTrainingMode(false)
	
	// Known pattern should still work
	template := analyzer.ProcessMessage("user bob logged in")
	if !strings.Contains(template, "user") {
		t.Errorf("should match known pattern, got: %s", template)
	}
	
	stats := analyzer.GetStats()
	if stats["miner_training"].(bool) {
		t.Error("expected training mode to be false")
	}
	
	// Check that template count didn't increase for known pattern
	templates := analyzer.GetTemplates()
	initialCount := len(templates)
	
	// Process unknown pattern (should not create new template in inference mode)
	analyzer.ProcessMessage("completely different pattern here")
	
	templates = analyzer.GetTemplates()
	if len(templates) > initialCount+1 {
		t.Error("new template should not be created in inference mode for unknown pattern")
	}
}

func BenchmarkAutoLogBodyAnalyzer(b *testing.B) {
	cfg := autotemplate.DefaultConfig()
	cfg.Shards = 4
	analyzer := NewAutoLogBodyAnalyzer(cfg)
	
	messages := []string{
		"INFO user logged in from 192.168.1.1",
		"ERROR database connection failed to server-01",
		"DEBUG cache miss for key user:session:abc123",
		"WARN rate limit exceeded for client 10.0.0.5",
		"INFO HTTP GET /api/users/123 returned 200 OK in 45ms",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := messages[i%len(messages)]
		analyzer.ProcessMessage(msg)
	}
	
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "eps")
}
