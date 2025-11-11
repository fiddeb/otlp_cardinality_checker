package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const schemaVersion = "1.0.0"

// InitializeSchema creates all required tables if they don't exist
func InitializeSchema(ctx context.Context, conn driver.Conn) error {
	// Create schema_version table first
	if err := createSchemaVersionTable(ctx, conn); err != nil {
		return fmt.Errorf("creating schema_version table: %w", err)
	}

	// Check current schema version
	currentVersion, err := getCurrentSchemaVersion(ctx, conn)
	if err != nil {
		return fmt.Errorf("checking schema version: %w", err)
	}

	if currentVersion != "" && currentVersion != schemaVersion {
		return fmt.Errorf("schema version mismatch: database has %s, code expects %s", currentVersion, schemaVersion)
	}

	// Create all tables
	tables := []struct {
		name string
		ddl  string
	}{
		{"metrics", metricsTableDDL},
		{"spans", spansTableDDL},
		{"logs", logsTableDDL},
		{"attribute_values", attributeValuesTableDDL},
		{"services", servicesTableDDL},
	}

	for _, table := range tables {
		if err := conn.Exec(ctx, table.ddl); err != nil {
			return fmt.Errorf("creating table %s: %w", table.name, err)
		}
	}

	// Update schema version
	if currentVersion == "" {
		if err := setSchemaVersion(ctx, conn, schemaVersion); err != nil {
			return fmt.Errorf("setting schema version: %w", err)
		}
	}

	return nil
}

func createSchemaVersionTable(ctx context.Context, conn driver.Conn) error {
	ddl := `
		CREATE TABLE IF NOT EXISTS schema_version (
			version String,
			applied_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = MergeTree()
		ORDER BY applied_at
	`
	return conn.Exec(ctx, ddl)
}

func getCurrentSchemaVersion(ctx context.Context, conn driver.Conn) (string, error) {
	var version string
	row := conn.QueryRow(ctx, "SELECT version FROM schema_version ORDER BY applied_at DESC LIMIT 1")
	err := row.Scan(&version)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return "", err
	}
	return version, nil
}

func setSchemaVersion(ctx context.Context, conn driver.Conn, version string) error {
	return conn.Exec(ctx, "INSERT INTO schema_version (version) VALUES (?)", version)
}

const metricsTableDDL = `
CREATE TABLE IF NOT EXISTS metrics (
    -- Identity
    name String,
    service_name String,
    
    -- Metric type info
    metric_type LowCardinality(String),
    unit String,
    
    -- Data point aggregation (for Sum metrics)
    aggregation_temporality LowCardinality(String),
    is_monotonic UInt8,
    
    -- Label/attribute keys (denormalized)
    label_keys Array(String),
    resource_keys Array(String),
    
    -- Sample counts
    sample_count UInt64,
    
    -- Timestamps
    first_seen DateTime64(3),
    last_seen DateTime64(3),
    
    -- Services using this metric
    services Array(String),
    service_count UInt32
    
) ENGINE = ReplacingMergeTree(last_seen)
ORDER BY (name, service_name)
SETTINGS index_granularity = 8192
`

const spansTableDDL = `
CREATE TABLE IF NOT EXISTS spans (
    -- Identity
    name String,
    service_name String,
    
    -- Span kind
    kind UInt8,
    kind_name LowCardinality(String),
    
    -- Attribute keys (denormalized)
    attribute_keys Array(String),
    resource_keys Array(String),
    
    -- Event and link metadata
    event_names Array(String),
    has_links UInt8,
    
    -- Status codes observed
    status_codes Array(LowCardinality(String)),
    
    -- Dropped counts (statistics)
    dropped_attrs_total UInt64,
    dropped_attrs_max UInt32,
    dropped_events_total UInt64,
    dropped_events_max UInt32,
    dropped_links_total UInt64,
    dropped_links_max UInt32,
    
    -- Sample counts
    sample_count UInt64,
    
    -- Timestamps
    first_seen DateTime64(3),
    last_seen DateTime64(3),
    
    -- Services
    services Array(String),
    service_count UInt32
    
) ENGINE = ReplacingMergeTree(last_seen)
ORDER BY (name, service_name)
SETTINGS index_granularity = 8192
`

const logsTableDDL = `
CREATE TABLE IF NOT EXISTS logs (
    -- Identity (pattern + severity + service)
    pattern_template String,
    severity LowCardinality(String),
    severity_number UInt8,
    service_name String,
    
    -- Attribute keys (denormalized)
    attribute_keys Array(String),
    resource_keys Array(String),
    
    -- Pattern examples
    example_body String,
    
    -- Context metadata
    has_trace_context UInt8,
    has_span_context UInt8,
    
    -- Dropped attributes
    dropped_attrs_total UInt64,
    dropped_attrs_max UInt32,
    
    -- Sample counts
    sample_count UInt64,
    
    -- Timestamps
    first_seen DateTime64(3),
    last_seen DateTime64(3),
    
    -- Services
    services Array(String),
    service_count UInt32
    
) ENGINE = ReplacingMergeTree(last_seen)
ORDER BY (pattern_template, severity, service_name)
SETTINGS index_granularity = 8192
`

const attributeValuesTableDDL = `
CREATE TABLE IF NOT EXISTS attribute_values (
    -- Key identity
    key String,
    value String,
    
    -- Signal type (metric, span, log)
    signal_type LowCardinality(String),
    
    -- Scope (resource, attribute)
    scope LowCardinality(String),
    
    -- Observation count
    observation_count UInt64,
    
    -- Timestamps
    first_seen DateTime64(3),
    last_seen DateTime64(3)
    
) ENGINE = SummingMergeTree(observation_count)
ORDER BY (key, value, signal_type, scope)
SETTINGS index_granularity = 8192
`

const servicesTableDDL = `
CREATE TABLE IF NOT EXISTS services (
    name String,
    
    -- Counts per signal type
    metric_count UInt32,
    span_count UInt32,
    log_pattern_count UInt32,
    
    -- Timestamps
    first_seen DateTime64(3),
    last_seen DateTime64(3)
    
) ENGINE = ReplacingMergeTree(last_seen)
ORDER BY name
SETTINGS index_granularity = 8192
`
