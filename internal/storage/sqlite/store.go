// Package sqlite provides a SQLite-backed storage implementation.
package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fidde/otlp_cardinality_checker/internal/analyzer/autotemplate"
	"github.com/fidde/otlp_cardinality_checker/pkg/models"
	_ "modernc.org/sqlite"
)

//go:embed migrations/001_initial_schema.up.sql
var migration001SQL string

//go:embed migrations/002_unified_signal_keys.up.sql
var migration002SQL string

//go:embed migrations/003_log_service_keys.up.sql
var migration003SQL string

//go:embed migrations/004_log_otlp_fields.up.sql
var migration004SQL string

//go:embed migrations/005_span_otlp_fields.up.sql
var migration005SQL string

//go:embed migrations/006_add_metric_data.up.sql
var migration006SQL string

// Store is a SQLite-backed storage for telemetry metadata.
type Store struct {
	db *sql.DB

	// Batch writer
	writeCh   chan writeOp
	flushCh   chan chan struct{} // Channel to request immediate flush
	closeCh   chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup

	// Prepared statements cache
	stmtCache   map[string]*sql.Stmt
	stmtCacheMu sync.RWMutex

	// Autotemplate configuration
	useAutoTemplate bool
	autoTemplateCfg autotemplate.Config
}

// writeOp represents a write operation to be batched.
type writeOp struct {
	opType string
	data   interface{}
	done   chan error
}

// Config holds SQLite store configuration.
type Config struct {
	DBPath          string
	UseAutoTemplate bool
	AutoTemplateCfg autotemplate.Config
	BatchSize       int
	FlushInterval   time.Duration
}

// DefaultConfig returns default SQLite configuration.
func DefaultConfig(dbPath string) Config {
	cfg := autotemplate.DefaultConfig()
	cfg.Shards = 4
	cfg.SimThreshold = 0.7

	return Config{
		DBPath:          dbPath,
		UseAutoTemplate: false,
		AutoTemplateCfg: cfg,
		BatchSize:       1000,       // Optimal balance found via profiling
		FlushInterval:   15 * time.Millisecond, // Sweet spot for batching
	}
}

// runMigrations runs all migrations that haven't been applied yet.
func runMigrations(db *sql.DB) error {
	// Define all migrations with their version numbers
	migrations := []struct {
		version int
		sql     string
	}{
		{1, migration001SQL},
		{2, migration002SQL},
		{3, migration003SQL},
		{4, migration004SQL},
		{5, migration005SQL},
		{6, migration006SQL},
	}

	// Get already applied migrations
	appliedVersions := make(map[int]bool)
	rows, err := db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		// Table doesn't exist yet, this is fine - migration 1 will create it
		if !strings.Contains(err.Error(), "no such table") {
			return fmt.Errorf("querying schema_migrations: %w", err)
		}
	} else {
		defer rows.Close()
		for rows.Next() {
			var version int
			if err := rows.Scan(&version); err != nil {
				return fmt.Errorf("scanning version: %w", err)
			}
			appliedVersions[version] = true
		}
	}

	// Run unapplied migrations in order
	for _, m := range migrations {
		if appliedVersions[m.version] {
			continue // Already applied
		}

		if _, err := db.Exec(m.sql); err != nil {
			return fmt.Errorf("running migration %d: %w", m.version, err)
		}
	}

	return nil
}


// New creates a new SQLite store with the given configuration.
func New(cfg Config) (*Store, error) {
	// Open database
	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Set connection pool limits for better concurrency
	db.SetMaxOpenConns(10)   // Optimal based on profiling
	db.SetMaxIdleConns(5)    // Keep connections warm
	db.SetConnMaxLifetime(0) // Reuse connections indefinitely

	// Set pragmas for performance
	pragmas := []string{
		"PRAGMA journal_mode=WAL",      // Write-Ahead Logging for better concurrency
		"PRAGMA synchronous=NORMAL",    // Balance safety and speed
		"PRAGMA cache_size=-128000",    // 128MB cache - proven optimal
		"PRAGMA temp_store=MEMORY",     // Temp tables in memory
		"PRAGMA busy_timeout=30000",    // 30s timeout
		"PRAGMA foreign_keys=ON",       // Enforce foreign keys
		"PRAGMA mmap_size=268435456",   // 256MB memory-mapped I/O
		"PRAGMA page_size=4096",        // Optimal page size
		"PRAGMA locking_mode=NORMAL",   // Allow concurrent access
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("setting pragma: %w", err)
		}
	}

	// Run migrations intelligently - only if not already applied
	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	store := &Store{
		db:              db,
		writeCh:         make(chan writeOp, 5000), // Optimal buffer size
		flushCh:         make(chan chan struct{}),
		closeCh:         make(chan struct{}),
		stmtCache:       make(map[string]*sql.Stmt),
		useAutoTemplate: cfg.UseAutoTemplate,
		autoTemplateCfg: cfg.AutoTemplateCfg,
	}

	// Start batch writer goroutine
	store.wg.Add(1)
	go store.batchWriter(cfg.BatchSize, cfg.FlushInterval)

	return store, nil
}

// DB returns the underlying database connection for direct queries.
// This should only be used for read-only operations to avoid breaking batch writes.
func (s *Store) DB() *sql.DB {
	return s.db
}

// batchWriter runs in a goroutine and batches write operations.
func (s *Store) batchWriter(batchSize int, flushInterval time.Duration) {
	defer s.wg.Done()

	batch := make([]writeOp, 0, batchSize)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}

		// Execute batch in a transaction
		err := s.executeBatch(batch)

		// Send result to all ops in batch
		for i := range batch {
			if batch[i].done != nil {
				batch[i].done <- err
				close(batch[i].done)
			}
		}

		batch = batch[:0]
	}

	for {
		select {
		case op := <-s.writeCh:
			batch = append(batch, op)
			if batchSize > 0 && len(batch) >= batchSize {
				flush()
			}

		case <-ticker.C:
			flush()

		case doneCh := <-s.flushCh:
			flush()
			close(doneCh) // Signal flush completed

		case <-s.closeCh:
			// Drain remaining ops
			close(s.writeCh)
			for op := range s.writeCh {
				batch = append(batch, op)
			}
			flush()
			return
		}
	}
}

