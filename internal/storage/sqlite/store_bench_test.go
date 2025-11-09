package sqlite

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/fidde/otlp_cardinality_checker/pkg/models"
)

// BenchmarkMetricWrites measures write throughput for metrics
func BenchmarkMetricWrites(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	cfg := Config{
		DBPath:          dbPath,
		UseAutoTemplate: false,
		BatchSize:       100,
		FlushInterval:   100 * time.Millisecond,
	}

	store, err := New(cfg)
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metric := &models.MetricMetadata{
			Name: fmt.Sprintf("metric_%d", i%1000), // 1000 unique metrics
			Data: &models.GaugeMetric{DataPointCount: 1},
			LabelKeys: map[string]*models.KeyMetadata{
				"host": {
					EstimatedCardinality: 10,
					ValueSamples:         []string{"host-1", "host-2", "host-3"},
				},
				"env": {
					EstimatedCardinality: 3,
					ValueSamples:         []string{"prod", "staging", "dev"},
				},
			},
			SampleCount: 1,
			Services: map[string]int64{
				fmt.Sprintf("service-%d", i%10): 1, // 10 unique services
			},
		}

		if err := store.StoreMetric(ctx, metric); err != nil {
			b.Fatalf("StoreMetric failed: %v", err)
		}
	}
	b.StopTimer()

	// Report ops/sec
	opsPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(opsPerSec, "ops/sec")
}

// BenchmarkSpanWrites measures write throughput for spans
func BenchmarkSpanWrites(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	cfg := Config{
		DBPath:          dbPath,
		UseAutoTemplate: false,
		BatchSize:       100,
		FlushInterval:   100 * time.Millisecond,
	}

	store, err := New(cfg)
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		span := &models.SpanMetadata{
			Name:     fmt.Sprintf("GET /api/resource/%d", i%500), // 500 unique spans
			Kind:     2,
			KindName: "Server",
			AttributeKeys: map[string]*models.KeyMetadata{
				"http.method": {
					EstimatedCardinality: 5,
					ValueSamples:         []string{"GET", "POST", "PUT"},
				},
				"http.status_code": {
					EstimatedCardinality: 10,
					ValueSamples:         []string{"200", "404", "500"},
				},
			},
			SampleCount: 1,
			Services: map[string]int64{
				fmt.Sprintf("service-%d", i%10): 1,
			},
		}

		if err := store.StoreSpan(ctx, span); err != nil {
			b.Fatalf("StoreSpan failed: %v", err)
		}
	}
	b.StopTimer()

	opsPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(opsPerSec, "ops/sec")
}

// BenchmarkLogWrites measures write throughput for logs
func BenchmarkLogWrites(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	cfg := Config{
		DBPath:          dbPath,
		UseAutoTemplate: false,
		BatchSize:       100,
		FlushInterval:   100 * time.Millisecond,
	}

	store, err := New(cfg)
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	severities := []string{"INFO", "WARN", "ERROR", "DEBUG"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log := &models.LogMetadata{
			Severity: severities[i%len(severities)],
			AttributeKeys: map[string]*models.KeyMetadata{
				"error.type": {
					EstimatedCardinality: 5,
					ValueSamples:         []string{"DatabaseError", "NetworkError"},
				},
			},
			SampleCount: 1,
			Services: map[string]int64{
				fmt.Sprintf("service-%d", i%10): 1,
			},
		}

		if err := store.StoreLog(ctx, log); err != nil {
			b.Fatalf("StoreLog failed: %v", err)
		}
	}
	b.StopTimer()

	opsPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(opsPerSec, "ops/sec")
}

