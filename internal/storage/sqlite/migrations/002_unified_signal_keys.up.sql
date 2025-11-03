-- Migration 002: Unified signal_keys table
-- Consolidates metric_keys, span_keys, and log_keys into a single table
-- This enables cross-signal cardinality analysis and simpler filtering

-- Record this migration
INSERT OR IGNORE INTO schema_migrations (version) VALUES (2);

-- ============================================================================
-- CREATE NEW UNIFIED TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS signal_keys (
    signal_type TEXT NOT NULL,      -- 'metric', 'span', 'log'
    signal_name TEXT NOT NULL,      -- metric name, span name, or log severity
    key_scope TEXT NOT NULL,        -- 'label', 'resource', 'attribute', 'event', 'link'
    key_name TEXT NOT NULL,
    event_name TEXT NOT NULL DEFAULT '',  -- Only for key_scope='event', else empty string
    key_count INTEGER DEFAULT 0,
    key_percentage REAL DEFAULT 0.0,
    estimated_cardinality INTEGER DEFAULT 0,
    value_samples TEXT,             -- JSON array of sample values
    hll_sketch BLOB,                -- HyperLogLog sketch for cardinality estimation
    PRIMARY KEY (signal_type, signal_name, key_scope, key_name, event_name)
) STRICT;

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_signal_keys_cardinality 
    ON signal_keys(estimated_cardinality DESC);

CREATE INDEX IF NOT EXISTS idx_signal_keys_type_name 
    ON signal_keys(signal_type, signal_name);

CREATE INDEX IF NOT EXISTS idx_signal_keys_scope 
    ON signal_keys(key_scope);

-- Composite index for pattern explorer (log patterns with high cardinality)
CREATE INDEX IF NOT EXISTS idx_signal_keys_log_cardinality 
    ON signal_keys(signal_type, signal_name, estimated_cardinality DESC) 
    WHERE signal_type = 'log';

-- ============================================================================
-- MIGRATE EXISTING DATA
-- ============================================================================

-- Migrate metric_keys to signal_keys
INSERT OR IGNORE INTO signal_keys (
    signal_type, signal_name, key_scope, key_name, event_name,
    key_count, key_percentage, estimated_cardinality, value_samples, hll_sketch
)
SELECT 
    'metric' as signal_type,
    metric_name as signal_name,
    key_scope,
    key_name,
    '' as event_name,
    key_count,
    key_percentage,
    estimated_cardinality,
    value_samples,
    hll_sketch
FROM metric_keys;

-- Migrate span_keys to signal_keys
INSERT OR IGNORE INTO signal_keys (
    signal_type, signal_name, key_scope, key_name, event_name,
    key_count, key_percentage, estimated_cardinality, value_samples, hll_sketch
)
SELECT 
    'span' as signal_type,
    span_name as signal_name,
    key_scope,
    key_name,
    event_name,
    key_count,
    key_percentage,
    estimated_cardinality,
    value_samples,
    hll_sketch
FROM span_keys;

-- Migrate log_keys to signal_keys
INSERT OR IGNORE INTO signal_keys (
    signal_type, signal_name, key_scope, key_name, event_name,
    key_count, key_percentage, estimated_cardinality, value_samples, hll_sketch
)
SELECT 
    'log' as signal_type,
    severity as signal_name,
    key_scope,
    key_name,
    '' as event_name,
    key_count,
    key_percentage,
    estimated_cardinality,
    value_samples,
    hll_sketch
FROM log_keys;

-- ============================================================================
-- KEEP OLD TABLES FOR NOW (can drop later after verifying migration)
-- ============================================================================
-- We'll keep metric_keys, span_keys, and log_keys for backward compatibility
-- They will be phased out in a future migration after verification
