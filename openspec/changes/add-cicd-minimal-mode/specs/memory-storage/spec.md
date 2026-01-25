# Spec Delta: Memory Storage

## ADDED Requirements

### Requirement: Bounded Memory Storage

**ID**: `memory-storage.bounded`  
**Status**: Draft  
**Priority**: Critical

#### Description
The system SHALL implement bounded in-memory storage with automatic eviction to prevent unbounded memory growth in resource-constrained CI/CD environments.

#### Requirements
1. Storage MUST be fully in-memory (no persistence)
2. Storage MUST enforce configurable maximum memory limit
3. Default limit MUST be 512MB
4. Storage MUST track current memory usage
5. Storage MUST evict data when approaching memory limit
6. Storage MUST be thread-safe for concurrent access
7. Storage SHALL use RWMutex for read-heavy workloads

#### Scenario: Storage respects memory limit
**GIVEN** storage initialized with 256MB limit  
**WHEN** OTLP data is ingested continuously  
**THEN** storage memory usage SHALL NOT exceed 256MB  
**AND** eviction SHALL occur when necessary

#### Scenario: Concurrent access safe
**GIVEN** storage is receiving OTLP writes  
**WHEN** API reads occur simultaneously  
**THEN** no data races SHALL occur  
**AND** reads SHALL return consistent data

---

### Requirement: Eviction Policy

**ID**: `memory-storage.eviction`  
**Status**: Draft  
**Priority**: High

#### Description
The system SHALL implement a Least Recently Used (LRU) eviction policy to remove oldest metrics when memory limit is reached, preserving recent data for accurate cardinality analysis.

#### Requirements
1. Eviction policy MUST be LRU based on `LastSeen` timestamp
2. Eviction MUST trigger at 90% memory usage
3. Eviction MUST remove at least 10% of data per cycle
4. Eviction MUST maintain cardinality accuracy for remaining data
5. Eviction events MUST be logged at INFO level
6. Eviction MUST complete within 1 second

#### Scenario: LRU eviction when limit reached
**GIVEN** storage at 90% capacity (460MB / 512MB)  
**AND** metric A last seen 10 minutes ago  
**AND** metric B last seen 1 minute ago  
**WHEN** eviction is triggered  
**THEN** metric A SHALL be evicted first  
**AND** metric B SHALL be retained

#### Scenario: Eviction logged
**GIVEN** storage triggers eviction  
**WHEN** eviction completes  
**THEN** log SHALL contain "Evicted N metrics, freed X MB"  
**AND** log level SHALL be INFO

---

### Requirement: Memory Usage Tracking

**ID**: `memory-storage.tracking`  
**Status**: Draft  
**Priority**: High

#### Description
The system SHALL continuously track memory usage and expose metrics for monitoring, enabling proactive management and troubleshooting.

#### Requirements
1. Storage MUST calculate memory usage using runtime.MemStats
2. Memory usage MUST be updated every 10 seconds
3. System SHALL log warning at 80% memory threshold
4. Memory usage MUST be exposed via `/api/v1/health` endpoint
5. Tracking overhead MUST be <1% CPU usage

#### Scenario: Warning at 80% threshold
**GIVEN** storage limit is 512MB  
**WHEN** memory usage reaches 410MB (80%)  
**THEN** warning SHALL be logged once  
**AND** log SHALL contain current usage and limit

#### Scenario: Memory exposed in health endpoint
**GIVEN** storage memory usage is 300MB  
**AND** storage limit is 512MB  
**WHEN** GET /api/v1/health is called  
**THEN** response SHALL include `memory_usage_mb: 300`  
**AND** response SHALL include `memory_limit_mb: 512`  
**AND** response SHALL include `memory_percent: 58.6`

---

### Requirement: Storage Data Model

**ID**: `memory-storage.data-model`  
**Status**: Draft  
**Priority**: High

#### Description
The system SHALL store metric and attribute metadata efficiently, optimized for cardinality queries while minimizing memory overhead.

