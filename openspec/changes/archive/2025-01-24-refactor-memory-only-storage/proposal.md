# Proposal: Refactor to Memory-Only Storage

## Summary

Remove all database persistence (ClickHouse, dual-mode) and simplify the application to use **only in-memory storage**. This refactoring eliminates ~2,400 lines of ClickHouse-specific code, removes external database dependencies, and aligns with the project's core principle as a lightweight metadata analyzer rather than a full telemetry backend.

## Status
- **Created**: 2026-01-24
- **State**: Complete
- **Completed**: 2026-01-24
- **Breaking**: Yes (API responses may change, deployment configuration changes)

## Why

### Current Problem

The codebase currently supports three storage backends:
1. **In-memory** (`internal/storage/memory/`) - Simple, fast, ephemeral
2. **ClickHouse** (`internal/storage/clickhouse/`) - Complex, persistent, requires external database (~2,400 LOC)
3. **Dual** (`internal/storage/dual/`) - Writes to both, adds complexity

**Issues:**
- **Over-engineered**: Application hasn't been released yet, no production users requiring persistence
- **Complexity**: ClickHouse integration adds significant code (~60% of storage layer)
- **External dependency**: Requires ClickHouse server (deployment complexity, k8s sidecar, docker-compose)
- **Maintenance burden**: Schema migrations, connection handling, batch buffering, integration tests
- **Resource overhead**: Memory AND disk storage, network I/O to database
- **Unclear value**: Metadata analysis is ephemeral by nature - restart and re-analyze as needed

### Root Cause

Feature creep during development. ClickHouse was added as "nice to have" for persistence, but:
- No user requirement for historical data retention
- Application is a diagnostic tool, not a monitoring platform
- In-memory storage already handles 500k+ metrics efficiently
- Pre-release state allows breaking changes without user impact

## What Changes

### Code Removal

**Delete entirely:**
- `internal/storage/clickhouse/` (4 files, ~2,400 LOC)
  - `store.go` (~1,489 lines)
  - `buffer.go` (~400 lines)
  - `schema.go` (~241 lines)
  - `connection.go` (~150 lines)
  - `store_integration_test.go` (~120 lines)
- `internal/storage/dual/` (2 files, ~350 LOC)
  - `store.go`
  - `store_test.go`
- Configuration files:
  - `config/clickhouse-config.xml`
  - `scripts/start-clickhouse.sh`
  - `scripts/verify-clickhouse-data.sh`
  - `data/clickhouse/` directory
- `openspec/changes/clickhouse-hll-sampling/` (abandoned, never implemented)

**Dependencies to remove from `go.mod`:**
```
github.com/ClickHouse/clickhouse-go/v2 v2.40.3
github.com/ClickHouse/ch-go v0.68.0
```

### Code Simplification

**`internal/storage/factory.go`** - Simplify to single backend:
```go
// Before: 3 backends (memory, clickhouse, dual)
func NewStorage(cfg Config) (Storage, error) {
    switch cfg.Backend {
    case "memory": ...
    case "clickhouse": ...
    default: ...
    }
}

// After: Always memory
func NewStorage(cfg Config) Storage {
    return memory.NewWithAutoTemplate(cfg.UseAutoTemplate)
}
```

**`cmd/server/main.go`** - Remove storage configuration:
```go
// Remove these env vars:
// - STORAGE_BACKEND
// - CLICKHOUSE_ADDR

// Before:
storageBackend := getEnv("STORAGE_BACKEND", "clickhouse")
clickhouseAddr := getEnv("CLICKHOUSE_ADDR", "localhost:9000")

// After:
// No configuration needed, always in-memory
```

**`internal/storage/interface.go`** - Keep unchanged (memory already implements it)

**`internal/storage/memory/store.go`** - Keep as-is, already complete

### Documentation Updates

**Update:**
- `README.md` - Remove ClickHouse setup, simplify deployment
- `openspec/project.md` - Update Tech Stack, remove ClickHouse references
- `docs/CLICKHOUSE.md` - Delete or mark as deprecated/archived
- `k8s/deployment.yaml` - Remove ClickHouse-related env vars
- `Dockerfile` - Already compatible (no changes needed)

**New sections:**
- `docs/PERSISTENCE.md` - Explain ephemeral design, alternative approaches (export/import if needed)

### Configuration Changes

**Before:**
```bash
# Multiple backends
STORAGE_BACKEND=clickhouse  # or memory, or dual
CLICKHOUSE_ADDR=localhost:9000

# Multiple modes, complex decision tree
```

**After:**
```bash
# Single backend, no configuration
# Always in-memory, ephemeral
```

