# ClickHouse Storage Backend

## Overview

OTLP Cardinality Checker uses ClickHouse as its primary persistent storage backend. ClickHouse is a columnar OLAP database optimized for analytical queries and high write throughput, making it ideal for telemetry metadata analysis.

## Why ClickHouse?

- **Exact Cardinality**: Uses `uniqExact()` aggregation function for precise cardinality counts (no estimation error)
- **High Write Throughput**: Batch buffer system achieves 6,400+ signals/sec with p95 latency of 161ms
- **Efficient Storage**: Columnar format compresses metadata efficiently
- **Scalability**: Handles millions of unique metrics, spans, and logs
- **Performance**: 5-10x faster reads compared to SQLite baseline

## Architecture

### Batch Buffer System

ClickHouse storage uses an asynchronous batch buffer to optimize write performance:

```
OTLP Receiver → Analyzer → Batch Buffer → ClickHouse
                              ↓
                         (1000 rows or 5s)
```

**Buffer Configuration:**
- Default batch size: 1000 rows
- Default flush interval: 5 seconds
- Graceful shutdown: max 10s wait for buffer flush

**How it works:**
1. Telemetry data is extracted by analyzers
2. Rows are added to in-memory buffer
3. Buffer flushes when size threshold OR time interval is reached
4. Batch INSERT is performed using `PrepareBatch()` API
5. On error, retries 3 times with exponential backoff

### Table Schema

ClickHouse uses four main tables with specialized engines:

#### 1. Metrics Table

```sql
CREATE TABLE IF NOT EXISTS metrics (
    name String,
    description String,
    unit String,
    type String,
    aggregation_temporality String,
    is_monotonic UInt8,
    label_keys String,           -- JSON array
    resource_keys String,        -- JSON array
    sample_count UInt64,
    service_name String,
    service_count UInt64,
    version UInt32,
    updated_at DateTime DEFAULT now()
) ENGINE = ReplacingMergeTree(version)
PARTITION BY toYYYYMM(updated_at)
ORDER BY (name, service_name);
```

**Engine:** `ReplacingMergeTree` - automatically deduplicates rows with same PRIMARY KEY, keeping latest version.

**Key Features:**
- Partitioned by month for efficient data management
- Ordered by metric name and service for fast queries
- Version field ensures latest data wins on merge

#### 2. Spans Table

```sql
CREATE TABLE IF NOT EXISTS spans (
    name String,
    kind Int32,
    kind_name String,
    has_trace_state UInt8,
    has_parent_span_id UInt8,
    status_codes String,         -- JSON array
    attribute_keys String,       -- JSON array
    resource_keys String,        -- JSON array
    event_names String,          -- JSON array
    sample_count UInt64,
    service_name String,
    service_count UInt64,
    version UInt32,
    updated_at DateTime DEFAULT now()
) ENGINE = ReplacingMergeTree(version)
PARTITION BY toYYYYMM(updated_at)
ORDER BY (name, service_name);
```

#### 3. Logs Table

```sql
CREATE TABLE IF NOT EXISTS logs (
    pattern_template String,
    severity String,
    severity_number Int32,
    has_trace_context UInt8,
    has_span_context UInt8,
    attribute_keys String,       -- JSON array
    resource_keys String,        -- JSON array
    event_names String,          -- JSON array
    example_body String,
    sample_count UInt64,
    service_name String,
    service_count UInt64,
    version UInt32,
    updated_at DateTime DEFAULT now()
) ENGINE = ReplacingMergeTree(version)
PARTITION BY toYYYYMM(updated_at)
ORDER BY (pattern_template, severity, service_name);
```

#### 4. Attribute Values Table

```sql
CREATE TABLE IF NOT EXISTS attribute_values (
    key String,
    value String,
    signal_type String,          -- 'metric', 'span', 'log'
    scope String,                -- 'label', 'resource', 'event', 'link'
    service_name String,
    count UInt64,
    first_seen DateTime DEFAULT now(),
    last_seen DateTime DEFAULT now()
) ENGINE = SummingMergeTree(count)
PARTITION BY toYYYYMM(first_seen)
ORDER BY (key, value, signal_type, scope, service_name);
```

