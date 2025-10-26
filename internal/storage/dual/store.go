package dual

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/fidde/otlp_cardinality_checker/internal/analyzer/autotemplate"
	"github.com/fidde/otlp_cardinality_checker/internal/storage"
	"github.com/fidde/otlp_cardinality_checker/pkg/models"
)

// Store wraps two storage backends for dual-write migration.
// Writes go to both primary and secondary.
// Reads come from primary only.
type Store struct {
	primary   storage.Storage
	secondary storage.Storage
	logger    *slog.Logger
}

// Config holds dual store configuration.
type Config struct {
	Primary   storage.Storage
	Secondary storage.Storage
	Logger    *slog.Logger
}

// New creates a new dual-write store.
func New(cfg Config) *Store {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &Store{
		primary:   cfg.Primary,
		secondary: cfg.Secondary,
		logger:    cfg.Logger,
	}
}

// dualWrite performs a write to both backends.
// Errors from secondary are logged but don't fail the operation.
func (s *Store) dualWrite(ctx context.Context, op string, primaryWrite, secondaryWrite func() error) error {
	// Write to primary (this determines success/failure)
	err := primaryWrite()
	if err != nil {
		return err
	}

	// Write to secondary (errors logged but don't fail)
	go func() {
		if err := secondaryWrite(); err != nil {
			s.logger.Error("dual-write to secondary failed",
				"operation", op,
				"error", err,
			)
		}
	}()

	return nil
}

// StoreMetric stores metric metadata in both backends.
func (s *Store) StoreMetric(ctx context.Context, metric *models.MetricMetadata) error {
	return s.dualWrite(ctx, "StoreMetric",
		func() error { return s.primary.StoreMetric(ctx, metric) },
		func() error { return s.secondary.StoreMetric(ctx, metric) },
	)
}

// GetMetric retrieves metric metadata from primary backend only.
func (s *Store) GetMetric(ctx context.Context, name string) (*models.MetricMetadata, error) {
	return s.primary.GetMetric(ctx, name)
}

// ListMetrics lists metrics from primary backend only.
func (s *Store) ListMetrics(ctx context.Context, serviceName string) ([]*models.MetricMetadata, error) {
	return s.primary.ListMetrics(ctx, serviceName)
}

// StoreSpan stores span metadata in both backends.
func (s *Store) StoreSpan(ctx context.Context, span *models.SpanMetadata) error {
	return s.dualWrite(ctx, "StoreSpan",
		func() error { return s.primary.StoreSpan(ctx, span) },
		func() error { return s.secondary.StoreSpan(ctx, span) },
	)
}

// GetSpan retrieves span metadata from primary backend only.
func (s *Store) GetSpan(ctx context.Context, name string) (*models.SpanMetadata, error) {
	return s.primary.GetSpan(ctx, name)
}

// ListSpans lists spans from primary backend only.
func (s *Store) ListSpans(ctx context.Context, serviceName string) ([]*models.SpanMetadata, error) {
	return s.primary.ListSpans(ctx, serviceName)
}

// StoreLog stores log metadata in both backends.
func (s *Store) StoreLog(ctx context.Context, log *models.LogMetadata) error {
	return s.dualWrite(ctx, "StoreLog",
		func() error { return s.primary.StoreLog(ctx, log) },
		func() error { return s.secondary.StoreLog(ctx, log) },
	)
}

// GetLog retrieves log metadata from primary backend only.
func (s *Store) GetLog(ctx context.Context, severityText string) (*models.LogMetadata, error) {
	return s.primary.GetLog(ctx, severityText)
}

// ListLogs lists logs from primary backend only.
func (s *Store) ListLogs(ctx context.Context, serviceName string) ([]*models.LogMetadata, error) {
	return s.primary.ListLogs(ctx, serviceName)
}

// ListServices lists services from primary backend only.
func (s *Store) ListServices(ctx context.Context) ([]string, error) {
	return s.primary.ListServices(ctx)
}

// GetServiceOverview gets service overview from primary backend only.
func (s *Store) GetServiceOverview(ctx context.Context, serviceName string) (*models.ServiceOverview, error) {
	return s.primary.GetServiceOverview(ctx, serviceName)
}

// UseAutoTemplate returns autotemplate config from primary backend.
func (s *Store) UseAutoTemplate() bool {
	return s.primary.UseAutoTemplate()
}

// AutoTemplateCfg returns autotemplate config from primary backend.
func (s *Store) AutoTemplateCfg() autotemplate.Config {
	return s.primary.AutoTemplateCfg()
}

// Clear clears both backends.
func (s *Store) Clear(ctx context.Context) error {
	// Clear primary first
	if err := s.primary.Clear(ctx); err != nil {
		return fmt.Errorf("clear primary: %w", err)
	}

	// Clear secondary (best effort)
	if err := s.secondary.Clear(ctx); err != nil {
		s.logger.Error("failed to clear secondary backend",
			"error", err,
		)
	}

	return nil
}

// Close closes both backends.
func (s *Store) Close() error {
	var primaryErr, secondaryErr error

	primaryErr = s.primary.Close()
	secondaryErr = s.secondary.Close()

	if primaryErr != nil {
		return fmt.Errorf("close primary: %w", primaryErr)
	}
	if secondaryErr != nil {
		return fmt.Errorf("close secondary: %w", secondaryErr)
	}

	return nil
}
