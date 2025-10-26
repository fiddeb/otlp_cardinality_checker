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
	"sync"
	"time"

	"github.com/fidde/otlp_cardinality_checker/internal/analyzer/autotemplate"
	"github.com/fidde/otlp_cardinality_checker/pkg/models"
	_ "modernc.org/sqlite"
)

//go:embed migrations/001_initial_schema.up.sql
var migrationSQL string

// Store is a SQLite-backed storage for telemetry metadata.
type Store struct {
	db *sql.DB

	// Batch writer
	writeCh   chan writeOp
	closeCh   chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup

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
		BatchSize:       100,
		FlushInterval:   100 * time.Millisecond,
	}
}

// New creates a new SQLite store with the given configuration.
func New(cfg Config) (*Store, error) {
	// Open database
	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Set pragmas for performance
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-64000", // 64MB cache
		"PRAGMA temp_store=MEMORY",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("setting pragma: %w", err)
		}
	}

	// Run migrations
	if _, err := db.Exec(migrationSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	store := &Store{
		db:              db,
		writeCh:         make(chan writeOp, 1000),
		closeCh:         make(chan struct{}),
		useAutoTemplate: cfg.UseAutoTemplate,
		autoTemplateCfg: cfg.AutoTemplateCfg,
	}

	// Start batch writer goroutine
	store.wg.Add(1)
	go store.batchWriter(cfg.BatchSize, cfg.FlushInterval)

	return store, nil
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
		err = s.db.Close()
	})
	return err
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
	done := make(chan error, 1)

	select {
	case s.writeCh <- writeOp{opType: "StoreMetric", data: metric, done: done}:
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closeCh:
		return errors.New("store is closed")
	}
}

