-- Migration 003: Per-service log key tracking
-- Adds log_service_keys table to track attribute/resource keys per service
-- This enables accurate per-service metadata analysis for log patterns

-- Record this migration
INSERT OR IGNORE INTO schema_migrations (version) VALUES (3);

-- ============================================================================
-- CREATE LOG SERVICE KEYS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS log_service_keys (
    severity TEXT NOT NULL,
    service_name TEXT NOT NULL,
    key_scope TEXT NOT NULL,        -- 'attribute' or 'resource'
    key_name TEXT NOT NULL,
    key_count INTEGER DEFAULT 0,
    key_percentage REAL DEFAULT 0.0,
    estimated_cardinality INTEGER DEFAULT 0,
    value_samples TEXT,             -- JSON array of sample values
    hll_sketch BLOB,                -- HyperLogLog sketch for cardinality estimation
    PRIMARY KEY (severity, service_name, key_scope, key_name),
    FOREIGN KEY (severity) REFERENCES logs(severity) ON DELETE CASCADE
) STRICT;

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_log_service_keys_severity_service 
    ON log_service_keys(severity, service_name);

CREATE INDEX IF NOT EXISTS idx_log_service_keys_cardinality 
    ON log_service_keys(severity, service_name, estimated_cardinality DESC);

CREATE INDEX IF NOT EXISTS idx_log_service_keys_scope 
    ON log_service_keys(severity, service_name, key_scope);
