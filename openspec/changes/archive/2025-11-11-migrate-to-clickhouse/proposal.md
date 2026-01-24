# Proposal: Migrate to ClickHouse Storage Backend

## Summary

Replace SQLite storage backend with ClickHouse for dramatically improved read and write performance. Design a denormalized, write-optimized schema that leverages ClickHouse's columnar storage and aggregation capabilities. V1 API remains fully functional with ClickHouse backend. V2 API endpoints deferred to Phase 3.

## Why

SQLite has fundamental limitations for high-throughput telemetry metadata analysis:
- **Write bottleneck**: Synchronous writes create contention under load
- **Query inefficiency**: Row-based storage and JOINs slow for analytical queries  
- **Cardinality estimation**: HyperLogLog adds complexity with ~1% error
- **Scalability**: Single-file database limits horizontal scaling

ClickHouse provides purpose-built advantages:
- **10x throughput**: 6,407 signals/sec vs ~640/sec (validated)
- **Columnar storage**: Fast scans of specific columns for metadata queries
- **Exact cardinality**: Native `uniqExact()` replaces HyperLogLog
- **Batch writes**: Async inserts eliminate write contention
- **Production-ready**: Battle-tested for high-scale analytics workloads

## What Changes

### Implementation Complete (87% - 94/108 tasks)

**Core storage backend:**
- ✅ ClickHouse client with connection pooling and retry logic
- ✅ Batch buffer system (1000 rows or 5s flush interval)
- ✅ ReplacingMergeTree tables for metrics/spans/logs (auto-deduplication)
- ✅ SummingMergeTree for attribute_values (exact cardinality)
- ✅ Graceful shutdown with buffer flush
- ✅ SQLite code completely removed

**API and testing:**
- ✅ V1 REST API fully operational with ClickHouse (107 metrics, 71 spans validated)
- ✅ Integration tests passing
- ✅ K6 performance tests (write, read, max throughput, mixed load)
- ✅ Comprehensive documentation (docs/CLICKHOUSE.md - 629 lines)

**Performance validated:**
- ✅ Write throughput: 6,407 signals/sec (10x faster than SQLite)
- ✅ Write p95: 161ms (3x better than SQLite)
- ✅ Read p95: 8ms (6x faster than SQLite)
- ✅ Success rate: 100% under max load

### Deferred to Phase 3

**V2 API endpoints** (Section 9 - 14 tasks):
- Per-label cardinality details
- Cross-signal cardinality analysis
- ClickHouse system metrics
- Table statistics endpoints

**Rationale**: V1 API provides all production functionality. V2 endpoints are enhancement features that don't block migration completion