// storeMetricTx stores metric metadata within a transaction.
func (s *Store) storeMetricTx(tx *sql.Tx, metric *models.MetricMetadata) error {
	// 1. Upsert base metric
	_, err := tx.Exec(`
		INSERT INTO metrics (name, type, unit, description, total_sample_count)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			type = excluded.type,
			unit = COALESCE(excluded.unit, unit),
			description = COALESCE(excluded.description, description),
			total_sample_count = total_sample_count + excluded.total_sample_count
	`, metric.Name, metric.Type, metric.Unit, metric.Description, metric.SampleCount)
	if err != nil {
		return fmt.Errorf("upserting metric: %w", err)
	}

	// 2. Upsert service mappings
	for service, count := range metric.Services {
		_, err := tx.Exec(`
			INSERT INTO metric_services (metric_name, service_name, sample_count)
			VALUES (?, ?, ?)
			ON CONFLICT(metric_name, service_name) DO UPDATE SET
				sample_count = sample_count + excluded.sample_count
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
func (s *Store) upsertKeysForMetric(tx *sql.Tx, metricName, keyScope string, keys map[string]*models.KeyMetadata) error {
	for keyName, keyMeta := range keys {
		samples, err := encodeJSON(keyMeta.ValueSamples)
		if err != nil {
			return fmt.Errorf("encoding samples for key %s: %w", keyName, err)
		}

		_, err = tx.Exec(`
			INSERT INTO metric_keys (
				metric_name, key_scope, key_name, key_count, key_percentage,
				estimated_cardinality, value_samples, hll_sketch
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(metric_name, key_scope, key_name) DO UPDATE SET
				key_count = key_count + excluded.key_count,
				key_percentage = excluded.key_percentage,
				estimated_cardinality = excluded.estimated_cardinality,
				value_samples = excluded.value_samples,
				hll_sketch = excluded.hll_sketch
		`, metricName, keyScope, keyName, keyMeta.Count, keyMeta.Percentage,
			keyMeta.EstimatedCardinality, samples, nil) // HLL sketch = nil for now

		if err != nil {
			return fmt.Errorf("upserting key %s: %w", keyName, err)
		}
	}
	return nil
}

// GetMetric retrieves metric metadata by name.
func (s *Store) GetMetric(ctx context.Context, name string) (*models.MetricMetadata, error) {
	// Get base metric
	var metric models.MetricMetadata
	err := s.db.QueryRowContext(ctx, `
		SELECT name, type, unit, description, total_sample_count
		FROM metrics WHERE name = ?
	`, name).Scan(&metric.Name, &metric.Type, &metric.Unit, &metric.Description, &metric.SampleCount)

	if err == sql.ErrNoRows {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying metric: %w", err)
	}

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

	return &metric, nil
}

// getKeysForMetric retrieves key metadata for a metric and scope.
func (s *Store) getKeysForMetric(ctx context.Context, metricName, keyScope string) (map[string]*models.KeyMetadata, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT key_name, key_count, key_percentage, estimated_cardinality, value_samples
		FROM metric_keys
		WHERE metric_name = ? AND key_scope = ?
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

		keys[keyName] = &keyMeta
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
	done := make(chan error, 1)

	select {
	case s.writeCh <- writeOp{opType: "StoreSpan", data: span, done: done}:
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closeCh:
		return errors.New("store is closed")
	}
}

// storeSpanTx stores span metadata within a transaction.
func (s *Store) storeSpanTx(tx *sql.Tx, span *models.SpanMetadata) error {
	// 1. Upsert base span
	_, err := tx.Exec(`
		INSERT INTO spans (name, kind, total_sample_count)
		VALUES (?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			kind = excluded.kind,
			total_sample_count = total_sample_count + excluded.total_sample_count
	`, span.Name, span.Kind, span.SampleCount)
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
func (s *Store) upsertKeysForSpan(tx *sql.Tx, spanName, keyScope string, keys map[string]*models.KeyMetadata, eventName string) error {
	for keyName, keyMeta := range keys {
		samples, err := encodeJSON(keyMeta.ValueSamples)
		if err != nil {
			return fmt.Errorf("encoding samples for key %s: %w", keyName, err)
		}

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
		`, spanName, keyScope, keyName, eventNameOrEmpty(eventName), keyMeta.Count, keyMeta.Percentage,
			keyMeta.EstimatedCardinality, samples, nil)

		if err != nil {
			return fmt.Errorf("upserting key %s: %w", keyName, err)
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
	// Get base span
	var span models.SpanMetadata
	err := s.db.QueryRowContext(ctx, `
		SELECT name, kind, total_sample_count
		FROM spans WHERE name = ?
	`, name).Scan(&span.Name, &span.Kind, &span.SampleCount)

	if err == sql.ErrNoRows {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying span: %w", err)
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
func (s *Store) getKeysForSpan(ctx context.Context, spanName, keyScope, eventName string) (map[string]*models.KeyMetadata, error) {
	var rows *sql.Rows
	var err error

	if eventName != "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT key_name, key_count, key_percentage, estimated_cardinality, value_samples
			FROM span_keys
			WHERE span_name = ? AND key_scope = ? AND event_name = ?
		`, spanName, keyScope, eventName)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT key_name, key_count, key_percentage, estimated_cardinality, value_samples
			FROM span_keys
			WHERE span_name = ? AND key_scope = ? AND event_name IS NULL
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
	done := make(chan error, 1)

	select {
	case s.writeCh <- writeOp{opType: "StoreLog", data: log, done: done}:
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closeCh:
		return errors.New("store is closed")
	}
}

// storeLogTx stores log metadata within a transaction.
func (s *Store) storeLogTx(tx *sql.Tx, log *models.LogMetadata) error {
	// 1. Upsert base log
	_, err := tx.Exec(`
		INSERT INTO logs (severity, total_sample_count)
		VALUES (?, ?)
		ON CONFLICT(severity) DO UPDATE SET
			total_sample_count = total_sample_count + excluded.total_sample_count
	`, log.Severity, log.SampleCount)
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

	// 4. Upsert resource keys
	if err := s.upsertKeysForLog(tx, log.Severity, "resource", log.ResourceKeys); err != nil {
		return fmt.Errorf("upserting resource keys: %w", err)
	}

	// 5. Upsert body templates (per service)
	if len(log.BodyTemplates) > 0 {
		// Body templates are stored per service in the model
		// We need to iterate through services and store their templates
		for _, tmpl := range log.BodyTemplates {
			// In the current model, templates don't have explicit service association
			// We'll store them for all services that have this severity
			for service := range log.Services {
				_, err := tx.Exec(`
					INSERT INTO log_body_templates (severity, service_name, template, example, count, percentage)
					VALUES (?, ?, ?, ?, ?, ?)
					ON CONFLICT(severity, service_name, template) DO UPDATE SET
						example = COALESCE(excluded.example, example),
						count = excluded.count,
						percentage = excluded.percentage
				`, log.Severity, service, tmpl.Template, tmpl.Example, tmpl.Count, tmpl.Percentage)
				if err != nil {
					return fmt.Errorf("upserting body template for service %s: %w", service, err)
				}
			}
		}

		// Recalculate percentages for each service
		for service := range log.Services {
			if err := s.recalculateTemplatePercentages(tx, log.Severity, service); err != nil {
				return fmt.Errorf("recalculating percentages for service %s: %w", service, err)
			}
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
	// Get base log
	var log models.LogMetadata
	err := s.db.QueryRowContext(ctx, `
		SELECT severity, total_sample_count
		FROM logs WHERE severity = ?
	`, severityText).Scan(&log.Severity, &log.SampleCount)

	if err == sql.ErrNoRows {
		return nil, models.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying log: %w", err)
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