**Engine:** `SummingMergeTree` - automatically sums the `count` column when rows with same PRIMARY KEY are merged.

**Use Case:** Track exact cardinality of attribute values across all signals.

## Querying ClickHouse

### Basic Queries

#### Get All Metrics

```bash
clickhouse-client --query="
SELECT 
    name,
    type,
    length(JSONExtract(label_keys, 'Array(String)')) as label_count,
    sample_count,
    service_name
FROM metrics FINAL
ORDER BY sample_count DESC
LIMIT 10"
```

**Note:** Always use `FINAL` modifier with ReplacingMergeTree to ensure deduplication.

#### Get Exact Cardinality for Attribute

```bash
clickhouse-client --query="
SELECT 
    key,
    uniqExact(value) as cardinality,
    sum(count) as total_observations,
    groupArray(5)(value) as sample_values
FROM attribute_values
WHERE key = 'http.method'
GROUP BY key"
```

#### Get High-Cardinality Keys

```bash
clickhouse-client --query="
SELECT 
    key,
    uniqExact(value) as unique_values,
    sum(count) as observations,
    groupUniqArray(signal_type) as used_in_signals
FROM attribute_values
GROUP BY key
HAVING unique_values > 100
ORDER BY unique_values DESC
LIMIT 20"
```

### Advanced Queries

#### Log Pattern Distribution by Service

```bash
clickhouse-client --query="
SELECT 
    pattern_template,
    severity,
    count(DISTINCT service_name) as service_count,
    sum(sample_count) as total_samples
FROM logs FINAL
WHERE severity = 'ERROR'
GROUP BY pattern_template, severity
HAVING service_count > 5
ORDER BY total_samples DESC"
```

#### Metric Complexity Analysis

```bash
clickhouse-client --query="
SELECT 
    m.name,
    length(JSONExtract(m.label_keys, 'Array(String)')) as label_count,
    max_cardinality,
    (label_count * max_cardinality) as complexity_score
FROM (
    SELECT 
        name,
        label_keys,
        max(sample_count) as max_cardinality
    FROM metrics FINAL
    GROUP BY name, label_keys
) m
ORDER BY complexity_score DESC
LIMIT 20"
```

## Performance Benchmarks

### Write Performance

**Test Configuration:**
- K6 load test with ramping VUs
- Duration: 4 minutes (30s warmup + 3×1m ramp stages + 30s cooldown)
- Peak load: 550 concurrent VUs (300 metrics + 150 spans + 100 logs)

**Results:**
```
Total Throughput:  6,407 signals/sec
├─ Metrics:        3,573/sec (56%)
├─ Spans:          1,731/sec (27%)
└─ Logs:           1,102/sec (17%)

Write Latency (p95):  161ms
Success Rate:         100%
Total Signals:        1,537,613 in 4 minutes
```

**Baseline Test (Constant Rate):**
```
Metrics:  100/sec × 2min = 12,001 written
Spans:     50/sec × 2min =  6,000 written  
Logs:      30/sec × 2min =  3,601 written

p95 Latency:  3ms
Success Rate: 100%
```

### Read Performance

**Test Configuration:**
- K6 load test hitting all v1 REST API endpoints
- 50 requests/sec sustained load
- 1 minute duration

**Results:**
```
Request Duration (p95):     8ms
Request Duration (avg):    5.16ms
Success Rate:              100%
Throughput:                50 req/sec
```

**Comparison vs SQLite:**
- Write throughput: ~10x faster
- Read latency: ~5-10x faster  
- Exact cardinality: No HyperLogLog estimation error (~1%)

## Configuration

### Environment Variables

```bash
# Storage backend selection (default: "clickhouse")
export STORAGE_BACKEND="clickhouse"

# ClickHouse server address (default: "localhost:9000")
export CLICKHOUSE_ADDR="localhost:9000"

# Optional: ClickHouse credentials
export CLICKHOUSE_USER="default"
export CLICKHOUSE_PASSWORD=""
export CLICKHOUSE_DATABASE="default"
```

