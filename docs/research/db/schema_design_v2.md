# Database Schema Design v2 - Revised Approach

**Date**: 2025-10-26  
**Status**: Design Phase  
**Approach**: Database-first optimization, willing to adapt Go models

## Problem with v1 Schema

The initial schema assumed per-service storage (e.g., `UNIQUE(service_id, name)`) but the actual model stores:
- ONE metric name with multiple services tracked in a map
- Rich KeyMetadata objects with cardinality estimation and samples
- Body templates with counts and percentages per severity

This mismatch would require multiple rows per metric, breaking the Get/Store API.

## Design Principles

1. **Optimize for query patterns** (list by service, get by name, filter, sort)
2. **Efficient UPSERTs** for high-throughput writes (20-30k EPS)
3. **Normalize where it helps**, denormalize where it performs better
4. **Use SQLite features** (JSON columns, partial indexes, generated columns)
5. **Keep it simple** - this is Phase A (SQLite), not final architecture

## Core Entities

### 1. Metrics

**Storage Strategy**: Separate base metadata from service tracking and key details.

```sql
-- Base metric info (one row per unique metric name)
CREATE TABLE metrics (
    name TEXT PRIMARY KEY,
    type TEXT NOT NULL,  -- Gauge, Sum, Histogram, etc.
    unit TEXT,
    description TEXT,
    total_sample_count INTEGER DEFAULT 0
);

-- Service tracking for metrics (many-to-many)
CREATE TABLE metric_services (
    metric_name TEXT NOT NULL REFERENCES metrics(name) ON DELETE CASCADE,
    service_name TEXT NOT NULL,
    sample_count INTEGER DEFAULT 0,
    PRIMARY KEY (metric_name, service_name)
);
CREATE INDEX idx_metric_services_service ON metric_services(service_name);

-- Label/attribute keys with cardinality tracking
CREATE TABLE metric_keys (
    metric_name TEXT NOT NULL REFERENCES metrics(name) ON DELETE CASCADE,
    key_scope TEXT NOT NULL,  -- 'label' or 'resource'
    key_name TEXT NOT NULL,
    key_count INTEGER DEFAULT 0,
    key_percentage REAL DEFAULT 0.0,
    estimated_cardinality INTEGER DEFAULT 0,
    value_samples TEXT,  -- JSON array of sample values
    hll_sketch BLOB,     -- HyperLogLog sketch for cardinality (future)
    PRIMARY KEY (metric_name, key_scope, key_name)
);
CREATE INDEX idx_metric_keys_name ON metric_keys(metric_name);
```

**Rationale**:
- Clean separation: base metadata vs service data vs key details
- Easy to query: "show all metrics for service X" = join on metric_services
- Efficient UPSERT: update counters on conflict
- JSON for samples: flexible, queryable with SQLite JSON functions
- HLL sketch: BLOB for future cardinality estimation

### 2. Spans

**Similar pattern to metrics:**

```sql
CREATE TABLE spans (
    name TEXT PRIMARY KEY,
    kind TEXT,  -- Client, Server, Internal, etc.
    total_sample_count INTEGER DEFAULT 0
);

CREATE TABLE span_services (
    span_name TEXT NOT NULL REFERENCES spans(name) ON DELETE CASCADE,
    service_name TEXT NOT NULL,
    sample_count INTEGER DEFAULT 0,
    PRIMARY KEY (span_name, service_name)
);
CREATE INDEX idx_span_services_service ON span_services(service_name);

CREATE TABLE span_keys (
    span_name TEXT NOT NULL REFERENCES spans(name) ON DELETE CASCADE,
    key_scope TEXT NOT NULL,  -- 'attribute', 'resource', 'event', 'link'
    key_name TEXT NOT NULL,
    event_name TEXT,  -- NULL unless key_scope = 'event'
    key_count INTEGER DEFAULT 0,
    key_percentage REAL DEFAULT 0.0,
    estimated_cardinality INTEGER DEFAULT 0,
    value_samples TEXT,  -- JSON array
    hll_sketch BLOB,
    PRIMARY KEY (span_name, key_scope, key_name, COALESCE(event_name, ''))
);
CREATE INDEX idx_span_keys_name ON span_keys(span_name);

-- Separate table for event names (lightweight)
CREATE TABLE span_events (
    span_name TEXT NOT NULL REFERENCES spans(name) ON DELETE CASCADE,
    event_name TEXT NOT NULL,
    PRIMARY KEY (span_name, event_name)
);
```

### 3. Logs

**Logs are unique - grouped by severity, not name:**

```sql
CREATE TABLE logs (
    severity TEXT PRIMARY KEY,  -- INFO, WARN, ERROR, etc.
    total_sample_count INTEGER DEFAULT 0
);

CREATE TABLE log_services (
    severity TEXT NOT NULL REFERENCES logs(severity) ON DELETE CASCADE,
    service_name TEXT NOT NULL,
    sample_count INTEGER DEFAULT 0,
    PRIMARY KEY (severity, service_name)
);
CREATE INDEX idx_log_services_service ON log_services(service_name);

CREATE TABLE log_keys (
    severity TEXT NOT NULL REFERENCES logs(severity) ON DELETE CASCADE,
    key_scope TEXT NOT NULL,  -- 'attribute' or 'resource'
    key_name TEXT NOT NULL,
    key_count INTEGER DEFAULT 0,
    key_percentage REAL DEFAULT 0.0,
    estimated_cardinality INTEGER DEFAULT 0,
    value_samples TEXT,  -- JSON array
    hll_sketch BLOB,
    PRIMARY KEY (severity, key_scope, key_name)
);
CREATE INDEX idx_log_keys_severity ON log_keys(severity);

-- Body templates (Drain output)
CREATE TABLE log_body_templates (
    severity TEXT NOT NULL REFERENCES logs(severity) ON DELETE CASCADE,
    service_name TEXT NOT NULL,
    template TEXT NOT NULL,
    example TEXT,  -- First matching log body
    count INTEGER DEFAULT 0,
    percentage REAL DEFAULT 0.0,
    PRIMARY KEY (severity, service_name, template)
);
CREATE INDEX idx_log_templates_severity_service 
    ON log_body_templates(severity, service_name);
-- Sort by count DESC for top-K queries
CREATE INDEX idx_log_templates_count 
    ON log_body_templates(severity, service_name, count DESC);
```

