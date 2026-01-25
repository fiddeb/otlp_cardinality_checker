# Spec Delta: Report Generation

## ADDED Requirements

### Requirement: Automated Report Generation

**ID**: `report-gen.automated`  
**Status**: Draft  
**Priority**: High

#### Description
The system SHALL automatically generate telemetry overview reports in minimal mode, showing what metrics, traces (spans), and logs were discovered. This is separate from session export which contains full data for UI loading.

#### Requirements
1. System SHALL generate report automatically on shutdown if in minimal mode
2. Report MUST be human-readable summary of telemetry (not full session data)
3. Report MUST include: discovered metrics, span names, log patterns, attribute keys, cardinality estimates, severity
4. Report MUST analyze attributes across all signal types (cross-signal analysis)
5. Report generation MUST complete within 5 seconds for up to 10,000 metrics + 5,000 spans + 5,000 logs
6. Report MUST be generated even if no data was received (empty report)
7. Report generation MUST NOT block OTLP ingestion while running
8. Session export (full data) MUST be optional via `--session-export` flag

#### Scenario: Report generated on shutdown
**GIVEN** OCC running in minimal mode with `--duration 1m`  
**AND** OTLP data has been received  
**WHEN** duration expires  
**THEN** report SHALL be generated  
**AND** report SHALL include all metrics received  
**AND** report SHALL be written to output destination

#### Scenario: Empty report when no data
**GIVEN** OCC running in minimal mode  
**AND** no OTLP data received  
**WHEN** shutdown is triggered  
**THEN** report SHALL still be generated  
**AND** report summary SHALL show 0 metrics  
**AND** report SHALL be valid according to schema

---

### Requirement: Report Schema

**ID**: `report-gen.schema`  
**Status**: Draft  
**Priority**: High

#### Description
The system SHALL generate reports with a simple, human-focused schema that summarizes telemetry. This is distinct from session format which contains full cardinality data.

#### Requirements
1. Report MUST include `version` field (report schema version)
2. Report MUST include `generated_at`, `duration`, `occ_version` fields
3. Report MUST include `summary` object with totals for metrics, spans, logs, attributes, and high_cardinality_count
4. Report MUST include `metrics` array with per-metric overview
5. Report MUST include `spans` array with per-span-name overview
6. Report MUST include `logs` array with per-log-pattern overview
7. Report MUST include `attributes` array showing which signals use each attribute (cross-signal)
8. Report SHOULD include `recommendations` array with actionable tips
9. Each metric MUST include: name, type, label_keys, estimated_cardinality, severity
10. Each span MUST include: name, attribute_keys, span_count, estimated_cardinality, severity
11. Each log MUST include: body_pattern, attribute_keys, log_count, estimated_cardinality, severity
12. Each attribute MUST include: key, signal_types, used_by (metrics/spans/logs), estimated_unique_values, severity
13. Report format MUST be optimized for human readability, not machine processing

#### Scenario: Valid JSON schema
**GIVEN** report has been generated  
**WHEN** report JSON is validated against schema  
**THEN** validation SHALL pass  
**AND** all required fields SHALL be present

#### Scenario: Report includes metadata
**GIVEN** OCC started at 2026-01-25T10:00:00Z with version 0.2.0  
**AND** duration was 5 minutes  
**WHEN** report is generated at 2026-01-25T10:05:00Z  
**THEN** report.metadata.generated_at SHALL equal "2026-01-25T10:05:00Z"  
**AND** report.metadata.duration SHALL equal "5m"  
**AND** report.metadata.occ_version SHALL equal "0.2.0"  
**AND** report.metadata.mode SHALL equal "minimal"

#### Scenario: Report shows telemetry overview
**GIVEN** minimal mode collected 42 metrics, 15 span names, 8 log types with 23 attributes  
**AND** metric "user_events" has cardinality 12000  
**AND** span "database.query" has cardinality 15000  
**WHEN** report is generated  
**THEN** report.summary.total_metrics SHALL equal 42  
**AND** report.summary.total_span_names SHALL equal 15  
**AND** report.summary.total_log_types SHALL equal 8  
**AND** report.summary.total_attributes SHALL equal 23  
**AND** report.metrics array SHALL contain "user_events" entry  
**AND** report.spans array SHALL contain "database.query" entry  
**AND** "user_events" entry SHALL show severity "critical"  
**AND** "database.query" entry SHALL show severity "critical"  
**AND** report.recommendations array SHALL suggest reviewing both signals