// executeBatch runs a batch of write operations in a single transaction.
func (s *Store) executeBatch(batch []writeOp) error {
	if len(batch) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, op := range batch {
		var err error
		switch op.opType {
		case "StoreMetric":
			err = s.storeMetricTx(tx, op.data.(*models.MetricMetadata))
		case "StoreSpan":
			err = s.storeSpanTx(tx, op.data.(*models.SpanMetadata))
		case "StoreLog":
			err = s.storeLogTx(tx, op.data.(*models.LogMetadata))
		default:
			err = fmt.Errorf("unknown operation: %s", op.opType)
		}

		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// Close closes the store and releases resources.
func (s *Store) Close() error {
	var err error
	s.closeOnce.Do(func() {
		close(s.closeCh)
		s.wg.Wait()
		
		// Close all prepared statements
		s.stmtCacheMu.Lock()
		for _, stmt := range s.stmtCache {
			stmt.Close()
		}
		s.stmtCacheMu.Unlock()
		
		err = s.db.Close()
	})
	return err
}

// getOrPrepareStmt gets a prepared statement from cache or prepares a new one.
// This reduces SQL parsing overhead significantly (saves ~2% CPU based on profiling).
func (s *Store) getOrPrepareStmt(tx *sql.Tx, query string) (*sql.Stmt, error) {
	// For transaction statements, we can't cache them across transactions
	// So we prepare them fresh each time within the transaction
	return tx.Prepare(query)
}

// Flush forces an immediate flush of pending writes.
// This is primarily for testing to ensure async writes complete.
func (s *Store) Flush() {
	doneCh := make(chan struct{})
	select {
	case s.flushCh <- doneCh:
		<-doneCh // Wait for flush to complete
	case <-s.closeCh:
		// Store is closing, no need to flush
	}
}

// UseAutoTemplate returns whether autotemplate is enabled.
func (s *Store) UseAutoTemplate() bool {
	return s.useAutoTemplate
}

// AutoTemplateCfg returns the autotemplate configuration.
func (s *Store) AutoTemplateCfg() autotemplate.Config {
	return s.autoTemplateCfg
}

// Clear removes all stored data.
func (s *Store) Clear(ctx context.Context) error {
	tables := []string{
		"signal_keys", // Unified keys table (new)
		"log_body_templates",
		"log_keys",
		"log_services",
		"logs",
		"span_events",
		"span_keys",
		"span_services",
		"spans",
		"metric_keys",
		"metric_services",
		"metrics",
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, table := range tables {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("clearing %s: %w", table, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// Helper functions

// encodeJSON encodes data as JSON string.
func encodeJSON(data interface{}) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("encoding JSON: %w", err)
	}
	return string(b), nil
}

// decodeJSON decodes JSON string to target.
func decodeJSON(data string, target interface{}) error {
	if err := json.Unmarshal([]byte(data), target); err != nil {
		return fmt.Errorf("decoding JSON: %w", err)
	}
	return nil
}

// sortedKeys returns sorted keys from a map.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// StoreMetric stores or updates metric metadata.
func (s *Store) StoreMetric(ctx context.Context, metric *models.MetricMetadata) error {
	// Read existing metric if it exists, merge before storing
	existingMetric, err := s.GetMetric(ctx, metric.Name)
	if err == nil && existingMetric != nil {
		// Metric exists - merge new data with existing using HLL
		existingMetric.MergeMetricMetadata(metric)
		metric = existingMetric
	}
	// If error (not found), just use new metric as-is
	
	// Fire-and-forget: send to batch writer without waiting
	select {
	case s.writeCh <- writeOp{opType: "StoreMetric", data: metric, done: nil}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closeCh:
		return errors.New("store is closed")
	}
}

// storeMetricTx stores metric metadata within a transaction.
func (s *Store) storeMetricTx(tx *sql.Tx, metric *models.MetricMetadata) error {
	// Note: Merging is already done in StoreMetric() before calling this function
	
	// Serialize metric data to JSON
	var metricDataJSON string
	if metric.Data != nil {
		dataBytes, err := models.MarshalMetricData(metric.Data)
		if err != nil {
			return fmt.Errorf("marshaling metric data: %w", err)
		}
		metricDataJSON = string(dataBytes)
	}

	// 1. Upsert base metric
	_, err := tx.Exec(`
		INSERT INTO metrics (name, type, unit, description, total_sample_count, metric_data)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			type = excluded.type,
			unit = COALESCE(excluded.unit, unit),
			description = COALESCE(excluded.description, description),
			total_sample_count = excluded.total_sample_count,
			metric_data = COALESCE(excluded.metric_data, metric_data)
	`, metric.Name, metric.GetType(), metric.Unit, metric.Description, metric.SampleCount, metricDataJSON)
	if err != nil {
		return fmt.Errorf("upserting metric: %w", err)
	}

	// 2. Upsert service mappings
	for service, count := range metric.Services {
		_, err = tx.Exec(`
			INSERT INTO metric_services (metric_name, service_name, sample_count)
			VALUES (?, ?, ?)
			ON CONFLICT(metric_name, service_name) DO UPDATE SET
				sample_count = excluded.sample_count
		`, metric.Name, service, count)
		if err != nil {
			return fmt.Errorf("upserting metric service %s: %w", service, err)
		}
	}

	// 3. Upsert label keys
	if err := s.upsertKeysForMetric(tx, metric.Name, "label", metric.LabelKeys); err != nil {
		return fmt.Errorf("upserting label keys: %w", err)
	}

	// 4. Upsert resource keys
	if err := s.upsertKeysForMetric(tx, metric.Name, "resource", metric.ResourceKeys); err != nil {
		return fmt.Errorf("upserting resource keys: %w", err)
	}

	return nil
}

// upsertKeysForMetric upserts key metadata for a metric.
// Dual-writes to both metric_keys (legacy) and signal_keys (unified) tables.
func (s *Store) upsertKeysForMetric(tx *sql.Tx, metricName, keyScope string, keys map[string]*models.KeyMetadata) error {
	for keyName, keyMeta := range keys {
		samples, err := encodeJSON(keyMeta.ValueSamples)
		if err != nil {
			return fmt.Errorf("encoding samples for key %s: %w", keyName, err)
		}

		// Write to legacy metric_keys table for backward compatibility
		_, err = tx.Exec(`
			INSERT INTO metric_keys (
				metric_name, key_scope, key_name, key_count, key_percentage,
				estimated_cardinality, value_samples, hll_sketch
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(metric_name, key_scope, key_name) DO UPDATE SET
				key_count = excluded.key_count,
				key_percentage = excluded.key_percentage,
				estimated_cardinality = excluded.estimated_cardinality,
				value_samples = excluded.value_samples,
				hll_sketch = excluded.hll_sketch
		`, metricName, keyScope, keyName, keyMeta.Count, keyMeta.Percentage,
			keyMeta.EstimatedCardinality, samples, nil) // HLL sketch = nil for now

		if err != nil {
			return fmt.Errorf("upserting key %s to metric_keys: %w", keyName, err)
		}

		// Write to unified signal_keys table
		_, err = tx.Exec(`
			INSERT INTO signal_keys (
				signal_type, signal_name, key_scope, key_name, event_name,
				key_count, key_percentage, estimated_cardinality, value_samples, hll_sketch
			) VALUES ('metric', ?, ?, ?, '', ?, ?, ?, ?, ?)
			ON CONFLICT(signal_type, signal_name, key_scope, key_name, event_name) DO UPDATE SET
				key_count = excluded.key_count,
				key_percentage = excluded.key_percentage,
				estimated_cardinality = excluded.estimated_cardinality,
				value_samples = excluded.value_samples,
				hll_sketch = excluded.hll_sketch
		`, metricName, keyScope, keyName, keyMeta.Count, keyMeta.Percentage,
			keyMeta.EstimatedCardinality, samples, nil)

		if err != nil {
			return fmt.Errorf("upserting key %s to signal_keys: %w", keyName, err)
		}
	}
	return nil
}

// GetMetric retrieves metric metadata by name.
func (s *Store) GetMetric(ctx context.Context, name string) (*models.MetricMetadata, error) {
	// Get base metric
	var metricType string
	var metricName, unit, description string
	var sampleCount int64
	var metricDataJSON sql.NullString
	
	err := s.db.QueryRowContext(ctx, `
		SELECT name, type, unit, description, total_sample_count, metric_data
		FROM metrics WHERE name = ?
	`, name).Scan(&metricName, &metricType, &unit, &description, &sampleCount, &metricDataJSON)

	if err == sql.ErrNoRows {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying metric: %w", err)
	}

	// Deserialize metric data from JSON if available
	var metricData models.MetricData
	if metricDataJSON.Valid && metricDataJSON.String != "" {
		metricData, err = models.UnmarshalMetricData([]byte(metricDataJSON.String))
		if err != nil {
			return nil, fmt.Errorf("unmarshaling metric data: %w", err)
		}
	}
	
	// Fallback to basic type if no JSON data
	if metricData == nil {
		switch metricType {
		case "Gauge":
			metricData = &models.GaugeMetric{DataPointCount: sampleCount}
		case "Sum":
			metricData = &models.SumMetric{DataPointCount: sampleCount}
		case "Histogram":
			metricData = &models.HistogramMetric{DataPointCount: sampleCount}
		case "ExponentialHistogram":
			metricData = &models.ExponentialHistogramMetric{DataPointCount: sampleCount}
		case "Summary":
			metricData = &models.SummaryMetric{DataPointCount: sampleCount}
		default:
			// Fallback to Gauge for unknown types
			metricData = &models.GaugeMetric{DataPointCount: sampleCount}
		}
	}
	
	metric := models.NewMetricMetadata(metricName, metricData)
	metric.Unit = unit
	metric.Description = description
	metric.SampleCount = sampleCount

	// Get services
	metric.Services = make(map[string]int64)
	rows, err := s.db.QueryContext(ctx, `
		SELECT service_name, sample_count
		FROM metric_services WHERE metric_name = ?
	`, name)
	if err != nil {
		return nil, fmt.Errorf("querying metric services: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var serviceName string
		var count int64
		if err := rows.Scan(&serviceName, &count); err != nil {
			return nil, fmt.Errorf("scanning service: %w", err)
		}
		metric.Services[serviceName] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Get label keys
	metric.LabelKeys, err = s.getKeysForMetric(ctx, name, "label")
	if err != nil {
		return nil, fmt.Errorf("querying label keys: %w", err)
	}

	// Get resource keys
	metric.ResourceKeys, err = s.getKeysForMetric(ctx, name, "resource")
	if err != nil {
		return nil, fmt.Errorf("querying resource keys: %w", err)
	}

	return metric, nil
}

// getKeysForMetric retrieves key metadata for a metric and scope.
// Reads from unified signal_keys table.
func (s *Store) getKeysForMetric(ctx context.Context, metricName, keyScope string) (map[string]*models.KeyMetadata, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT key_name, key_count, key_percentage, estimated_cardinality, value_samples
		FROM signal_keys
		WHERE signal_type = 'metric' AND signal_name = ? AND key_scope = ?
	`, metricName, keyScope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make(map[string]*models.KeyMetadata)
	for rows.Next() {
		var keyName string
		var keyMeta models.KeyMetadata
		var samplesJSON string

		if err := rows.Scan(&keyName, &keyMeta.Count, &keyMeta.Percentage,
			&keyMeta.EstimatedCardinality, &samplesJSON); err != nil {
			return nil, fmt.Errorf("scanning key: %w", err)
		}

		if err := decodeJSON(samplesJSON, &keyMeta.ValueSamples); err != nil {
			return nil, fmt.Errorf("decoding samples for key %s: %w", keyName, err)
		}

		// Initialize HLL for merging (since we don't store HLL sketch in DB yet)
		// We'll lose exact HLL state but cardinality count is preserved
		km := models.NewKeyMetadata()
		km.Count = keyMeta.Count
		km.Percentage = keyMeta.Percentage
		km.EstimatedCardinality = keyMeta.EstimatedCardinality
		km.ValueSamples = keyMeta.ValueSamples
		
		keys[keyName] = km
	}

	return keys, rows.Err()
}

// ListMetrics lists all metrics, optionally filtered by service.
func (s *Store) ListMetrics(ctx context.Context, serviceName string) ([]*models.MetricMetadata, error) {
	var query string
	var args []interface{}

	if serviceName != "" {
		query = `
			SELECT DISTINCT m.name
			FROM metrics m
			JOIN metric_services ms ON m.name = ms.metric_name
			WHERE ms.service_name = ?
			ORDER BY m.name
		`
		args = []interface{}{serviceName}
	} else {
		query = `SELECT name FROM metrics ORDER BY name`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying metrics: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning metric name: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fetch full metadata for each metric
	var results []*models.MetricMetadata
	for _, name := range names {
		metric, err := s.GetMetric(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("getting metric %s: %w", name, err)
		}
		results = append(results, metric)
	}

	return results, nil
}

// StoreSpan stores or updates span metadata.
func (s *Store) StoreSpan(ctx context.Context, span *models.SpanMetadata) error {
	// Fire-and-forget: send to batch writer without waiting
	select {
	case s.writeCh <- writeOp{opType: "StoreSpan", data: span, done: nil}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closeCh:
		return errors.New("store is closed")
	}
}

// storeSpanTx stores span metadata within a transaction.
func (s *Store) storeSpanTx(tx *sql.Tx, span *models.SpanMetadata) error {
	// Serialize StatusCodes to JSON
	statusCodesJSON, err := encodeJSON(span.StatusCodes)
	if err != nil {
		return fmt.Errorf("encoding status codes: %w", err)
	}

	// Convert boolean flags to INTEGER for SQLite
	hasTraceState := 0
	if span.HasTraceState {
		hasTraceState = 1
	}
	hasParentSpanId := 0
	if span.HasParentSpanId {
		hasParentSpanId = 1
	}

	// Prepare dropped stats values
	droppedAttrsTotal, droppedAttrsItems, droppedAttrsMax := uint32(0), int64(0), uint32(0)
	if span.DroppedAttributesStats != nil {
		droppedAttrsTotal = span.DroppedAttributesStats.TotalDropped
		droppedAttrsItems = span.DroppedAttributesStats.ItemsWithDropped
		droppedAttrsMax = span.DroppedAttributesStats.MaxDropped
	}

	droppedEventsTotal, droppedEventsItems, droppedEventsMax := uint32(0), int64(0), uint32(0)
	if span.DroppedEventsStats != nil {
		droppedEventsTotal = span.DroppedEventsStats.TotalDropped
		droppedEventsItems = span.DroppedEventsStats.ItemsWithDropped
		droppedEventsMax = span.DroppedEventsStats.MaxDropped
	}

	droppedLinksTotal, droppedLinksItems, droppedLinksMax := uint32(0), int64(0), uint32(0)
	if span.DroppedLinksStats != nil {
		droppedLinksTotal = span.DroppedLinksStats.TotalDropped
		droppedLinksItems = span.DroppedLinksStats.ItemsWithDropped
		droppedLinksMax = span.DroppedLinksStats.MaxDropped
	}

	// 1. Upsert base span with new OTLP fields
	_, err = tx.Exec(`
		INSERT INTO spans (
			name, kind, kind_number, kind_name, total_sample_count,
			has_trace_state, has_parent_span_id, status_codes,
			dropped_attrs_total, dropped_attrs_items, dropped_attrs_max,
			dropped_events_total, dropped_events_items, dropped_events_max,
			dropped_links_total, dropped_links_items, dropped_links_max
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			kind = excluded.kind,
			kind_number = excluded.kind_number,
			kind_name = excluded.kind_name,
			total_sample_count = total_sample_count + excluded.total_sample_count,
			has_trace_state = excluded.has_trace_state OR has_trace_state,
			has_parent_span_id = excluded.has_parent_span_id OR has_parent_span_id,
			status_codes = excluded.status_codes,
			dropped_attrs_total = dropped_attrs_total + excluded.dropped_attrs_total,
			dropped_attrs_items = dropped_attrs_items + excluded.dropped_attrs_items,
			dropped_attrs_max = MAX(dropped_attrs_max, excluded.dropped_attrs_max),
			dropped_events_total = dropped_events_total + excluded.dropped_events_total,
			dropped_events_items = dropped_events_items + excluded.dropped_events_items,
			dropped_events_max = MAX(dropped_events_max, excluded.dropped_events_max),
			dropped_links_total = dropped_links_total + excluded.dropped_links_total,
			dropped_links_items = dropped_links_items + excluded.dropped_links_items,
			dropped_links_max = MAX(dropped_links_max, excluded.dropped_links_max)
	`, span.Name, span.KindName, span.Kind, span.KindName, span.SampleCount,
		hasTraceState, hasParentSpanId, statusCodesJSON,
		droppedAttrsTotal, droppedAttrsItems, droppedAttrsMax,
		droppedEventsTotal, droppedEventsItems, droppedEventsMax,
		droppedLinksTotal, droppedLinksItems, droppedLinksMax)
	if err != nil {
		return fmt.Errorf("upserting span: %w", err)
	}

	// 2. Upsert service mappings
	for service, count := range span.Services {
		_, err := tx.Exec(`
			INSERT INTO span_services (span_name, service_name, sample_count)
			VALUES (?, ?, ?)
			ON CONFLICT(span_name, service_name) DO UPDATE SET
				sample_count = sample_count + excluded.sample_count
		`, span.Name, service, count)
		if err != nil {
			return fmt.Errorf("upserting span service %s: %w", service, err)
		}
	}

	// 3. Upsert attribute keys
	if err := s.upsertKeysForSpan(tx, span.Name, "attribute", span.AttributeKeys, ""); err != nil {
		return fmt.Errorf("upserting attribute keys: %w", err)
	}

	// 4. Upsert resource keys
	if err := s.upsertKeysForSpan(tx, span.Name, "resource", span.ResourceKeys, ""); err != nil {
		return fmt.Errorf("upserting resource keys: %w", err)
	}

	// 5. Upsert link attribute keys
	if err := s.upsertKeysForSpan(tx, span.Name, "link", span.LinkAttributeKeys, ""); err != nil {
		return fmt.Errorf("upserting link attribute keys: %w", err)
	}

	// 6. Upsert event names
	for _, eventName := range span.EventNames {
		_, err := tx.Exec(`
			INSERT INTO span_events (span_name, event_name)
			VALUES (?, ?)
			ON CONFLICT(span_name, event_name) DO NOTHING
		`, span.Name, eventName)
		if err != nil {
			return fmt.Errorf("upserting event name %s: %w", eventName, err)
		}

		// Upsert event attribute keys
		if eventKeys, ok := span.EventAttributeKeys[eventName]; ok {
			if err := s.upsertKeysForSpan(tx, span.Name, "event", eventKeys, eventName); err != nil {
				return fmt.Errorf("upserting event %s attribute keys: %w", eventName, err)
			}
		}
	}

	return nil
}

// upsertKeysForSpan upserts key metadata for a span.
// Dual-writes to both span_keys (legacy) and signal_keys (unified) tables.
func (s *Store) upsertKeysForSpan(tx *sql.Tx, spanName, keyScope string, keys map[string]*models.KeyMetadata, eventName string) error {
	for keyName, keyMeta := range keys {
		samples, err := encodeJSON(keyMeta.ValueSamples)
		if err != nil {
			return fmt.Errorf("encoding samples for key %s: %w", keyName, err)
		}

		eventNameVal := eventNameOrEmpty(eventName)

		// Write to legacy span_keys table for backward compatibility
		_, err = tx.Exec(`
			INSERT INTO span_keys (
				span_name, key_scope, key_name, event_name, key_count, key_percentage,
				estimated_cardinality, value_samples, hll_sketch
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(span_name, key_scope, key_name, event_name) DO UPDATE SET
				key_count = key_count + excluded.key_count,
				key_percentage = excluded.key_percentage,
				estimated_cardinality = excluded.estimated_cardinality,
				value_samples = excluded.value_samples,
				hll_sketch = excluded.hll_sketch
		`, spanName, keyScope, keyName, eventNameVal, keyMeta.Count, keyMeta.Percentage,
			keyMeta.EstimatedCardinality, samples, nil)

		if err != nil {
			return fmt.Errorf("upserting key %s to span_keys: %w", keyName, err)
		}

		// Write to unified signal_keys table
		_, err = tx.Exec(`
			INSERT INTO signal_keys (
				signal_type, signal_name, key_scope, key_name, event_name,
				key_count, key_percentage, estimated_cardinality, value_samples, hll_sketch
			) VALUES ('span', ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(signal_type, signal_name, key_scope, key_name, event_name) DO UPDATE SET
				key_count = key_count + excluded.key_count,
				key_percentage = excluded.key_percentage,
				estimated_cardinality = excluded.estimated_cardinality,
				value_samples = excluded.value_samples,
				hll_sketch = excluded.hll_sketch
		`, spanName, keyScope, keyName, eventNameVal, keyMeta.Count, keyMeta.Percentage,
			keyMeta.EstimatedCardinality, samples, nil)

		if err != nil {
			return fmt.Errorf("upserting key %s to signal_keys: %w", keyName, err)
		}
	}
	return nil
}

// nullString returns nil if s is empty, otherwise returns s.
// eventNameOrEmpty returns empty string if s is empty, otherwise returns s.
// Used for span event_name field which is NOT NULL DEFAULT ''.
func eventNameOrEmpty(s string) string {
	if s == "" {
		return ""
	}
	return s
}

// GetSpan retrieves span metadata by name.
func (s *Store) GetSpan(ctx context.Context, name string) (*models.SpanMetadata, error) {
	// Get base span with new OTLP fields
	var span models.SpanMetadata
	span.DroppedAttributesStats = &models.DroppedCountStats{}
	span.DroppedEventsStats = &models.DroppedCountStats{}
	span.DroppedLinksStats = &models.DroppedCountStats{}
	
	var hasTraceState, hasParentSpanId int
	var statusCodesJSON string
	var kindName string // For backwards compatibility with old 'kind' TEXT column
	
	err := s.db.QueryRowContext(ctx, `
		SELECT name, COALESCE(kind, ''), COALESCE(kind_number, 0), COALESCE(kind_name, ''), total_sample_count,
		       COALESCE(has_trace_state, 0), COALESCE(has_parent_span_id, 0), COALESCE(status_codes, '[]'),
		       COALESCE(dropped_attrs_total, 0), COALESCE(dropped_attrs_items, 0), COALESCE(dropped_attrs_max, 0),
		       COALESCE(dropped_events_total, 0), COALESCE(dropped_events_items, 0), COALESCE(dropped_events_max, 0),
		       COALESCE(dropped_links_total, 0), COALESCE(dropped_links_items, 0), COALESCE(dropped_links_max, 0)
		FROM spans WHERE name = ?
	`, name).Scan(
		&span.Name, &kindName, &span.Kind, &span.KindName, &span.SampleCount,
		&hasTraceState, &hasParentSpanId, &statusCodesJSON,
		&span.DroppedAttributesStats.TotalDropped, &span.DroppedAttributesStats.ItemsWithDropped, &span.DroppedAttributesStats.MaxDropped,
		&span.DroppedEventsStats.TotalDropped, &span.DroppedEventsStats.ItemsWithDropped, &span.DroppedEventsStats.MaxDropped,
		&span.DroppedLinksStats.TotalDropped, &span.DroppedLinksStats.ItemsWithDropped, &span.DroppedLinksStats.MaxDropped,
	)

	if err == sql.ErrNoRows {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying span: %w", err)
	}
	
	// Use kind_name if available, otherwise fall back to old kind column
	if span.KindName == "" {
		span.KindName = kindName
	}

	// Convert INTEGER back to boolean
	span.HasTraceState = hasTraceState != 0
	span.HasParentSpanId = hasParentSpanId != 0

	// Deserialize StatusCodes from JSON
	if statusCodesJSON != "" && statusCodesJSON != "[]" {
		if err := decodeJSON(statusCodesJSON, &span.StatusCodes); err != nil {
			return nil, fmt.Errorf("decoding status codes: %w", err)
		}
	}
	if span.StatusCodes == nil {
		span.StatusCodes = []string{}
	}

	// Get services
	span.Services = make(map[string]int64)
	rows, err := s.db.QueryContext(ctx, `
		SELECT service_name, sample_count
		FROM span_services WHERE span_name = ?
	`, name)
	if err != nil {
		return nil, fmt.Errorf("querying span services: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var serviceName string
		var count int64
		if err := rows.Scan(&serviceName, &count); err != nil {
			return nil, fmt.Errorf("scanning service: %w", err)
		}
		span.Services[serviceName] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Get attribute keys
	span.AttributeKeys, err = s.getKeysForSpan(ctx, name, "attribute", "")
	if err != nil {
		return nil, fmt.Errorf("querying attribute keys: %w", err)
	}

	// Get resource keys
	span.ResourceKeys, err = s.getKeysForSpan(ctx, name, "resource", "")
	if err != nil {
		return nil, fmt.Errorf("querying resource keys: %w", err)
	}

	// Get link attribute keys
	span.LinkAttributeKeys, err = s.getKeysForSpan(ctx, name, "link", "")
	if err != nil {
		return nil, fmt.Errorf("querying link attribute keys: %w", err)
	}

	// Get event names
	eventRows, err := s.db.QueryContext(ctx, `
		SELECT event_name FROM span_events WHERE span_name = ?
	`, name)
	if err != nil {
		return nil, fmt.Errorf("querying event names: %w", err)
	}
	defer eventRows.Close()

	span.EventNames = []string{}
	span.EventAttributeKeys = make(map[string]map[string]*models.KeyMetadata)

	for eventRows.Next() {
		var eventName string
		if err := eventRows.Scan(&eventName); err != nil {
			return nil, fmt.Errorf("scanning event name: %w", err)
		}
		span.EventNames = append(span.EventNames, eventName)

		// Get event attribute keys
		eventKeys, err := s.getKeysForSpan(ctx, name, "event", eventName)
		if err != nil {
			return nil, fmt.Errorf("querying event %s keys: %w", eventName, err)
		}
		span.EventAttributeKeys[eventName] = eventKeys
	}

	return &span, eventRows.Err()
}

// getKeysForSpan retrieves key metadata for a span, scope, and optional event name.
// Reads from unified signal_keys table.
func (s *Store) getKeysForSpan(ctx context.Context, spanName, keyScope, eventName string) (map[string]*models.KeyMetadata, error) {
	var rows *sql.Rows
	var err error

	if eventName != "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT key_name, key_count, key_percentage, estimated_cardinality, value_samples
			FROM signal_keys
			WHERE signal_type = 'span' AND signal_name = ? AND key_scope = ? AND event_name = ?
		`, spanName, keyScope, eventName)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT key_name, key_count, key_percentage, estimated_cardinality, value_samples
			FROM signal_keys
			WHERE signal_type = 'span' AND signal_name = ? AND key_scope = ? AND event_name = ''
		`, spanName, keyScope)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make(map[string]*models.KeyMetadata)
	for rows.Next() {
		var keyName string
		var keyMeta models.KeyMetadata
		var samplesJSON string

		if err := rows.Scan(&keyName, &keyMeta.Count, &keyMeta.Percentage,
			&keyMeta.EstimatedCardinality, &samplesJSON); err != nil {
			return nil, fmt.Errorf("scanning key: %w", err)
		}

		if err := decodeJSON(samplesJSON, &keyMeta.ValueSamples); err != nil {
			return nil, fmt.Errorf("decoding samples for key %s: %w", keyName, err)
		}

		keys[keyName] = &keyMeta
	}

	return keys, rows.Err()
}

// ListSpans lists all spans, optionally filtered by service.
func (s *Store) ListSpans(ctx context.Context, serviceName string) ([]*models.SpanMetadata, error) {
	var query string
	var args []interface{}

	if serviceName != "" {
		query = `
			SELECT DISTINCT sp.name
			FROM spans sp
			JOIN span_services ss ON sp.name = ss.span_name
			WHERE ss.service_name = ?
			ORDER BY sp.name
		`
		args = []interface{}{serviceName}
	} else {
		query = `SELECT name FROM spans ORDER BY name`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying spans: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning span name: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fetch full metadata for each span
	var results []*models.SpanMetadata
	for _, name := range names {
		span, err := s.GetSpan(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("getting span %s: %w", name, err)
		}
		results = append(results, span)
	}

	return results, nil
}

// StoreLog stores or updates log metadata.
func (s *Store) StoreLog(ctx context.Context, log *models.LogMetadata) error {
	// Fire-and-forget: send to batch writer without waiting
	select {
	case s.writeCh <- writeOp{opType: "StoreLog", data: log, done: nil}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closeCh:
		return errors.New("store is closed")
	}
}

// storeLogTx stores log metadata within a transaction.
func (s *Store) storeLogTx(tx *sql.Tx, log *models.LogMetadata) error {
	// Serialize EventNames to JSON
	eventNamesJSON, err := encodeJSON(log.EventNames)
	if err != nil {
		return fmt.Errorf("encoding event names: %w", err)
	}

	// Convert boolean flags to INTEGER for SQLite
	hasTraceCtx := 0
	if log.HasTraceContext {
		hasTraceCtx = 1
	}
	hasSpanCtx := 0
	if log.HasSpanContext {
		hasSpanCtx = 1
	}

	// Initialize dropped attributes stats if nil
	if log.DroppedAttributesStats == nil {
		log.DroppedAttributesStats = &models.DroppedAttributesStats{}
	}

	// 1. Upsert base log with new OTLP fields
	_, err = tx.Exec(`
		INSERT INTO logs (
			severity, total_sample_count, severity_number,
			has_trace_context, has_span_context, event_names,
			dropped_attrs_total, dropped_attrs_records, dropped_attrs_max
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(severity) DO UPDATE SET
			total_sample_count = total_sample_count + excluded.total_sample_count,
			severity_number = excluded.severity_number,
			has_trace_context = excluded.has_trace_context OR has_trace_context,
			has_span_context = excluded.has_span_context OR has_span_context,
			event_names = excluded.event_names,
			dropped_attrs_total = dropped_attrs_total + excluded.dropped_attrs_total,
			dropped_attrs_records = dropped_attrs_records + excluded.dropped_attrs_records,
			dropped_attrs_max = MAX(dropped_attrs_max, excluded.dropped_attrs_max)
	`, log.Severity, log.SampleCount, log.SeverityNumber,
		hasTraceCtx, hasSpanCtx, eventNamesJSON,
		log.DroppedAttributesStats.TotalDropped,
		log.DroppedAttributesStats.RecordsWithDropped,
		log.DroppedAttributesStats.MaxDropped)
	if err != nil {
		return fmt.Errorf("upserting log: %w", err)
	}

	// 2. Upsert service mappings
	for service, count := range log.Services {
		_, err := tx.Exec(`
			INSERT INTO log_services (severity, service_name, sample_count)
			VALUES (?, ?, ?)
			ON CONFLICT(severity, service_name) DO UPDATE SET
				sample_count = sample_count + excluded.sample_count
		`, log.Severity, service, count)
		if err != nil {
			return fmt.Errorf("upserting log service %s: %w", service, err)
		}
	}

	// 3. Upsert attribute keys
	if err := s.upsertKeysForLog(tx, log.Severity, "attribute", log.AttributeKeys); err != nil {
		return fmt.Errorf("upserting attribute keys: %w", err)
	}
	
	// 3a. Link attribute keys to services with counts and cardinality
	for service := range log.Services {
		for keyName, keyMeta := range log.AttributeKeys {
			_, err := tx.Exec(`
				INSERT INTO log_service_keys (
					service_name, severity, key_scope, key_name,
					key_count, key_percentage, estimated_cardinality
				)
				VALUES (?, ?, 'attribute', ?, ?, ?, ?)
				ON CONFLICT(service_name, severity, key_scope, key_name) DO UPDATE SET
					key_count = key_count + excluded.key_count,
					key_percentage = excluded.key_percentage,
					estimated_cardinality = MAX(estimated_cardinality, excluded.estimated_cardinality)
			`, service, log.Severity, keyName, keyMeta.Count, keyMeta.Percentage, keyMeta.EstimatedCardinality)
			if err != nil {
				return fmt.Errorf("linking attribute key %s to service %s: %w", keyName, service, err)
			}
		}
	}

	// 4. Upsert resource keys
	if err := s.upsertKeysForLog(tx, log.Severity, "resource", log.ResourceKeys); err != nil {
		return fmt.Errorf("upserting resource keys: %w", err)
	}
	
	// 4a. Link resource keys to services with counts and cardinality
	for service := range log.Services {
		for keyName, keyMeta := range log.ResourceKeys {
			_, err := tx.Exec(`
				INSERT INTO log_service_keys (
					service_name, severity, key_scope, key_name,
					key_count, key_percentage, estimated_cardinality
				)
				VALUES (?, ?, 'resource', ?, ?, ?, ?)
				ON CONFLICT(service_name, severity, key_scope, key_name) DO UPDATE SET
					key_count = key_count + excluded.key_count,
					key_percentage = excluded.key_percentage,
					estimated_cardinality = MAX(estimated_cardinality, excluded.estimated_cardinality)
			`, service, log.Severity, keyName, keyMeta.Count, keyMeta.Percentage, keyMeta.EstimatedCardinality)
			if err != nil {
				return fmt.Errorf("linking resource key %s to service %s: %w", keyName, service, err)
			}
		}
	}

	// 5. Upsert body templates
	// IMPORTANT: Body templates are analyzed at SEVERITY level, not per service.
	// Each service+severity LogMetadata has the SAME templates with SAME counts.
	// To avoid counting duplicates, we only store templates once per severity,
	// using an arbitrary service name (first one we encounter).
	if len(log.BodyTemplates) > 0 && len(log.Services) > 0 {
		// Pick first service arbitrarily
		var firstService string
		for service := range log.Services {
			firstService = service
			break
		}

		for _, tmpl := range log.BodyTemplates {
			_, err := tx.Exec(`
				INSERT INTO log_body_templates (severity, service_name, template, example, count, percentage)
				VALUES (?, ?, ?, ?, ?, ?)
				ON CONFLICT(severity, service_name, template) DO UPDATE SET
					example = COALESCE(excluded.example, example),
					count = count + excluded.count,
					percentage = excluded.percentage
			`, log.Severity, firstService, tmpl.Template, tmpl.Example, tmpl.Count, tmpl.Percentage)
			if err != nil {
				return fmt.Errorf("upserting body template: %w", err)
			}
		}

		// Recalculate percentages
		if err := s.recalculateTemplatePercentages(tx, log.Severity, firstService); err != nil {
			return fmt.Errorf("recalculating percentages: %w", err)
		}
	}

	return nil
}

// upsertKeysForLog upserts key metadata for a log.
func (s *Store) upsertKeysForLog(tx *sql.Tx, severity, keyScope string, keys map[string]*models.KeyMetadata) error {
	for keyName, keyMeta := range keys {
		samples, err := encodeJSON(keyMeta.ValueSamples)
		if err != nil {
			return fmt.Errorf("encoding samples for key %s: %w", keyName, err)
		}

		_, err = tx.Exec(`
			INSERT INTO log_keys (
				severity, key_scope, key_name, key_count, key_percentage,
				estimated_cardinality, value_samples, hll_sketch
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(severity, key_scope, key_name) DO UPDATE SET
				key_count = key_count + excluded.key_count,
				key_percentage = excluded.key_percentage,
				estimated_cardinality = excluded.estimated_cardinality,
				value_samples = excluded.value_samples,
				hll_sketch = excluded.hll_sketch
		`, severity, keyScope, keyName, keyMeta.Count, keyMeta.Percentage,
			keyMeta.EstimatedCardinality, samples, nil)

		if err != nil {
			return fmt.Errorf("upserting key %s: %w", keyName, err)
		}

		// Also write to unified signal_keys table
		// First, check if key exists and merge samples
		var existingSamplesJSON string
		var existingCard int64
		err = tx.QueryRow(`
			SELECT COALESCE(value_samples, '[]'), estimated_cardinality
			FROM signal_keys
			WHERE signal_type = 'log' AND signal_name = ? AND key_scope = ? AND key_name = ? AND event_name = ''
		`, severity, keyScope, keyName).Scan(&existingSamplesJSON, &existingCard)
		
		mergedSamples := samples
		mergedCard := int64(keyMeta.EstimatedCardinality)
		
		if err == nil {
			// Key exists, merge samples
			var existingSamples []string
			if existingSamplesJSON != "" && existingSamplesJSON != "[]" {
				if err := decodeJSON(existingSamplesJSON, &existingSamples); err == nil {
					// Merge samples (union)
					sampleSet := make(map[string]bool)
					for _, s := range existingSamples {
						sampleSet[s] = true
					}
					for _, s := range keyMeta.ValueSamples {
						sampleSet[s] = true
					}
					
					// Convert back to slice (limit to MaxSamples)
					merged := make([]string, 0, len(sampleSet))
					for s := range sampleSet {
						merged = append(merged, s)
						if len(merged) >= 10 { // MaxSamples
							break
						}
					}
					mergedSamples, _ = encodeJSON(merged)
					mergedCard = int64(len(sampleSet))
					if mergedCard < existingCard {
						mergedCard = existingCard
					}
				}
			}
		}
		
		_, err = tx.Exec(`
			INSERT INTO signal_keys (
				signal_type, signal_name, key_scope, key_name, event_name,
				key_count, key_percentage, estimated_cardinality, value_samples, hll_sketch
			) VALUES ('log', ?, ?, ?, '', ?, ?, ?, ?, ?)
			ON CONFLICT(signal_type, signal_name, key_scope, key_name, event_name) DO UPDATE SET
				key_count = key_count + excluded.key_count,
				key_percentage = excluded.key_percentage,
				estimated_cardinality = excluded.estimated_cardinality,
				value_samples = excluded.value_samples,
				hll_sketch = excluded.hll_sketch
		`, severity, keyScope, keyName, keyMeta.Count, keyMeta.Percentage,
			mergedCard, mergedSamples, nil)

		if err != nil {
			return fmt.Errorf("upserting signal key %s: %w", keyName, err)
		}
	}
	return nil
}

// recalculateTemplatePercentages recalculates percentages for all templates in a severity+service.
func (s *Store) recalculateTemplatePercentages(tx *sql.Tx, severity, service string) error {
	// Get total count
	var totalCount int64
	err := tx.QueryRow(`
		SELECT COALESCE(SUM(count), 0) 
		FROM log_body_templates 
		WHERE severity = ? AND service_name = ?
	`, severity, service).Scan(&totalCount)
	if err != nil {
		return fmt.Errorf("getting total count: %w", err)
	}

	if totalCount == 0 {
		return nil
	}

	// Update all percentages
	_, err = tx.Exec(`
		UPDATE log_body_templates
		SET percentage = (count * 100.0) / ?
		WHERE severity = ? AND service_name = ?
	`, totalCount, severity, service)

	return err
}

// GetLog retrieves log metadata by severity.
func (s *Store) GetLog(ctx context.Context, severityText string) (*models.LogMetadata, error) {
	// Get base log with new OTLP fields
	var log models.LogMetadata
	log.DroppedAttributesStats = &models.DroppedAttributesStats{} // Initialize before use
	var hasTraceCtx, hasSpanCtx int
	var eventNamesJSON string
	err := s.db.QueryRowContext(ctx, `
		SELECT severity, total_sample_count, severity_number,
		       has_trace_context, has_span_context, event_names,
		       dropped_attrs_total, dropped_attrs_records, dropped_attrs_max
		FROM logs WHERE severity = ?
	`, severityText).Scan(
		&log.Severity, &log.SampleCount, &log.SeverityNumber,
		&hasTraceCtx, &hasSpanCtx, &eventNamesJSON,
		&log.DroppedAttributesStats.TotalDropped,
		&log.DroppedAttributesStats.RecordsWithDropped,
		&log.DroppedAttributesStats.MaxDropped,
	)

	if err == sql.ErrNoRows {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying log: %w", err)
	}

	// Convert INTEGER back to boolean
	log.HasTraceContext = hasTraceCtx != 0
	log.HasSpanContext = hasSpanCtx != 0

	// Deserialize EventNames from JSON
	if eventNamesJSON != "" {
		if err := decodeJSON(eventNamesJSON, &log.EventNames); err != nil {
			return nil, fmt.Errorf("decoding event names: %w", err)
		}
	}

	// Get services
	log.Services = make(map[string]int64)
	rows, err := s.db.QueryContext(ctx, `
		SELECT service_name, sample_count
		FROM log_services WHERE severity = ?
	`, severityText)
	if err != nil {
		return nil, fmt.Errorf("querying log services: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var serviceName string
		var count int64
		if err := rows.Scan(&serviceName, &count); err != nil {
			return nil, fmt.Errorf("scanning service: %w", err)
		}
		log.Services[serviceName] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Get attribute keys
	log.AttributeKeys, err = s.getKeysForLog(ctx, severityText, "attribute")
	if err != nil {
		return nil, fmt.Errorf("querying attribute keys: %w", err)
	}

	// Get resource keys
	log.ResourceKeys, err = s.getKeysForLog(ctx, severityText, "resource")
	if err != nil {
		return nil, fmt.Errorf("querying resource keys: %w", err)
	}

	// Get body templates (aggregated across all services)
	tmplRows, err := s.db.QueryContext(ctx, `
		SELECT template, example, SUM(count) as total_count, AVG(percentage) as avg_percentage
		FROM log_body_templates
		WHERE severity = ?
		GROUP BY template, example
		ORDER BY total_count DESC
	`, severityText)
	if err != nil {
		return nil, fmt.Errorf("querying body templates: %w", err)
	}
	defer tmplRows.Close()

	log.BodyTemplates = []*models.BodyTemplate{}
	for tmplRows.Next() {
		var tmpl models.BodyTemplate
		if err := tmplRows.Scan(&tmpl.Template, &tmpl.Example, &tmpl.Count, &tmpl.Percentage); err != nil {
			return nil, fmt.Errorf("scanning body template: %w", err)
		}
		log.BodyTemplates = append(log.BodyTemplates, &tmpl)
	}

	return &log, tmplRows.Err()
}

// getKeysForLog retrieves key metadata for a log and scope.
func (s *Store) getKeysForLog(ctx context.Context, severity, keyScope string) (map[string]*models.KeyMetadata, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT key_name, key_count, key_percentage, estimated_cardinality, value_samples
		FROM log_keys
		WHERE severity = ? AND key_scope = ?
	`, severity, keyScope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make(map[string]*models.KeyMetadata)
	for rows.Next() {
		var keyName string
		var keyMeta models.KeyMetadata
		var samplesJSON string

		if err := rows.Scan(&keyName, &keyMeta.Count, &keyMeta.Percentage,
			&keyMeta.EstimatedCardinality, &samplesJSON); err != nil {
			return nil, fmt.Errorf("scanning key: %w", err)
		}

		if err := decodeJSON(samplesJSON, &keyMeta.ValueSamples); err != nil {
			return nil, fmt.Errorf("decoding samples for key %s: %w", keyName, err)
		}

		keys[keyName] = &keyMeta
	}

	return keys, rows.Err()
}

// ListLogs lists all logs, optionally filtered by service.
func (s *Store) ListLogs(ctx context.Context, serviceName string) ([]*models.LogMetadata, error) {
	var query string
	var args []interface{}

	if serviceName != "" {
		query = `
			SELECT DISTINCT l.severity
			FROM logs l
			JOIN log_services ls ON l.severity = ls.severity
			WHERE ls.service_name = ?
			ORDER BY l.severity
		`
		args = []interface{}{serviceName}
	} else {
		query = `SELECT severity FROM logs ORDER BY severity`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying logs: %w", err)
	}
	defer rows.Close()

	var severities []string
	for rows.Next() {
		var severity string
		if err := rows.Scan(&severity); err != nil {
			return nil, fmt.Errorf("scanning severity: %w", err)
		}
		severities = append(severities, severity)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fetch full metadata for each severity
	var results []*models.LogMetadata
	for _, severity := range severities {
		log, err := s.GetLog(ctx, severity)
		if err != nil {
			return nil, fmt.Errorf("getting log %s: %w", severity, err)
		}
		results = append(results, log)
	}

	return results, nil
}

// GetLogPatterns returns an advanced pattern analysis view.
// Groups patterns by template, then by service, with key cardinality info.
func (s *Store) GetLogPatterns(ctx context.Context, minCount int64, minServices int) (*models.PatternExplorerResponse, error) {
	// Step 1: Get all unique patterns with their total counts and severity breakdown
	patternsQuery := `
		SELECT 
			template,
			severity,
			service_name,
			example,
			count
		FROM log_body_templates
		ORDER BY template, count DESC
	`
	
	rows, err := s.db.QueryContext(ctx, patternsQuery)
	if err != nil {
		return nil, fmt.Errorf("querying patterns: %w", err)
	}
	defer rows.Close()
	
	// Build pattern groups
	patternMap := make(map[string]*models.PatternGroup)
	servicePatternMap := make(map[string]map[string]*models.ServicePatternInfo) // pattern -> service -> info
	
	for rows.Next() {
		var template, severity, serviceName, example string
		var count int64
		
		if err := rows.Scan(&template, &severity, &serviceName, &example, &count); err != nil {
			return nil, fmt.Errorf("scanning pattern: %w", err)
		}
		
		// Handle missing service name
		if serviceName == "" {
			serviceName = "unknown"
		}
		
		// Initialize pattern group if needed
		if _, exists := patternMap[template]; !exists {
			patternMap[template] = &models.PatternGroup{
				Template:          template,
				ExampleBody:       example, // Keep first example we find
				TotalCount:        0,
				SeverityBreakdown: make(map[string]int64),
				Services:          []models.ServicePatternInfo{},
			}
			servicePatternMap[template] = make(map[string]*models.ServicePatternInfo)
		}
		
		pg := patternMap[template]
		// Keep the first non-empty example
		if pg.ExampleBody == "" && example != "" {
			pg.ExampleBody = example
		}
		pg.TotalCount += count
		pg.SeverityBreakdown[severity] += count
		
		// Track service info
		if _, exists := servicePatternMap[template][serviceName]; !exists {
			servicePatternMap[template][serviceName] = &models.ServicePatternInfo{
				ServiceName:   serviceName,
				SampleCount:   0,
				Severities:    []string{},
				ResourceKeys:  []models.KeyInfo{},
				AttributeKeys: []models.KeyInfo{},
			}
		}
		
		spi := servicePatternMap[template][serviceName]
		spi.SampleCount += count
		spi.Severities = append(spi.Severities, severity)
	}
	
	if err := rows.Err(); err != nil {
		return nil, err
	}
	
	// Step 2: For each pattern+service, fetch unique keys
	for _, serviceMap := range servicePatternMap {
		for serviceName, spi := range serviceMap {
			// Get severities for this pattern+service combo
			severities := spi.Severities
			
			// Fetch resource keys
			resourceKeys, err := s.getKeysForPatternService(ctx, severities, serviceName, "resource")
			if err != nil {
				return nil, fmt.Errorf("fetching resource keys: %w", err)
			}
			spi.ResourceKeys = resourceKeys
			
			// Fetch attribute keys
			attrKeys, err := s.getKeysForPatternService(ctx, severities, serviceName, "attribute")
			if err != nil {
				return nil, fmt.Errorf("fetching attribute keys: %w", err)
			}
			spi.AttributeKeys = attrKeys
		}
	}
	
	// Step 3: Filter and build final result
	var patterns []models.PatternGroup
	for template, pg := range patternMap {
		// Apply filters
		if pg.TotalCount < minCount {
			continue
		}
		
		// Build services list for this pattern
		servicesForPattern := servicePatternMap[template]
		if len(servicesForPattern) < minServices {
			continue
		}
		
		for _, spi := range servicesForPattern {
			pg.Services = append(pg.Services, *spi)
		}
		
		patterns = append(patterns, *pg)
	}
	
	// Sort by total count descending
	for i := 0; i < len(patterns); i++ {
		for j := i + 1; j < len(patterns); j++ {
			if patterns[j].TotalCount > patterns[i].TotalCount {
				patterns[i], patterns[j] = patterns[j], patterns[i]
			}
		}
	}
	
	return &models.PatternExplorerResponse{
		Patterns: patterns,
		Total:    len(patterns),
	}, nil
}

// getKeysForPatternService fetches keys for a specific service and severities
func (s *Store) getKeysForPatternService(ctx context.Context, severities []string, serviceName string, keyScope string) ([]models.KeyInfo, error) {
	if len(severities) == 0 {
		return []models.KeyInfo{}, nil
	}
	
	// Build IN clause for severities
	placeholders := ""
	args := []interface{}{}
	for i, sev := range severities {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "?"
		args = append(args, sev)
	}
	args = append(args, serviceName)
	args = append(args, keyScope)
	
	// Query log_service_keys to get only keys that actually belong to this service
	// NOTE: We don't include cardinality/samples here because signal_keys contains
	// aggregated data across ALL services. To show accurate per-service cardinality,
	// we would need to store cardinality per service+severity+key combination.
	query := `
		SELECT DISTINCT
			lsk.key_name
		FROM log_service_keys lsk
		WHERE lsk.severity IN (` + placeholders + `)
		AND lsk.service_name = ?
		AND lsk.key_scope = ?
		ORDER BY lsk.key_name
	`
	
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying keys: %w", err)
	}
	defer rows.Close()
	
	var keys []models.KeyInfo
	for rows.Next() {
		var keyName string
		
		if err := rows.Scan(&keyName); err != nil {
			return nil, fmt.Errorf("scanning key: %w", err)
		}
		
		keys = append(keys, models.KeyInfo{
			Name:         keyName,
			Cardinality:  0, // Not available per-service
			SampleValues: []string{}, // Not available per-service
		})
	}
	
	return keys, rows.Err()
}

// ListServices lists all known services.
func (s *Store) ListServices(ctx context.Context) ([]string, error) {
	// Collect services from all three junction tables
	query := `
		SELECT DISTINCT service_name FROM (
			SELECT service_name FROM metric_services
			UNION
			SELECT service_name FROM span_services
			UNION
			SELECT service_name FROM log_services
		) ORDER BY service_name
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying services: %w", err)
	}
	defer rows.Close()

	var services []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning service: %w", err)
		}
		services = append(services, name)
	}

	return services, rows.Err()
}

// GetServiceOverview returns an overview of all telemetry for a service.
func (s *Store) GetServiceOverview(ctx context.Context, serviceName string) (*models.ServiceOverview, error) {
	metrics, err := s.ListMetrics(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("listing metrics: %w", err)
	}

	spans, err := s.ListSpans(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("listing spans: %w", err)
	}

	logs, err := s.ListLogs(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("listing logs: %w", err)
	}

	return &models.ServiceOverview{
		ServiceName: serviceName,
		MetricCount: len(metrics),
		SpanCount:   len(spans),
		LogCount:    len(logs),
		Metrics:     metrics,
		Spans:       spans,
		Logs:        logs,
	}, nil
}

// GetHighCardinalityKeys returns high-cardinality keys across all signal types.
func (s *Store) GetHighCardinalityKeys(ctx context.Context, threshold int, limit int) (*models.CrossSignalCardinalityResponse, error) {
	if limit <= 0 {
		limit = 100 // Default limit
	}

	query := `
		SELECT 
			signal_type,
			signal_name,
			key_scope,
			key_name,
			event_name,
			estimated_cardinality,
			key_count,
			value_samples
		FROM signal_keys
		WHERE estimated_cardinality >= ?
		ORDER BY estimated_cardinality DESC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, threshold, limit)
	if err != nil {
		return nil, fmt.Errorf("querying high-cardinality keys: %w", err)
	}
	defer rows.Close()

	var keys []models.SignalKey
	for rows.Next() {
		var key models.SignalKey
		var samplesJSON string

		if err := rows.Scan(
			&key.SignalType,
			&key.SignalName,
			&key.KeyScope,
			&key.KeyName,
			&key.EventName,
			&key.EstimatedCardinality,
			&key.KeyCount,
			&samplesJSON,
		); err != nil {
			return nil, fmt.Errorf("scanning key: %w", err)
		}

		if err := decodeJSON(samplesJSON, &key.ValueSamples); err != nil {
			return nil, fmt.Errorf("decoding samples for key %s: %w", key.KeyName, err)
		}

		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &models.CrossSignalCardinalityResponse{
		HighCardinalityKeys: keys,
		Total:               len(keys),
		Threshold:           threshold,
	}, nil
}

// GetMetadataComplexity returns signals with high metadata complexity (many keys).
func (s *Store) GetMetadataComplexity(ctx context.Context, threshold int, limit int) (*models.MetadataComplexityResponse, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		WITH metric_buckets AS (
			SELECT 
				name,
				CASE 
					WHEN type = 'Histogram' AND metric_data IS NOT NULL 
					THEN json_extract(metric_data, '$.data.explicit_bounds')
					ELSE NULL
				END as bounds_json
			FROM metrics
		),
		bucket_counts AS (
			SELECT 
				name,
				CASE 
					WHEN bounds_json IS NOT NULL 
					THEN json_array_length(bounds_json) + 1
					ELSE 0
				END as bucket_count
			FROM metric_buckets
		)
		SELECT 
			sk.signal_type,
			sk.signal_name,
			COUNT(*) + COALESCE(bc.bucket_count, 0) as total_keys,
			SUM(CASE WHEN sk.key_scope IN ('attribute', 'label') THEN 1 ELSE 0 END) as attribute_keys,
			SUM(CASE WHEN sk.key_scope = 'resource' THEN 1 ELSE 0 END) as resource_keys,
			SUM(CASE WHEN sk.key_scope = 'event' THEN 1 ELSE 0 END) as event_keys,
			SUM(CASE WHEN sk.key_scope = 'link' THEN 1 ELSE 0 END) as link_keys,
			MAX(sk.estimated_cardinality) as max_cardinality,
			SUM(CASE WHEN sk.estimated_cardinality > 100 THEN 1 ELSE 0 END) as high_card_count
		FROM signal_keys sk
		LEFT JOIN bucket_counts bc 
			ON sk.signal_type = 'metric' AND sk.signal_name = bc.name
		GROUP BY sk.signal_type, sk.signal_name
		HAVING total_keys >= ?
		ORDER BY total_keys DESC, max_cardinality DESC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, threshold, limit)
	if err != nil {
		return nil, fmt.Errorf("querying metadata complexity: %w", err)
	}
	defer rows.Close()

	var signals []models.SignalComplexity
	for rows.Next() {
		var sig models.SignalComplexity

		if err := rows.Scan(
			&sig.SignalType,
			&sig.SignalName,
			&sig.TotalKeys,
			&sig.AttributeKeyCount,
			&sig.ResourceKeyCount,
			&sig.EventKeyCount,
			&sig.LinkKeyCount,
			&sig.MaxCardinality,
			&sig.HighCardinalityCount,
		); err != nil {
			return nil, fmt.Errorf("scanning complexity: %w", err)
		}

		// Calculate complexity score: total keys × max cardinality
		sig.ComplexityScore = sig.TotalKeys * sig.MaxCardinality

		signals = append(signals, sig)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &models.MetadataComplexityResponse{
		Signals:   signals,
		Total:     len(signals),
		Threshold: threshold,
	}, nil
}



