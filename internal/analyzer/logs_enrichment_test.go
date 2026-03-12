package analyzer

import (
	"context"
	"testing"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
)

func TestInferSeverityFromBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"ERROR keyword", "[RedisCacheHandler] Redis connection error: ECONNREFUSED", "ERROR"},
		{"ERROR uppercase", "ERROR: something went wrong", "ERROR"},
		{"WARN keyword", "warn: disk usage high", "WARN"},
		{"warning keyword", "Warning: disk usage high", "WARN"},
		{"INFO keyword", "info starting service", "INFO"},
		{"DEBUG keyword", "debug trace entry", "DEBUG"},
		{"case-insensitive error", "Error connecting to DB", "ERROR"},
		{"case-insensitive warn", "WARN rate limit exceeded", "WARN"},
		{"error beats warn", "error and warn in same message", "ERROR"},
		{"warn beats info", "warn and info in same message", "WARN"},
		{"no keyword", "connection established successfully", "UNSET"},
		{"empty body", "", "UNSET"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inferSeverityFromBody(tt.body); got != tt.want {
				t.Fatalf("inferSeverityFromBody(%q) = %q, want %q", tt.body, got, tt.want)
			}
		})
	}
}

// makeAttr returns a string KV proto attribute.
func makeAttr(key, val string) *commonpb.KeyValue {
	return &commonpb.KeyValue{
		Key: key,
		Value: &commonpb.AnyValue{
			Value: &commonpb.AnyValue_StringValue{StringValue: val},
		},
	}
}

// makeLogsRequest builds a minimal OTLP logs request with the supplied resource
// attributes, severity text, and body.
func makeLogsRequest(resourceAttrs []*commonpb.KeyValue, severityText string, body string) *collogspb.ExportLogsServiceRequest {
	return &collogspb.ExportLogsServiceRequest{
		ResourceLogs: []*logspb.ResourceLogs{
			{
				Resource: &resourcepb.Resource{Attributes: resourceAttrs},
				ScopeLogs: []*logspb.ScopeLogs{
					{
						LogRecords: []*logspb.LogRecord{
							{
								SeverityText: severityText,
								Body: &commonpb.AnyValue{
									Value: &commonpb.AnyValue_StringValue{StringValue: body},
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestLogsAnalyzer_PodLogEnrichment(t *testing.T) {
	defaultLabels := []string{
		"service_name", "service", "app", "application", "name",
		"app_kubernetes_io_name", "k8s.container.name", "k8s.deployment.name",
		"k8s.pod.name", "container", "component", "workload", "job",
	}

	t.Run("service name resolved from k8s.container.name", func(t *testing.T) {
		a := NewLogsAnalyzerWithCatalog(nil)
		a.SetPodLogEnrichment(true, defaultLabels)

		req := makeLogsRequest(
			[]*commonpb.KeyValue{makeAttr("k8s.container.name", "nova-powerplay-app")},
			"", "starting up",
		)
		results, err := a.AnalyzeWithContext(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected at least one result")
		}
		if _, ok := results[0].Services["nova-powerplay-app"]; !ok {
			t.Fatalf("expected service nova-powerplay-app, got %v", results[0].Services)
		}
	})

	t.Run("severity inferred from body when empty", func(t *testing.T) {
		a := NewLogsAnalyzerWithCatalog(nil)
		a.SetPodLogEnrichment(true, defaultLabels)

		req := makeLogsRequest(
			[]*commonpb.KeyValue{makeAttr("k8s.container.name", "my-app")},
			"", "[RedisCacheHandler] Redis connection error: ECONNREFUSED",
		)
		results, err := a.AnalyzeWithContext(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected at least one result")
		}
		if results[0].Severity != "ERROR" {
			t.Fatalf("expected severity ERROR, got %q", results[0].Severity)
		}
	})

	t.Run("existing severity is not overridden", func(t *testing.T) {
		a := NewLogsAnalyzerWithCatalog(nil)
		a.SetPodLogEnrichment(true, defaultLabels)

		req := makeLogsRequest(
			[]*commonpb.KeyValue{makeAttr("k8s.container.name", "my-app")},
			"INFO", "some info message",
		)
		results, err := a.AnalyzeWithContext(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected at least one result")
		}
		if results[0].Severity != "INFO" {
			t.Fatalf("expected severity INFO, got %q", results[0].Severity)
		}
	})

	t.Run("enrichment disabled preserves unknown and UNSET", func(t *testing.T) {
		a := NewLogsAnalyzerWithCatalog(nil)
		// No SetPodLogEnrichment call — default off.

		req := makeLogsRequest(
			[]*commonpb.KeyValue{makeAttr("k8s.container.name", "nova-powerplay-app")},
			"", "Redis connection error: ECONNREFUSED",
		)
		results, err := a.AnalyzeWithContext(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected at least one result")
		}
		// Service name must fall back to "unknown" (not resolved from k8s.container.name)
		if _, ok := results[0].Services["unknown"]; !ok {
			t.Fatalf("expected service unknown, got %v", results[0].Services)
		}
		// Severity must be UNSET (not inferred from body)
		if results[0].Severity != "UNSET" {
			t.Fatalf("expected severity UNSET, got %q", results[0].Severity)
		}
	})

	t.Run("no label match falls back to unknown_service", func(t *testing.T) {
		a := NewLogsAnalyzerWithCatalog(nil)
		a.SetPodLogEnrichment(true, defaultLabels)

		req := makeLogsRequest(
			[]*commonpb.KeyValue{makeAttr("host.name", "worker-1")},
			"INFO", "some message",
		)
		results, err := a.AnalyzeWithContext(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected at least one result")
		}
		if _, ok := results[0].Services["unknown_service"]; !ok {
			t.Fatalf("expected service unknown_service, got %v", results[0].Services)
		}
	})
}
