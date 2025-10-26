package sqlite

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/fidde/otlp_cardinality_checker/internal/analyzer/autotemplate"
	"github.com/fidde/otlp_cardinality_checker/pkg/models"
)

// setupTestStore creates a temporary SQLite database for testing
func setupTestStore(t *testing.T) (*Store, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	cfg := Config{
		DBPath:          dbPath,
		UseAutoTemplate: true,
		AutoTemplateCfg: autotemplate.Config{
			Shards:       4,
			SimThreshold: 0.7,
		},
		BatchSize:     100,
		FlushInterval: 100 * time.Millisecond,
	}

	store, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}

	cleanup := func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

func TestStoreMetric(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	metric := &models.MetricMetadata{
		Name: "http_requests_total",
		Type: "counter",
		Unit: "1",
		LabelKeys: map[string]*models.KeyMetadata{
			"method": {
				EstimatedCardinality: 3,
				ValueSamples:         []string{"GET", "POST", "DELETE"},
			},
			"status": {
				EstimatedCardinality: 2,
				ValueSamples:         []string{"200", "404"},
			},
		},
		ResourceKeys: map[string]*models.KeyMetadata{
			"service.name": {
				EstimatedCardinality: 1,
				ValueSamples:         []string{"api-gateway"},
			},
		},
		SampleCount: 100,
		Services: map[string]int64{
			"api-gateway": 100,
		},
	}

	// Store metric
	err := store.StoreMetric(ctx, metric)
	if err != nil {
		t.Fatalf("StoreMetric failed: %v", err)
	}

	// Retrieve metric
	retrieved, err := store.GetMetric(ctx, "http_requests_total")
	if err != nil {
		t.Fatalf("GetMetric failed: %v", err)
	}

	if retrieved.Name != metric.Name {
		t.Errorf("expected name %s, got %s", metric.Name, retrieved.Name)
	}
	if retrieved.Type != metric.Type {
		t.Errorf("expected type %s, got %s", metric.Type, retrieved.Type)
	}
	if retrieved.SampleCount != metric.SampleCount {
		t.Errorf("expected sample_count %d, got %d", metric.SampleCount, retrieved.SampleCount)
	}

	// Check label keys
	if len(retrieved.LabelKeys) != 2 {
		t.Errorf("expected 2 label keys, got %d", len(retrieved.LabelKeys))
	}
	if _, ok := retrieved.LabelKeys["method"]; !ok {
		t.Error("expected label key 'method' not found")
	}
	if _, ok := retrieved.LabelKeys["status"]; !ok {
		t.Error("expected label key 'status' not found")
	}

	// Check services
	if len(retrieved.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(retrieved.Services))
	}
	if count, ok := retrieved.Services["api-gateway"]; !ok || count != 100 {
		t.Errorf("expected service api-gateway with count 100, got %d", count)
	}
}

func TestStoreMetricUpdate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Store initial metric
	metric1 := &models.MetricMetadata{
		Name: "cpu_usage",
		Type: "gauge",
		LabelKeys: map[string]*models.KeyMetadata{
			"host": {
				EstimatedCardinality: 1,
				ValueSamples:         []string{"host-1"},
			},
		},
		SampleCount: 50,
		Services: map[string]int64{
			"service-a": 50,
		},
	}

	err := store.StoreMetric(ctx, metric1)
	if err != nil {
		t.Fatalf("StoreMetric failed: %v", err)
	}

	// Store update with new label key and different service
	metric2 := &models.MetricMetadata{
		Name: "cpu_usage",
		Type: "gauge",
		LabelKeys: map[string]*models.KeyMetadata{
			"host": {
				EstimatedCardinality: 2,
				ValueSamples:         []string{"host-1", "host-2"},
			},
			"cpu": {
				EstimatedCardinality: 4,
				ValueSamples:         []string{"0", "1", "2", "3"},
			},
		},
		SampleCount: 30,
		Services: map[string]int64{
			"service-b": 30,
		},
	}

	err = store.StoreMetric(ctx, metric2)
	if err != nil {
		t.Fatalf("StoreMetric update failed: %v", err)
	}

	// Retrieve and verify merged result
	retrieved, err := store.GetMetric(ctx, "cpu_usage")
	if err != nil {
		t.Fatalf("GetMetric failed: %v", err)
	}

	// Sample count should be summed
	expectedCount := int64(80)
	if retrieved.SampleCount != expectedCount {
		t.Errorf("expected sample_count %d, got %d", expectedCount, retrieved.SampleCount)
	}

	// Both label keys should exist (union)
	if len(retrieved.LabelKeys) != 2 {
		t.Errorf("expected 2 label keys after merge, got %d", len(retrieved.LabelKeys))
	}

	// Both services should exist
	if len(retrieved.Services) != 2 {
		t.Errorf("expected 2 services after merge, got %d", len(retrieved.Services))
	}
	if count, ok := retrieved.Services["service-a"]; !ok || count != 50 {
		t.Errorf("expected service-a with count 50, got %d", count)
	}
	if count, ok := retrieved.Services["service-b"]; !ok || count != 30 {
		t.Errorf("expected service-b with count 30, got %d", count)
	}
}

