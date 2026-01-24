# Design: Memory-Only Storage Architecture

## Overview

This design document explains the architectural decisions for refactoring the OTLP Cardinality Checker to use **only in-memory storage**, removing all database persistence layers (ClickHouse and dual-mode).

## Problem Statement

The current storage architecture has three backends with significant complexity:

```
┌─────────────────────────────────────────┐
│ Storage Interface (10 methods)          │
└───────┬─────────────────┬──────────────┬┘
        │                 │              │
   ┌────▼────┐      ┌─────▼──────┐  ┌───▼────┐
   │ Memory  │      │ ClickHouse │  │  Dual  │
   │ 500 LOC │      │ 2,400 LOC  │  │ 350 LOC│
   └─────────┘      └────────────┘  └────────┘
                           │
                    ┌──────▼────────┐
                    │ ClickHouse DB │
                    │ (external)    │
                    └───────────────┘
```

**Complexity issues:**
- 3 implementations of same interface (testing burden)
- ClickHouse requires connection management, schema migrations, batch buffering
- Dual-mode adds async write complexity and potential data inconsistency
- Factory pattern adds indirection for limited value
- ~60% of storage code dedicated to unused persistence

## Design Principles

### 1. Ephemeral by Design
**Rationale**: Metadata analysis is diagnostic, not archival.

A cardinality checker should:
- ✅ Show **current** metadata structure
- ✅ Identify **active** high-cardinality keys
- ✅ Detect **live** patterns and anomalies
- ❌ NOT store historical telemetry data
- ❌ NOT replace observability backends

**Analogy**: Like `htop` for system monitoring - shows current state, data lost on exit.

### 2. Simplicity Over Features
**Trade-off**: Lose persistence, gain maintainability.

```
Complexity = Code × Dependencies × Configuration
Memory-only: 500 LOC × 0 external deps × 0 config = Low
ClickHouse:  2,400 LOC × 2 external deps × 5 config vars = High
```

### 3. Restart = Re-analyze
**Pattern**: Stateless diagnostic tool

If you need to analyze telemetry:
1. Start the checker
2. Point collector at it
3. Analyze metadata
4. Stop when done

No need for persistence between runs.

## Proposed Architecture

### Simplified Storage Layer

```
┌─────────────────────────────────────────┐
│ Storage Interface (unchanged)           │
└───────┬─────────────────────────────────┘
        │
   ┌────▼────────────────────────────┐
   │ Memory Store (500 LOC)          │
   │ • sync.RWMutex for concurrency  │
   │ • HyperLogLog for cardinality   │
   │ • Drain for log patterns        │
   │ • 500k+ metrics capacity        │
   └─────────────────────────────────┘
```

**No factory pattern needed** - Direct instantiation:
```go
// Before (complex)
cfg := storage.Config{
    Backend: "clickhouse",
    ClickHouseAddr: "localhost:9000",
    UseAutoTemplate: true,
}
store, err := storage.NewStorage(cfg)

// After (simple)
store := memory.NewWithAutoTemplate(true)
```

### Memory Store Capabilities

Already implements full interface:
- ✅ Metrics: `StoreMetric()`, `GetMetric()`, `ListMetrics()`
- ✅ Spans: `StoreSpan()`, `GetSpan()`, `ListSpans()`
- ✅ Logs: `StoreLog()`, `GetLog()`, `ListLogs()`, `GetLogPatterns()`
- ✅ Attributes: `StoreAttributeValue()`, `GetAttribute()`, `ListAttributes()`
- ✅ Services: `ListServices()`, `GetServiceOverview()`
- ✅ Analysis: `GetHighCardinalityKeys()`, `GetMetadataComplexity()`
- ✅ Drain: Automatic log template extraction (20-30k EPS)
- ✅ HLL: Cardinality estimation with ~0.81% error

**Performance:** Already exceeds requirements (500k metrics, <256MB, 30k-50k signals/sec).

## Removed Components

### ClickHouse Backend (~2,400 LOC)

**Files deleted:**
```
internal/storage/clickhouse/
├── store.go              (1,489 LOC) - CRUD operations, batch writes
├── buffer.go             (400 LOC)   - Async batch buffer
├── schema.go             (241 LOC)   - Table DDL, migrations
├── connection.go         (150 LOC)   - Connection pooling
└── store_integration_test.go (120 LOC)
```

**Why remove:**
- Never reached production
- Adds operational complexity (database deployment)
- Slower than memory for reads
- Write batching adds latency
- Schema migrations add maintenance burden

### Dual Backend (~350 LOC)

**Files deleted:**
```
internal/storage/dual/
├── store.go              (200 LOC) - Write-through to two backends
└── store_test.go         (150 LOC)
```

**Why remove:**
- Only useful when ClickHouse exists
- Adds async write complexity
- Potential for data inconsistency (secondary write failures)
- No clear use case without persistence

### Configuration Files

**Deleted:**
```
config/clickhouse-config.xml     - ClickHouse server config
scripts/start-clickhouse.sh      - Local dev database startup
scripts/verify-clickhouse-data.sh - Integration test helper
data/clickhouse/*                - Database storage directory
```

