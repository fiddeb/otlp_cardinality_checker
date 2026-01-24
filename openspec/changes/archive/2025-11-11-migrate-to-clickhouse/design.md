# Design: ClickHouse Schema & Architecture

## Architecture Overview

```
OTLP Endpoints → Analyzer → Batch Buffer → ClickHouse Writer → ClickHouse DB
                                ↓
                           In-Memory Cache (read-through)
                                ↓
                         API v2 Handlers → Query ClickHouse
```

**Write Path:**
1. OTLP data arrives → Analyzer extracts metadata
2. Metadata buffered in memory (1-5 second window)
3. Batch writer flushes buffer to ClickHouse
4. In-memory cache invalidated/updated

**Read Path:**
1. API request → Check in-memory cache
2. Cache miss → Query ClickHouse
3. Update cache, return result

## ClickHouse Schema Design

### Core Principle: Denormalization

Every table stores all context needed for queries to avoid JOINs. Service names, timestamps, and common filters are duplicated across rows.

### Table: metrics

Stores metadata about each unique metric name with all its label keys.

```sql
CREATE TABLE metrics (
    -- Identity
    name String,
    service_name String,
    
    -- Metric type info
    metric_type LowCardinality(String),  -- Gauge, Sum, Histogram, etc.
    unit String,
    
    -- Data point aggregation (for Sum metrics)
    aggregation_temporality LowCardinality(String),  -- DELTA, CUMULATIVE
    is_monotonic UInt8,
    
    -- Label/attribute keys (denormalized)
    label_keys Array(String),           -- All label keys seen
    resource_keys Array(String),        -- All resource keys seen
    
    -- Cardinality per key (computed via GROUP BY on separate table)
    -- Will query attribute_values table for cardinality
    
    -- Sample counts
    sample_count UInt64,
    
    -- Timestamps
    first_seen DateTime64(3),
    last_seen DateTime64(3),
    
    -- Services using this metric (for multi-service metrics)
    services Array(String),
    service_count UInt32
    
) ENGINE = ReplacingMergeTree(last_seen)
ORDER BY (name, service_name)
SETTINGS index_granularity = 8192;
```

**ReplacingMergeTree:** Automatically deduplicates rows with same `(name, service_name)` key, keeping the row with latest `last_seen`.

**Why denormalize label_keys?** Querying "list all metrics with their labels" becomes a single table scan instead of JOIN to separate keys table.

### Table: spans

Stores metadata about unique span names.

```sql
CREATE TABLE spans (
    -- Identity
    name String,
    service_name String,
    
    -- Span kind (INTERNAL=1, SERVER=2, CLIENT=3, etc.)
    kind UInt8,
    kind_name LowCardinality(String),
    
    -- Attribute keys (denormalized)
    attribute_keys Array(String),
    resource_keys Array(String),
    
    -- Event and link metadata
    event_names Array(String),           -- Unique event names seen
    has_links UInt8,                     -- Boolean: any spans have links
    
    -- Status codes observed
    status_codes Array(LowCardinality(String)),  -- [OK, ERROR, UNSET]
    
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
SETTINGS index_granularity = 8192;
```

### Table: logs

Stores log pattern metadata (from Drain algorithm).

```sql
CREATE TABLE logs (
    -- Identity (pattern + severity + service)
    pattern_template String,             -- "user <*> logged in from <IP>"
    severity LowCardinality(String),     -- INFO, ERROR, WARN, etc.
    severity_number UInt8,               -- 1-24 per OTLP spec
    service_name String,
    
    -- Attribute keys (denormalized)
    attribute_keys Array(String),
    resource_keys Array(String),
    
    -- Pattern examples
    example_body String,                 -- One example log message
    
    -- Context metadata
    has_trace_context UInt8,             -- Boolean
    has_span_context UInt8,              -- Boolean
    
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
SETTINGS index_granularity = 8192;
```

**Note:** Log patterns extracted in-memory (Drain algorithm), only unique patterns stored in ClickHouse.

### Table: attribute_values

Tracks every attribute key-value observation across all signal types for cardinality estimation.

```sql
CREATE TABLE attribute_values (
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
SETTINGS index_granularity = 8192;
```

**SummingMergeTree:** Automatically sums `observation_count` for duplicate `(key, value, signal_type, scope)` rows.

