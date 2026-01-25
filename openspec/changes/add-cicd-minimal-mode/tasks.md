# Tasks: Add CI/CD Minimal Mode

## Phase 1: Configuration & Flags (Foundation)

### Task 1.1: Add CLI flags
- [ ] Add `--minimal` / `--cicd` flag to main command
- [ ] Add `--duration` flag with duration parsing
- [ ] Add `--report-output` flag (file path)
- [ ] Add `--report-format` flag (json/yaml/text enum)
- [ ] Add `--report-verbosity` flag (basic/verbose enum, default: basic)
- [ ] Add `--report-max-items` flag (integer, default: 20, 0=unlimited)
- [ ] Add `--session-export` flag (optional session file path)
- [ ] Add `--api-only` flag (no auto-shutdown)
- [ ] Add `--max-memory` flag (MB limit)
- [ ] Add `--exit-on-threshold` flag (boolean)
- [ ] **Validation**: All flags parse correctly, help text accurate
- [ ] **Tests**: Unit tests for flag parsing and validation

### Task 1.2: Environment variable support
- [ ] Map `OCC_MINIMAL` to `--minimal`
- [ ] Map `OCC_DURATION` to `--duration`
- [ ] Map `OCC_REPORT_OUTPUT` to `--report-output`
- [ ] Map `OCC_REPORT_FORMAT` to `--report-format`
- [ ] Map `OCC_MAX_MEMORY` to `--max-memory`
- [ ] Map `OCC_EXIT_ON_THRESHOLD` to `--exit-on-threshold`
- [ ] Implement precedence: CLI > env > config > defaults
- [ ] **Validation**: Env vars override defaults but not CLI flags
- [ ] **Tests**: Integration tests for precedence

### Task 1.3: Config struct updates
- [ ] Add `MinimalMode bool` field
- [ ] Add `Duration time.Duration` field
- [ ] Add `ReportOutput string` field
- [ ] Add `ReportFormat string` field
- [ ] Add `ReportVerbosity string` field (basic/verbose)
- [ ] Add `ReportMaxItems int` field (default: 20)
- [ ] Add `APIOnly bool` field
- [ ] Add `MaxMemoryMB int` field
- [ ] Add `ExitOnThreshold bool` field
- [ ] **Validation**: Config validates on load
- [ ] **Tests**: Config serialization/deserialization tests

## Phase 2: Minimal Mode Runtime (Core Logic)

### Task 2.1: Server startup logic
- [ ] Detect minimal mode from config
- [ ] Initialize memory-only storage backend
- [ ] Start OTLP gRPC receiver
- [ ] Start OTLP HTTP receiver
- [ ] Start API server
- [ ] **Skip** UI server initialization if minimal mode
- [ ] Log "Running in minimal mode" with config details
- [ ] **Validation**: UI server not started, ports not bound
- [ ] **Tests**: Integration test verifying component states

### Task 2.2: Duration timer
- [ ] Create timer from `--duration` if set
- [ ] Register timer callback for shutdown
- [ ] Handle timer cancellation on signal
- [ ] Log timer events (started, remaining time, triggered)
- [ ] **Validation**: Timer triggers shutdown at correct time
- [ ] **Tests**: Unit tests with mock timers

### Task 2.3: Signal handling
- [ ] Register SIGINT handler
- [ ] Register SIGTERM handler
- [ ] Cancel duration timer on signal
- [ ] Initiate graceful shutdown on signal
- [ ] Propagate context cancellation to all components
- [ ] **Validation**: Clean shutdown on signals
- [ ] **Tests**: Integration tests sending signals

### Task 2.4: Graceful shutdown sequence
- [ ] Stop accepting new OTLP data
- [ ] Drain in-flight OTLP requests (5s timeout)
- [ ] Trigger report generation
- [ ] Shutdown API server (30s timeout)
- [ ] Close storage backend
- [ ] Log shutdown completion
- [ ] Return exit code
- [ ] **Validation**: No data loss, all resources closed
- [ ] **Tests**: Integration tests with active connections

## Phase 3: Memory-Only Storage (Bounded Storage)

### Task 3.1: Memory storage implementation
- [ ] Create `MinimalStorage` struct
- [ ] Implement `Store()` method
- [ ] Implement `Query()` method
- [ ] Implement `Close()` method
- [ ] Thread-safe with RWMutex
- [ ] **Validation**: Concurrent access safe
- [ ] **Tests**: Unit tests for storage operations, race detector

