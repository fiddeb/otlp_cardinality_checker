package report

import (
	"context"

	"github.com/fidde/otlp_cardinality_checker/pkg/autotemplate"
	"github.com/fidde/otlp_cardinality_checker/pkg/models"
)

type mockStorage struct {
	metrics []*models.MetricMetadata
	spans   []*models.SpanMetadata
	logs    []*models.LogMetadata
	attrs   []*models.AttributeMetadata
}

func (m *mockStorage) StoreMetric(_ context.Context, _ *models.MetricMetadata) error {
	return nil
}

func (m *mockStorage) GetMetric(_ context.Context, _ string) (*models.MetricMetadata, error) {
	return nil, nil
}

func (m *mockStorage) ListMetrics(_ context.Context, _ string) ([]*models.MetricMetadata, error) {
	return m.metrics, nil
}

func (m *mockStorage) StoreSpan(_ context.Context, _ *models.SpanMetadata) error {
	return nil
}

func (m *mockStorage) GetSpan(_ context.Context, _ string) (*models.SpanMetadata, error) {
	return nil, nil
}

func (m *mockStorage) ListSpans(_ context.Context, _ string) ([]*models.SpanMetadata, error) {
	return m.spans, nil
}

func (m *mockStorage) StoreLog(_ context.Context, _ *models.LogMetadata) error {
	return nil
}

func (m *mockStorage) GetLog(_ context.Context, _ string) (*models.LogMetadata, error) {
	return nil, nil
}

func (m *mockStorage) ListLogs(_ context.Context, _ string) ([]*models.LogMetadata, error) {
	return m.logs, nil
}

func (m *mockStorage) GetLogPatterns(_ context.Context, _ int64, _ int) (*models.PatternExplorerResponse, error) {
	return nil, nil
}

func (m *mockStorage) CountLogPatterns(_ context.Context) (int, error) {
	return 0, nil
}

func (m *mockStorage) GetSpanPatterns(_ context.Context) (*models.SpanPatternResponse, error) {
	return nil, nil
}

func (m *mockStorage) GetHighCardinalityKeys(_ context.Context, _ int, _ int) (*models.CrossSignalCardinalityResponse, error) {
	return nil, nil
}

func (m *mockStorage) GetMetadataComplexity(_ context.Context, _ int, _ int) (*models.MetadataComplexityResponse, error) {
	return nil, nil
}

func (m *mockStorage) StoreAttributeValue(_ context.Context, _, _, _, _ string) error {
	return nil
}

func (m *mockStorage) GetAttribute(_ context.Context, _ string) (*models.AttributeMetadata, error) {
	return nil, nil
}

func (m *mockStorage) ListAttributes(_ context.Context, _ *models.AttributeFilter) ([]*models.AttributeMetadata, error) {
	return m.attrs, nil
}

func (m *mockStorage) WatchAttribute(_ context.Context, _ string) error {
	return nil
}

func (m *mockStorage) UnwatchAttribute(_ context.Context, _ string) error {
	return nil
}

func (m *mockStorage) GetWatchedAttribute(_ context.Context, _ string) (*models.WatchedAttribute, error) {
	return nil, nil
}

func (m *mockStorage) ListWatchedAttributes(_ context.Context) ([]*models.WatchedAttribute, error) {
	return nil, nil
}

func (m *mockStorage) ListServices(_ context.Context) ([]string, error) {
	return nil, nil
}

func (m *mockStorage) GetServiceOverview(_ context.Context, _ string) (*models.ServiceOverview, error) {
	return nil, nil
}

func (m *mockStorage) UseAutoTemplate() bool {
	return false
}

func (m *mockStorage) AutoTemplateCfg() autotemplate.Config {
	return autotemplate.Config{}
}

func (m *mockStorage) PodLogEnrichment() bool {
	return false
}

func (m *mockStorage) PodLogServiceLabels() []string {
	return nil
}

func (m *mockStorage) Clear(_ context.Context) error {
	return nil
}

func (m *mockStorage) Close() error {
	return nil
}