func TestStoreSpan(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	span := &models.SpanMetadata{
		Name: "GET /api/users",
		Kind: "server",
		AttributeKeys: map[string]*models.KeyMetadata{
			"http.method": {
				EstimatedCardinality: 1,
				ValueSamples:         []string{"GET"},
			},
			"http.status_code": {
				EstimatedCardinality: 2,
				ValueSamples:         []string{"200", "404"},
			},
		},
		EventNames: []string{"exception", "log"},
		EventAttributeKeys: map[string]map[string]*models.KeyMetadata{
			"exception": {
				"exception.type": {
					EstimatedCardinality: 1,
					ValueSamples:         []string{"NullPointerException"},
				},
			},
		},
		SampleCount: 200,
		Services: map[string]int64{
			"user-service": 200,
		},
	}

	// Store span
	err := store.StoreSpan(ctx, span)
	if err != nil {
		t.Fatalf("StoreSpan failed: %v", err)
	}

	// Retrieve span
	retrieved, err := store.GetSpan(ctx, "GET /api/users")
	if err != nil {
		t.Fatalf("GetSpan failed: %v", err)
	}

	if retrieved.Name != span.Name {
		t.Errorf("expected name %s, got %s", span.Name, retrieved.Name)
	}
	if retrieved.Kind != span.Kind {
		t.Errorf("expected kind %s, got %s", span.Kind, retrieved.Kind)
	}
	if retrieved.SampleCount != span.SampleCount {
		t.Errorf("expected sample_count %d, got %d", span.SampleCount, retrieved.SampleCount)
	}

	// Check attribute keys
	if len(retrieved.AttributeKeys) != 2 {
		t.Errorf("expected 2 attribute keys, got %d", len(retrieved.AttributeKeys))
	}

	// Check event names
	if len(retrieved.EventNames) != 2 {
		t.Errorf("expected 2 event names, got %d", len(retrieved.EventNames))
	}

	// Check services
	if len(retrieved.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(retrieved.Services))
	}
}

func TestStoreLog(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	log := &models.LogMetadata{
		Severity: "ERROR",
		AttributeKeys: map[string]*models.KeyMetadata{
			"error.type": {
				EstimatedCardinality: 2,
				ValueSamples:         []string{"DatabaseError", "NetworkError"},
			},
		},
		SampleCount: 150,
		Services: map[string]int64{
			"service-a": 150,
		},
	}

	// Store log
	err := store.StoreLog(ctx, log)
	if err != nil {
		t.Fatalf("StoreLog failed: %v", err)
	}

	// Retrieve log
	retrieved, err := store.GetLog(ctx, "ERROR")
	if err != nil {
		t.Fatalf("GetLog failed: %v", err)
	}

	if retrieved.Severity != log.Severity {
		t.Errorf("expected severity %s, got %s", log.Severity, retrieved.Severity)
	}
	if retrieved.SampleCount != log.SampleCount {
		t.Errorf("expected sample_count %d, got %d", log.SampleCount, retrieved.SampleCount)
	}

	// Check attribute keys
	if len(retrieved.AttributeKeys) != 1 {
		t.Errorf("expected 1 attribute key, got %d", len(retrieved.AttributeKeys))
	}

	// Check services
	if len(retrieved.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(retrieved.Services))
	}
}

func TestListServices(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Store metrics with different services
	metric1 := &models.MetricMetadata{
		Name:        "metric1",
		Type:        "gauge",
		SampleCount: 10,
		Services:    map[string]int64{"service-a": 10},
	}
	metric2 := &models.MetricMetadata{
		Name:        "metric2",
		Type:        "counter",
		SampleCount: 20,
		Services:    map[string]int64{"service-b": 20},
	}

	// Store span with another service
	span := &models.SpanMetadata{
		Name:        "span1",
		Kind:        "server",
		SampleCount: 30,
		Services:    map[string]int64{"service-c": 30},
	}

	// Store log with overlapping service
	log := &models.LogMetadata{
		Severity:    "INFO",
		SampleCount: 40,
		Services:    map[string]int64{"service-a": 40},
	}

	if err := store.StoreMetric(ctx, metric1); err != nil {
		t.Fatalf("StoreMetric failed: %v", err)
	}
	if err := store.StoreMetric(ctx, metric2); err != nil {
		t.Fatalf("StoreMetric failed: %v", err)
	}
	if err := store.StoreSpan(ctx, span); err != nil {
		t.Fatalf("StoreSpan failed: %v", err)
	}
	if err := store.StoreLog(ctx, log); err != nil {
		t.Fatalf("StoreLog failed: %v", err)
	}

	// List all services
	services, err := store.ListServices(ctx)
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}

	// Should have 3 unique services
	if len(services) != 3 {
		t.Errorf("expected 3 services, got %d: %v", len(services), services)
	}

	// Verify service names (order doesn't matter)
	serviceSet := make(map[string]bool)
	for _, s := range services {
		serviceSet[s] = true
	}
	if !serviceSet["service-a"] || !serviceSet["service-b"] || !serviceSet["service-c"] {
		t.Errorf("expected services [service-a, service-b, service-c], got %v", services)
	}
}

