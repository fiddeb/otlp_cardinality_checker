# Storage Specification Changes

## MODIFIED Requirements

### Requirement: Attribute Value Storage Efficiency

**Description**: The storage backend SHALL track attribute cardinality using HyperLogLog sketches and MUST NOT store all unique values, preventing cardinality explosion in the metadata database itself.

**Requirements**:
- The system SHALL use HyperLogLog (HLL) for cardinality estimation
- The system MUST store at most 5 sample values per attribute key
- Storage growth SHALL be O(number of keys), not O(cardinality)
- Each attribute key SHALL consume at most 20KB of memory (12KB HLL + 8KB samples/metadata)

**Rationale**: Storing every unique value (current ClickHouse behavior) defeats the purpose of a metadata analyzer - we should track **what keys exist** and **their cardinality**, not become a full telemetry backend.

**Current Behavior**:
```
attribute_values table:
- Row per (key, value) pair
- 1000 unique user.id values → 1000 rows
- Storage: O(cardinality) ❌
```

**New Behavior**:
```
attribute_catalog table:
- Row per key
- 1000 unique user.id values → 1 row with HLL sketch
- Storage: O(num_keys) ✅
```

#### Scenario 1: Store High-Cardinality Attribute

**Given**: Application sends 10,000 unique `user.id` values  
**When**: OTLP receiver processes metrics/spans/logs with different user IDs  
**Then**: 
- ✅ Only **1 row** created in `attribute_catalog` for key="user.id"
- ✅ HLL sketch updated incrementally in memory
- ✅ Estimated cardinality = ~10,000 (±2% error)
- ✅ 5 sample values stored: ["user-1", "user-42", "user-999", ...]

#### Scenario 2: Query Cardinality

**Given**: `attribute_catalog` has key="http.status_code" with 301 unique values  
**When**: API calls `GET /api/v1/cardinality/complexity`  
**Then**:
- ✅ Returns `max_cardinality: 301` from `estimated_cardinality` column
- ✅ Query completes in <50ms (no `uniqExact` on millions of rows)
- ✅ No need to scan all values in memory or disk

#### Scenario 3: Periodic Flush

**Given**: 100 dirty keys in `attrCache` after 60s  
**When**: Flush timer triggers  
**Then**:
- ✅ Batch INSERT/UPDATE to `attribute_catalog` completes in <100ms
- ✅ HLL state serialized correctly (binary or ClickHouse AggregateFunction)
- ✅ Value samples truncated to 5 most recent
- ✅ `attrDirty` map cleared after successful flush

#### Scenario 4: Graceful Shutdown

**Given**: Server receives SIGTERM with 50 dirty keys  
**When**: Shutdown hook executes  
**Then**:
- ✅ Final `flushAttributes()` called before exit
- ✅ All dirty keys persisted to ClickHouse
- ✅ No data loss
- ✅ On restart, cache hydrated from `attribute_catalog`

### Requirement: Cardinality Estimation Accuracy

**Description**: HyperLogLog-based cardinality estimation SHALL provide sufficient accuracy for metadata analysis purposes.

**Requirements**:
- The system SHALL estimate cardinality within ±5% for values > 1000
- The system SHALL estimate cardinality within ±15% for values < 100
- The system MUST correctly classify attributes as "high cardinality" (>100 unique values)
- Cardinality estimates SHALL be monotonically increasing (never decrease as more data arrives)

**Rationale**: Perfect accuracy is unnecessary - users need to identify "high cardinality" attributes (>100 values), not exact counts.

#### Scenario 1: Low Cardinality (< 100)

**Given**: Attribute `http.method` has exactly 7 unique values  
**When**: HLL estimates cardinality  
**Then**:
- ✅ Estimated cardinality between 6-8 (within ±1)
- ✅ Error rate < 15% acceptable for low cardinality

#### Scenario 2: High Cardinality (1000+)

**Given**: Attribute `user.id` has exactly 10,000 unique values  
**When**: HLL estimates cardinality  
**Then**:
- ✅ Estimated cardinality between 9,800-10,200 (±2%)
- ✅ Correctly identified as "high cardinality" (>100)
- ✅ UI shows warning about potential issues

#### Scenario 3: Very High Cardinality (1M+)

**Given**: Attribute `trace.id` has 1,000,000 unique values  
**When**: HLL estimates cardinality  
**Then**:
- ✅ Estimated cardinality between 980,000-1,020,000 (±2%)
- ✅ HLL memory usage remains constant (~12KB)
- ✅ No memory explosion from storing actual values

### Requirement: Value Sampling for Debugging

**Description**: The system SHALL store up to 5 example values per attribute key to help users understand what data looks like.

**Requirements**:
- The system SHALL store at most 5 sample values per attribute key
- Sample values SHALL be representative of actual data (first N or random sampling)
- Sample values MUST be persisted on flush to survive restarts
- The system SHALL display samples in the UI Details view

**Rationale**: Users need examples to debug high-cardinality issues (e.g., "Oh, we're logging UUIDs as attributes!").

#### Scenario 1: Sample Collection

**Given**: Attribute `user.id` receives 1000 different values  
**When**: In-memory cache collects samples  
**Then**:
- ✅ Stores first 5 unique values encountered
- ✅ OR stores 5 random samples (reservoir sampling)
- ✅ Samples persisted on flush

#### Scenario 2: Sample Display in UI

**Given**: `attribute_catalog` has 5 samples for `http.url`  
**When**: User views Details page for a metric/span  
**Then**:
- ✅ Displays: "/api/users", "/api/orders", "/health", "/metrics", "/login"
- ✅ Shows "...and 1245 more unique values" if high cardinality
- ✅ Helps user identify patterns (URLs not parameterized)

## REMOVED Requirements

### ~~Requirement: Store All Unique Values~~

**Reason**: This requirement caused cardinality explosion. Replaced with HLL + sampling approach.

## NEW Requirements

### Requirement: In-Memory Cache for Performance

**Description**: Attribute metadata SHALL be cached in memory to avoid database writes on every OTLP message.

**Requirements**:
- The system SHALL maintain an in-memory cache of all attribute metadata
- Cache writes MUST complete in <1μs per attribute value
- The system SHALL periodically flush dirty cache entries to ClickHouse (default: 60s)
- The system MUST flush all dirty entries on graceful shutdown
- Cache hydration from database SHALL occur on first access (lazy loading)

**Rationale**: Writing to ClickHouse on every span/metric/log is too slow. Batching via in-memory cache + periodic flush is required.

#### Scenario 1: Cache Hit

**Given**: Cache contains key="service.name"  
**When**: New span arrives with service.name="my-svc"  
**Then**:
- ✅ No database query
- ✅ HLL.Add() called in memory
- ✅ Marked as dirty for next flush
- ✅ Process completes in <1μs

#### Scenario 2: Cache Miss (Cold Start)

**Given**: Cache empty after restart  
**When**: First span arrives with key="service.name"  
**Then**:
- ✅ Query `attribute_catalog` for existing data
- ✅ Load HLL state into memory
- ✅ Subsequent writes hit cache
- ✅ One-time DB read penalty acceptable

#### Scenario 3: Memory Limit

**Given**: 10,000 unique attribute keys in system  
**When**: All keys loaded in cache  
**Then**:
- ✅ Memory usage ~120MB (12KB HLL * 10k keys)
- ✅ Acceptable for production workloads
- ✅ No eviction policy needed (bounded by key count)