### ClickHouse Server Setup

#### macOS (Homebrew)

```bash
# Install
brew install clickhouse

# Start server
clickhouse-server --config-file=config/clickhouse-config.xml

# Or use the helper script
./scripts/start-clickhouse.sh
```

#### Ubuntu/Debian

```bash
# Add repository
sudo apt-get install apt-transport-https ca-certificates dirmngr
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv 8919F6BD2B48D754
echo "deb https://packages.clickhouse.com/deb stable main" | \
  sudo tee /etc/apt/sources.list.d/clickhouse.list

# Install
sudo apt-get update && sudo apt-get install -y clickhouse-server clickhouse-client

# Start server
sudo systemctl start clickhouse-server
```

#### Docker

```bash
docker run -d --name clickhouse-server \
  -p 9000:9000 \
  -p 8123:8123 \
  --ulimit nofile=262144:262144 \
  clickhouse/clickhouse-server:latest
```

### ClickHouse Configuration

Example `config/clickhouse-config.xml`:

```xml
<clickhouse>
    <logger>
        <level>warning</level>
        <log>/tmp/clickhouse-server.log</log>
        <errorlog>/tmp/clickhouse-server.err.log</errorlog>
    </logger>

    <http_port>8123</http_port>
    <tcp_port>9000</tcp_port>

    <path>/tmp/clickhouse/</path>
    
    <users>
        <default>
            <password></password>
            <networks>
                <ip>::/0</ip>
            </networks>
            <profile>default</profile>
            <quota>default</quota>
        </default>
    </users>

    <profiles>
        <default>
            <max_memory_usage>10000000000</max_memory_usage>
            <use_uncompressed_cache>0</use_uncompressed_cache>
            <load_balancing>random</load_balancing>
        </default>
    </profiles>

    <quotas>
        <default>
            <interval>
                <duration>3600</duration>
                <queries>0</queries>
                <errors>0</errors>
                <result_rows>0</result_rows>
                <read_rows>0</read_rows>
                <execution_time>0</execution_time>
            </interval>
        </default>
    </quotas>
</clickhouse>
```

## Testing

### Integration Tests

Run full integration test suite:

```bash
./scripts/test-clickhouse-integration.sh
```

This validates:
- ClickHouse connection
- Application startup
- Health check endpoint
- OTLP write operations
- Buffer flush mechanism
- REST API reads
- Data persistence in ClickHouse

### Go Integration Tests

```bash
# Run Go-based integration tests
go test -tags=integration ./internal/storage/clickhouse/... -v

# Run specific test
go test -tags=integration ./internal/storage/clickhouse/ -run TestClickHouseStore_StoreAndRetrieveMetric -v
```

### Load Tests

```bash
# Baseline constant-rate test
k6 run scripts/k6-clickhouse-write.js

# Max throughput test
k6 run scripts/k6-clickhouse-max-throughput.js

# Read performance test
k6 run scripts/k6-clickhouse-read.js
```

## Monitoring

### System Tables

ClickHouse provides system tables for monitoring:

```sql
-- Query execution statistics
SELECT 
    query,
    type,
    event_time,
    query_duration_ms,
    read_rows,
    read_bytes
FROM system.query_log
WHERE type = 'QueryFinish'
ORDER BY event_time DESC
LIMIT 10;

-- Table sizes
SELECT 
    database,
    table,
    formatReadableSize(sum(bytes)) as size,
    sum(rows) as rows,
    count() as parts
FROM system.parts
WHERE active
GROUP BY database, table
ORDER BY sum(bytes) DESC;

-- Current processes
SELECT 
    query_id,
    user,
    elapsed,
    read_rows,
    memory_usage
FROM system.processes
WHERE query NOT LIKE '%system.processes%';
```

### Application Metrics

The application exposes buffer metrics via the health endpoint:

```bash
curl http://localhost:8080/health | jq '.storage'
```

Response:
```json
{
  "backend": "clickhouse",
  "address": "localhost:9000",
  "buffer": {
    "pending_metrics": 245,
    "pending_spans": 102,
    "pending_logs": 67,
    "last_flush": "2025-11-09T15:23:45Z"
  }
}
```