func TestGetServiceOverview(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	serviceName := "test-service"

	// Store data for specific service
	metric := &models.MetricMetadata{
		Name:        "test_metric",
		Type:        "gauge",
		SampleCount: 10,
		Services:    map[string]int64{serviceName: 10},
	}
	span := &models.SpanMetadata{
		Name:        "test_span",
		Kind:        "server",
		SampleCount: 20,
		Services:    map[string]int64{serviceName: 20},
	}
	log := &models.LogMetadata{
		Severity:    "INFO",
		SampleCount: 30,
		Services:    map[string]int64{serviceName: 30},
	}

	if err := store.StoreMetric(ctx, metric); err != nil {
		t.Fatalf("StoreMetric failed: %v", err)
	}
	if err := store.StoreSpan(ctx, span); err != nil {
		t.Fatalf("StoreSpan failed: %v", err)
	}
	if err := store.StoreLog(ctx, log); err != nil {
		t.Fatalf("StoreLog failed: %v", err)
	}

	// Get service overview
	overview, err := store.GetServiceOverview(ctx, serviceName)
	if err != nil {
		t.Fatalf("GetServiceOverview failed: %v", err)
	}

	if overview.ServiceName != serviceName {
		t.Errorf("expected service name %s, got %s", serviceName, overview.ServiceName)
	}
	if overview.MetricCount != 1 {
		t.Errorf("expected 1 metric, got %d", overview.MetricCount)
	}
	if overview.SpanCount != 1 {
		t.Errorf("expected 1 span, got %d", overview.SpanCount)
	}
	if overview.LogCount != 1 {
		t.Errorf("expected 1 log, got %d", overview.LogCount)
	}
}

func TestConcurrentReadsAndWrites(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	var wg sync.WaitGroup
	numWriters := 10
	numReaders := 5
	opsPerWriter := 20

	// Start writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < opsPerWriter; j++ {
				metric := &models.MetricMetadata{
					Name:        fmt.Sprintf("metric_%d_%d", writerID, j),
					Type:        "gauge",
					SampleCount: int64(j + 1),
					Services: map[string]int64{
						fmt.Sprintf("service-%d", writerID): int64(j + 1),
					},
				}
				if err := store.StoreMetric(ctx, metric); err != nil {
					t.Errorf("writer %d failed: %v", writerID, err)
				}
			}
		}(i)
	}

	// Start readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for j := 0; j < opsPerWriter*2; j++ {
				// List metrics (might return partial results during writes)
				_, err := store.ListMetrics(ctx, "")
				if err != nil {
					t.Errorf("reader %d failed: %v", readerID, err)
				}
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Verify final state
	services, err := store.ListServices(ctx)
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}
	if len(services) != numWriters {
		t.Errorf("expected %d services, got %d", numWriters, len(services))
	}
}

func TestNotFound(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Try to get non-existent metric
	_, err := store.GetMetric(ctx, "non_existent")
	if err != models.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	// Try to get non-existent span
	_, err = store.GetSpan(ctx, "non_existent")
	if err != models.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	// Try to get non-existent log
	_, err = store.GetLog(ctx, "NON_EXISTENT")
	if err != models.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestClear(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Store some data
	metric := &models.MetricMetadata{
		Name:        "test_metric",
		Type:        "gauge",
		SampleCount: 10,
		Services:    map[string]int64{"service-a": 10},
	}
	if err := store.StoreMetric(ctx, metric); err != nil {
		t.Fatalf("StoreMetric failed: %v", err)
	}

	// Verify data exists
	_, err := store.GetMetric(ctx, "test_metric")
	if err != nil {
		t.Fatalf("GetMetric failed: %v", err)
	}

	// Clear all data
	if err := store.Clear(ctx); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify data is gone
	_, err = store.GetMetric(ctx, "test_metric")
	if err != models.ErrNotFound {
		t.Errorf("expected ErrNotFound after Clear, got %v", err)
	}

	services, err := store.ListServices(ctx)
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}
	if len(services) != 0 {
		t.Errorf("expected 0 services after Clear, got %d", len(services))
	}
}
