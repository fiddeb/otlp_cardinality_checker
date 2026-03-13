// Package receiver implements OTLP HTTP and gRPC endpoints.
package receiver

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/fidde/otlp_cardinality_checker/internal/analyzer"
	"github.com/fidde/otlp_cardinality_checker/internal/patterns"
	"github.com/fidde/otlp_cardinality_checker/internal/storage"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Log level configuration
var verboseLogging = strings.ToLower(os.Getenv("VERBOSE_LOGGING")) == "true"

// bodyBufPool reuses bytes.Buffer allocations for reading HTTP request bodies.
// Each OTLP batch is typically 50–200 KB; pooling avoids allocating a fresh
// backing array on every request (which was the #1 allocation source).
var bodyBufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// readAndDecompressBody reads the full request body and decompresses it if
// needed. Decompression is triggered by an explicit "Content-Encoding: gzip"
// header OR by the gzip magic bytes (0x1f 0x8b) at the start of the body.
// The latter handles collectors that compress without advertising it, which
// is the root cause of "cannot parse invalid wire-format data" errors.
//
// The caller MUST call release() after consuming the returned slice (i.e.
// after proto.Unmarshal / protojson.Unmarshal returns) to return the buffer
// to the pool. Holding the slice after release() is a use-after-free.
func readAndDecompressBody(req *http.Request) (data []byte, release func(), err error) {
	buf := bodyBufPool.Get().(*bytes.Buffer)
	buf.Reset()

	if _, err = buf.ReadFrom(req.Body); err != nil {
		bodyBufPool.Put(buf)
		return nil, nil, fmt.Errorf("read body: %w", err)
	}

	raw := buf.Bytes()
	isGzipHeader := strings.EqualFold(req.Header.Get("Content-Encoding"), "gzip")
	hasMagicBytes := len(raw) >= 2 && raw[0] == 0x1f && raw[1] == 0x8b

	if isGzipHeader || hasMagicBytes {
		gzr, gzErr := gzip.NewReader(bytes.NewReader(raw))
		if gzErr != nil {
			bodyBufPool.Put(buf)
			return nil, nil, fmt.Errorf("gzip reader: %w", gzErr)
		}
		defer gzr.Close()

		// Decompress into a second pooled buffer.
		decompBuf := bodyBufPool.Get().(*bytes.Buffer)
		decompBuf.Reset()
		if _, err = decompBuf.ReadFrom(gzr); err != nil {
			bodyBufPool.Put(buf)
			bodyBufPool.Put(decompBuf)
			return nil, nil, fmt.Errorf("decompress: %w", err)
		}
		bodyBufPool.Put(buf) // raw (compressed) buffer no longer needed
		return decompBuf.Bytes(), func() { bodyBufPool.Put(decompBuf) }, nil
	}

	return raw, func() { bodyBufPool.Put(buf) }, nil
}

// sanitizeUTF8 replaces invalid UTF-8 byte sequences with the Unicode
// replacement character before JSON parsing. protojson strictly validates UTF-8
// in string fields; Kafka-sourced metrics may embed binary data in attribute
// values, which OCC never stores. Allocates a copy only when invalid bytes are
// found.
func sanitizeUTF8(b []byte) []byte {
	if utf8.Valid(b) {
		return b
	}
	return []byte(strings.ToValidUTF8(string(b), string(utf8.RuneError)))
}

// isJSONContentType returns true when the Content-Type header indicates
// JSON-encoded OTLP (e.g. "application/json").
func isJSONContentType(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "json")
}

// HTTPReceiver handles OTLP HTTP requests.
type HTTPReceiver struct {
	store           storage.Storage
	metricsAnalyzer *analyzer.MetricsAnalyzer
	tracesAnalyzer  *analyzer.TracesAnalyzer
	logsAnalyzer    *analyzer.LogsAnalyzer
	server          *http.Server
}

