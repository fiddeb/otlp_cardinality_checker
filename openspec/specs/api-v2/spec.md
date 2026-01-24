# api-v2 Specification

## Purpose
TBD - created by archiving change migrate-to-clickhouse. Update Purpose after archive.
## Requirements
### Requirement: API Versioning Strategy

The system SHALL expose both `/v1/` and `/v2/` API prefixes with different capabilities.

#### Scenario: v1 endpoints remain unchanged
- **GIVEN** existing `/v1/` endpoints serve current UI
- **WHEN** ClickHouse backend is enabled
- **THEN** `/v1/` endpoints continue to function
- **AND** `/v1/` endpoints use same Storage interface (now ClickHouse implementation)
- **AND** response formats are identical to previous behavior

#### Scenario: v2 endpoints expose ClickHouse features
- **GIVEN** ClickHouse backend provides advanced aggregations
- **WHEN** client requests `/v2/` endpoints
- **THEN** responses include additional fields not in v1
- **AND** queries leverage ClickHouse-specific functions (uniqExact, topK, etc.)
- **AND** `/v2/` returns HTTP 501 if backend is Memory (not ClickHouse)

#### Scenario: API version negotiation
- **GIVEN** client supports both v1 and v2
- **WHEN** client sends Accept: application/vnd.otlp.v2+json header
- **THEN** server routes to v2 endpoints
- **AND** response includes API-Version: v2 header
- **AND** default (no header) routes to v1 for compatibility

### Requirement: Enhanced Metrics Endpoint

The system SHALL provide `/v2/metrics` endpoint with additional cardinality and complexity metadata.

#### Scenario: List metrics with per-label cardinality
- **GIVEN** ClickHouse attribute_values table tracks cardinality
- **WHEN** GET /v2/metrics?service=api-gateway
- **THEN** response includes array of metrics with:
  - name, metric_type, unit (same as v1)
  - label_keys with estimated_cardinality per key
  - resource_keys with estimated_cardinality per key
  - complexity_score = label_count × max_label_cardinality
  - sample_count, first_seen, last_seen
- **AND** cardinality computed via ClickHouse query joining metrics + attribute_values

#### Scenario: Filter by complexity threshold
- **GIVEN** metrics have varying complexity scores
- **WHEN** GET /v2/metrics?min_complexity=500
- **THEN** only metrics with complexity_score >= 500 are returned
- **AND** results sorted by complexity_score DESC
- **AND** query uses HAVING clause on computed complexity

### Requirement: Aggregated Cardinality Endpoint

The system SHALL provide `/v2/cardinality` endpoint for cross-signal attribute analysis.

#### Scenario: Top cardinality keys across all signals
- **GIVEN** attribute_values table tracks keys from metrics, spans, logs
- **WHEN** GET /v2/cardinality?limit=20
- **THEN** response includes top 20 keys by cardinality with:
  - key name
  - estimated_cardinality (uniqExact across all signals)
  - signal_types (array: ["metric", "span", "log"])
  - total_observations count
  - value_samples (first 5 unique values)
- **AND** query groups by key, no JOINs needed

#### Scenario: Filter by signal type
- **GIVEN** some keys appear only in spans
- **WHEN** GET /v2/cardinality?signal_type=span&min_cardinality=100
- **THEN** only keys from span signal_type with cardinality >= 100 are returned
- **AND** WHERE clause filters attribute_values by signal_type column

#### Scenario: Cardinality trend over time
- **GIVEN** attribute_values has first_seen and last_seen timestamps
- **WHEN** GET /v2/cardinality/http.user_id?period=1h
- **THEN** response includes hourly buckets with cardinality per hour
- **AND** query uses toStartOfHour(first_seen) for bucketing
- **AND** cardinality computed per bucket using uniqExact(value)

### Requirement: Pattern Explorer V2 Endpoint

The system SHALL provide `/v2/logs/patterns` endpoint with enhanced log pattern analytics.

#### Scenario: Patterns with service distribution
- **GIVEN** logs table tracks patterns per service
- **WHEN** GET /v2/logs/patterns?min_services=3
- **THEN** response includes patterns appearing in 3+ services with:
  - pattern_template (e.g., "user <*> logged in")
  - severity levels where pattern appears
  - service_count and services array
  - total_count across all services
  - example_body for pattern
- **AND** query groups by pattern_template, aggregates service_count

#### Scenario: Pattern similarity search
- **GIVEN** Drain algorithm produces similar patterns
- **WHEN** GET /v2/logs/patterns/similar?template=user%20*%20logged%20in
- **THEN** response includes patterns with edit distance < 3 from query
- **AND** ClickHouse uses ngramDistance() function for similarity
- **AND** results sorted by similarity score DESC

### Requirement: Performance Metrics Endpoint

The system SHALL provide `/v2/stats` endpoint exposing ClickHouse query performance metrics.

#### Scenario: Query execution statistics
- **GIVEN** ClickHouse tracks query execution time and rows scanned
- **WHEN** GET /v2/stats
- **THEN** response includes:
  - total_queries_executed counter
  - avg_query_duration_ms gauge
  - p95_query_duration_ms gauge
  - rows_scanned_per_second gauge
  - active_connections gauge
- **AND** metrics queried from ClickHouse system.query_log table

#### Scenario: Table size and row counts
- **GIVEN** ClickHouse system tables track storage size
- **WHEN** GET /v2/stats/tables
- **THEN** response includes per-table statistics:
  - table_name (metrics, spans, logs, attribute_values)
  - row_count (approximate via count())
  - compressed_size_mb
  - uncompressed_size_mb
  - compression_ratio
- **AND** data queried from system.parts table

### Requirement: Batch Query Endpoint

The system SHALL provide `/v2/query/batch` endpoint for executing multiple queries in one request.

#### Scenario: Batch query execution
- **GIVEN** client needs data from multiple endpoints
- **WHEN** POST /v2/query/batch with body:
  ```json
  {
    "queries": [
      {"id": "metrics", "endpoint": "/v2/metrics", "params": {"service": "api"}},
      {"id": "cardinality", "endpoint": "/v2/cardinality", "params": {"limit": 10}}
    ]
  }
  ```
- **THEN** both queries execute in parallel against ClickHouse
- **AND** response contains results keyed by id:
  ```json
  {
    "results": {
      "metrics": {...},
      "cardinality": {...}
    }
  }
  ```
- **AND** if any query fails, its result includes error field

### Requirement: Streaming Response for Large Datasets

The system SHALL support streaming JSON responses for queries returning >1000 rows.

#### Scenario: Stream large attribute list
- **GIVEN** attribute_values table has 50k unique keys
- **WHEN** GET /v2/cardinality?limit=10000
- **THEN** response uses chunked transfer encoding
- **AND** JSON array items stream as completed by ClickHouse
- **AND** client receives first rows before query completes
- **AND** memory usage remains bounded (no full result buffering)

