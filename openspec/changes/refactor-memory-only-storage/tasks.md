# Tasks: Refactor to Memory-Only Storage

## 1. Code Removal

- [ ] 1.1 Delete `internal/storage/clickhouse/` directory (all 5 files)
- [ ] 1.2 Delete `internal/storage/dual/` directory (2 files)
- [ ] 1.3 Delete `config/clickhouse-config.xml`
- [ ] 1.4 Delete `data/clickhouse/` directory
- [ ] 1.5 Delete `scripts/start-clickhouse.sh`
- [ ] 1.6 Delete `scripts/verify-clickhouse-data.sh`
- [ ] 1.7 Delete `openspec/changes/clickhouse-hll-sampling/` (unimplemented)

## 2. Dependency Cleanup

- [ ] 2.1 Remove ClickHouse imports from `go.mod` (run `go mod tidy`)
- [ ] 2.2 Verify no ClickHouse imports remain in codebase (`rg "clickhouse-go"`)
- [ ] 2.3 Update `go.sum` (automatic with `go mod tidy`)

## 3. Code Refactoring

- [ ] 3.1 Simplify `internal/storage/factory.go`:
  - Remove `Config.Backend` field
  - Remove `Config.ClickHouseAddr` field
  - Simplify `NewStorage()` to always return memory backend
  - Remove `switch` statement for backend selection
- [ ] 3.2 Update `cmd/server/main.go`:
  - Remove `STORAGE_BACKEND` env var handling
  - Remove `CLICKHOUSE_ADDR` env var handling
  - Simplify storage initialization
- [ ] 3.3 Verify `internal/storage/interface.go` unchanged (memory already implements it)
- [ ] 3.4 Verify `internal/storage/memory/store.go` unchanged (already complete)

## 4. Test Updates

- [ ] 4.1 Run all tests: `go test ./...`
- [ ] 4.2 Remove/update tests that reference ClickHouse
- [ ] 4.3 Remove/update tests that reference dual storage
- [ ] 4.4 Verify memory storage tests pass
- [ ] 4.5 Run integration test suite (if exists)

## 5. Documentation Updates

- [ ] 5.1 Update `README.md`:
  - Remove ClickHouse setup instructions
  - Remove storage backend configuration section
  - Update Quick Start (simpler, no config needed)
  - Update deployment instructions
- [ ] 5.2 Update `openspec/project.md`:
  - Remove ClickHouse from Tech Stack
  - Update Storage section (memory-only)
  - Remove storage configuration examples
  - Update Development Setup
- [ ] 5.3 Delete or archive `docs/CLICKHOUSE.md`
- [ ] 5.4 Create `docs/PERSISTENCE.md`:
  - Explain ephemeral design rationale
  - Document memory limits and behavior
  - Describe future export/import options (if needed)
- [ ] 5.5 Update `docs/USAGE.md` (if ClickHouse referenced)
- [ ] 5.6 Update `docs/API.md` (if storage backend mentioned)

## 6. Deployment Configuration

- [ ] 6.1 Update `k8s/deployment.yaml`:
  - Remove `STORAGE_BACKEND` env var
  - Remove `CLICKHOUSE_ADDR` env var
  - Review resource limits (may reduce memory if no batch buffer)
- [ ] 6.2 Update `k8s/README.md` (simplify deployment steps)
- [ ] 6.3 Verify `Dockerfile` (already compatible, no changes needed)
- [ ] 6.4 Create example `docker-compose.yml` if it doesn't exist (simple, no database)
- [ ] 6.5 Update any Helm charts (if they exist)

## 7. Script Cleanup

- [ ] 7.1 Update or remove k6 scripts that reference ClickHouse:
  - `scripts/k6-clickhouse-write.js`
  - `scripts/k6-clickhouse-read.js`
  - `scripts/k6-clickhouse-mixed.js`
  - `scripts/k6-clickhouse-max-throughput.js`
  - `scripts/k6-clickhouse-optimized-test.js`
- [ ] 7.2 Update `scripts/README.md` (if ClickHouse tests documented)
- [ ] 7.3 Remove ClickHouse test results files:
  - `k6-clickhouse-read-results.json`
  - `k6-clickhouse-write-results.json`

## 8. Web UI Updates (Breaking Changes OK)

- [ ] 8.1 Review UI components for ClickHouse-specific features
- [ ] 8.2 Update UI documentation if storage backend mentioned
- [ ] 8.3 Test UI with memory-only backend
- [ ] 8.4 Verify all REST API endpoints work with memory storage

## 9. Validation & Testing

- [ ] 9.1 Full codebase search for "clickhouse" (case-insensitive): `rg -i clickhouse`
- [ ] 9.2 Full codebase search for "ClickHouse": `rg ClickHouse`
- [ ] 9.3 Full codebase search for "dual" storage: `rg "dual" internal/storage`
- [ ] 9.4 Build application: `make build`
- [ ] 9.5 Run all tests: `go test ./...`
- [ ] 9.6 Start server locally and verify endpoints
- [ ] 9.7 Run load test with memory backend: `k6 run scripts/load-test-metrics.js`
- [ ] 9.8 Verify memory usage stays reasonable (<256MB target)

## 10. Final Cleanup

- [ ] 10.1 Review `go.mod` - ensure only necessary dependencies remain
- [ ] 10.2 Review `.gitignore` - remove ClickHouse-specific entries
- [ ] 10.3 Update `CHANGELOG.md` (if exists) with breaking change notice
- [ ] 10.4 Update version or add migration note (pre-release, so just docs)

## 11. Git & Version Control

- [ ] 11.1 Create feature branch: `git checkout -b refactor/memory-only-storage`
- [ ] 11.2 Commit changes with semantic message: `refactor: remove database persistence, use memory-only storage`
- [ ] 11.3 Verify no unintended files deleted: `git status`
- [ ] 11.4 Push branch and create Pull Request
- [ ] 11.5 Add breaking change warning in PR description

## Completion Checklist

- [ ] All ClickHouse code removed (verified with search)
- [ ] All dual storage code removed
- [ ] Dependencies cleaned up (`go mod tidy` run)
- [ ] All tests pass
- [ ] Documentation updated and accurate
- [ ] Application builds and runs successfully
- [ ] Load tests pass with memory backend
- [ ] No references to removed backends in codebase
- [ ] Deployment manifests simplified
- [ ] PR created with clear breaking change notice

## Success Metrics

- **LOC removed**: ~2,750 lines
- **Files removed**: ~15 files
- **Dependencies removed**: 2 (ClickHouse client libraries)
- **Build time improvement**: ~10-20% (fewer deps)
- **Deployment complexity**: Reduced (no external database)
- **Startup time**: Faster (no DB connection)
- **Test suite speed**: Faster (no integration tests)

## Notes

- This is a **breaking change** but acceptable since application is pre-release
- Users must restart to re-analyze data (ephemeral design)
- Future: Can add export/import if persistence needed
- Memory limit: Design target is 500k metrics in <256MB