---

### Requirement: Session Export

**ID**: `report-gen.session-export`  
**Status**: Draft  
**Priority**: Medium

#### Description
The system SHALL optionally export full session data when `--session-export` flag is provided, enabling detailed UI analysis of cardinality issues.

#### Requirements
1. Session export MUST be optional (only if `--session-export PATH` specified)
2. Session format MUST be compatible with `occ session load` command
3. Session MUST include all metrics, spans, logs, attributes, and cardinality data
4. Session MUST use standard OCC session schema
5. Session export MUST complete within 10 seconds for 10,000 metrics + 5,000 spans + 5,000 logs
6. Both report and session export MAY be generated in same run

#### Scenario: Session export loadable in UI
**GIVEN** minimal mode run with `--session-export output.json`  
**AND** OTLP data was received (metrics, spans, logs)  
**WHEN** shutdown completes  
**THEN** file "output.json" SHALL be created  
**AND** file SHALL use OCC session format  
**WHEN** normal mode OCC executes `occ session load output.json`  
**THEN** session SHALL load successfully  
**AND** UI SHALL display all metrics with full cardinality details  
**AND** UI SHALL display all spans with full cardinality details  
**AND** UI SHALL display all logs with full cardinality details

#### Scenario: Report without session export
**GIVEN** minimal mode run with only `--report-output`  
**AND** no `--session-export` specified  
**WHEN** shutdown completes  
**THEN** report SHALL be generated  
**AND** session export SHALL NOT be created

---

### Requirement: Report Output Formats

**ID**: `report-gen.formats`  
**Status**: Draft  
**Priority**: Medium

#### Description
The system SHALL support multiple report output formats to accommodate different CI/CD tooling and human readability needs.

#### Requirements
1. System MUST support JSON format (default)
2. System SHOULD support YAML format
3. System SHOULD support plain text/table format
4. Format MUST be selectable via `--report-format` flag
5. JSON and YAML formats MUST be parseable by standard tools (jq, yq)
6. Text format MUST be human-readable with aligned columns

#### Scenario: JSON format output
**GIVEN** `--report-format json` is set  
**WHEN** report is generated  
**THEN** output SHALL be valid JSON  
**AND** output SHALL be parseable by jq

#### Scenario: YAML format output
**GIVEN** `--report-format yaml` is set  
**WHEN** report is generated  
**THEN** output SHALL be valid YAML  
**AND** output SHALL be parseable by yq

#### Scenario: Text format human-readable
**GIVEN** `--report-format text` is set  
**WHEN** report is generated  
**THEN** output SHALL contain table headers  
**AND** output SHALL contain aligned columns  
**AND** output SHALL be readable without tools

---

### Requirement: Report Output Destination

**ID**: `report-gen.output`  
**Status**: Draft  
**Priority**: High

#### Description
The system SHALL support writing reports to stdout or a specified file path, with appropriate error handling for file I/O failures.

#### Requirements
1. Default output MUST be stdout if `--report-output` not specified
2. System SHALL write to file if `--report-output` path provided
3. File SHALL be created with 0644 permissions
4. System SHALL create parent directories if they don't exist
5. System MUST log error if file write fails
6. System MUST exit with code 1 if report generation fails
7. Existing file MUST be overwritten

#### Scenario: Write to stdout by default
**GIVEN** no `--report-output` flag set  
**WHEN** report is generated  
**THEN** report SHALL be written to stdout  
**AND** stdout SHALL contain valid report

#### Scenario: Write to file successfully
**GIVEN** `--report-output /tmp/report.json` is set  
**WHEN** report is generated  
**THEN** file /tmp/report.json SHALL be created  
**AND** file SHALL contain valid report  
**AND** file permissions SHALL be 0644

#### Scenario: Handle file write error
**GIVEN** `--report-output /readonly/report.json` is set  
**AND** /readonly directory is not writable  
**WHEN** report generation is attempted  
**THEN** error SHALL be logged  
**AND** OCC SHALL exit with code 1

---

### Requirement: Cardinality Severity Assessment

**ID**: `report-gen.severity`  
**Status**: Draft  
**Priority**: High

#### Description
The system SHALL classify metrics and attributes by cardinality severity levels (ok, warning, critical) based on configurable thresholds.

