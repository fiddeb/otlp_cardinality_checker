package clickhouse

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const (
	defaultBatchSize     = 1000
	defaultFlushInterval = 5 * time.Second
	defaultShutdownWait  = 10 * time.Second
	maxRetries           = 3
)

// MetricRow represents a row in the metrics table
type MetricRow struct {
	Name                    string
	ServiceName             string
	MetricType              string
	Unit                    string
	AggregationTemporality  string
	IsMonotonic             uint8
	LabelKeys               []string
	ResourceKeys            []string
	SampleCount             uint64
	FirstSeen               time.Time
	LastSeen                time.Time
	Services                []string
	ServiceCount            uint32
}

// SpanRow represents a row in the spans table
type SpanRow struct {
	Name                string
	ServiceName         string
	Kind                uint8
	KindName            string
	AttributeKeys       []string
	ResourceKeys        []string
	EventNames          []string
	HasLinks            uint8
	StatusCodes         []string
	DroppedAttrsTotal   uint64
	DroppedAttrsMax     uint32
	DroppedEventsTotal  uint64
	DroppedEventsMax    uint32
	DroppedLinksTotal   uint64
	DroppedLinksMax     uint32
	SampleCount         uint64
	FirstSeen           time.Time
	LastSeen            time.Time
	Services            []string
	ServiceCount        uint32
}

// LogRow represents a row in the logs table
type LogRow struct {
	PatternTemplate     string
	Severity            string
	SeverityNumber      uint8
	ServiceName         string
	AttributeKeys       []string
	ResourceKeys        []string
	ExampleBody         string
	HasTraceContext     uint8
	HasSpanContext      uint8
	DroppedAttrsTotal   uint64
	DroppedAttrsMax     uint32
	SampleCount         uint64
	FirstSeen           time.Time
	LastSeen            time.Time
	Services            []string
	ServiceCount        uint32
}

// AttributeRow represents a row in the attribute_values table
type AttributeRow struct {
	Key              string
	Value            string
	SignalType       string
	Scope            string
	ObservationCount uint64
	FirstSeen        time.Time
	LastSeen         time.Time
}

// BatchBuffer manages batched writes to ClickHouse with automatic flushing
type BatchBuffer struct {
	conn driver.Conn

	mu              sync.Mutex
	metricRows      []MetricRow
	spanRows        []SpanRow
	logRows         []LogRow
	attributeRows   []AttributeRow

	batchSize     int
	flushInterval time.Duration
	shutdownWait  time.Duration

	flushTimer *time.Timer
	stopCh     chan struct{}
	closeOnce  sync.Once
	wg         sync.WaitGroup
	logger     *slog.Logger
}

// NewBatchBuffer creates a new batch buffer
func NewBatchBuffer(conn driver.Conn, logger *slog.Logger) *BatchBuffer {
	if logger == nil {
		logger = slog.Default()
	}

	b := &BatchBuffer{
		conn:          conn,
		batchSize:     defaultBatchSize,
		flushInterval: defaultFlushInterval,
		shutdownWait:  defaultShutdownWait,
		stopCh:        make(chan struct{}),
		logger:        logger,
	}

	b.flushTimer = time.NewTimer(b.flushInterval)

	// Start flush goroutine
	b.wg.Add(1)
	go b.flushLoop()

	return b
}

// AddMetric adds a metric row to the buffer
func (b *BatchBuffer) AddMetric(row MetricRow) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.metricRows = append(b.metricRows, row)

	if len(b.metricRows) >= b.batchSize {
		return b.flushMetricsLocked()
	}

	return nil
}

// AddSpan adds a span row to the buffer
func (b *BatchBuffer) AddSpan(row SpanRow) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.spanRows = append(b.spanRows, row)

	if len(b.spanRows) >= b.batchSize {
		return b.flushSpansLocked()
	}

	return nil
}

// AddLog adds a log row to the buffer
func (b *BatchBuffer) AddLog(row LogRow) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.logRows = append(b.logRows, row)

	if len(b.logRows) >= b.batchSize {
		return b.flushLogsLocked()
	}

	return nil
}

