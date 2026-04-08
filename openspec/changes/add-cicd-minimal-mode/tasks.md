# Tasks: Add CI/CD Minimal Mode

## Phase 1: CLI Flags & Server Configuration

### Task 1.1: Minimal mode flags
- [x] Add `--minimal` flag to disable UI and enable headless mode
- [x] Add `--duration` flag with Go duration parsing (e.g. `5m`, `1h`)
- [x] Add `--exit-on-threshold` flag for cardinality-based exit codes

### Task 1.2: Report flags
- [x] Add `--report-output` flag (file path for report)
- [x] Add `--report-format` flag (`json` or `text`)
- [x] Add `--session-export` flag (export session JSON on shutdown)

### Task 1.3: Server DisableUI option
- [x] Add `DisableUI` field to `ServerConfig`
- [x] Skip UI static file serving and SPA fallback when `DisableUI` is true
- [x] Log "Running in minimal mode" at startup

## Phase 2: Report Generation

### Task 2.1: Report data model (`internal/report/model.go`)
- [x] Define `Report` struct with version, generated_at, duration, summary
- [x] Define `ReportSummary` with totals per signal type and high cardinality count
- [x] Define `MetricReport`, `SpanReport`, `LogReport`, `AttributeReport` structs
- [x] Include severity classification (ok/warning/critical) based on cardinality

### Task 2.2: Report generator (`internal/report/generator.go`)
- [x] Create `Generator` that queries storage for all telemetry signals
- [x] Build metric reports from `GetAllMetrics()` with cardinality from `GetActiveSeries()`
- [x] Build span reports from `GetAllSpans()` with attribute cardinality
- [x] Build log reports from `GetAllLogs()` with attribute cardinality
- [x] Build attribute reports from `GetAllAttributes()` with cross-signal analysis
- [x] Sort all items by cardinality descending
- [x] Calculate exit code (0=ok, 1=warning, 2=critical) based on thresholds

### Task 2.3: Report formatters
- [x] JSON formatter (`internal/report/format_json.go`)
- [x] Text/table formatter (`internal/report/format_text.go`) with sections per signal

### Task 2.4: Report tests
- [x] Model marshaling tests (`report_test.go`)
- [x] Generator tests with mock storage (`generator_test.go`, `mock_storage_test.go`)
- [x] Severity classification tests
- [x] Exit code calculation tests

## Phase 3: Shutdown & Integration

### Task 3.1: Duration timer in main
- [x] Start `time.AfterFunc` timer when `--duration` is set
- [x] Timer triggers context cancellation for graceful shutdown

### Task 3.2: Graceful shutdown sequence
- [x] On shutdown: generate report if `--report-output` or `--report-format` set
- [x] Write report to file or stdout
- [x] Export session JSON if `--session-export` set (uses existing session serializer)
- [x] Return exit code from report generator when `--exit-on-threshold` is set

### Task 3.3: Signal handling
- [x] SIGINT/SIGTERM trigger context cancellation (existing behavior preserved)
- [x] Duration timer cancelled on signal

## Verification

- [x] `go build ./...` passes
- [x] `go test ./...` passes (all packages, 0 regressions)
- [x] Report package tests: 7/7 pass
- [x] API tests: no regressions

## Out of Scope (deferred)

Items from proposal not included in this first pass:
- Environment variable support (`OCC_MINIMAL`, etc.)
- `--api-only` flag (no auto-shutdown mode)
- `--max-memory` flag and memory eviction
- `--report-verbosity` and `--report-max-items` flags
- YAML report format
- `GET /api/v1/report` API endpoint
- Health endpoint mode/uptime fields
- CI/CD example workflows and documentation
- Spec delta files
