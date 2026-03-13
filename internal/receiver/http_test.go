package receiver

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/protobuf/proto"

	"github.com/fidde/otlp_cardinality_checker/internal/storage"
)

// gzipBytes compresses src with gzip and returns the compressed bytes.
func gzipBytes(t *testing.T, src []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(src); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

// minimalMetricsProto returns a serialised ExportMetricsServiceRequest with
// one gauge metric, suitable for round-trip parse tests.
func minimalMetricsProto(t *testing.T) []byte {
	t.Helper()
	req := &colmetricspb.ExportMetricsServiceRequest{
		ResourceMetrics: []*metricspb.ResourceMetrics{
			{
				Resource: &resourcepb.Resource{
					Attributes: []*commonpb.KeyValue{
						{Key: "service.name", Value: &commonpb.AnyValue{
							Value: &commonpb.AnyValue_StringValue{StringValue: "test-service"},
						}},
					},
				},
				ScopeMetrics: []*metricspb.ScopeMetrics{
					{
						Metrics: []*metricspb.Metric{
							{
								Name: "test.counter",
								Data: &metricspb.Metric_Gauge{
									Gauge: &metricspb.Gauge{
										DataPoints: []*metricspb.NumberDataPoint{
											{
												Attributes: []*commonpb.KeyValue{
													{Key: "env", Value: &commonpb.AnyValue{
														Value: &commonpb.AnyValue_StringValue{StringValue: "prod"},
													}},
												},
											},
										},
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

// --- readAndDecompressBody unit tests ---

func TestReadAndDecompressBody_PlainProtobuf(t *testing.T) {
	payload := minimalMetricsProto(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/metrics", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/x-protobuf")

	got, release, err := readAndDecompressBody(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer release()
	if !bytes.Equal(got, payload) {
		t.Fatalf("body mismatch: got %d bytes, want %d bytes", len(got), len(payload))
	}
}

func TestReadAndDecompressBody_GzipWithHeader(t *testing.T) {
	payload := minimalMetricsProto(t)
	compressed := gzipBytes(t, payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/metrics", bytes.NewReader(compressed))
	req.Header.Set("Content-Encoding", "gzip")

	got, release, err := readAndDecompressBody(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer release()
	if !bytes.Equal(got, payload) {
		t.Fatalf("decompressed body mismatch: got %d bytes, want %d bytes", len(got), len(payload))
	}
}

// TestReadAndDecompressBody_GzipWithoutHeader is the regression test for the
// reported bug: the OTel Collector sends gzip-compressed protobuf but omits
// the Content-Encoding header. The magic-byte detection must handle this.
func TestReadAndDecompressBody_GzipWithoutHeader(t *testing.T) {
	payload := minimalMetricsProto(t)
	compressed := gzipBytes(t, payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/metrics", bytes.NewReader(compressed))
	// Intentionally omit Content-Encoding header.

	got, release, err := readAndDecompressBody(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer release()
	if !bytes.Equal(got, payload) {
		t.Fatalf("decompressed body mismatch: got %d bytes, want %d bytes", len(got), len(payload))
	}
}

func TestReadAndDecompressBody_EmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/metrics", bytes.NewReader(nil))
	got, release, err := readAndDecompressBody(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer release()
	if len(got) != 0 {
		t.Fatalf("expected empty body, got %d bytes", len(got))
	}
}

// --- handleMetrics integration smoke tests ---

func newTestReceiver(t *testing.T) *HTTPReceiver {
	t.Helper()
	store := storage.NewStorage(storage.DefaultConfig())
	r := NewHTTPReceiver(":0", store)
	return r
}

func TestHandleMetrics_GzipNoHeader(t *testing.T) {
	r := newTestReceiver(t)

	payload := minimalMetricsProto(t)
	compressed := gzipBytes(t, payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/metrics", bytes.NewReader(compressed))
	// Content-Type set; Content-Encoding intentionally absent.
	req.Header.Set("Content-Type", "application/x-protobuf")

	w := httptest.NewRecorder()
	r.handleMetrics(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("expected 200, got %d: %s", res.StatusCode, body)
	}
}

func TestHandleMetrics_GzipWithHeader(t *testing.T) {
	r := newTestReceiver(t)

	payload := minimalMetricsProto(t)
	compressed := gzipBytes(t, payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/metrics", bytes.NewReader(compressed))
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "gzip")

	w := httptest.NewRecorder()
	r.handleMetrics(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("expected 200, got %d: %s", res.StatusCode, body)
	}
}

func TestHandleMetrics_PlainProtobuf(t *testing.T) {
	r := newTestReceiver(t)

	payload := minimalMetricsProto(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/metrics", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/x-protobuf")

	w := httptest.NewRecorder()
	r.handleMetrics(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("expected 200, got %d: %s", res.StatusCode, body)
	}
}

// TestHandleMetrics_JSONInvalidUTF8 is the regression test for the production
// bug: the OTel Collector is configured with encoding:json and forwards
// Kafka-sourced metrics whose attribute values contain invalid UTF-8 bytes.
// OCC must accept and process the request (keys are stored, values are
// discarded) rather than returning 400.
func TestHandleMetrics_JSONInvalidUTF8(t *testing.T) {
	r := newTestReceiver(t)

	// Build a valid JSON OTLP payload that contains an attribute value with
	// invalid UTF-8 bytes (0xC0 0xAF is an overlong encoding, always invalid).
	// We construct the JSON manually so the invalid bytes survive intact.
	rawJSON := []byte(`{"resourceMetrics":[{"resource":{"attributes":[` +
		`{"key":"service.name","value":{"stringValue":"test-svc"}}]},"scopeMetrics":[{"metrics":[{"name":"test.counter","gauge":{"dataPoints":[{"attributes":[` +
		`{"key":"env","value":{"stringValue":"` + "prod\xc0\xaf" + `"}}],"asDouble":1}]}}]}]}]}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/metrics", bytes.NewReader(rawJSON))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.handleMetrics(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("expected 200 for JSON with invalid UTF-8, got %d: %s", res.StatusCode, body)
	}
}
