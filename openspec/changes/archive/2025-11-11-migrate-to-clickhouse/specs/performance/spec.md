# Performance Requirements

> Capability: performance
> Status: PROPOSED
> Related Specs: clickhouse-storage, api-v2

## Overview

This spec defines performance benchmarks and testing requirements for the ClickHouse migration. The primary goal is 10x improvement in write throughput and 5-10x improvement in read latency compared to SQLite.

## ADDED Requirements

### Requirement: Write Throughput Performance

The system SHALL achieve 10x higher write throughput with ClickHouse compared to SQLite baseline.

#### Scenario: Sustained high write load
- **GIVEN** k6 load test sends 50,000 metric observations per second
- **WHEN** test runs for 5 minutes with ClickHouse backend
- **THEN** all writes complete without errors
- **AND** average write latency < 10ms (includes buffering, not full flush)
- **AND** p95 write latency < 50ms
- **AND** CPU usage < 50% on 4-core machine
- **AND** throughput is 10x SQLite baseline (~5k writes/sec)

#### Scenario: Batch write efficiency
- **GIVEN** batch buffer accumulates 1000 rows over 2 seconds
- **WHEN** buffer flushes to ClickHouse
- **THEN** flush completes in <100ms
- **AND** ClickHouse receives rows in single batch INSERT
- **AND** network round trips = 1 (not 1000 individual inserts)
- **AND** flush latency logged for monitoring

#### Scenario: Write during read load
- **GIVEN** concurrent read queries hitting ClickHouse
- **WHEN** batch writes occur simultaneously
- **THEN** write throughput does not degrade >10%
- **AND** read query latency does not increase >20%
- **AND** no write errors occur due to lock contention (columnar storage benefit)

### Requirement: Read Query Performance

The system SHALL achieve 5-10x lower query latency with ClickHouse compared to SQLite baseline.

#### Scenario: List metrics query
- **GIVEN** metrics table contains 100,000 unique metrics
- **WHEN** GET /v1/metrics?service=api-gateway executes
- **THEN** query completes in <50ms (vs ~200ms SQLite)
- **AND** query scans only metrics table (no JOINs)
- **AND** ClickHouse uses index on (name, service_name)
- **AND** result contains 100-500 metrics typically

#### Scenario: High cardinality analysis query
- **GIVEN** attribute_values table has 10M rows
- **WHEN** GET /v2/cardinality?min_cardinality=1000 executes
- **THEN** query completes in <200ms (vs >1s SQLite)
- **AND** query uses uniqExact() aggregation (exact, not estimated)
- **AND** query scans value column efficiently (columnar storage)
- **AND** result contains top 100 high-cardinality keys

#### Scenario: Complex aggregation query
- **GIVEN** logs table has 500k pattern rows across 20 services
- **WHEN** GET /v2/logs/patterns?min_services=5&min_count=1000 executes
- **THEN** query completes in <100ms (vs ~500ms SQLite)
- **AND** query groups by pattern_template, filters HAVING clauses
- **AND** no JOINs required (denormalized schema)
- **AND** result contains 50-200 patterns typically

### Requirement: Concurrent Query Performance

The system SHALL handle 100+ concurrent read queries without degradation.

#### Scenario: High concurrency load test
- **GIVEN** k6 test runs 100 virtual users querying APIs simultaneously
- **WHEN** test runs for 2 minutes
- **THEN** all queries complete successfully (0 errors)
- **AND** median query latency remains <50ms
- **AND** p95 query latency < 200ms
- **AND** ClickHouse connection pool provides sufficient connections (10)
- **AND** no connection exhaustion errors occur

#### Scenario: Query isolation
- **GIVEN** one client executes slow aggregation query (e.g., full table scan)
- **WHEN** other clients execute fast lookup queries
- **THEN** fast queries complete in normal time (<50ms)
- **AND** slow query does not block fast queries
- **AND** ClickHouse query priorities can be configured if needed

### Requirement: Cardinality Estimation Accuracy

