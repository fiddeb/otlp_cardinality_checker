package dual

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/fidde/otlp_cardinality_checker/internal/analyzer/autotemplate"
	"github.com/fidde/otlp_cardinality_checker/internal/storage/memory"
	"github.com/fidde/otlp_cardinality_checker/pkg/models"
)

func TestDualWrite(t *testing.T) {
	// Create two in-memory backends for testing
	primary := memory.New()
	secondary := memory.New()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	store := New(Config{
		Primary:   primary,
		Secondary: secondary,
		Logger:    logger,
	})
	defer store.Close()

	ctx := context.Background()

	// Test metric dual write
	metric := &models.MetricMetadata{
		Name:        "test_metric",
		Type:        "gauge",
		SampleCount: 100,
		Services: map[string]int64{
			"test-service": 100,
		},
	}

	err := store.StoreMetric(ctx, metric)
	if err != nil {
		t.Fatalf("StoreMetric failed: %v", err)
	}

	// Give async secondary write time to complete
	time.Sleep(50 * time.Millisecond)

	// Verify in primary
	retrieved, err := primary.GetMetric(ctx, "test_metric")
	if err != nil {
		t.Fatalf("primary GetMetric failed: %v", err)
	}
	if retrieved.Name != metric.Name {
		t.Errorf("primary: expected name %s, got %s", metric.Name, retrieved.Name)
	}

	// Verify in secondary
	retrieved, err = secondary.GetMetric(ctx, "test_metric")
	if err != nil {
		t.Fatalf("secondary GetMetric failed: %v", err)
	}
	if retrieved.Name != metric.Name {
		t.Errorf("secondary: expected name %s, got %s", metric.Name, retrieved.Name)
	}
}

func TestReadFromPrimary(t *testing.T) {
	primary := memory.New()
	secondary := memory.New()

	store := New(Config{
		Primary:   primary,
		Secondary: secondary,
	})
	defer store.Close()

	ctx := context.Background()

	// Write directly to primary only
	metric := &models.MetricMetadata{
		Name:        "primary_only",
		Type:        "counter",
		SampleCount: 50,
	}
	err := primary.StoreMetric(ctx, metric)
	if err != nil {
		t.Fatalf("primary StoreMetric failed: %v", err)
	}

	// Read via DualStore should return primary's data
	retrieved, err := store.GetMetric(ctx, "primary_only")
	if err != nil {
		t.Fatalf("GetMetric failed: %v", err)
	}
	if retrieved.Name != metric.Name {
		t.Errorf("expected name %s, got %s", metric.Name, retrieved.Name)
	}

	// Secondary should not have it
	_, err = secondary.GetMetric(ctx, "primary_only")
	if !errors.Is(err, models.ErrNotFound) {
		t.Errorf("expected ErrNotFound in secondary, got %v", err)
	}
}

func TestSecondaryWriteFailure(t *testing.T) {
	primary := memory.New()
	secondary := &failingStore{}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	store := New(Config{
		Primary:   primary,
		Secondary: secondary,
		Logger:    logger,
	})
	defer store.Close()

	ctx := context.Background()

	// Write should succeed even if secondary fails
	metric := &models.MetricMetadata{
		Name:        "test_metric",
		Type:        "gauge",
		SampleCount: 100,
	}

	err := store.StoreMetric(ctx, metric)
	if err != nil {
		t.Fatalf("StoreMetric should succeed even if secondary fails: %v", err)
	}

	// Verify in primary
	retrieved, err := primary.GetMetric(ctx, "test_metric")
	if err != nil {
		t.Fatalf("primary GetMetric failed: %v", err)
	}
	if retrieved.Name != metric.Name {
		t.Errorf("expected name %s, got %s", metric.Name, retrieved.Name)
	}
}

func TestDualWriteAllSignals(t *testing.T) {
	primary := memory.New()
	secondary := memory.New()

	store := New(Config{
		Primary:   primary,
		Secondary: secondary,
	})
	defer store.Close()

	ctx := context.Background()

	// Test span dual write
	span := &models.SpanMetadata{
		Name:        "test_span",
		Kind:        "server",
		SampleCount: 50,
	}
	if err := store.StoreSpan(ctx, span); err != nil {
		t.Fatalf("StoreSpan failed: %v", err)
	}

	// Test log dual write
	log := &models.LogMetadata{
		Severity:    "INFO",
		SampleCount: 30,
	}
	if err := store.StoreLog(ctx, log); err != nil {
		t.Fatalf("StoreLog failed: %v", err)
	}

	// Wait for async writes
	time.Sleep(50 * time.Millisecond)

	// Verify span in both
	_, err := primary.GetSpan(ctx, "test_span")
	if err != nil {
		t.Errorf("primary GetSpan failed: %v", err)
	}
	_, err = secondary.GetSpan(ctx, "test_span")
	if err != nil {
		t.Errorf("secondary GetSpan failed: %v", err)
	}

	// Verify log in both
	_, err = primary.GetLog(ctx, "INFO")
	if err != nil {
		t.Errorf("primary GetLog failed: %v", err)
	}
	_, err = secondary.GetLog(ctx, "INFO")
	if err != nil {
		t.Errorf("secondary GetLog failed: %v", err)
	}
}

// failingStore is a mock storage that always fails writes
type failingStore struct{}

func (f *failingStore) StoreMetric(ctx context.Context, metric *models.MetricMetadata) error {
	return errors.New("simulated failure")
}

func (f *failingStore) GetMetric(ctx context.Context, name string) (*models.MetricMetadata, error) {
	return nil, models.ErrNotFound
}

func (f *failingStore) ListMetrics(ctx context.Context, serviceName string) ([]*models.MetricMetadata, error) {
	return nil, nil
}

func (f *failingStore) StoreSpan(ctx context.Context, span *models.SpanMetadata) error {
	return errors.New("simulated failure")
}

func (f *failingStore) GetSpan(ctx context.Context, name string) (*models.SpanMetadata, error) {
	return nil, models.ErrNotFound
}

func (f *failingStore) ListSpans(ctx context.Context, serviceName string) ([]*models.SpanMetadata, error) {
	return nil, nil
}

func (f *failingStore) StoreLog(ctx context.Context, log *models.LogMetadata) error {
	return errors.New("simulated failure")
}

func (f *failingStore) GetLog(ctx context.Context, severityText string) (*models.LogMetadata, error) {
	return nil, models.ErrNotFound
}

func (f *failingStore) ListLogs(ctx context.Context, serviceName string) ([]*models.LogMetadata, error) {
	return nil, nil
}

func (f *failingStore) ListServices(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (f *failingStore) GetServiceOverview(ctx context.Context, serviceName string) (*models.ServiceOverview, error) {
	return nil, models.ErrNotFound
}

func (f *failingStore) UseAutoTemplate() bool {
	return false
}

func (f *failingStore) AutoTemplateCfg() autotemplate.Config {
	return autotemplate.Config{}
}

func (f *failingStore) Clear(ctx context.Context) error {
	return nil
}

func (f *failingStore) Close() error {
	return nil
}