### Task 3.2: Memory tracking
- [ ] Track current memory usage (runtime.MemStats)
- [ ] Implement memory usage calculation
- [ ] Log memory usage periodically
- [ ] Warn at 80% threshold
- [ ] **Validation**: Accurate memory reporting
- [ ] **Tests**: Unit tests with known data sizes

### Task 3.3: Eviction policy
- [ ] Implement LRU eviction
- [ ] Track `LastSeen` timestamp per metric
- [ ] Evict oldest metrics when max memory reached
- [ ] Log eviction events
- [ ] Maintain cardinality accuracy after eviction
- [ ] **Validation**: Memory stays under limit
- [ ] **Tests**: Integration tests with memory pressure

## Phase 4: Report Generation (Output)

### Task 4.1: Report data model
- [ ] Define `Report` struct (telemetry overview, not session format)
- [ ] Add `version`, `generated_at`, `duration` fields
- [ ] Define `ReportSummary` struct (metrics, spans, logs totals, high_cardinality_count)
- [ ] Define `MetricReport` struct (name, type, label_keys, cardinality, severity)
- [ ] Define `SpanReport` struct (name, attribute_keys, span_count, cardinality, severity)
- [ ] Define `LogReport` struct (body_pattern, attribute_keys, log_count, cardinality, severity)
- [ ] Define `AttributeReport` struct (key, signal_types, used_by, unique_values, severity)
- [ ] Define `Recommendations` array (human-readable tips)
- [ ] **Validation**: Report schema covers all telemetry signals
- [ ] **Tests**: Unit tests for struct marshaling

### Task 4.2: Report generator
- [ ] Create `ReportGenerator` service
- [ ] Implement `Generate()` method
- [ ] Query storage for all metrics
- [ ] Query storage for all span names
- [ ] Query storage for all log patterns
- [ ] Query storage for all attributes (cross-signal)
- [ ] Sort items by cardinality (highest first)
- [ ] Apply max items limit based on config
- [ ] Implement basic mode (top N items)
- [ ] Implement verbose mode (all items)
- [ ] Calculate cardinality statistics per signal type
- [ ] Determine severity levels for each signal
- [ ] Generate cross-signal attribute analysis
- [ ] Calculate exit code
- [ ] **Validation**: Report accurate vs storage for all signals
- [ ] **Validation**: Basic mode shows top N, verbose shows all
- [ ] **Tests**: Unit tests with mock storage for both modes

### Task 4.3: Report formatters
- [ ] Implement JSON formatter
- [ ] Implement YAML formatter
- [ ] Implement text/table formatter
- [ ] Format according to schema
- [ ] **Validation**: Output parseable by standard tools (jq, yq)
- [ ] **Tests**: Unit tests for each format

### Task 4.4: Report output
- [ ] Write report to file if `--report-output` set
- [ ] Write report to stdout if no file specified
- [ ] Handle file write errors gracefully
- [ ] Set file permissions (0644)
- [ ] Log report location
- [ ] **Validation**: File created with correct content
- [ ] **Tests**: Integration tests with file I/O

### Task 4.5: Session export (optional)
- [ ] If `--session-export` specified, export full session data
- [ ] Use standard session format (compatible with `occ session load`)
- [ ] Include all metrics, spans, logs with attributes and cardinality data
- [ ] Handle file write errors gracefully
- [ ] Log session export location
- [ ] **Validation**: Session file loadable in normal mode UI with all signals
- [ ] **Tests**: Export session, load in normal mode, verify data accuracy for all signals

### Task 4.6: Exit code logic
- [ ] Exit code 0 if no thresholds exceeded
- [ ] Exit code 1 if warning thresholds exceeded (if `--exit-on-threshold`)
- [ ] Exit code 2 if critical thresholds exceeded (if `--exit-on-threshold`)
- [ ] Exit code 0 if `--exit-on-threshold` not set
- [ ] **Validation**: Correct exit codes for various scenarios
- [ ] **Tests**: Integration tests checking $?

## Phase 5: API Enhancements (Query Interface)

### Task 5.1: Existing API validation
- [ ] Test all API endpoints in minimal mode
- [ ] Ensure no UI dependencies in API handlers
- [ ] Verify CORS not required in minimal mode
- [ ] **Validation**: All endpoints return 200 OK
- [ ] **Tests**: API integration tests in minimal mode