The system SHALL provide exact cardinality counts using ClickHouse uniqExact() function.

#### Scenario: Exact cardinality for attribute key
- **GIVEN** attribute_values table has key "http.method" with 9 unique values
- **WHEN** cardinality is computed via uniqExact(value)
- **THEN** result is exactly 9 (not estimated)
- **AND** no HyperLogLog approximation error (previous: ±0.81%)
- **AND** query completes in <20ms even for 100k unique values

#### Scenario: Compare HyperLogLog vs uniqExact
- **GIVEN** baseline HyperLogLog implementation has ~1% error
- **WHEN** same dataset is computed with uniqExact()
- **THEN** results are exact (0% error)
- **AND** query time is comparable or faster than HyperLogLog in-memory computation
- **AND** no per-key memory overhead (16KB HLL sketch eliminated)

### Requirement: Memory Efficiency

The system SHALL reduce in-memory footprint by offloading storage to ClickHouse.

#### Scenario: Memory usage with ClickHouse backend
- **GIVEN** application tracks 100k metrics with 10M attribute observations
- **WHEN** ClickHouse backend is used
- **THEN** application memory usage < 500MB (batch buffer + cache)
- **AND** ClickHouse server memory usage varies (query-dependent)
- **AND** no in-memory HyperLogLog sketches (16KB × 100k keys eliminated)

#### Scenario: Compare memory vs SQLite
- **GIVEN** SQLite implementation used in-memory cache for attributes
- **WHEN** same workload runs with ClickHouse
- **THEN** application memory usage reduced by 50%+
- **AND** no need for 5-second batch flush cache (writes async)
- **AND** total system memory (app + DB) is comparable

### Requirement: Load Test Scenarios

The system SHALL include comprehensive k6 load tests for benchmarking.

#### Scenario: Write-heavy load test
- **GIVEN** k6 script `scripts/k6-clickhouse-write.js`
- **WHEN** test sends 10k metrics/sec, 5k spans/sec, 3k logs/sec for 5 min
- **THEN** test measures:
  - requests per second (RPS) achieved
  - p50, p95, p99 write latency
  - error rate (target: <0.1%)
  - ClickHouse CPU and memory usage
- **AND** results compared to SQLite baseline

#### Scenario: Read-heavy load test
- **GIVEN** k6 script `scripts/k6-clickhouse-read.js`
- **WHEN** test executes 100 req/sec across all API endpoints for 3 min
- **THEN** test measures:
  - p50, p95, p99 read latency per endpoint
  - throughput (queries/sec)
  - ClickHouse query_log statistics
- **AND** results compared to SQLite baseline

#### Scenario: Mixed workload test
- **GIVEN** k6 script `scripts/k6-clickhouse-mixed.js`
- **WHEN** test runs 70% writes, 30% reads for 5 min
- **THEN** test measures:
  - write throughput not degraded by concurrent reads
  - read latency not degraded by concurrent writes
  - overall system stability (no OOM, no crashes)
- **AND** results validate 10x improvement claim

### Requirement: Performance Regression Testing

The system SHALL include automated performance regression detection.

#### Scenario: Baseline performance capture
- **GIVEN** initial ClickHouse implementation passes all load tests
- **WHEN** baseline metrics are recorded
- **THEN** results stored as JSON: `benchmarks/clickhouse-baseline.json`
- **AND** metrics include: write_rps, read_p95_ms, cardinality_query_p95_ms
- **AND** baseline serves as comparison for future changes

#### Scenario: Regression detection on changes
- **GIVEN** code changes are made to ClickHouse storage
- **WHEN** load tests re-run
- **THEN** results compared to baseline
- **AND** if write throughput degrades >20%, test fails
- **AND** if read latency increases >30%, test fails
- **AND** CI/CD pipeline blocks merge on regression

## Cross-References

- **clickhouse-storage**: Performance tests validate storage implementation
- **api-v2**: V2 endpoint performance measured separately from v1
- **storage (existing spec)**: Performance requirements supersede SQLite performance goals