#### Requirements
1. Default warning threshold MUST be 1000 unique combinations
2. Default critical threshold MUST be 10000 unique combinations
3. Thresholds SHOULD be configurable via flags (future enhancement)
4. Each metric SHALL have severity: "ok", "warning", or "critical"
5. Each attribute SHALL have severity: "ok", "warning", or "critical"
6. Report summary MUST indicate overall status (highest severity)

#### Scenario: Metric classified as warning
**GIVEN** metric "http_requests_total" has cardinality 5000  
**AND** warning threshold is 1000  
**AND** critical threshold is 10000  
**WHEN** report is generated  
**THEN** metric severity SHALL be "warning"

#### Scenario: Metric classified as critical
**GIVEN** metric "user_events" has cardinality 15000  
**AND** critical threshold is 10000  
**WHEN** report is generated  
**THEN** metric severity SHALL be "critical"

#### Scenario: Overall status reflects highest severity
**GIVEN** 3 metrics with severity "ok"  
**AND** 1 metric with severity "critical"  
**WHEN** report is generated  
**THEN** report.thresholds.status SHALL equal "critical"

---

### Requirement: Exit Code Based on Thresholds

**ID**: `report-gen.exit-code`  
**Status**: Draft  
**Priority**: High

#### Description
The system SHALL set process exit code based on cardinality threshold violations when `--exit-on-threshold` flag is enabled, allowing CI/CD pipelines to fail builds on high cardinality.

#### Requirements
1. Exit code MUST be 0 if all metrics are severity "ok"
2. Exit code MUST be 1 if any metric is severity "warning" (if flag enabled)
3. Exit code MUST be 2 if any metric is severity "critical" (if flag enabled)
4. Exit code MUST be 0 if `--exit-on-threshold` not set, regardless of severity
5. Exit code MUST be set before process termination
6. Exit code SHALL be logged

#### Scenario: Exit 0 when all ok
**GIVEN** `--exit-on-threshold` is set  
**AND** all metrics have severity "ok"  
**WHEN** OCC exits  
**THEN** exit code SHALL be 0

#### Scenario: Exit 1 on warning
**GIVEN** `--exit-on-threshold` is set  
**AND** one metric has severity "warning"  
**WHEN** OCC exits  
**THEN** exit code SHALL be 1

#### Scenario: Exit 2 on critical
**GIVEN** `--exit-on-threshold` is set  
**AND** one metric has severity "critical"  
**WHEN** OCC exits  
**THEN** exit code SHALL be 2

#### Scenario: Always exit 0 without flag
**GIVEN** `--exit-on-threshold` is NOT set  
**AND** metrics have severity "critical"  
**WHEN** OCC exits  
**THEN** exit code SHALL be 0

---

### Requirement: On-Demand Report API

**ID**: `report-gen.api-endpoint`  
**Status**: Draft  
**Priority**: Medium

#### Description
The system SHALL expose an API endpoint to generate reports on-demand without triggering shutdown, enabling periodic querying during long-running tests.

#### Requirements
1. Endpoint MUST be `GET /api/v1/report`
2. Endpoint SHALL accept `?format=json|yaml|text` query parameter
3. Default format MUST be JSON if not specified
4. Endpoint SHALL return current state report
5. Endpoint SHALL NOT trigger shutdown
6. Endpoint SHALL return 503 Service Unavailable if shutting down
7. Concurrent requests SHOULD be safe (idempotent)

#### Scenario: On-demand JSON report
**GIVEN** OCC is running in minimal mode  
**AND** OTLP data has been received  
**WHEN** GET request sent to `/api/v1/report?format=json`  
**THEN** response status SHALL be 200 OK  
**AND** response body SHALL be valid JSON report  
**AND** OCC SHALL continue running

#### Scenario: Report during shutdown returns 503
**GIVEN** OCC is shutting down  
**WHEN** GET request sent to `/api/v1/report`  
**THEN** response status SHALL be 503 Service Unavailable  
**AND** response body SHOULD indicate shutdown in progress

#### Scenario: Format parameter respected
**GIVEN** OCC is running  
**WHEN** GET request sent to `/api/v1/report?format=yaml`  
**THEN** response Content-Type SHALL be "application/x-yaml"  
**AND** response body SHALL be valid YAML
