# Scalability Improvements

This document describes the scalability optimizations implemented in the OTLP Cardinality Checker.

## Performance Optimizations

### 1. Optimized Value Sample Collection

**Problem:** Previous implementation sorted value samples on every insert, resulting in O(n log n) performance per metric update.

**Solution:**
- Removed sort operation from `AddValue()` method
- Added custom JSON marshaler that sorts only during serialization
- Improved cardinality tracking to count unique values beyond `MaxSamples` limit

**Impact:**
- Insert performance: O(n log n) → O(1)
- Memory usage: Unchanged (still respects MaxSamples=100)
- Cardinality accuracy: Improved (tracks all unique values, not just sampled ones)

### 2. API Pagination

**Problem:** Listing 10,000+ metrics in a single response could result in 100+ MB JSON payloads and slow response times.

**Solution:**
- Added pagination support to all list endpoints
- Default limit: 100 items per page
- Maximum limit: 1,000 items per page
- Offset-based pagination

**Usage:**

```bash
# Get first page (default 100 items)
curl http://localhost:8080/api/v1/metrics

# Get specific page
curl http://localhost:8080/api/v1/metrics?limit=50&offset=100

# Filter by service with pagination
curl http://localhost:8080/api/v1/metrics?service=my-service&limit=100&offset=0
```

**Response format:**

```json
{
  "data": [...],
  "total": 10000,
  "limit": 100,
  "offset": 0,
  "has_more": true
}
```

**Endpoints with pagination:**
- `GET /api/v1/metrics` - List all metrics
- `GET /api/v1/spans` - List all spans
- `GET /api/v1/logs` - List all log metadata

## Current Performance Characteristics

### With 10,000 Metrics

**Memory usage per metric:**
- Base metadata: ~200 bytes
- Per label key: ~150 bytes + (100 samples × avg 20 bytes) = ~2,150 bytes
- Typical metric with 5 label keys: ~11 KB

**Total for 10,000 metrics with 5 label keys each:**
- Estimated: ~110 MB in memory
- With Go overhead: ~150-200 MB

**API response times (estimated):**
- List 100 metrics: <10ms
- List 1,000 metrics: <50ms
- Get single metric: <1ms

**OTLP ingestion:**
- Per metric update: O(1) for value tracking
- Merge operation: O(k) where k = number of unique label keys
- Concurrent writes: Thread-safe with RWMutex per data structure

## Future Optimizations

### 3. HyperLogLog Cardinality Tracking (Planned)

**Goal:** Reduce memory usage for high-cardinality tracking

**Current approach:**
- Uses set-based cardinality: `EstimatedCardinality = len(valueSampleSet)`
- Accurate up to MaxSamples (100 values)
- Memory: O(n) where n = unique values (capped at 100)

**HyperLogLog approach:**
- Memory: Fixed ~1.5 KB per key (0.81% standard error)
- Accuracy: ±1% for any cardinality
- Trade-off: Slightly less accurate, much more memory-efficient

**When to implement:** When tracking cardinality beyond 1,000 unique values per key becomes common.

### 4. Sharded Maps (Planned)

**Goal:** Reduce lock contention under high concurrency

**Current approach:**
- Single RWMutex per signal type (metrics, spans, logs)
- All concurrent readers/writers compete for same lock
- Fine for <1,000 requests/second

**Sharded approach:**
- Split storage into N shards (e.g., 32)
- Hash metric name to determine shard
- Each shard has independent lock
- Reduces contention by factor of N

**When to implement:** When profiling shows lock contention is a bottleneck (typically >10,000 metrics with high update frequency).

## Scalability Limits

### Current Architecture Limits

**In-Memory Storage:**
- Practical limit: ~50,000-100,000 metrics (depending on label cardinality)
- Memory: ~5-10 GB for 50,000 metrics with average label keys
- Recovery: All metadata lost on restart

**Recommendations for very large deployments:**
1. **PostgreSQL persistence** (planned) - Survive restarts
2. **Horizontal sharding** - Run multiple instances, shard by service name
3. **Metric aggregation** - Pre-aggregate at Collector level
4. **TTL/expiry** - Automatically remove stale metrics after N days

### When to Consider PostgreSQL

Switch to PostgreSQL persistence when:
- You have >50,000 unique metrics
- Restart downtime is unacceptable
- You need historical metadata analysis
- You want to query across time ranges

## Testing Scalability

### Load Testing with 10,000 Metrics

```bash
# Generate test data
for i in {1..10000}; do
  echo '{
    "resource_metrics": [{
      "resource": {"attributes": [{"key": "service.name", "value": {"string_value": "test-service"}}]},
      "scope_metrics": [{
        "metrics": [{
          "name": "metric_'$i'",
          "sum": {
            "data_points": [{
              "attributes": [
                {"key": "label1", "value": {"string_value": "value1"}},
                {"key": "label2", "value": {"string_value": "value2"}}
              ],
              "as_int": 100
            }]
          }
        }]
      }]
    }]
  }' | curl -X POST -H "Content-Type: application/x-protobuf" \
    --data-binary @- http://localhost:4318/v1/metrics
done

# Test pagination performance
time curl -s "http://localhost:8080/api/v1/metrics?limit=1000&offset=0" | jq '.total'
time curl -s "http://localhost:8080/api/v1/metrics?limit=1000&offset=5000" | jq '.total'
```

### Memory Profiling

```bash
# Build with profiling
go build -o otlp-cardinality-checker ./cmd/server

# Run with profiling enabled
./otlp-cardinality-checker &

# Take heap snapshot
curl http://localhost:8080/debug/pprof/heap > heap.prof

# Analyze
go tool pprof -http=:8081 heap.prof
```

## Summary

The current implementation can comfortably handle:
- ✅ 10,000 metrics with standard label cardinality
- ✅ 1,000 requests/second ingestion rate
- ✅ Concurrent queries with pagination
- ✅ Multiple services (filtered queries)

For larger deployments:
- 🔄 Consider PostgreSQL persistence (>50k metrics)
- 🔄 Implement HyperLogLog (high cardinality labels)
- 🔄 Add sharded maps (>10k req/sec)
- 🔄 Horizontal scaling (>100k metrics)
