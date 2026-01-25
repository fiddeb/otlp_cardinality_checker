# Proposal: Add CI/CD Minimal Mode

## Change ID
`add-cicd-minimal-mode`

## Status
Draft

## Summary
Introduce a minimal/headless operating mode for OCC optimized for CI/CD pipelines. This mode runs only the OTLP receiver and API server (no UI), uses minimal resources, and optionally generates reports after a specified duration. Reports show all telemetry signals (metrics, traces, logs) with their attributes and cardinality information. This enables pre-production telemetry analysis during automated testing.

## Problem Statement
Currently, OCC runs with all components active (OTLP receiver, API, UI, memory storage with session management). For CI/CD scenarios where:
- Applications need telemetry analysis before going live
- Load tests generate telemetry that should be analyzed
- Resource constraints exist (containers, CI runners)
- No human interaction is needed during the run
- Quick feedback on cardinality issues is critical
- Session persistence is not needed (ephemeral analysis)

The full OCC deployment is resource-heavy and includes unnecessary components.

## Proposed Solution
Implement a `--minimal` or `--cicd` mode that:
1. Starts only OTLP receiver (HTTP/gRPC) and API endpoints
2. Uses in-memory storage (no persistence requirement)
3. Disables UI server
4. Optionally auto-generates a **report** (telemetry overview) after `--duration` timeout
5. Optionally exports **session file** for loading in full UI later (deep analysis)
6. Exposes API for programmatic querying during/after test execution
7. Exits cleanly after report generation or on signal

**Report vs Session Export:**
- **Report**: Human-readable summary of discovered telemetry (metrics, traces/spans, logs, attributes, basic cardinality stats)
- **Session Export**: Full data snapshot in OCC session format for loading in UI (detailed cardinality analysis)

## Use Cases

### UC0: Normal Mode (Full Version)
```bash
# Start OCC in normal mode (default) - all features enabled
occ start

# Or explicitly:
occ start --normal

# This starts:
# - OTLP receivers (gRPC + HTTP)
# - API server
# - UI server (web interface)
# - Persistent storage (database/file)
# - No auto-shutdown
```

### UC1: CI Pipeline Cardinality Check
```bash
# Start OCC in minimal mode
occ start --minimal --duration 5m --report-format json --report-output /tmp/report.json &
OCC_PID=$!

# Run application with OCC as OTLP endpoint
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
./run-app-tests.sh

# Wait for report or query API
curl http://localhost:8080/api/v1/metrics/cardinality

# Check report and fail CI if thresholds exceeded
wait $OCC_PID
jq '.high_cardinality_metrics | length' /tmp/report.json
```

### UC2: Load Test Analysis
```bash
occ start --minimal --api-only
# API stays active, no auto-shutdown
k6 run load-test.js  # Sends OTLP to OCC
curl http://localhost:8080/api/v1/attributes/top-cardinality
```

### UC3: Post-CI Analysis in UI
```bash
# In CI pipeline: Generate report
occ start --minimal --duration 5m --report-output ci-run-$BUILD_ID.json
./run-tests.sh

# Later: Developer loads report in full UI for investigation
occ start  # Start with UI
occ session load ci-run-$BUILD_ID.json
# Browse cardinality issues in web interface at http://localhost:3000
```

## Benefits
- **Resource Efficient**: 50-70% lower memory/CPU usage (no UI rendering, minimal storage)
- **CI/CD Native**: Designed for automated pipelines
- **Fast Feedback**: Reports show telemetry overview automatically
- **Scriptable**: API-first design for automation
- **Zero Configuration**: Sensible defaults for CI environments
- **Dual Output**: Quick report for CI + optional session export for deep UI analysis

## Affected Components
- **Runtime/Server**: New startup mode logic
- **Configuration**: New flags and environment variables
- **Storage**: Memory-only mode requirement
- **API**: Ensure full functionality without UI
- **Reporting**: New auto-report generation capability

## Alternatives Considered

### Alternative 1: Docker Compose with lightweight config
**Rejected**: Still runs all components, just with smaller limits. Doesn't solve the auto-report or duration-based shutdown.

### Alternative 2: Separate "occ-lite" binary
**Rejected**: Maintenance burden of two binaries. Better to have mode flags.

### Alternative 3: Configuration file based
**Rejected**: CLI flags are more CI-friendly than managing config files.

## Dependencies
- Requires memory-only storage backend (exists per archive change)
- API must work independently of UI (likely already true)
- Report generation capability (new)
- Session save/load should be disabled in minimal mode (existing feature)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Memory overflow in long-running tests | High | Implement configurable max-entries limits; auto-eviction of old data |
| Missing features vs full mode | Medium | Document clearly; ensure API parity |
| Report format changes break CI | Medium | Version report schema; provide migration guide |
| Graceful shutdown complexity | Low | Use context cancellation; drain OTLP receiver before exit |

## Design Decisions
1. **Report purpose**: Human-readable telemetry overview (what metrics/attributes exist, basic stats)
2. **Session export**: Optional full data export in session format for UI loading (deep cardinality analysis)
3. **Report formats**: JSON (machine-readable), YAML, and text (human-readable) formats supported
4. **Exit codes**: Configurable via `--exit-on-threshold` flag based on cardinality thresholds
5. **Memory limits**: Default 512MB, configurable via `--max-memory`

## Open Questions
1. What are the exact cardinality thresholds (warning/critical)? Should they be configurable?
2. Should we support streaming reports (incremental updates) for long-running tests?

## Success Criteria
- [ ] OCC starts in <2s in minimal mode
- [ ] Memory usage <100MB for typical CI workload
- [ ] Report generated automatically after duration
- [ ] API fully functional for querying
- [ ] Exit code reflects cardinality health (0=ok, 1=warnings, 2=critical)
- [ ] Documentation with CI/CD examples (GitHub Actions, GitLab CI, etc.)

## Related Changes
- References archived change `2025-01-24-refactor-memory-only-storage`
- May relate to `add-snapshot-sessions` for defining test boundaries

## Reviewers
- @fiddeb (proposer)

## Timeline
- Proposal: 2026-01-25
- Design review: TBD
- Implementation: TBD
- Release target: Next minor version