### Task 5.2: New report endpoint
- [ ] Add `GET /api/v1/report` endpoint
- [ ] Accept `?format=` query parameter
- [ ] Return generated report
- [ ] Handle concurrent report generation
- [ ] Return 503 if shutting down
- [ ] **Validation**: Endpoint returns valid report
- [ ] **Tests**: API tests for new endpoint

### Task 5.3: Health endpoint updates
- [ ] Add `mode` field to health response
- [ ] Add `uptime` field
- [ ] Add `shutdown_in` field if duration set
- [ ] Add `memory_usage` field
- [ ] **Validation**: Health reflects minimal mode state
- [ ] **Tests**: Health endpoint tests

## Phase 6: Documentation (Enablement)

### Task 6.1: README updates
- [ ] Add "Minimal Mode" section
- [ ] Add quick start example
- [ ] Add use cases
- [ ] Document all flags
- [ ] Add CI/CD examples
- [ ] **Validation**: Examples run successfully

### Task 6.2: CI/CD guides
- [ ] Create GitHub Actions example workflow
- [ ] Create GitLab CI example (.gitlab-ci.yml)
- [ ] Create Jenkins Jenkinsfile example
- [ ] Create CircleCI config example
- [ ] Create Docker Compose example
- [ ] **Validation**: All examples tested and working

### Task 6.3: API documentation
- [ ] Document new `/api/v1/report` endpoint (OpenAPI/Swagger)
- [ ] Update API examples
- [ ] Document report schema
- [ ] **Validation**: API docs match implementation

### Task 6.4: Troubleshooting guide
- [ ] Document common CI issues
- [ ] Memory limit tuning guide
- [ ] Timeout configuration guide
- [ ] Exit code reference
- [ ] **Validation**: Guide covers real issues from testing

## Phase 7: Testing & Validation (Quality)

### Task 7.1: Unit tests
- [ ] Flag parsing tests (100% coverage)
- [ ] Report generation tests (100% coverage)
- [ ] Memory eviction tests (100% coverage)
- [ ] Exit code calculation tests (100% coverage)
- [ ] **Target**: >90% overall coverage for new code

### Task 7.2: Integration tests
- [ ] End-to-end minimal mode test
- [ ] Duration timeout test
- [ ] Signal handling test (SIGTERM, SIGINT)
- [ ] API availability test
- [ ] Memory limit enforcement test
- [ ] Report generation accuracy test
- [ ] Session compatibility test: Generate report, load in normal mode
- [ ] **Target**: All happy paths + error cases covered

### Task 7.3: E2E CI tests
- [ ] GitHub Actions workflow testing OCC itself
- [ ] Docker-based test sending OTLP data
- [ ] Verify report generated correctly
- [ ] Verify exit codes
- [ ] **Target**: Validates real CI/CD usage

### Task 7.4: Performance validation
- [ ] Measure startup time (<2s)
- [ ] Measure baseline memory (<50MB)
- [ ] Measure memory under load (<100MB for 1000 metrics)
- [ ] Measure shutdown time (<5s)
- [ ] Measure report generation time (<1s for 10k metrics)
- [ ] **Target**: All performance targets met

## Phase 8: Spec Deltas (Formal Specification)

See individual spec delta files in `specs/` directory:
- `specs/runtime-modes/spec.md` - Runtime mode capability
- `specs/report-generation/spec.md` - Report generation capability
- `specs/memory-storage/spec.md` - Bounded memory storage spec

## Dependencies

- **Blocks**: None (independent feature)
- **Blocked by**: None
- **Related**: 
  - Archive change `2025-01-24-refactor-memory-only-storage` (memory backend)
  - Change `add-snapshot-sessions` (may define test boundaries)

## Parallel Work
Phases 1-3 can be developed in parallel with Phase 4.
Phase 5 can start once Phase 2 is complete.
Phase 6 can start once Phases 1-5 are feature-complete.

## Estimated Effort
- Phase 1: 2-3 days
- Phase 2: 3-4 days
- Phase 3: 3-4 days
- Phase 4: 4-5 days
- Phase 5: 1-2 days
- Phase 6: 2-3 days
- Phase 7: 3-4 days
- Phase 8: 1 day (writing specs)

**Total**: ~20-30 days (one engineer)

## Definition of Done
- [ ] All tasks completed and validated
- [ ] All tests passing (unit + integration + E2E)
- [ ] Performance targets met
- [ ] Documentation complete and reviewed
- [ ] Spec deltas validated with `openspec validate`
- [ ] CI/CD examples tested in real pipelines
- [ ] Code reviewed and merged
- [ ] Release notes written