## Data Model (Unchanged)

Memory store already uses efficient data structures:

```go
type Store struct {
    // Concurrency
    mu sync.RWMutex
    
    // Core data
    metrics map[string]*models.MetricMetadata
    spans   map[string]*models.SpanMetadata
    logs    map[string]*models.LogMetadata
    
    // Cross-signal tracking
    attributes map[string]*models.AttributeMetadata // HLL-based
    services   map[string]*models.ServiceMetadata
    
    // Drain log templating
    drainProcessor *autotemplate.Processor
    useAutoTemplate bool
}
```

**Memory efficiency:**
- Metadata keys only (not values) → O(num_keys) not O(cardinality)
- HyperLogLog sketches (~16KB per attribute) for cardinality
- Sample values (max 5-10) instead of all values
- Expected: <256MB for 500k metrics with moderate cardinality

## Configuration Changes

### Before (Complex)

```bash
# Main config
STORAGE_BACKEND=clickhouse  # or "memory", "dual"
CLICKHOUSE_ADDR=localhost:9000

# ClickHouse-specific (batch buffer)
CLICKHOUSE_BATCH_SIZE=1000
CLICKHOUSE_FLUSH_INTERVAL=5s

# Dual-mode behavior
# (implicit: write to both, read from primary)
```

Decision tree:
1. Which backend? (3 options)
2. If ClickHouse: What address?
3. If ClickHouse: Batch size?
4. If ClickHouse: Flush interval?

### After (Simple)

```bash
# No storage configuration needed!
# Optionally:
USE_AUTOTEMPLATE=true  # Enable Drain log templating (default: true)
```

## API Behavior

### No Breaking Changes

REST API remains identical:
- `GET /api/v1/metrics` - Returns metrics from memory
- `GET /api/v1/attributes?signal_type=metric` - HLL cardinality from memory
- `GET /api/v1/logs/patterns` - Drain patterns from memory
- All other endpoints unchanged

**Behavioral change:** Data lost on restart (documented, expected).

### Performance Characteristics

| Operation | Memory | ClickHouse | Change |
|-----------|--------|------------|--------|
| Write latency | <1ms | 5-50ms (batched) | ✅ Faster |
| Read latency | <1ms | 10-100ms | ✅ Faster |
| Cardinality query | <10ms (HLL) | 50-200ms (uniqExact) | ✅ Faster |
| Startup time | Instant | 100-500ms (connect) | ✅ Faster |
| Memory usage | <256MB | <256MB + DB disk | ✅ Lower |

## Deployment Simplification

### Before (Complex)

```yaml
# docker-compose.yml
services:
  clickhouse:
    image: clickhouse/clickhouse-server:latest
    ports:
      - "9000:9000"
      - "8123:8123"
    volumes:
      - ./data/clickhouse:/var/lib/clickhouse
      - ./config/clickhouse-config.xml:/etc/clickhouse-server/config.xml
    healthcheck:
      test: ["CMD", "clickhouse-client", "--query", "SELECT 1"]
      
  occ:
    image: occ:latest
    depends_on:
      clickhouse:
        condition: service_healthy
    environment:
      STORAGE_BACKEND: clickhouse
      CLICKHOUSE_ADDR: clickhouse:9000
    ports:
      - "4317:4317"
      - "4318:4318"
      - "8080:8080"
```

**Steps to deploy:**
1. Start ClickHouse (wait for healthy)
2. Run schema migrations
3. Configure connection
4. Start OCC

### After (Simple)

```yaml
# docker-compose.yml
services:
  occ:
    image: occ:latest
    ports:
      - "4317:4317"
      - "4318:4318"
      - "8080:8080"
```

**Steps to deploy:**
1. Start OCC

### Kubernetes

**Before:** 2 pods (OCC + ClickHouse StatefulSet), PersistentVolume, Service mesh  
**After:** 1 pod (OCC Deployment), no volumes

## Migration Strategy

### For Unreleased Project

**ADVANTAGE:** No production users, breaking changes acceptable!

Steps:
1. Delete code (one PR)
2. Update docs
3. Tag as v0.1.0 or v1.0.0-beta (pre-release)
4. Clearly document ephemeral design in README

**No migration** needed - fresh start.

### For Future Persistence (If Needed)

**Option 1: Export/Import API** (Simple, deferred)
```bash
# Export current state
curl http://localhost:8080/api/v1/export > metadata.json

# Import on new instance
curl -X POST -d @metadata.json http://localhost:8080/api/v1/import
```

**Option 2: External Persistence** (User-managed)
- User can periodically call `/api/v1/metrics`, save to file
- Restart → re-analyze from source (preferred)

**Option 3: Add back ClickHouse** (If strong user demand)
- Would require justification
- Only if users need historical queries (not current use case)

## Risk Analysis

### Risk: Data Loss on Restart
**Likelihood:** High (expected behavior)  
**Impact:** Low (diagnostic tool, re-analyze source)  
**Mitigation:** Document clearly, consider export API if needed

