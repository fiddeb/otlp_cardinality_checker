package receiver

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/protobuf/proto"
)

// gzipCompress compresses src and returns the compressed bytes.
func gzipCompress(t *testing.T, src []byte) []byte {
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

// metricsProtoWithPadding returns a serialised ExportMetricsServiceRequest
// padded to approximately targetSize bytes via a long attribute value.
func metricsProtoWithPadding(t *testing.T, targetSize int) []byte {
	t.Helper()
	req := &colmetricspb.ExportMetricsServiceRequest{
		ResourceMetrics: []*metricspb.ResourceMetrics{
			{
				Resource: &resourcepb.Resource{
					Attributes: []*commonpb.KeyValue{
						{
							Key: "padding",
							Value: &commonpb.AnyValue{
								Value: &commonpb.AnyValue_StringValue{
									StringValue: strings.Repeat("x", targetSize),
								},
							},
						},
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

// TestOversizedContentLengthRejected verifies that a request advertising a
// Content-Length above 32 MiB is rejected with HTTP 413 before any body is read.
func TestOversizedContentLengthRejected(t *testing.T) {
	r := newTestReceiver(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/metrics", http.NoBody)
	req.ContentLength = (32 << 20) + 1 // 32 MiB + 1 byte
	req.Header.Set("Content-Type", "application/x-protobuf")

	w := httptest.NewRecorder()
	r.handleMetrics(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", w.Code)
	}
}

// TestOversizedStreamingBodyRejected verifies that a streaming body (no
// Content-Length) exceeding 32 MiB is rejected with HTTP 413 via MaxBytesReader.
func TestOversizedStreamingBodyRejected(t *testing.T) {
	r := newTestReceiver(t)

	// Allocate 32 MiB + 1 byte of zero data (non-gzip, won't hit magic bytes).
	body := make([]byte, (32<<20)+1)
	body[0] = 0x00 // ensure no gzip magic bytes
	body[1] = 0x00

	req := httptest.NewRequest(http.MethodPost, "/v1/metrics", bytes.NewReader(body))
	req.ContentLength = -1 // suppress Content-Length to test the streaming path
	req.Header.Set("Content-Type", "application/x-protobuf")

	w := httptest.NewRecorder()
	r.handleMetrics(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", w.Code)
	}
}

// TestGzipBombRejected verifies that a gzip-encoded body whose decompressed
// size exceeds 32 MiB is rejected with HTTP 413.
func TestGzipBombRejected(t *testing.T) {
	r := newTestReceiver(t)

	// Compress 33 MiB of zeros; gzip compresses zeros to a tiny payload.
	bomb := gzipCompress(t, make([]byte, (33<<20)+1))

	req := httptest.NewRequest(http.MethodPost, "/v1/metrics", bytes.NewReader(bomb))
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "gzip")

	w := httptest.NewRecorder()
	r.handleMetrics(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for gzip bomb, got %d", w.Code)
	}
}

// TestNormalBodyAccepted verifies that a valid protobuf payload of ~1 MiB is
// accepted with HTTP 200.
func TestNormalBodyAccepted(t *testing.T) {
	r := newTestReceiver(t)

	body := metricsProtoWithPadding(t, 1<<20) // ~1 MiB

	req := httptest.NewRequest(http.MethodPost, "/v1/metrics", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")

	w := httptest.NewRecorder()
	r.handleMetrics(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("expected 200, got %d: %s", res.StatusCode, b)
	}
}