// BenchmarkMixedWorkload simulates realistic mixed workload (metrics, spans, logs)
func BenchmarkMixedWorkload(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	cfg := Config{
		DBPath:          dbPath,
		UseAutoTemplate: false,
		BatchSize:       100,
		FlushInterval:   50 * time.Millisecond, // More aggressive flushing
	}

	store, err := New(cfg)
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		switch i % 3 {
		case 0: // Metric
			metric := &models.MetricMetadata{
				Name:        fmt.Sprintf("metric_%d", i%1000),
				Data:        &models.GaugeMetric{DataPointCount: 1},
				SampleCount: 1,
				Services:    map[string]int64{fmt.Sprintf("svc-%d", i%10): 1},
			}
			if err := store.StoreMetric(ctx, metric); err != nil {
				b.Fatalf("StoreMetric failed: %v", err)
			}

		case 1: // Span
			span := &models.SpanMetadata{
				Name:        fmt.Sprintf("span_%d", i%500),
				Kind:        2,
				KindName:    "Server",
				SampleCount: 1,
				Services:    map[string]int64{fmt.Sprintf("svc-%d", i%10): 1},
			}
			if err := store.StoreSpan(ctx, span); err != nil {
				b.Fatalf("StoreSpan failed: %v", err)
			}

		case 2: // Log
			log := &models.LogMetadata{
				Severity:    "INFO",
				SampleCount: 1,
				Services:    map[string]int64{fmt.Sprintf("svc-%d", i%10): 1},
			}
			if err := store.StoreLog(ctx, log); err != nil {
				b.Fatalf("StoreLog failed: %v", err)
			}
		}
	}
	b.StopTimer()

	opsPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(opsPerSec, "ops/sec")
}

// BenchmarkConcurrentWrites measures throughput with concurrent writers
func BenchmarkConcurrentWrites(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	cfg := Config{
		DBPath:          dbPath,
		UseAutoTemplate: false,
		BatchSize:       100,
		FlushInterval:   50 * time.Millisecond,
	}

	store, err := New(cfg)
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Run with different concurrency levels
	for _, concurrency := range []int{1, 2, 4, 8, 16} {
		b.Run(fmt.Sprintf("writers=%d", concurrency), func(b *testing.B) {
			b.ResetTimer()

			var wg sync.WaitGroup
			opsPerWorker := b.N / concurrency

			for w := 0; w < concurrency; w++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()

					for i := 0; i < opsPerWorker; i++ {
						metric := &models.MetricMetadata{
							Name:        fmt.Sprintf("metric_%d_%d", workerID, i%100),
							Data:        &models.GaugeMetric{DataPointCount: 1},
							SampleCount: 1,
							Services:    map[string]int64{fmt.Sprintf("svc-%d", workerID): 1},
						}
						if err := store.StoreMetric(ctx, metric); err != nil {
							b.Errorf("StoreMetric failed: %v", err)
							return
						}
					}
				}(w)
			}

			wg.Wait()
			b.StopTimer()

			opsPerSec := float64(b.N) / b.Elapsed().Seconds()
			b.ReportMetric(opsPerSec, "ops/sec")
		})
	}
}

// BenchmarkReadWhileWriting measures read performance during writes
func BenchmarkReadWhileWriting(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	cfg := Config{
		DBPath:          dbPath,
		UseAutoTemplate: false,
		BatchSize:       100,
		FlushInterval:   50 * time.Millisecond,
	}

	store, err := New(cfg)
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Pre-populate with some data
	for i := 0; i < 100; i++ {
		metric := &models.MetricMetadata{
			Name:        fmt.Sprintf("metric_%d", i),
			Data:        &models.GaugeMetric{DataPointCount: 1},
			SampleCount: 1,
			Services:    map[string]int64{"test-service": 1},
		}
		store.StoreMetric(ctx, metric)
	}

	// Start background writer
	stopWriter := make(chan struct{})
	var writerWg sync.WaitGroup
	writerWg.Add(1)

	go func() {
		defer writerWg.Done()
		i := 100
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stopWriter:
				return
			case <-ticker.C:
				metric := &models.MetricMetadata{
					Name:        fmt.Sprintf("metric_%d", i%1000),
					Data:        &models.GaugeMetric{DataPointCount: 1},
					SampleCount: 1,
					Services:    map[string]int64{"test-service": 1},
				}
				store.StoreMetric(ctx, metric)
				i++
			}
		}
	}()

	// Benchmark reads
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metricName := fmt.Sprintf("metric_%d", i%100)
		_, err := store.GetMetric(ctx, metricName)
		if err != nil && err != models.ErrNotFound {
			b.Fatalf("GetMetric failed: %v", err)
		}
	}
	b.StopTimer()

	// Stop writer
	close(stopWriter)
	writerWg.Wait()

	opsPerSec := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(opsPerSec, "reads/sec")
}