### Risk: Memory Limits in Large Deployments
**Likelihood:** Medium  
**Impact:** Medium (OOM if >500k unique keys)  
**Mitigation:** 
- Document limits in README
- Monitor memory usage via `/api/v1/health`
- Add eviction policy if needed (LRU, TTL)

### Risk: User Expectations (Want Persistence)
**Likelihood:** Low (no users yet)  
**Impact:** Low (can add later)  
**Mitigation:** Validate with early users, add export/import if requested

### Risk: Code Complexity Rebound
**Likelihood:** Low  
**Impact:** Medium (re-add persistence later)  
**Mitigation:** Stick to design principles, require clear justification

## Alternatives Considered

### Alternative 1: Keep ClickHouse as Optional

**Pros:**
- No breaking change
- Users can choose persistence

**Cons:**
- Maintain two code paths (testing burden)
- Increases complexity significantly
- No clear user demand

**Decision:** Rejected - No users exist yet, can add back if needed.

### Alternative 2: Add SQLite Instead

**Pros:**
- Embedded database (no external dependency)
- File-based persistence

**Cons:**
- Still adds ~1,000 LOC
- Write performance issues (WAL, locking)
- Doesn't solve core problem (persistence not needed)

**Decision:** Rejected - Adds complexity without clear value.

### Alternative 3: Remote State Store (Redis/etcd)

**Pros:**
- Distributed state (multi-instance)
- Fast in-memory persistence

**Cons:**
- External dependency
- Network I/O overhead
- Over-engineered for single diagnostic instance

**Decision:** Rejected - Not needed for current use case.

## Implementation Plan

### Phase 1: Code Removal (30 min)
1. Delete `internal/storage/clickhouse/`
2. Delete `internal/storage/dual/`
3. Delete config/script files
4. Run `go mod tidy`

### Phase 2: Simplification (15 min)
1. Refactor `factory.go` → direct memory instantiation
2. Update `cmd/server/main.go` → remove config
3. Remove env vars from deployment manifests

### Phase 3: Documentation (30 min)
1. Update README (remove ClickHouse setup)
2. Update `openspec/project.md` (Tech Stack)
3. Create `docs/PERSISTENCE.md` (explain design)
4. Update k8s README (simpler deployment)

### Phase 4: Validation (30 min)
1. Search codebase for "clickhouse", "ClickHouse", "dual"
2. Run full test suite
3. Build application
4. Test locally
5. Run load tests

**Total:** ~2 hours

## Success Criteria

- [ ] Codebase search for "clickhouse" returns 0 results (except this doc)
- [ ] Application builds without errors
- [ ] All tests pass
- [ ] Memory storage handles 500k metrics in <256MB
- [ ] Load test passes (30k signals/sec)
- [ ] Documentation clear and accurate
- [ ] Deployment simpler (docker-compose, k8s)

## Appendix: Code Snippets

### Before: Factory Pattern

```go
// internal/storage/factory.go
type Config struct {
    Backend         string
    ClickHouseAddr  string
    UseAutoTemplate bool
    AutoTemplateCfg autotemplate.Config
}

func NewStorage(cfg Config) (Storage, error) {
    switch cfg.Backend {
    case "memory":
        return memory.NewWithAutoTemplate(cfg.UseAutoTemplate), nil
    case "clickhouse":
        chCfg := clickhouse.DefaultConfig()
        chCfg.Addr = cfg.ClickHouseAddr
        logger := slog.Default()
        store, err := clickhouse.NewStore(context.Background(), chCfg, logger)
        if err != nil {
            return nil, fmt.Errorf("creating ClickHouse store: %w", err)
        }
        return store, nil
    default:
        return nil, fmt.Errorf("unknown storage backend: %s", cfg.Backend)
    }
}
```

### After: Direct Instantiation

```go
// internal/storage/memory.go (no factory needed)
func New() *Store {
    return NewWithAutoTemplate(false)
}

func NewWithAutoTemplate(useAutoTemplate bool) *Store {
    store := &Store{
        metrics:         make(map[string]*models.MetricMetadata),
        spans:           make(map[string]*models.SpanMetadata),
        logs:            make(map[string]*models.LogMetadata),
        attributes:      make(map[string]*models.AttributeMetadata),
        services:        make(map[string]*models.ServiceMetadata),
        useAutoTemplate: useAutoTemplate,
    }
    
    if useAutoTemplate {
        store.drainProcessor = autotemplate.NewProcessor(autotemplate.DefaultConfig())
    }
    
    return store
}
```

Usage:
```go
// cmd/server/main.go
store := memory.NewWithAutoTemplate(getEnvBool("USE_AUTOTEMPLATE", true))
defer store.Close()
```

## References

- Memory store: `internal/storage/memory/store.go` (500 LOC, complete)
- ClickHouse store: `internal/storage/clickhouse/store.go` (1,489 LOC, to be removed)
- Project principles: `openspec/project.md` - "lightweight metadata analyzer"
- Current design: `openspec/changes/clickhouse-hll-sampling/` (unimplemented, will be deleted)
