# Proposal: add-podlog-enrichment

## Problem Statement

Pod logs collected via the OpenTelemetry Collector `filelog/podlogs` receiver arrive with two systematic gaps that make them hard to analyse in this tool:

1. **Missing service name**: The Kubernetes resource attributes (`k8s.pod.name`, `k8s.container.name`, `k8s.namespace.name`, etc.) are present, but `service.name` is absent. The analyzer therefore falls back to `"unknown"` for every pod log record, making per-service grouping meaningless.

2. **Unspecified severity**: The Collector does not parse Kubernetes container log severity at the filelog level, so `SeverityText` is empty and `SeverityNumber` is `Unspecified(0)` for every record. All pod logs collapse into a single `"UNSET"` bucket.

These issues are structural to the pod-log pipeline and do not affect application logs that already carry `service.name` and a parsed severity. An opt-in feature flag avoids changing default behaviour for other log sources.

## Proposed Solution

Add a **POD_LOG_ENRICHMENT** opt-in mode (env var, defaulting to `false`) that, when enabled, activates two enrichment steps inside `LogsAnalyzer`:

### 1. Service Name Discovery

Before falling back to `"unknown"`, check a configurable ordered list of resource attribute keys. The default priority list mirrors the Loki convention:

```
service_name, service, app, application, name, app_kubernetes_io_name,
k8s.container.name, k8s.deployment.name, k8s.pod.name, container, component, workload, job
```

The first non-empty match is used as the service name. If none match, fall back to `"unknown_service"` (matching the OpenTelemetry SDK convention) rather than `"unknown"`.

### 2. Severity Inference from Body

When `SeverityText` is empty and `SeverityNumber` is 0, scan the log body string against a set of keyword patterns (e.g. `error`, `warn`, `info`, `debug`) and assign a normalised severity text. Patterns are case-insensitive and matched in priority order (ERROR before WARN before INFO before DEBUG). If no pattern matches, the record is categorised as `"UNSET"`.

The configurable label list for service discovery SHALL be overridable via the `POD_LOG_SERVICE_LABELS` environment variable (comma-separated ordered list).

**Performance note:** On the hot path, severity inference adds ~110 ns per record using `strings.ToLower` + `strings.Contains`. This represents ~6% overhead on top of the Drain body template step (~1825 ns). Records that already have a `SeverityText` skip inference entirely. A hand-rolled zero-alloc scan provides no benefit — Go's stdlib string functions use SIMD acceleration and are faster in practice.

## Why

- Pod logs represent a large and growing share of telemetry in Kubernetes-native deployments. Without this enrichment they produce a single opaque `unknown / UNSET` bucket that provides no diagnostic value.
- The feature is opt-in to preserve backward compatibility for existing deployments that use structured application logs with pre-populated `service.name` and severity fields.
- The Loki project solves the exact same problem with an identical attribute priority approach, confirming this is idiomatic for the ecosystem.
- Severity inference covers the large fraction of application logs that omit severity but embed level keywords in the body (node.js, python stdlib, many custom loggers).

## What Changes

- `internal/storage/factory.go` — add `PodLogEnrichment bool` and `PodLogServiceLabels []string` to `Config`; expose defaults.
- `cmd/server/main.go` — read `POD_LOG_ENRICHMENT` and `POD_LOG_SERVICE_LABELS` env vars; pass into `Config`.
- `internal/analyzer/common.go` — extend `getServiceName` to accept an optional ordered label list instead of hard-coding `service.name` only.
- `internal/analyzer/logs.go` — **when enrichment is enabled**: invoke extended service name discovery and severity body inference; wire the config flag via `LogsAnalyzer`.
- New spec: `openspec/changes/add-podlog-enrichment/specs/log-enrichment/spec.md`

## Benefits

- Per-pod / per-container cardinality breakdown for logs that previously all appeared as `unknown`.
- Severity distribution becomes meaningful for pod logs, enabling operators to spot error spikes.
- Zero behaviour change for deployments that do not set `POD_LOG_ENRICHMENT=true`.

## Scope

**In scope:**
- Service name discovery from configurable resource attribute priority list.
- Severity inference from log body keyword matching (case-insensitive).
- New `Config` fields wired from environment variables.
- Unit tests for both enrichment steps.

**Out of scope:**
- Parsing structured JSON log bodies to extract severity (separate feature).
- Multiline log assembly (logtag=P partial records).
- Any changes to storage schema or API response shape.

## Success Criteria

- [ ] `POD_LOG_ENRICHMENT=true` with a pod log payload resolves service name from `k8s.container.name` (or whichever label is matched first).
- [ ] Logs with an empty `SeverityText` and a body containing `"error"` are stored under severity `"ERROR"`.
- [ ] Default behaviour (enrichment disabled) is unchanged — existing tests pass without modification.
- [ ] Unit tests cover: label priority ordering, no-match fallback, severity pattern matching, case-insensitivity.