// BenchmarkBatchSizeImpact measures impact of different batch sizes
func BenchmarkBatchSizeImpact(b *testing.B) {
	for _, batchSize := range []int{10, 50, 100, 200, 500} {
		b.Run(fmt.Sprintf("batch=%d", batchSize), func(b *testing.B) {
			tmpDir := b.TempDir()
			dbPath := filepath.Join(tmpDir, "bench.db")

			cfg := Config{
				DBPath:          dbPath,
				UseAutoTemplate: false,
				BatchSize:       batchSize,
				FlushInterval:   100 * time.Millisecond,
			}

			store, err := New(cfg)
			if err != nil {
				b.Fatalf("failed to create store: %v", err)
			}
			defer store.Close()

			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				metric := &models.MetricMetadata{
					Name:        fmt.Sprintf("metric_%d", i%1000),
					Data:        &models.GaugeMetric{DataPointCount: 1},
					SampleCount: 1,
					Services:    map[string]int64{"test-service": 1},
				}
				if err := store.StoreMetric(ctx, metric); err != nil {
					b.Fatalf("StoreMetric failed: %v", err)
				}
			}
			b.StopTimer()

			opsPerSec := float64(b.N) / b.Elapsed().Seconds()
			b.ReportMetric(opsPerSec, "ops/sec")
		})
	}
}

// BenchmarkHighThroughput simulates 30k EPS target
func BenchmarkHighThroughput(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	cfg := Config{
		DBPath:          dbPath,
		UseAutoTemplate: false,
		BatchSize:       200,
		FlushInterval:   50 * time.Millisecond,
	}

	store, err := New(cfg)
	if err != nil {
		b.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Target: 30k ops/sec for 10 seconds = 300k operations
	targetOps := 300000
	duration := 10 * time.Second

	// Use multiple goroutines to generate load
	numWorkers := 8
	opsPerWorker := targetOps / numWorkers

	start := time.Now()
	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalOps int
	var errors int

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for i := 0; i < opsPerWorker; i++ {
				// Stop if we've exceeded duration
				if time.Since(start) > duration {
					return
				}

				metric := &models.MetricMetadata{
					Name:        fmt.Sprintf("metric_%d_%d", workerID, i%1000),
					Data:        &models.GaugeMetric{DataPointCount: 1},
					SampleCount: 1,
					Services:    map[string]int64{fmt.Sprintf("svc-%d", workerID): 1},
				}

				if err := store.StoreMetric(ctx, metric); err != nil {
					mu.Lock()
					errors++
					mu.Unlock()
					continue
				}

				mu.Lock()
				totalOps++
				mu.Unlock()
			}
		}(w)
	}

	wg.Wait()
	elapsed := time.Since(start)

	actualRate := float64(totalOps) / elapsed.Seconds()

	b.ReportMetric(actualRate, "ops/sec")
	b.ReportMetric(float64(totalOps), "total_ops")
	b.ReportMetric(elapsed.Seconds(), "duration_sec")
	b.ReportMetric(float64(errors), "errors")

	// Log results
	b.Logf("Achieved: %.0f ops/sec (target: 30000)", actualRate)
	b.Logf("Total operations: %d in %.2fs", totalOps, elapsed.Seconds())
	b.Logf("Errors: %d", errors)

	// Verify we hit target
	if actualRate < 20000 {
		b.Errorf("Failed to achieve minimum 20k ops/sec (got %.0f)", actualRate)
	}
}

// Cleanup helper
func init() {
	// Ensure clean state between benchmarks
	os.RemoveAll(filepath.Join(os.TempDir(), "bench*.db"))
}
