// Package receiver implements OTLP HTTP and gRPC endpoints.
package receiver

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/fidde/otlp_cardinality_checker/internal/analyzer"
	"github.com/fidde/otlp_cardinality_checker/internal/storage/memory"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// GRPCReceiver handles OTLP gRPC requests.
type GRPCReceiver struct {
	colmetricspb.UnimplementedMetricsServiceServer
	store           *memory.Store
	metricsAnalyzer *analyzer.MetricsAnalyzer
	tracesAnalyzer  *analyzer.TracesAnalyzer
	logsAnalyzer    *analyzer.LogsAnalyzer
	server          *grpc.Server
	listener        net.Listener
	addr            string
}

// NewGRPCReceiver creates a new gRPC receiver.
func NewGRPCReceiver(addr string, store *memory.Store) *GRPCReceiver {
	return &GRPCReceiver{
		store:           store,
		metricsAnalyzer: analyzer.NewMetricsAnalyzer(),
		tracesAnalyzer:  analyzer.NewTracesAnalyzer(),
		logsAnalyzer:    analyzer.NewLogsAnalyzer(),
		addr:            addr,
	}
}

// Start starts the gRPC server.
func (r *GRPCReceiver) Start() error {
	lis, err := net.Listen("tcp", r.addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	r.listener = lis

	r.server = grpc.NewServer()

	// Register OTLP services with wrapper types to avoid method name conflicts
	colmetricspb.RegisterMetricsServiceServer(r.server, r)
	coltracepb.RegisterTraceServiceServer(r.server, &traceService{
		UnimplementedTraceServiceServer: coltracepb.UnimplementedTraceServiceServer{},
		GRPCReceiver:                    r,
	})
	collogspb.RegisterLogsServiceServer(r.server, &logsService{
		UnimplementedLogsServiceServer: collogspb.UnimplementedLogsServiceServer{},
		GRPCReceiver:                   r,
	})

	// Register reflection service for debugging with grpcurl
	reflection.Register(r.server)

	log.Printf("gRPC server listening on %s", r.addr)
	return r.server.Serve(lis)
}

// Shutdown gracefully shuts down the gRPC server.
func (r *GRPCReceiver) Shutdown(ctx context.Context) error {
	if r.server != nil {
		r.server.GracefulStop()
	}
	return nil
}

// MetricsService implementation

// Export implements the MetricsService Export RPC.
func (r *GRPCReceiver) Export(ctx context.Context, req *colmetricspb.ExportMetricsServiceRequest) (*colmetricspb.ExportMetricsServiceResponse, error) {
	// Analyze metrics metadata
	metadata, err := r.metricsAnalyzer.Analyze(req)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze metrics: %w", err)
	}

	// Store metadata
	for _, m := range metadata {
		r.store.StoreMetric(ctx, m)
	}

	// Return success response
	return &colmetricspb.ExportMetricsServiceResponse{
		PartialSuccess: &colmetricspb.ExportMetricsPartialSuccess{
			RejectedDataPoints: 0,
		},
	}, nil
}

// TraceService implementation - uses separate type to avoid method name conflicts
type traceService struct {
	coltracepb.UnimplementedTraceServiceServer
	*GRPCReceiver
}

// Export implements the TraceService Export RPC.
func (s *traceService) Export(ctx context.Context, req *coltracepb.ExportTraceServiceRequest) (*coltracepb.ExportTraceServiceResponse, error) {
	// Analyze traces metadata
	metadata, err := s.tracesAnalyzer.Analyze(req)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze traces: %w", err)
	}

	// Store metadata
	for _, m := range metadata {
		s.store.StoreSpan(ctx, m)
	}

	// Return success response
	return &coltracepb.ExportTraceServiceResponse{
		PartialSuccess: &coltracepb.ExportTracePartialSuccess{
			RejectedSpans: 0,
		},
	}, nil
}

// LogsService implementation - uses separate type to avoid method name conflicts
type logsService struct {
	collogspb.UnimplementedLogsServiceServer
	*GRPCReceiver
}

// Export implements the LogsService Export RPC.
func (s *logsService) Export(ctx context.Context, req *collogspb.ExportLogsServiceRequest) (*collogspb.ExportLogsServiceResponse, error) {
	// Analyze logs metadata
	metadata, err := s.logsAnalyzer.Analyze(req)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze logs: %w", err)
	}

	// Store metadata
	for _, m := range metadata {
		s.store.StoreLog(ctx, m)
	}

	// Return success response
	return &collogspb.ExportLogsServiceResponse{
		PartialSuccess: &collogspb.ExportLogsPartialSuccess{
			RejectedLogRecords: 0,
		},
	}, nil
}
