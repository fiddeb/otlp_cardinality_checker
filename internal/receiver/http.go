// Package receiver implements OTLP HTTP and gRPC endpoints.
package receiver

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/fidde/otlp_cardinality_checker/internal/analyzer"
	"github.com/fidde/otlp_cardinality_checker/internal/storage/memory"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// decompressGzip decompresses gzip-encoded data
func decompressGzip(r io.Reader) (io.ReadCloser, error) {
	return gzip.NewReader(r)
}

// HTTPReceiver handles OTLP HTTP requests.
type HTTPReceiver struct {
	store           *memory.Store
	metricsAnalyzer *analyzer.MetricsAnalyzer
	tracesAnalyzer  *analyzer.TracesAnalyzer
	logsAnalyzer    *analyzer.LogsAnalyzer
	server          *http.Server
}

// NewHTTPReceiver creates a new HTTP receiver.
func NewHTTPReceiver(addr string, store *memory.Store) *HTTPReceiver {
	r := &HTTPReceiver{
		store:           store,
		metricsAnalyzer: analyzer.NewMetricsAnalyzer(),
		tracesAnalyzer:  analyzer.NewTracesAnalyzer(),
		logsAnalyzer:    analyzer.NewLogsAnalyzer(),
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

	// Handle compression
	reader := req.Body
	if req.Header.Get("Content-Encoding") == "gzip" {
		var err error
		reader, err = decompressGzip(req.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to decompress: %v", err), http.StatusBadRequest)
			return
		}
		defer reader.Close()
	}

	// Read request body
	body, err := io.ReadAll(reader)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read body: %v", err), http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Parse request based on Content-Type
	var exportReq colmetricspb.ExportMetricsServiceRequest
	contentType := req.Header.Get("Content-Type")
	
	// Log for debugging
	fmt.Printf("Received metrics request: Content-Type=%s, Content-Encoding=%s, Body length=%d\n", 
		contentType, req.Header.Get("Content-Encoding"), len(body))

	// Always try protobuf first (default for OTLP), then fallback to JSON
	if err := proto.Unmarshal(body, &exportReq); err != nil {
		// If protobuf fails, try JSON
		unmarshaler := protojson.UnmarshalOptions{
			DiscardUnknown: true,
		}
		if jsonErr := unmarshaler.Unmarshal(body, &exportReq); jsonErr != nil {
			fmt.Printf("Failed to parse as both protobuf and JSON\n")
			fmt.Printf("Protobuf error: %v\n", err)
			fmt.Printf("JSON error: %v\n", jsonErr)
			fmt.Printf("Body preview: %s\n", string(body[:min(len(body), 100)]))
			http.Error(w, fmt.Sprintf("Failed to parse request: protobuf error: %v, json error: %v", err, jsonErr), http.StatusBadRequest)
			return
		}
		fmt.Println("Parsed as JSON")
	} else {
		fmt.Println("Parsed as protobuf")
	}

	// Analyze metrics
	metricsMetadata, err := r.metricsAnalyzer.Analyze(&exportReq)
	if err != nil {
		fmt.Printf("Analysis error: %v\n", err)
		http.Error(w, fmt.Sprintf("Failed to analyze metrics: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Printf("Successfully analyzed %d metrics\n", len(metricsMetadata))

	// Store metadata
	for _, metadata := range metricsMetadata {
		if err := r.store.StoreMetric(ctx, metadata); err != nil {
			fmt.Printf("Storage error: %v\n", err)
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

	// Handle compression
	reader := req.Body
	if req.Header.Get("Content-Encoding") == "gzip" {
		var err error
		reader, err = decompressGzip(req.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to decompress: %v", err), http.StatusBadRequest)
			return
		}
		defer reader.Close()
	}

	// Read request body
	body, err := io.ReadAll(reader)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read body: %v", err), http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Parse request - try protobuf first, then JSON
	var exportReq coltracepb.ExportTraceServiceRequest
	if err := proto.Unmarshal(body, &exportReq); err != nil {
		unmarshaler := protojson.UnmarshalOptions{DiscardUnknown: true}
		if jsonErr := unmarshaler.Unmarshal(body, &exportReq); jsonErr != nil {
			fmt.Printf("Failed to parse traces as both protobuf and JSON\n")
			fmt.Printf("Protobuf error: %v\n", err)
			fmt.Printf("JSON error: %v\n", jsonErr)
			fmt.Printf("Body preview: %s\n", string(body[:min(len(body), 100)]))
			http.Error(w, fmt.Sprintf("Failed to parse request: protobuf error: %v, json error: %v", err, jsonErr), http.StatusBadRequest)
			return
		}
		fmt.Println("Parsed traces as JSON")
	} else {
		fmt.Println("Parsed traces as protobuf")
	}

	// Analyze traces
	spansMetadata, err := r.tracesAnalyzer.Analyze(&exportReq)
	if err != nil {
		fmt.Printf("Trace analysis error: %v\n", err)
		http.Error(w, fmt.Sprintf("Failed to analyze traces: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Printf("Successfully analyzed %d spans\n", len(spansMetadata))

	// Store metadata
	for _, metadata := range spansMetadata {
		if err := r.store.StoreSpan(ctx, metadata); err != nil {
			fmt.Printf("Span storage error: %v\n", err)
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

	// Handle compression
	reader := req.Body
	if req.Header.Get("Content-Encoding") == "gzip" {
		var err error
		reader, err = decompressGzip(req.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to decompress: %v", err), http.StatusBadRequest)
			return
		}
		defer reader.Close()
	}

	// Read request body
	body, err := io.ReadAll(reader)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read body: %v", err), http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Parse request - try protobuf first, then JSON
	var exportReq collogspb.ExportLogsServiceRequest
	if err := proto.Unmarshal(body, &exportReq); err != nil {
		unmarshaler := protojson.UnmarshalOptions{DiscardUnknown: true}
		if jsonErr := unmarshaler.Unmarshal(body, &exportReq); jsonErr != nil {
			fmt.Printf("Failed to parse logs as both protobuf and JSON\n")
			fmt.Printf("Protobuf error: %v\n", err)
			fmt.Printf("JSON error: %v\n", jsonErr)
			fmt.Printf("Body preview: %s\n", string(body[:min(len(body), 100)]))
			http.Error(w, fmt.Sprintf("Failed to parse request: protobuf error: %v, json error: %v", err, jsonErr), http.StatusBadRequest)
			return
		}
		fmt.Println("Parsed logs as JSON")
	} else {
		fmt.Println("Parsed logs as protobuf")
	}

	// Analyze logs
	logsMetadata, err := r.logsAnalyzer.Analyze(&exportReq)
	if err != nil {
		fmt.Printf("Log analysis error: %v\n", err)
		http.Error(w, fmt.Sprintf("Failed to analyze logs: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Printf("Successfully analyzed %d log severities\n", len(logsMetadata))

	// Store metadata
	for _, metadata := range logsMetadata {
		if err := r.store.StoreLog(ctx, metadata); err != nil {
			fmt.Printf("Log storage error: %v\n", err)
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
