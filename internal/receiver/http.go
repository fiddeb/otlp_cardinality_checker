// Package receiver implements OTLP HTTP and gRPC endpoints.
package receiver

import (
	"bytes"
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

	// Read request body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read body: %v", err), http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Parse request based on Content-Type
	var exportReq colmetricspb.ExportMetricsServiceRequest
	contentType := req.Header.Get("Content-Type")

	switch contentType {
	case "application/json":
		if err := protojson.Unmarshal(body, &exportReq); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse JSON: %v", err), http.StatusBadRequest)
			return
		}
	case "application/x-protobuf", "":
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse protobuf: %v", err), http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "Unsupported Content-Type", http.StatusUnsupportedMediaType)
		return
	}

	// Analyze metrics
	metricsMetadata, err := r.metricsAnalyzer.Analyze(&exportReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to analyze metrics: %v", err), http.StatusInternalServerError)
		return
	}

	// Store metadata
	for _, metadata := range metricsMetadata {
		if err := r.store.StoreMetric(ctx, metadata); err != nil {
			http.Error(w, fmt.Sprintf("Failed to store metric: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Return success response
	resp := &colmetricspb.ExportMetricsServiceResponse{}
	r.writeResponse(w, resp, contentType)
}

// handleTraces handles OTLP traces export requests.
func (r *HTTPReceiver) handleTraces(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := req.Context()

	// Read request body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read body: %v", err), http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Parse request based on Content-Type
	var exportReq coltracepb.ExportTraceServiceRequest
	contentType := req.Header.Get("Content-Type")

	switch contentType {
	case "application/json":
		if err := protojson.Unmarshal(body, &exportReq); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse JSON: %v", err), http.StatusBadRequest)
			return
		}
	case "application/x-protobuf", "":
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse protobuf: %v", err), http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "Unsupported Content-Type", http.StatusUnsupportedMediaType)
		return
	}

	// Analyze traces
	spansMetadata, err := r.tracesAnalyzer.Analyze(&exportReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to analyze traces: %v", err), http.StatusInternalServerError)
		return
	}

	// Store metadata
	for _, metadata := range spansMetadata {
		if err := r.store.StoreSpan(ctx, metadata); err != nil {
			http.Error(w, fmt.Sprintf("Failed to store span: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Return success response
	resp := &coltracepb.ExportTraceServiceResponse{}
	r.writeResponse(w, resp, contentType)
}

// handleLogs handles OTLP logs export requests.
func (r *HTTPReceiver) handleLogs(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := req.Context()

	// Read request body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read body: %v", err), http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Parse request based on Content-Type
	var exportReq collogspb.ExportLogsServiceRequest
	contentType := req.Header.Get("Content-Type")

	switch contentType {
	case "application/json":
		if err := protojson.Unmarshal(body, &exportReq); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse JSON: %v", err), http.StatusBadRequest)
			return
		}
	case "application/x-protobuf", "":
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse protobuf: %v", err), http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "Unsupported Content-Type", http.StatusUnsupportedMediaType)
		return
	}

	// Analyze logs
	logsMetadata, err := r.logsAnalyzer.Analyze(&exportReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to analyze logs: %v", err), http.StatusInternalServerError)
		return
	}

	// Store metadata
	for _, metadata := range logsMetadata {
		if err := r.store.StoreLog(ctx, metadata); err != nil {
			http.Error(w, fmt.Sprintf("Failed to store log: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Return success response
	resp := &collogspb.ExportLogsServiceResponse{}
	r.writeResponse(w, resp, contentType)
}

// handleHealth handles health check requests.
func (r *HTTPReceiver) handleHealth(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// writeResponse writes a protobuf response based on Content-Type.
func (r *HTTPReceiver) writeResponse(w http.ResponseWriter, resp proto.Message, contentType string) {
	var respBytes []byte
	var err error

	switch contentType {
	case "application/json":
		respBytes, err = protojson.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
	default:
		respBytes, err = proto.Marshal(resp)
		w.Header().Set("Content-Type", "application/x-protobuf")
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal response: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	io.Copy(w, bytes.NewReader(respBytes))
}