// NewHTTPReceiver creates a new HTTP receiver.
func NewHTTPReceiver(addr string, store storage.Storage) *HTTPReceiver {
	// Load patterns from config
	pats, err := patterns.LoadPatterns("config/patterns.yaml")
	if err != nil {
		log.Printf("Warning: Failed to load patterns: %v", err)
		pats = nil
	}
	
	// Create logs analyzer based on store configuration
	var logsAnalyzer *analyzer.LogsAnalyzer
	if store.UseAutoTemplate() {
		logsAnalyzer = analyzer.NewLogsAnalyzerWithAutoTemplateAndCatalog(store.AutoTemplateCfg(), pats, store)
	} else {
		logsAnalyzer = analyzer.NewLogsAnalyzerWithCatalog(store)
	}
	if store.PodLogEnrichment() {
		logsAnalyzer.SetPodLogEnrichment(true, store.PodLogServiceLabels())
	}
	
	r := &HTTPReceiver{
		store:           store,
		metricsAnalyzer: analyzer.NewMetricsAnalyzerWithCatalog(store),
		tracesAnalyzer:  analyzer.NewTracesAnalyzerWithCatalog(store),
		logsAnalyzer:    logsAnalyzer,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/metrics", r.handleMetrics)
	mux.HandleFunc("/v1/traces", r.handleTraces)
	mux.HandleFunc("/v1/logs", r.handleLogs)
	mux.HandleFunc("/health", r.handleHealth)

	r.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return r
}

// Start starts the HTTP server.
func (r *HTTPReceiver) Start() error {
	return r.server.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server.
func (r *HTTPReceiver) Shutdown(ctx context.Context) error {
	return r.server.Shutdown(ctx)
}

// handleMetrics handles OTLP metrics export requests.
func (r *HTTPReceiver) handleMetrics(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := req.Context()

	body, releaseBody, err := readAndDecompressBody(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read body: %v", err), http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Parse request based on Content-Type
	var exportReq colmetricspb.ExportMetricsServiceRequest
	contentType := req.Header.Get("Content-Type")
	
	// Log for debugging (only if verbose logging is enabled)
	if verboseLogging {
		fmt.Printf("Received metrics request: Content-Type=%s, Content-Encoding=%s, Body length=%d\n", 
			contentType, req.Header.Get("Content-Encoding"), len(body))
	}

	unmarshaler := protojson.UnmarshalOptions{DiscardUnknown: true}

	if isJSONContentType(contentType) {
		// Collector sent JSON (e.g. encoding: json). Sanitize invalid UTF-8
		// that may be embedded in Kafka-sourced attribute values before parsing.
		if jsonErr := unmarshaler.Unmarshal(sanitizeUTF8(body), &exportReq); jsonErr != nil {
			// Fall back to protobuf in case Content-Type was set incorrectly.
			if protoErr := proto.Unmarshal(body, &exportReq); protoErr != nil {
				releaseBody()
				log.Printf("Failed to parse metrics request: json error: %v, protobuf error: %v", jsonErr, protoErr)
				http.Error(w, fmt.Sprintf("Failed to parse request: %v", jsonErr), http.StatusBadRequest)
				return
			}
			if verboseLogging {
				fmt.Println("Parsed metrics as protobuf (Content-Type mismatch)")
			}
		} else if verboseLogging {
			fmt.Println("Parsed metrics as JSON")
		}
	} else {
		// Default: try protobuf first, fall back to JSON.
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			if jsonErr := unmarshaler.Unmarshal(sanitizeUTF8(body), &exportReq); jsonErr != nil {
				releaseBody()
				log.Printf("Failed to parse metrics request: protobuf error: %v, json error: %v", err, jsonErr)
				if verboseLogging {
					fmt.Printf("Body preview: %s\n", string(body[:min(len(body), 100)]))
				}
				http.Error(w, fmt.Sprintf("Failed to parse request: protobuf error: %v, json error: %v", err, jsonErr), http.StatusBadRequest)
				return
			}
			if verboseLogging {
				fmt.Println("Parsed metrics as JSON")
			}
		} else if verboseLogging {
			fmt.Println("Parsed metrics as protobuf")
		}
	}
	releaseBody()

	// Analyze metrics
	metricsMetadata, err := r.metricsAnalyzer.AnalyzeWithContext(ctx, &exportReq)
	if err != nil {
		log.Printf("Metrics analysis error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to analyze metrics: %v", err), http.StatusInternalServerError)
		return
	}

	if verboseLogging {
		fmt.Printf("Successfully analyzed %d metrics\n", len(metricsMetadata))
	}

	// Store metadata
	for _, metadata := range metricsMetadata {
		if err := r.store.StoreMetric(ctx, metadata); err != nil {
			log.Printf("Storage error: %v\n", err)
			http.Error(w, fmt.Sprintf("Failed to store metric: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Return success response (always protobuf for OTLP)
	resp := &colmetricspb.ExportMetricsServiceResponse{}
	r.writeResponse(w, resp)
}

// handleTraces handles OTLP traces export requests.
func (r *HTTPReceiver) handleTraces(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := req.Context()

	body, releaseBody, err := readAndDecompressBody(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read body: %v", err), http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var exportReq coltracepb.ExportTraceServiceRequest
	traceUnmarshaler := protojson.UnmarshalOptions{DiscardUnknown: true}
	traceContentType := req.Header.Get("Content-Type")

	if isJSONContentType(traceContentType) {
		if jsonErr := traceUnmarshaler.Unmarshal(sanitizeUTF8(body), &exportReq); jsonErr != nil {
			if protoErr := proto.Unmarshal(body, &exportReq); protoErr != nil {
				releaseBody()
				log.Printf("Failed to parse traces request: json error: %v, protobuf error: %v", jsonErr, protoErr)
				http.Error(w, fmt.Sprintf("Failed to parse request: %v", jsonErr), http.StatusBadRequest)
				return
			}
		} else if verboseLogging {
			fmt.Println("Parsed traces as JSON")
		}
	} else {
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			if jsonErr := traceUnmarshaler.Unmarshal(sanitizeUTF8(body), &exportReq); jsonErr != nil {
				releaseBody()
				log.Printf("Failed to parse traces request: protobuf error: %v, json error: %v", err, jsonErr)
				if verboseLogging {
					fmt.Printf("Body preview: %s\n", string(body[:min(len(body), 100)]))
				}
				http.Error(w, fmt.Sprintf("Failed to parse request: protobuf error: %v, json error: %v", err, jsonErr), http.StatusBadRequest)
				return
			}
			if verboseLogging {
				fmt.Println("Parsed traces as JSON")
			}
		} else if verboseLogging {
			fmt.Println("Parsed traces as protobuf")
		}
	}
	releaseBody()

	// Analyze traces
	spansMetadata, err := r.tracesAnalyzer.AnalyzeWithContext(ctx, &exportReq)
	if err != nil {
		log.Printf("Trace analysis error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to analyze traces: %v", err), http.StatusInternalServerError)
		return
	}

	if verboseLogging {
		fmt.Printf("Successfully analyzed %d spans\n", len(spansMetadata))
	}

	// Store metadata
	for _, metadata := range spansMetadata {
		if err := r.store.StoreSpan(ctx, metadata); err != nil {
			log.Printf("Span storage error: %v\n", err)
			http.Error(w, fmt.Sprintf("Failed to store span: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Return success response (always protobuf for OTLP)
	resp := &coltracepb.ExportTraceServiceResponse{}
	r.writeResponse(w, resp)
}

// handleLogs handles OTLP logs export requests.
func (r *HTTPReceiver) handleLogs(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := req.Context()

	body, releaseBody, err := readAndDecompressBody(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read body: %v", err), http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var exportReq collogspb.ExportLogsServiceRequest
	logsUnmarshaler := protojson.UnmarshalOptions{DiscardUnknown: true}
	logsContentType := req.Header.Get("Content-Type")

	if isJSONContentType(logsContentType) {
		if jsonErr := logsUnmarshaler.Unmarshal(sanitizeUTF8(body), &exportReq); jsonErr != nil {
			if protoErr := proto.Unmarshal(body, &exportReq); protoErr != nil {
				releaseBody()
				log.Printf("Failed to parse logs request: json error: %v, protobuf error: %v", jsonErr, protoErr)
				http.Error(w, fmt.Sprintf("Failed to parse request: %v", jsonErr), http.StatusBadRequest)
				return
			}
		} else if verboseLogging {
			fmt.Println("Parsed logs as JSON")
		}
	} else {
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			if jsonErr := logsUnmarshaler.Unmarshal(sanitizeUTF8(body), &exportReq); jsonErr != nil {
				releaseBody()
				log.Printf("Failed to parse logs request: protobuf error: %v, json error: %v", err, jsonErr)
				if verboseLogging {
					fmt.Printf("Body preview: %s\n", string(body[:min(len(body), 100)]))
				}
				http.Error(w, fmt.Sprintf("Failed to parse request: protobuf error: %v, json error: %v", err, jsonErr), http.StatusBadRequest)
				return
			}
			if verboseLogging {
				fmt.Println("Parsed logs as JSON")
			}
		} else if verboseLogging {
			fmt.Println("Parsed logs as protobuf")
		}
	}
	releaseBody()

	// Analyze logs
	logsMetadata, err := r.logsAnalyzer.AnalyzeWithContext(ctx, &exportReq)
	if err != nil {
		log.Printf("Log analysis error: %v\n", err)
		http.Error(w, fmt.Sprintf("Failed to analyze logs: %v", err), http.StatusInternalServerError)
		return
	}

	if verboseLogging {
		fmt.Printf("Successfully analyzed %d log severities\n", len(logsMetadata))
	}

	// Store metadata
	for _, metadata := range logsMetadata {
		if err := r.store.StoreLog(ctx, metadata); err != nil {
			log.Printf("Log storage error: %v\n", err)
			http.Error(w, fmt.Sprintf("Failed to store log: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Return success response (always protobuf for OTLP)
	resp := &collogspb.ExportLogsServiceResponse{}
	r.writeResponse(w, resp)
}

// handleHealth handles health check requests.
func (r *HTTPReceiver) handleHealth(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// writeResponse writes a protobuf response.
// OTLP always uses protobuf for responses.
func (r *HTTPReceiver) writeResponse(w http.ResponseWriter, resp proto.Message) {
	respBytes, err := proto.Marshal(resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-protobuf")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, bytes.NewReader(respBytes))
}