## Troubleshooting

### Connection Issues

**Problem:** Application can't connect to ClickHouse

```
Error: dial tcp 127.0.0.1:9000: connect: connection refused
```

**Solution:**
1. Verify ClickHouse is running: `ps aux | grep clickhouse`
2. Check port availability: `lsof -i :9000`
3. Test connection: `clickhouse-client --query="SELECT 1"`

### Buffer Not Flushing

**Problem:** Data not appearing in ClickHouse immediately

**Explanation:** This is expected behavior. The batch buffer:
- Flushes every 5 seconds OR
- Flushes when 1000 rows accumulated

**Solution:** Wait 5 seconds, or send more data to trigger size threshold.

### High Memory Usage

**Problem:** Application memory grows continuously

**Possible Causes:**
1. Buffer not flushing (check logs for errors)
2. ClickHouse connection lost (retries accumulate in buffer)
3. Very high cardinality data

**Solution:**
1. Check ClickHouse connectivity
2. Monitor buffer size via `/health` endpoint
3. Review application logs for flush errors
4. Consider reducing batch size if memory constrained

### Slow Queries

**Problem:** REST API responses are slow

**Diagnostics:**
```sql
-- Find slow queries in ClickHouse
SELECT 
    query,
    query_duration_ms,
    read_rows
FROM system.query_log
WHERE type = 'QueryFinish'
  AND query_duration_ms > 100
ORDER BY query_duration_ms DESC
LIMIT 10;
```

**Solutions:**
- Always use `FINAL` modifier for ReplacingMergeTree queries
- Add `LIMIT` clauses to large result sets
- Use pagination in API requests
- Consider adding materialized views for common aggregations

## Migration Notes

### From SQLite

The ClickHouse migration removes SQLite completely:

**Removed:**
- `internal/storage/sqlite/` directory
- SQLite dependency from `go.mod`
- SQL migration files

**Added:**
- `internal/storage/clickhouse/` package
- Batch buffer system
- ReplacingMergeTree and SummingMergeTree engines

**Benefits:**
- 10x write throughput improvement
- 5-10x faster read latency
- Exact cardinality (no HyperLogLog estimation)
- Better scalability for production workloads

### From Memory Storage

Memory storage remains available for development/testing:

```bash
# Use memory backend (no persistence)
export STORAGE_BACKEND="memory"
./bin/occ
```

**When to use Memory:**
- Local development
- Unit testing
- Temporary analysis
- Resource-constrained environments

**When to use ClickHouse:**
- Production deployments
- Long-running analysis
- Historical data tracking
- High-throughput scenarios

## Best Practices

1. **Always use FINAL**: Queries against ReplacingMergeTree tables should use `FINAL` modifier
2. **Partition management**: Monitor partition sizes, consider TTL for old data
3. **Connection pooling**: Application uses connection pool, don't override defaults
4. **Batch writes**: Let buffer handle batching, don't force manual flushes
5. **Query optimization**: Use `LIMIT`, indexes, and materialized views for common queries
6. **Monitoring**: Track buffer sizes, flush latency, and query performance
7. **Backup strategy**: ClickHouse data is in `/tmp` by default, use persistent volume in production

## Future Enhancements

Planned improvements:
- [ ] Configurable batch buffer size and interval via environment variables
- [ ] Parallel batch writers for higher throughput
- [ ] Materialized views for common aggregation queries
- [ ] API v2 endpoints with per-label cardinality details
- [ ] ClickHouse cluster support for distributed deployments
- [ ] Custom retention policies per signal type

## References

- [ClickHouse Documentation](https://clickhouse.com/docs)
- [ReplacingMergeTree Engine](https://clickhouse.com/docs/en/engines/table-engines/mergetree-family/replacingmergetree)
- [SummingMergeTree Engine](https://clickhouse.com/docs/en/engines/table-engines/mergetree-family/summingmergetree)
- [ClickHouse Go Client](https://github.com/ClickHouse/clickhouse-go)
