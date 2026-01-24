# clickhouse-storage Specification

## Purpose
TBD - created by archiving change migrate-to-clickhouse. Update Purpose after archive.
## Requirements
### Requirement: ClickHouse Connection Management

The system SHALL establish and maintain a connection to ClickHouse server on localhost:9000 using native protocol.

#### Scenario: Successful connection on startup
- **GIVEN** ClickHouse server is running on localhost:9000
- **WHEN** the application starts with `STORAGE_TYPE=clickhouse`
- **THEN** a connection pool is established with native protocol
- **AND** schema initialization queries execute successfully
- **AND** the application logs "ClickHouse storage initialized"

#### Scenario: Connection failure handling
- **GIVEN** ClickHouse server is not available
- **WHEN** the application attempts to connect
- **THEN** connection retries occur with exponential backoff (1s, 2s, 4s, 8s)
- **AND** after 5 failed attempts, the application logs an error and exits
- **AND** error message includes connection details for debugging

#### Scenario: Connection pool management
- **GIVEN** an active ClickHouse connection
- **WHEN** concurrent requests require database access
- **THEN** the connection pool provides up to 10 concurrent connections
- **AND** connections are reused across requests
- **AND** idle connections close after 5 minutes

### Requirement: Schema Initialization

The system SHALL create ClickHouse tables on first startup if they do not exist.

#### Scenario: Fresh database initialization
- **GIVEN** ClickHouse database has no tables
- **WHEN** the application starts
- **THEN** CREATE TABLE statements execute for: metrics, spans, logs, attribute_values, services
- **AND** all tables use appropriate ENGINE (ReplacingMergeTree or SummingMergeTree)
- **AND** all tables have PRIMARY KEY defined
- **AND** schema version is recorded in a `schema_version` table

#### Scenario: Existing schema detected
- **GIVEN** ClickHouse database already has tables
- **WHEN** the application starts
- **THEN** no CREATE TABLE statements execute
- **AND** schema version is validated against expected version
- **AND** if version mismatch, application logs warning and continues

### Requirement: Batch Write Operations

The system SHALL buffer incoming metadata and write to ClickHouse in batches for optimal throughput.

#### Scenario: Batch size threshold triggers flush
- **GIVEN** batch buffer is configured with size threshold 1000
- **WHEN** 1000 rows are accumulated in buffer
- **THEN** buffer flushes to ClickHouse immediately
- **AND** buffer is cleared after successful write
- **AND** write latency is logged

#### Scenario: Time interval triggers flush
- **GIVEN** batch buffer is configured with 5-second flush interval
- **WHEN** 5 seconds elapse since last flush
- **THEN** buffer flushes to ClickHouse regardless of size
- **AND** buffer may contain 1-999 rows
- **AND** empty buffers are not flushed

#### Scenario: Graceful shutdown flushes buffer
- **GIVEN** batch buffer contains 200 unflushed rows
- **WHEN** application receives shutdown signal
- **THEN** buffer flushes immediately
- **AND** shutdown waits for flush completion (max 10 seconds)
- **AND** connections close after flush

#### Scenario: Batch write error handling
- **GIVEN** batch contains 500 rows
- **WHEN** batch write fails with network error
- **THEN** write is retried up to 3 times with exponential backoff
- **AND** if all retries fail, rows are logged to disk for manual recovery
- **AND** error counter metric is incremented

### Requirement: Denormalized Schema for Fast Queries

The system SHALL store metadata in denormalized tables optimized for columnar scans without JOINs.

#### Scenario: Metric storage includes all context
- **GIVEN** a metric with name "http_requests_total" from service "api-gateway"
- **WHEN** metric metadata is stored
- **THEN** metrics table row includes: name, service_name, label_keys (array), resource_keys (array), metric_type, sample_count, timestamps
- **AND** all label keys are stored in a single Array(String) column
- **AND** no separate metric_keys table exists (denormalized)

#### Scenario: Query metrics without JOINs
- **GIVEN** metrics table contains 100k rows
- **WHEN** API queries "list all metrics for service X with >5 labels"
- **THEN** query scans only metrics table (no JOINs)
- **AND** WHERE clause filters on service_name column
- **AND** HAVING clause filters on length(label_keys) > 5
- **AND** query completes in <50ms

### Requirement: Attribute Cardinality Tracking

The system SHALL track attribute key-value pairs in attribute_values table and compute cardinality using ClickHouse native functions.

#### Scenario: Store attribute observation
- **GIVEN** an attribute key "http.method" with value "GET" from a metric
- **WHEN** attribute is observed
- **THEN** attribute_values table is inserted with: key, value, signal_type="metric", scope="attribute", observation_count=1, timestamps
- **AND** SummingMergeTree automatically merges duplicate (key, value, signal_type, scope) rows
- **AND** observation_count sums across merges

#### Scenario: Compute cardinality for attribute key
- **GIVEN** attribute_values contains 1000 rows for key "user_id" with 523 unique values
- **WHEN** API requests cardinality for "user_id"
- **THEN** query uses uniqExact(value) aggregation
- **AND** result returns estimated_cardinality = 523 (exact)
- **AND** query includes groupArray(5)(value) for first 5 samples
- **AND** query completes in <20ms

#### Scenario: Cross-signal cardinality analysis
- **GIVEN** attribute "service.name" appears in metrics, spans, and logs
- **WHEN** API requests high-cardinality keys across all signals
- **THEN** query groups by key, aggregates uniqExact(value), groupArrayDistinct(signal_type)
- **AND** result includes keys with cardinality >1000 from any signal
- **AND** query scans attribute_values table only (no JOINs)

### Requirement: ReplacingMergeTree Deduplication

The system SHALL use ReplacingMergeTree engine to automatically deduplicate metadata rows based on primary key.

#### Scenario: Metric metadata update merges with existing row
- **GIVEN** metrics table has row for ("http_requests_total", "api-gateway")
- **WHEN** same metric is observed again with updated last_seen timestamp
- **THEN** new row is inserted with same (name, service_name)
- **AND** ClickHouse eventually merges rows, keeping the one with latest last_seen
- **AND** sample_count from both rows is NOT summed (latest wins)

#### Scenario: Query returns deduplicated data
- **GIVEN** metrics table has duplicate rows awaiting merge
- **WHEN** API queries metrics
- **THEN** FINAL modifier is used in SELECT query
- **AND** ClickHouse returns deduplicated rows immediately
- **AND** merge happens asynchronously in background

### Requirement: Storage Interface Compatibility

The system SHALL implement the existing Storage interface for drop-in replacement of SQLite.

#### Scenario: StoreMetric implementation
- **GIVEN** Storage interface defines StoreMetric(ctx, *MetricMetadata) error
- **WHEN** ClickHouse store implements StoreMetric
- **THEN** method buffers metric row in batch buffer
- **AND** method returns immediately (async write)
- **AND** errors are logged but do not block caller

#### Scenario: GetMetric implementation
- **GIVEN** Storage interface defines GetMetric(ctx, name string) (*MetricMetadata, error)
- **WHEN** ClickHouse store implements GetMetric
- **THEN** method queries metrics table with WHERE name = ?
- **AND** method uses FINAL modifier for deduplicated results
- **AND** method returns ErrNotFound if no rows match

#### Scenario: ListAttributes implementation
- **GIVEN** Storage interface defines ListAttributes(ctx, filter) ([]*AttributeMetadata, error)
- **WHEN** ClickHouse store implements ListAttributes
- **THEN** method queries attribute_values table with dynamic WHERE clauses
- **AND** method aggregates cardinality using uniqExact(value)
- **AND** method applies pagination using LIMIT/OFFSET