// AddAttribute adds an attribute row to the buffer
func (b *BatchBuffer) AddAttribute(row AttributeRow) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.attributeRows = append(b.attributeRows, row)

	if len(b.attributeRows) >= b.batchSize {
		return b.flushAttributesLocked()
	}

	return nil
}

// flushLoop periodically flushes buffers on timer
func (b *BatchBuffer) flushLoop() {
	defer b.wg.Done()

	for {
		select {
		case <-b.flushTimer.C:
			b.mu.Lock()
			_ = b.flushAllLocked()
			b.mu.Unlock()
			b.flushTimer.Reset(b.flushInterval)

		case <-b.stopCh:
			return
		}
	}
}

// flushAllLocked flushes all buffers (must hold lock)
func (b *BatchBuffer) flushAllLocked() error {
	var errs []error

	if err := b.flushMetricsLocked(); err != nil {
		errs = append(errs, err)
	}
	if err := b.flushSpansLocked(); err != nil {
		errs = append(errs, err)
	}
	if err := b.flushLogsLocked(); err != nil {
		errs = append(errs, err)
	}
	if err := b.flushAttributesLocked(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("flush errors: %v", errs)
	}
	return nil
}

// flushMetricsLocked flushes metric rows (must hold lock)
func (b *BatchBuffer) flushMetricsLocked() error {
	if len(b.metricRows) == 0 {
		return nil
	}

	start := time.Now()
	rows := b.metricRows
	b.metricRows = nil

	// Release lock during insert
	b.mu.Unlock()
	err := b.insertMetrics(rows)
	b.mu.Lock()

	if err != nil {
		b.logger.Error("failed to flush metrics",
			"error", err,
			"row_count", len(rows),
		)
		return err
	}

	b.logger.Debug("flushed metrics",
		"row_count", len(rows),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return nil
}

// flushSpansLocked flushes span rows (must hold lock)
func (b *BatchBuffer) flushSpansLocked() error {
	if len(b.spanRows) == 0 {
		return nil
	}

	start := time.Now()
	rows := b.spanRows
	b.spanRows = nil

	b.mu.Unlock()
	err := b.insertSpans(rows)
	b.mu.Lock()

	if err != nil {
		b.logger.Error("failed to flush spans",
			"error", err,
			"row_count", len(rows),
		)
		return err
	}

	b.logger.Debug("flushed spans",
		"row_count", len(rows),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return nil
}

// flushLogsLocked flushes log rows (must hold lock)
func (b *BatchBuffer) flushLogsLocked() error {
	if len(b.logRows) == 0 {
		return nil
	}

	start := time.Now()
	rows := b.logRows
	b.logRows = nil

	b.mu.Unlock()
	err := b.insertLogs(rows)
	b.mu.Lock()

	if err != nil {
		b.logger.Error("failed to flush logs",
			"error", err,
			"row_count", len(rows),
		)
		return err
	}

	b.logger.Debug("flushed logs",
		"row_count", len(rows),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return nil
}

// flushAttributesLocked flushes attribute rows (must hold lock)
func (b *BatchBuffer) flushAttributesLocked() error {
	if len(b.attributeRows) == 0 {
		return nil
	}

	start := time.Now()
	rows := b.attributeRows
	b.attributeRows = nil

	b.mu.Unlock()
	err := b.insertAttributes(rows)
	b.mu.Lock()

	if err != nil {
		b.logger.Error("failed to flush attributes",
			"error", err,
			"row_count", len(rows),
		)
		return err
	}

	b.logger.Debug("flushed attributes",
		"row_count", len(rows),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return nil
}

// Close gracefully shuts down the buffer, flushing remaining data
func (b *BatchBuffer) Close(ctx context.Context) error {
	var finalErr error
	
	// Use sync.Once to ensure we only close once
	b.closeOnce.Do(func() {
		// Stop flush loop
		close(b.stopCh)

		// Create shutdown context with timeout
		shutdownCtx, cancel := context.WithTimeout(ctx, b.shutdownWait)
		defer cancel()

		// Wait for flush loop to stop
		done := make(chan struct{})
		go func() {
			b.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Flush loop stopped
		case <-shutdownCtx.Done():
			b.logger.Warn("flush loop did not stop within timeout")
		}

		// Final flush
		b.mu.Lock()
		defer b.mu.Unlock()

		finalErr = b.flushAllLocked()
	})

	return finalErr
}

// Insert methods with retry logic

func (b *BatchBuffer) insertMetrics(rows []MetricRow) error {
	return b.retryInsert(func(ctx context.Context) error {
		batch, err := b.conn.PrepareBatch(ctx, "INSERT INTO metrics")
		if err != nil {
			return err
		}

		for _, row := range rows {
			err = batch.Append(
				row.Name,
				row.ServiceName,
				row.MetricType,
				row.Unit,
				row.AggregationTemporality,
				row.IsMonotonic,
				row.LabelKeys,
				row.ResourceKeys,
				row.SampleCount,
				row.FirstSeen,
				row.LastSeen,
				row.Services,
				row.ServiceCount,
			)
			if err != nil {
				return err
			}
		}

		return batch.Send()
	})
}

func (b *BatchBuffer) insertSpans(rows []SpanRow) error {
	return b.retryInsert(func(ctx context.Context) error {
		batch, err := b.conn.PrepareBatch(ctx, "INSERT INTO spans")
		if err != nil {
			return err
		}

		for _, row := range rows {
			err = batch.Append(
				row.Name,
				row.ServiceName,
				row.Kind,
				row.KindName,
				row.AttributeKeys,
				row.ResourceKeys,
				row.EventNames,
				row.HasLinks,
				row.StatusCodes,
				row.DroppedAttrsTotal,
				row.DroppedAttrsMax,
				row.DroppedEventsTotal,
				row.DroppedEventsMax,
				row.DroppedLinksTotal,
				row.DroppedLinksMax,
				row.SampleCount,
				row.FirstSeen,
				row.LastSeen,
				row.Services,
				row.ServiceCount,
			)
			if err != nil {
				return err
			}
		}

		return batch.Send()
	})
}

func (b *BatchBuffer) insertLogs(rows []LogRow) error {
	return b.retryInsert(func(ctx context.Context) error {
		batch, err := b.conn.PrepareBatch(ctx, "INSERT INTO logs")
		if err != nil {
			return err
		}

		for _, row := range rows {
			err = batch.Append(
				row.PatternTemplate,
				row.Severity,
				row.SeverityNumber,
				row.ServiceName,
				row.AttributeKeys,
				row.ResourceKeys,
				row.ExampleBody,
				row.HasTraceContext,
				row.HasSpanContext,
				row.DroppedAttrsTotal,
				row.DroppedAttrsMax,
				row.SampleCount,
				row.FirstSeen,
				row.LastSeen,
				row.Services,
				row.ServiceCount,
			)
			if err != nil {
				return err
			}
		}

		return batch.Send()
	})
}

func (b *BatchBuffer) insertAttributes(rows []AttributeRow) error {
	return b.retryInsert(func(ctx context.Context) error {
		batch, err := b.conn.PrepareBatch(ctx, "INSERT INTO attribute_values")
		if err != nil {
			return err
		}

		for _, row := range rows {
			err = batch.Append(
				row.Key,
				row.Value,
				row.SignalType,
				row.Scope,
				row.ObservationCount,
				row.FirstSeen,
				row.LastSeen,
			)
			if err != nil {
				return err
			}
		}

		return batch.Send()
	})
}

// retryInsert retries insert operation with exponential backoff
func (b *BatchBuffer) retryInsert(fn func(context.Context) error) error {
	var err error
	retryDelay := 100 * time.Millisecond

	for attempt := 1; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err = fn(ctx)
		cancel()

		if err == nil {
			return nil
		}

		if attempt < maxRetries {
			time.Sleep(retryDelay)
			retryDelay *= 2
		}
	}

	return fmt.Errorf("insert failed after %d attempts: %w", maxRetries, err)
}
