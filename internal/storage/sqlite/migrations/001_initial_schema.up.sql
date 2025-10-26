-- Initial schema for OTLP metadata storage
-- Design: Normalized tables with service tracking and key metadata

-- ============================================================================
-- METRICS
-- ============================================================================

-- Base metric metadata (one row per unique metric name)
CREATE TABLE metrics (
    name TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    unit TEXT,
    description TEXT,
    total_sample_count INTEGER DEFAULT 0
) STRICT;

-- Service tracking for metrics (many-to-many)
CREATE TABLE metric_services (
    metric_name TEXT NOT NULL,
    service_name TEXT NOT NULL,
    sample_count INTEGER DEFAULT 0,
    PRIMARY KEY (metric_name, service_name),
    FOREIGN KEY (metric_name) REFERENCES metrics(name) ON DELETE CASCADE
) STRICT;

CREATE INDEX idx_metric_services_service ON metric_services(service_name);

-- Label and resource keys with cardinality tracking
CREATE TABLE metric_keys (
    metric_name TEXT NOT NULL,
    key_scope TEXT NOT NULL,  -- 'label' or 'resource'
    key_name TEXT NOT NULL,
    key_count INTEGER DEFAULT 0,
    key_percentage REAL DEFAULT 0.0,
    estimated_cardinality INTEGER DEFAULT 0,
    value_samples TEXT,  -- JSON array of sample values
    hll_sketch BLOB,     -- HyperLogLog sketch (future)
    PRIMARY KEY (metric_name, key_scope, key_name),
    FOREIGN KEY (metric_name) REFERENCES metrics(name) ON DELETE CASCADE
) STRICT;

CREATE INDEX idx_metric_keys_name ON metric_keys(metric_name);

-- ============================================================================
-- SPANS
-- ============================================================================

-- Base span metadata
CREATE TABLE spans (
    name TEXT PRIMARY KEY,
    kind TEXT,  -- Client, Server, Internal, Producer, Consumer
    total_sample_count INTEGER DEFAULT 0
) STRICT;

-- Service tracking for spans
CREATE TABLE span_services (
    span_name TEXT NOT NULL,
    service_name TEXT NOT NULL,
    sample_count INTEGER DEFAULT 0,
    PRIMARY KEY (span_name, service_name),
    FOREIGN KEY (span_name) REFERENCES spans(name) ON DELETE CASCADE
) STRICT;

CREATE INDEX idx_span_services_service ON span_services(service_name);

-- Span attribute and resource keys
CREATE TABLE span_keys (
    span_name TEXT NOT NULL,
    key_scope TEXT NOT NULL,  -- 'attribute', 'resource', 'event', 'link'
    key_name TEXT NOT NULL,
    event_name TEXT NOT NULL DEFAULT '',  -- Only for key_scope='event', else empty string
    key_count INTEGER DEFAULT 0,
    key_percentage REAL DEFAULT 0.0,
    estimated_cardinality INTEGER DEFAULT 0,
    value_samples TEXT,  -- JSON array
    hll_sketch BLOB,
    PRIMARY KEY (span_name, key_scope, key_name, event_name),
    FOREIGN KEY (span_name) REFERENCES spans(name) ON DELETE CASCADE
) STRICT;

CREATE INDEX idx_span_keys_name ON span_keys(span_name);

-- Event names observed in spans
CREATE TABLE span_events (
    span_name TEXT NOT NULL,
    event_name TEXT NOT NULL,
    PRIMARY KEY (span_name, event_name),
    FOREIGN KEY (span_name) REFERENCES spans(name) ON DELETE CASCADE
) STRICT;

-- ============================================================================
-- LOGS
-- ============================================================================

-- Base log metadata (grouped by severity)
CREATE TABLE logs (
    severity TEXT PRIMARY KEY,
    total_sample_count INTEGER DEFAULT 0
) STRICT;

-- Service tracking for logs
CREATE TABLE log_services (
    severity TEXT NOT NULL,
    service_name TEXT NOT NULL,
    sample_count INTEGER DEFAULT 0,
    PRIMARY KEY (severity, service_name),
    FOREIGN KEY (severity) REFERENCES logs(severity) ON DELETE CASCADE
) STRICT;

CREATE INDEX idx_log_services_service ON log_services(service_name);

-- Log attribute and resource keys
CREATE TABLE log_keys (
    severity TEXT NOT NULL,
    key_scope TEXT NOT NULL,  -- 'attribute' or 'resource'
    key_name TEXT NOT NULL,
    key_count INTEGER DEFAULT 0,
    key_percentage REAL DEFAULT 0.0,
    estimated_cardinality INTEGER DEFAULT 0,
    value_samples TEXT,  -- JSON array
    hll_sketch BLOB,
    PRIMARY KEY (severity, key_scope, key_name),
    FOREIGN KEY (severity) REFERENCES logs(severity) ON DELETE CASCADE
) STRICT;

CREATE INDEX idx_log_keys_severity ON log_keys(severity);

-- Body templates extracted from log messages (Drain algorithm output)
-- Stored per-service because different services have different log patterns
CREATE TABLE log_body_templates (
    severity TEXT NOT NULL,
    service_name TEXT NOT NULL,
    template TEXT NOT NULL,
    example TEXT,  -- First log body that matched this template
    count INTEGER DEFAULT 0,
    percentage REAL DEFAULT 0.0,
    PRIMARY KEY (severity, service_name, template),
    FOREIGN KEY (severity) REFERENCES logs(severity) ON DELETE CASCADE
) STRICT;

CREATE INDEX idx_log_templates_severity_service 
    ON log_body_templates(severity, service_name);

-- Optimize for top-K template queries (ORDER BY count DESC)
CREATE INDEX idx_log_templates_count 
    ON log_body_templates(severity, service_name, count DESC);

-- ============================================================================
-- INSTRUMENTATION SCOPE (optional, future use)
-- ============================================================================

-- Scope information can be added later if needed
-- For now, we focus on the core metadata structure