#### Requirements
1. Storage MUST index metrics by name (map key)
2. Storage MUST index attributes by key (map key)
3. Storage MUST track per-metric: name, label_keys, sample_count, last_seen, cardinality
4. Storage MUST track per-attribute: key, usage_count, estimated_cardinality, last_seen
5. Storage MUST NOT store actual label values (only keys and cardinality estimates)
6. Cardinality estimation MUST use HyperLogLog or equivalent probabilistic algorithm
7. Memory per metric MUST average <1KB

#### Scenario: Store metric metadata only
**GIVEN** OTLP metric "http_requests_total" with labels {method="GET", path="/api"}  
**WHEN** metric is stored  
**THEN** storage SHALL contain metric name  
**AND** storage SHALL contain label keys ["method", "path"]  
**AND** storage SHALL NOT contain label values ["GET", "/api"]  
**AND** storage SHALL update cardinality estimate

#### Scenario: Update last seen on re-observation
**GIVEN** metric "http_requests_total" exists with last_seen=T0  
**WHEN** same metric observed again at T1  
**THEN** last_seen SHALL be updated to T1  
**AND** sample_count SHALL be incremented

---

### Requirement: Cardinality Estimation

**ID**: `memory-storage.cardinality`  
**Status**: Draft  
**Priority**: High

#### Description
The system SHALL estimate metric cardinality using probabilistic algorithms to minimize memory usage while maintaining acceptable accuracy for high-cardinality detection.

#### Requirements
1. Cardinality estimation MUST use HyperLogLog (HLL) algorithm
2. HLL precision MUST provide <2% error rate for cardinalities up to 100,000
3. Each HLL sketch MUST consume <16KB memory
4. Cardinality MUST be queryable in O(1) time
5. Estimation MUST update on each new label value combination observed

#### Scenario: Accurate cardinality estimation
**GIVEN** metric with exactly 5000 unique label combinations  
**WHEN** cardinality is queried  
**THEN** estimated cardinality SHALL be within 2% of 5000  
**AND** estimated cardinality SHALL be between 4900 and 5100

#### Scenario: Memory efficient estimation
**GIVEN** 1000 metrics with cardinality tracking  
**WHEN** memory usage is measured  
**THEN** HLL sketches SHALL consume <16MB total (16KB * 1000)

---

### Requirement: Query Performance

**ID**: `memory-storage.query-performance`  
**Status**: Draft  
**Priority**: Medium

#### Description
The system SHALL provide fast query performance for metric and attribute lookups to support real-time API responses.

#### Requirements
1. Metric lookup by name MUST complete in O(1) time
2. List all metrics MUST complete in <100ms for 10,000 metrics
3. Top-cardinality query MUST complete in <500ms for 10,000 metrics
4. Query performance MUST NOT degrade during eviction
5. Concurrent reads MUST NOT block each other

#### Scenario: Fast metric lookup
**GIVEN** storage contains 5000 metrics  
**WHEN** metric lookup by name is performed  
**THEN** query SHALL complete in <1ms

#### Scenario: List all metrics performance
**GIVEN** storage contains 10,000 metrics  
**WHEN** API requests all metrics  
**THEN** query SHALL complete in <100ms  
**AND** all metrics SHALL be returned

---

### Requirement: Storage Lifecycle

**ID**: `memory-storage.lifecycle`  
**Status**: Draft  
**Priority**: High

#### Description
The system SHALL properly initialize and cleanup memory storage resources during startup and shutdown.

#### Requirements
1. Storage MUST initialize in <100ms
2. Storage MUST start empty (no pre-loaded data)
3. Storage MUST flush all data on Close()
4. Storage MUST be idempotent for multiple Close() calls
5. Storage MUST NOT accept writes after Close() called

#### Scenario: Initialize empty storage
**GIVEN** storage is being initialized  
**WHEN** initialization completes  
**THEN** metric count SHALL be 0  
**AND** attribute count SHALL be 0  
**AND** memory usage SHALL be <10MB

#### Scenario: Reject writes after close
**GIVEN** storage has been closed  
**WHEN** write is attempted  
**THEN** write SHALL return error  
**AND** error message SHALL indicate storage closed

## MODIFIED Requirements

None (no existing memory storage requirements to modify)

## REMOVED Requirements

None