**Cardinality query:**
```sql
-- Get cardinality and first 5 sample values for a key
SELECT
    key,
    uniqExact(value) AS estimated_cardinality,
    groupArray(5)(value) AS value_samples,
    sum(observation_count) AS total_count
FROM attribute_values
WHERE key = 'http.method'
GROUP BY key;
```

### Table: services

Aggregated service-level metadata (materialized view or computed on-the-fly).

```sql
CREATE TABLE services (
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
SETTINGS index_granularity = 8192;
```

**Could be materialized view** instead of separate table, computed from metrics/spans/logs tables.

## Query Patterns

### High Cardinality Keys (Cross-Signal)

```sql
-- Find top 10 highest cardinality keys across all signals
SELECT
    key,
    uniqExact(value) AS cardinality,
    groupArrayArray(signal_type) AS signals,
    sum(observation_count) AS total_observations
FROM attribute_values
GROUP BY key
HAVING cardinality > 1000
ORDER BY cardinality DESC
LIMIT 10;
```

**No JOINs needed** - all data in one table, columnar scan of `key`, `value`, `signal_type`.

### Metrics for Service

```sql
-- List all metrics for a service with label counts
SELECT
    name,
    metric_type,
    length(label_keys) AS label_count,
    length(resource_keys) AS resource_count,
    sample_count,
    last_seen
FROM metrics
WHERE service_name = 'my-service'
ORDER BY sample_count DESC;
```

**Index on (name, service_name)** makes this query very fast - single partition scan.

### Pattern Explorer (Log Patterns)

```sql
-- Find log patterns with high occurrence across multiple services
SELECT
    pattern_template,
    severity,
    service_count,
    sum(sample_count) AS total_count,
    max(last_seen) AS last_seen
FROM logs
GROUP BY pattern_template, severity
HAVING service_count >= 3 AND total_count >= 100
ORDER BY total_count DESC
LIMIT 50;
```

**Pre-aggregated service_count** eliminates need to count services in query.

## Write Strategy

### Batch Buffer

```go
type BatchBuffer struct {
    metrics []MetricRow
    spans   []SpanRow
    logs    []LogRow
    attrs   []AttributeRow
    
    mu          sync.Mutex
    lastFlush   time.Time
    flushSize   int  // e.g., 1000 rows
    flushInterval time.Duration  // e.g., 5 seconds
}
```

**Flush triggers:**
1. Buffer reaches `flushSize` rows (e.g., 1000)
2. `flushInterval` elapsed since last flush (e.g., 5s)
3. Shutdown signal received

### Insert Query Pattern

```go
// Batch insert with native ClickHouse protocol
batch, err := conn.PrepareBatch(ctx, "INSERT INTO metrics")
for _, m := range buffer.metrics {
    batch.Append(
        m.Name,
        m.ServiceName,
        m.MetricType,
        // ... all columns
    )
}
err := batch.Send()
```

**Native protocol** (`clickhouse://` connection) is faster than HTTP for bulk inserts.

## Performance Expectations

**Write throughput:**
- Current SQLite: ~2k-5k writes/sec (with cache)
- Target ClickHouse: 50k+ writes/sec (batch inserts)
- **10x+ improvement**

**Read latency:**
- Current SQLite: 50-200ms (complex JOINs)
- Target ClickHouse: 5-20ms (columnar scans, no JOINs)
- **5-10x improvement**

**Cardinality estimation:**
- Current: HyperLogLog in Go (~1% error, 16KB per key)
- New: ClickHouse `uniqExact()` (exact count, computed on-demand)
- **Simpler code, exact results**

## Migration Notes

**No data migration** - SQLite data will not be migrated. Fresh start with ClickHouse.

**Development setup:**
```bash
# Start ClickHouse (using external binary)
./external/clickhouse server --config-file=config/clickhouse-config.xml

# Server listens on localhost:9000 (native protocol)
# HTTP interface on localhost:8123 (for admin queries)
```

**Schema initialization:**
```go
// On startup, create tables if not exist
for _, ddl := range schema.CreateTableStatements {
    _, err := conn.Exec(ctx, ddl)
}
```

## Rollback Plan

If ClickHouse performs worse than expected:
1. Memory backend still available (no changes)
2. SQLite code removed but in git history
3. Revert commits, restore SQLite if needed (unlikely)

This design prioritizes **performance through denormalization** and **simplicity through elimination of JOINs**.
