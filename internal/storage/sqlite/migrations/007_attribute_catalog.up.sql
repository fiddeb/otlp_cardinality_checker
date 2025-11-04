-- Migration 007: Add attribute catalog table for global attribute tracking
-- This table tracks all unique attribute keys across all signals (metrics, spans, logs)
-- with cardinality estimation using HyperLogLog

CREATE TABLE IF NOT EXISTS attribute_catalog (
    key TEXT PRIMARY KEY,
    hll_sketch BLOB NOT NULL,           -- HyperLogLog binary sketch for cardinality
    count INTEGER NOT NULL DEFAULT 0,   -- Number of times this attribute was seen
    estimated_cardinality INTEGER NOT NULL DEFAULT 0, -- Estimated unique value count
    value_samples TEXT,                 -- JSON array of sample values (max 10)
    signal_types TEXT NOT NULL,         -- JSON array of signal types using this attribute
    scope TEXT NOT NULL,                -- 'resource', 'attribute', or 'both'
    first_seen TIMESTAMP NOT NULL,
    last_seen TIMESTAMP NOT NULL
);

-- Index for filtering by signal type (stored in JSON, so we use pattern matching)
CREATE INDEX IF NOT EXISTS idx_attribute_catalog_signal_types ON attribute_catalog(signal_types);

-- Index for filtering by scope
CREATE INDEX IF NOT EXISTS idx_attribute_catalog_scope ON attribute_catalog(scope);

-- Index for sorting by cardinality
CREATE INDEX IF NOT EXISTS idx_attribute_catalog_cardinality ON attribute_catalog(estimated_cardinality);

-- Index for sorting by count
CREATE INDEX IF NOT EXISTS idx_attribute_catalog_count ON attribute_catalog(count);

-- Index for time-based queries
CREATE INDEX IF NOT EXISTS idx_attribute_catalog_last_seen ON attribute_catalog(last_seen);
