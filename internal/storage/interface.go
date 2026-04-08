// Package storage defines the storage interface for telemetry metadata.
package storage

import (
	"context"

	"github.com/fidde/otlp_cardinality_checker/pkg/autotemplate"
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
	// CountLogPatterns returns the total number of unique log templates without building the full response.
	CountLogPatterns(ctx context.Context) (int, error)
	
	// Span pattern analysis - aggregate span names into patterns
	GetSpanPatterns(ctx context.Context) (*models.SpanPatternResponse, error)

	// Cross-signal cardinality analysis
	GetHighCardinalityKeys(ctx context.Context, threshold int, limit int) (*models.CrossSignalCardinalityResponse, error)

	// Metadata complexity analysis
	GetMetadataComplexity(ctx context.Context, threshold int, limit int) (*models.MetadataComplexityResponse, error)

	// Attribute catalog operations
	StoreAttributeValue(ctx context.Context, key, value, signalType, scope string) error
	GetAttribute(ctx context.Context, key string) (*models.AttributeMetadata, error)
	ListAttributes(ctx context.Context, filter *models.AttributeFilter) ([]*models.AttributeMetadata, error)

	// Deep watch operations
	WatchAttribute(ctx context.Context, key string) error
	UnwatchAttribute(ctx context.Context, key string) error
	GetWatchedAttribute(ctx context.Context, key string) (*models.WatchedAttribute, error)
	ListWatchedAttributes(ctx context.Context) ([]*models.WatchedAttribute, error)

	// Service operations
	ListServices(ctx context.Context) ([]string, error)
	GetServiceOverview(ctx context.Context, serviceName string) (*models.ServiceOverview, error)

	// Configuration (for autotemplate support)
	UseAutoTemplate() bool
	AutoTemplateCfg() autotemplate.Config

	// Configuration (for pod log enrichment)
	PodLogEnrichment() bool
	PodLogServiceLabels() []string

	// Clear all data
	Clear(ctx context.Context) error

	// Close the storage (for cleanup, e.g., DB connections)
	Close() error
}