**Rationale for logs**:
- Body templates are **per-service AND per-severity** (not shared across services)
- This matches the actual behavior: each service has its own log patterns
- Easy top-K query: `WHERE severity=? AND service_name=? ORDER BY count DESC LIMIT 10`

## Revised Go Models

**Option A**: Keep current nested maps, serialize/deserialize to DB

**Option B**: Flatten models to match DB better

```go
// Flatter model matching DB structure
type MetricMetadata struct {
    Name        string
    Type        string
    Unit        string
    Description string
    
    // Aggregate counts
    TotalSampleCount int64
    
    // Loaded on demand or embedded
    Services     map[string]int64       // service -> count
    LabelKeys    map[string]*KeyMetadata
    ResourceKeys map[string]*KeyMetadata
}

type KeyMetadata struct {
    KeyName              string
    Count                int64
    Percentage           float64
    EstimatedCardinality int64
    ValueSamples         []string
    HLLSketch            []byte  // serialized sketch
}
```

**Recommendation**: Start with Option A (keep current API), optimize later if needed.

## Query Patterns & Performance

### Common Queries

1. **List all metrics for a service**:
```sql
SELECT m.* 
FROM metrics m
JOIN metric_services ms ON m.name = ms.metric_name
WHERE ms.service_name = ?
ORDER BY m.name;
```

2. **Get metric with all details** (N+1 or JOIN):
```sql
-- Base
SELECT * FROM metrics WHERE name = ?;

-- Services
SELECT service_name, sample_count 
FROM metric_services 
WHERE metric_name = ?;

-- Keys
SELECT key_scope, key_name, key_count, key_percentage, 
       estimated_cardinality, value_samples
FROM metric_keys
WHERE metric_name = ?;
```

3. **Top-K body templates for severity + service**:
```sql
SELECT template, example, count, percentage
FROM log_body_templates
WHERE severity = ? AND service_name = ?
ORDER BY count DESC
LIMIT 10;
```

### Indexes Strategy

- **Primary keys**: Natural keys (name, severity) - always indexed
- **Foreign keys**: Indexed for joins (service_name in junction tables)
- **Sort columns**: Indexes on commonly sorted fields (count DESC for templates)
- **Partial indexes**: Future optimization (e.g., only index high-cardinality keys)

## UPSERT Patterns

### Metric Storage

```sql
-- 1. Insert or ignore base metric
INSERT INTO metrics (name, type, unit, description)
VALUES (?, ?, ?, ?)
ON CONFLICT(name) DO UPDATE SET
    unit = COALESCE(excluded.unit, unit),
    description = COALESCE(excluded.description, description);

-- 2. Update service count
INSERT INTO metric_services (metric_name, service_name, sample_count)
VALUES (?, ?, ?)
ON CONFLICT(metric_name, service_name) DO UPDATE SET
    sample_count = sample_count + excluded.sample_count;

-- 3. Upsert keys (read-merge-write or application-level merge)
-- Option: Read existing keys, merge in Go, write back
-- Option: Use JSON_PATCH for samples (more complex)
```

### Log Template Storage

```sql
-- Templates need recalculation of percentages per service+severity
-- Strategy: Batch updates in transaction
BEGIN;
  INSERT INTO log_body_templates (severity, service_name, template, example, count)
  VALUES (?, ?, ?, ?, ?)
  ON CONFLICT(severity, service_name, template) DO UPDATE SET
      count = excluded.count,
      example = COALESCE(excluded.example, example);
  
  -- Recalc percentages for this severity+service
  UPDATE log_body_templates
  SET percentage = (count * 100.0) / (
      SELECT SUM(count) FROM log_body_templates 
      WHERE severity = ? AND service_name = ?
  )
  WHERE severity = ? AND service_name = ?;
COMMIT;
```

## Migration from In-Memory

### Phase 1: Dual Write (safest)
1. Keep in-memory store as primary
2. Also write to SQLite (async, best-effort)
3. Compare results for consistency
4. Switch read path when confident

### Phase 2: Seed from Memory
1. On startup: if DB empty, bulk insert from memory snapshot
2. Switch to DB for all reads/writes
3. Memory becomes optional fallback

### Phase 3: DB-only
1. Remove in-memory store
2. All state in SQLite
3. Restart = cold start from empty DB

## Concurrency Model

- **Single writer goroutine**: Batches writes into transactions
- **WAL mode**: Readers don't block writer
- **Batch size**: 50-100 ops per transaction (tune via benchmarks)
- **Flush interval**: 100ms max latency

## Open Questions

1. **KeyMetadata merge strategy**: Read-merge-write vs SQL-level merge?
2. **HLL sketches**: Integrate now or defer to Phase B?
3. **Scope info**: Store instrumentation scope separately?
4. **Time bucketing**: Add `recorded_at` timestamp for future time-series queries?

## Next Steps

1. Implement migration SQL with this schema
2. Create store.go with batch writer
3. Write integration tests
4. Benchmark against in-memory
5. Feature flag rollout
