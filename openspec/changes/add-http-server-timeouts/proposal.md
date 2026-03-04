# Proposal: add-http-server-timeouts

## Problem Statement

Both the OTLP HTTP receiver (`internal/receiver/http.go`) and the REST API server (`internal/api/server.go`) construct `http.Server` with no timeout fields set:

```go
r.server = &http.Server{
    Addr:    addr,
    Handler: mux,
}
```

Go's net/http uses no deadlines when `ReadTimeout`, `WriteTimeout`, and `IdleTimeout` are zero. A misbehaving OTel Collector, a slow REST client, or a TCP connection that never sends FIN will hold a goroutine and a file descriptor open indefinitely. Under a sustained anomaly this exhausts the goroutine pool and crashes the process. The OTLP receiver also reads the request body without any size cap, so a compressed (gzip) zip-bomb payload can expand in memory before any limit is applied — this is addressed in a companion fix (F-02) but the timeout boundary is a prerequisite.

## Proposed Solution

Add explicit timeout configuration to both server structs:

| Field | Value |
|---|---|
| `ReadTimeout` | 30 s |
| `WriteTimeout` | 30 s |
| `IdleTimeout` | 120 s |

For the OTLP receiver, also wrap each handler's body reader with `http.MaxBytesReader(w, req.Body, 32<<20)` so that body reading is bounded independently of the `ReadTimeout` guard (the two defences are complementary).

Add an integration test for each server that sends a slow/partial request and confirms the connection is terminated within the configured `ReadTimeout`.

## Why

- **Goroutine leak**: the current zero-value configuration means one crashed-but-connected client occupies a goroutine permanently.
- **File descriptor exhaustion**: each stalled connection holds an OS file descriptor; under load this reaches the process or OS limit.
- **Correctness signal**: an internal diagnostic tool that cannot defend itself against its own inputs undermines trust in its telemetry data.
- **Defence in depth**: timeouts are the cheapest, most reliable layer of protection available. There is no good reason to omit them.

## What Changes

- `internal/receiver/http.go` — add `ReadTimeout`, `WriteTimeout`, `IdleTimeout` to `http.Server` literal; add `http.MaxBytesReader` call in each handler.
- `internal/api/server.go` — add `ReadTimeout`, `WriteTimeout`, `IdleTimeout` to `http.Server` literal.
- New capability spec: `openspec/specs/receiver/spec.md` introduced via delta `openspec/changes/add-http-server-timeouts/specs/receiver/spec.md`.
- Existing capability spec `api` extended via delta `openspec/changes/add-http-server-timeouts/specs/api/spec.md`.
- New integration tests in `internal/receiver/` and `internal/api/` covering timeout behaviour.

## Benefits

- Eliminates goroutine and file descriptor leaks caused by stalled HTTP connections.
- Prevents the tool from becoming a victim of the same cardinality/volume problems it is designed to detect in others.
- Brings both servers to the minimum acceptable production configuration for Go HTTP services.

## Scope

**In scope:**
- `ReadTimeout`, `WriteTimeout`, `IdleTimeout` on both HTTP servers.
- `http.MaxBytesReader` on OTLP receiver handlers.
- Integration tests confirming timeout enforcement.

**Out of scope:**
- gRPC server timeouts (separate concern, no deadline set there either — tracked separately).
- TLS configuration.
- F-02 (zip bomb / body size limit) — related but filed as a distinct finding; the `MaxBytesReader` guard added here is a prerequisite step, not the full F-02 fix.

## Success Criteria

- [ ] `ReadTimeout: 30*time.Second`, `WriteTimeout: 30*time.Second`, `IdleTimeout: 120*time.Second` present in both server constructors.
- [ ] `http.MaxBytesReader` applied in all three OTLP receiver handlers (metrics, traces, logs).
- [ ] Integration test for receiver: slow-body request is closed within `ReadTimeout + 1s` margin.
- [ ] Integration test for API server: slow-body request is closed within `ReadTimeout + 1s` margin.
- [ ] `go build ./...` and `go test ./...` pass with no regressions.