**Environment variables removed:**
- `STORAGE_BACKEND`
- `CLICKHOUSE_ADDR`

### API Changes

**No breaking API changes expected** - The REST API interface remains identical:
- `GET /api/v1/metrics` - Works same with memory backend
- `GET /api/v1/spans` - Works same with memory backend
- All endpoints already implemented by memory storage

**Behavioral change:**
- Data loss on restart (expected for diagnostic tool)
- No historical queries (never implemented anyway)

### Deployment Simplification

**Before (ClickHouse):**
```yaml
# docker-compose.yml (required)
services:
  clickhouse:
    image: clickhouse/clickhouse-server
    volumes: ...
    
  occ:
    depends_on:
      - clickhouse
    environment:
      STORAGE_BACKEND: clickhouse
      CLICKHOUSE_ADDR: clickhouse:9000
```

**After (Memory only):**
```yaml
# Single container, no dependencies
services:
  occ:
    image: occ:latest
    # No external dependencies!
```

**Kubernetes:** Same simplification - no StatefulSet for ClickHouse, no PersistentVolumes.

## Benefits

### Developer Experience
- **Less code to maintain**: -2,750 LOC (~60% of storage layer)
- **Simpler architecture**: Single storage implementation, no factory pattern needed
- **Easier testing**: No integration tests requiring database
- **Faster builds**: Fewer dependencies, faster `go mod download`

### Operations
- **Simpler deployment**: No database setup, no connection strings, no schema migrations
- **Lower resource usage**: No database server RAM/CPU/disk
- **Faster startup**: No database connection initialization
- **Easier troubleshooting**: Fewer moving parts

### User Experience
- **Faster setup**: `docker run` → works immediately
- **Lower cognitive load**: No storage backend decisions, no config tuning
- **Clearer purpose**: "Metadata analyzer" not "telemetry database"

## Trade-offs

### What We Lose
1. **Persistence across restarts** - Data lost on pod restart
   - **Mitigation**: Export/import API (future if needed), or re-analyze source data
2. **Historical analysis** - Can't query past observations
   - **Mitigation**: Not a use case for metadata analysis (analyze live data)
3. **Large-scale deployments** - Single instance memory limit
   - **Mitigation**: Sufficient for stated design (500k metrics in <256MB)

### What We Keep
- All current functionality (metrics, traces, logs, cardinality, patterns)
- Performance characteristics (in-memory already fastest)
- REST API compatibility (no breaking changes)
- HyperLogLog cardinality estimation
- Drain log templating

## Alternatives Considered

### 1. Keep ClickHouse as Optional
**Rejected**: Increases maintenance burden, need to support both code paths

### 2. Add SQLite Instead
**Rejected**: Still adds persistence complexity, not addressing core issue

### 3. Export/Import JSON
**Deferred**: Add later if users request it, simpler than full database

## Migration Path

### For Existing Deployments
**N/A** - Application not yet released, no production users

### For Development/Testing
1. Remove ClickHouse containers from local setup
2. Delete `data/clickhouse/` directory
3. Update `.env` files to remove `STORAGE_BACKEND`, `CLICKHOUSE_ADDR`
4. Rebuild and restart

### Rollback Plan
**If needed:** Git revert is straightforward (all code in version control)

## Success Criteria

- [x] All ClickHouse code removed (`internal/storage/clickhouse/`, `internal/storage/dual/`)
- [x] ClickHouse dependencies removed from `go.mod`
- [x] All tests pass (remove ClickHouse integration tests)
- [x] Application builds successfully
- [x] Memory storage handles full feature set (metrics, traces, logs, attributes, patterns)
- [x] Documentation updated (README, project.md, etc.)
- [x] Deployment manifests updated (k8s, Dockerfile)
- [x] No references to ClickHouse remain in codebase (except archive)

## Timeline

**Estimated effort:** 1-2 hours
- Code removal: 30 min (delete files, update imports)
- Factory simplification: 15 min
- Config cleanup: 15 min
- Documentation: 30 min
- Testing/validation: 30 min

**No phased rollout needed** - Single PR, atomic change

## Open Questions

1. **Export/Import**: Should we add JSON export/import for metadata backup? (Can defer to future)
2. **Metrics**: Any observability metrics we want to expose about memory usage? (Already have `/api/v1/health`)
3. **Limits**: Should we add configurable memory limits or eviction policies? (Can add later if needed)

## References

- Current code: `internal/storage/clickhouse/` (~2,400 LOC)
- Memory storage: `internal/storage/memory/store.go` (~500 LOC, complete)
- Factory: `internal/storage/factory.go` (needs simplification)
- Project principles: `openspec/project.md` - "lightweight metadata analyzer"
