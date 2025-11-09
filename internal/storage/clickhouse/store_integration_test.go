// +build integration

package clickhouse

import (
	"context"
	"testing"
	"time"

	"github.com/fidde/otlp_cardinality_checker/pkg/models"
)

// TestClickHouseIntegration tests basic ClickHouse operations
// Run with: go test -tags=integration ./internal/storage/clickhouse -v
func TestClickHouseIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	// Connect to ClickHouse
	config := DefaultConnectionConfig()
	conn, err := Connect(ctx, config)
	if err != nil {
		t.Skipf("ClickHouse not available: %v", err)
	}
	defer conn.Close()

	// Initialize schema
	if err := InitializeSchema(ctx, conn); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Create store
	store, err := NewStore(ctx, conn)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	t.Run("StoreAndGetMetric", func(t *testing.T) {
		metric := &models.MetricMetadata{
			Name:        "test_metric",
			Description: "Test metric",
			Unit:        "ms",
			Data: &models.GaugeMetric{
				DataPointCount: 100,
			},
			LabelKeys: map[string]*models.KeyMetadata{
				"method": {Count: 100},
				"status": {Count: 100},
			},
			ResourceKeys: map[string]*models.KeyMetadata{
				"service.name": {Count: 100},
			},
			SampleCount: 100,
			Services: map[string]int64{
				"test-service": 100,
			},
		}

		// Store metric
		if err := store.StoreMetric(ctx, metric); err != nil {
			t.Fatalf("Failed to store metric: %v", err)
		}

		// Wait for buffer flush
		time.Sleep(6 * time.Second)

		// Get metric
		retrieved, err := store.GetMetric(ctx, "test_metric")
		if err != nil {
			t.Fatalf("Failed to get metric: %v", err)
		}

		if retrieved.Name != "test_metric" {
			t.Errorf("Expected name 'test_metric', got '%s'", retrieved.Name)
		}

		if len(retrieved.LabelKeys) != 2 {
			t.Errorf("Expected 2 label keys, got %d", len(retrieved.LabelKeys))
		}
	})

	t.Run("StoreAndGetSpan", func(t *testing.T) {
		span := &models.SpanMetadata{
			Name:     "test_span",
			Kind:     2,
			KindName: "SERVER",
			AttributeKeys: map[string]*models.KeyMetadata{
				"http.method": {Count: 50},
				"http.url":    {Count: 50},
			},
			ResourceKeys: map[string]*models.KeyMetadata{
				"service.name": {Count: 50},
			},
			SampleCount: 50,
			Services: map[string]int64{
				"test-service": 50,
			},
		}

		if err := store.StoreSpan(ctx, span); err != nil {
			t.Fatalf("Failed to store span: %v", err)
		}

		time.Sleep(6 * time.Second)

		retrieved, err := store.GetSpan(ctx, "test_span")
		if err != nil {
			t.Fatalf("Failed to get span: %v", err)
		}

		if retrieved.Name != "test_span" {
			t.Errorf("Expected name 'test_span', got '%s'", retrieved.Name)
		}
	})

	t.Run("StoreAndGetLog", func(t *testing.T) {
		log := &models.LogMetadata{
			Severity:       "INFO",
			SeverityNumber: 9,
			AttributeKeys: map[string]*models.KeyMetadata{
				"log.level": {Count: 30},
			},
			ResourceKeys: map[string]*models.KeyMetadata{
				"service.name": {Count: 30},
			},
			SampleCount: 30,
			Services: map[string]int64{
				"test-service": 30,
			},
		}

		if err := store.StoreLog(ctx, log); err != nil {
			t.Fatalf("Failed to store log: %v", err)
		}

		time.Sleep(6 * time.Second)

		retrieved, err := store.GetLog(ctx, "INFO")
		if err != nil {
			t.Fatalf("Failed to get log: %v", err)
		}

		if retrieved.Severity != "INFO" {
			t.Errorf("Expected severity 'INFO', got '%s'", retrieved.Severity)
		}
	})

	t.Run("StoreAndGetAttribute", func(t *testing.T) {
		// Store some attribute values
		values := []string{"value1", "value2", "value3"}
		for _, val := range values {
			if err := store.StoreAttributeValue(ctx, "test_key", val, "metric", "label"); err != nil {
				t.Fatalf("Failed to store attribute value: %v", err)
			}
		}

		time.Sleep(6 * time.Second)

		// Get attribute metadata
		attr, err := store.GetAttribute(ctx, "test_key", "metric", "label")
		if err != nil {
			t.Fatalf("Failed to get attribute: %v", err)
		}

		if attr.Key != "test_key" {
			t.Errorf("Expected key 'test_key', got '%s'", attr.Key)
		}

		if attr.Cardinality != 3 {
			t.Errorf("Expected cardinality 3, got %d", attr.Cardinality)
		}
	})

	t.Run("ListServices", func(t *testing.T) {
		services, err := store.ListServices(ctx)
		if err != nil {
			t.Fatalf("Failed to list services: %v", err)
		}

		if len(services) == 0 {
			t.Error("Expected at least one service")
		}

		found := false
		for _, svc := range services {
			if svc == "test-service" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find 'test-service' in service list")
		}
	})
}
