# Tasks

- [x] Add `PodLogEnrichment bool` and `PodLogServiceLabels []string` to `storage.Config`; set defaults; read `POD_LOG_ENRICHMENT` and `POD_LOG_SERVICE_LABELS` env vars in `main.go`
- [x] Extend `getServiceName(attrs, labels []string)` in `internal/analyzer/common.go` to iterate the priority list before falling back to `"unknown_service"`
- [x] Add severity body inference function in `internal/analyzer/logs.go`; activate both enrichments when the flag is enabled in `LogsAnalyzer`
- [x] Unit tests for service name discovery: correct priority ordering, first-match wins, no-match fallback, empty labels list falls back to `service.name`-only behaviour
- [x] Unit tests for severity inference: ERROR/WARN/INFO/DEBUG patterns, case-insensitivity, no-match returns `"UNSET"`
- [x] Integration smoke test: send a synthetic pod log export (no `service.name`, empty severity) and assert enriched service name and severity appear in stored metadata
- [x] Update README or configuration docs to document `POD_LOG_ENRICHMENT` and `POD_LOG_SERVICE_LABELS` env vars
