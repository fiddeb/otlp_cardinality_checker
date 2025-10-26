// Package storage defines the storage interface for telemetry metadata.
package storage

import (
	"context"

	"github.com/fidde/otlp_cardinality_checker/internal/analyzer/autotemplate"
	"github.com/fidde/otlp_cardinality_checker/pkg/models"
)

// Storage is the interface for storing and retrieving telemetry metadata.
// Implementations must be safe for concurrent use.
type Storage interface {
	// Metric operations
	StoreMetric(ctx context.Context, metric *models.MetricMetadata) error
	GetMetric(ctx context.Context, name string) (*models.MetricMetadata, error)
	ListMetrics(ctx context.Context, serviceName string) ([]*models.MetricMetadata, error)

	// Span operations
	StoreSpan(ctx context.Context, span *models.SpanMetadata) error
	GetSpan(ctx context.Context, name string) (*models.SpanMetadata, error)
	ListSpans(ctx context.Context, serviceName string) ([]*models.SpanMetadata, error)

	// Log operations
	StoreLog(ctx context.Context, log *models.LogMetadata) error
	GetLog(ctx context.Context, severityText string) (*models.LogMetadata, error)
	ListLogs(ctx context.Context, serviceName string) ([]*models.LogMetadata, error)
	
	// Pattern explorer - advanced log pattern analysis
	GetLogPatterns(ctx context.Context, minCount int64, minServices int) (*models.PatternExplorerResponse, error)

	// Cross-signal cardinality analysis
	GetHighCardinalityKeys(ctx context.Context, threshold int, limit int) (*models.CrossSignalCardinalityResponse, error)

	// Service operations
	ListServices(ctx context.Context) ([]string, error)
	GetServiceOverview(ctx context.Context, serviceName string) (*models.ServiceOverview, error)

	// Configuration (for autotemplate support)
	UseAutoTemplate() bool
	AutoTemplateCfg() autotemplate.Config

	// Clear all data
	Clear(ctx context.Context) error

	// Close the storage (for cleanup, e.g., DB connections)
	Close() error
}
