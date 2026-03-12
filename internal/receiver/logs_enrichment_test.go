package receiver

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/protobuf/proto"

	"github.com/fidde/otlp_cardinality_checker/internal/storage"
)

// newTestReceiverWithEnrichment creates an HTTPReceiver with pod log enrichment
// enabled and returns both the receiver and the store for result inspection.
func newTestReceiverWithEnrichment(t *testing.T) (*HTTPReceiver, storage.Storage) {
	t.Helper()
	cfg := storage.DefaultConfig()
	cfg.PodLogEnrichment = true
	store := storage.NewStorage(cfg)
	r := NewHTTPReceiver(":0", store)
	return r, store
}

// minimalPodLogsProto returns a serialised ExportLogsServiceRequest that mimics
// a pod log collected by the filelog/podlogs receiver: no service.name, empty
// SeverityText, and a body that contains the keyword "error".
func minimalPodLogsProto(t *testing.T) []byte {
	t.Helper()
	req := &collogspb.ExportLogsServiceRequest{
		ResourceLogs: []*logspb.ResourceLogs{
			{
				Resource: &resourcepb.Resource{
					Attributes: []*commonpb.KeyValue{
						{Key: "k8s.container.name", Value: &commonpb.AnyValue{
							Value: &commonpb.AnyValue_StringValue{StringValue: "nova-powerplay-app"},
						}},
						{Key: "k8s.namespace.name", Value: &commonpb.AnyValue{
							Value: &commonpb.AnyValue_StringValue{StringValue: "production"},
						}},
					},
				},
				ScopeLogs: []*logspb.ScopeLogs{
					{
						LogRecords: []*logspb.LogRecord{
							{
								// SeverityText intentionally empty — enrichment should infer from body.
								SeverityText: "",
								Body: &commonpb.AnyValue{
									Value: &commonpb.AnyValue_StringValue{
										StringValue: "[RedisCacheHandler] Redis connection error: ECONNREFUSED",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	b, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// TestHandleLogs_PodLogEnrichment is the integration smoke test defined by task 6.
// It verifies that, with enrichment enabled, a pod log export with no service.name
// and empty severity is stored under the enriched service name and inferred severity.
func TestHandleLogs_PodLogEnrichment(t *testing.T) {
	r, store := newTestReceiverWithEnrichment(t)

	payload := minimalPodLogsProto(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/logs", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/x-protobuf")

	w := httptest.NewRecorder()
	r.handleLogs(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", res.StatusCode)
	}

	// Introspect stored log metadata.
	logs, err := store.ListLogs(context.Background(), "")
	if err != nil {
		t.Fatalf("ListLogs: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("expected at least one log metadata entry")
	}

	// Find the entry with service nova-powerplay-app.
	found := false
	for _, lm := range logs {
		if _, ok := lm.Services["nova-powerplay-app"]; ok {
			found = true
			if lm.Severity != "ERROR" {
				t.Fatalf("expected inferred severity ERROR, got %q", lm.Severity)
			}
		}
	}
	if !found {
		t.Fatalf("expected service nova-powerplay-app in stored logs; got entries: %v",
			func() []string {
				var ss []string
				for _, lm := range logs {
					for svc := range lm.Services {
						ss = append(ss, svc)
					}
				}
				return ss
			}(),
		)
	}
}
