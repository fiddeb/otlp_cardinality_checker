# Proposal: add-otlp-body-size-limit

## Problem Statement

The OTLP HTTP receiver reads the request body unconditionally after optional gzip
decompression. Two distinct attack surfaces exist:

1. **Zip-bomb risk**: `Content-Encoding: gzip` is handled before any size limit is
   applied. A 1 MB compressed payload can expand to a multi-gigabyte decompressed
   body in memory. `http.MaxBytesReader` (added by the companion change
   `add-http-server-timeouts` / `receiver-002`) caps the raw read but does not
   defend against the expansion that occurs *inside* the gzip reader.

2. **No early Content-Length rejection**: A sender that advertises a large
   `Content-Length` is allowed to begin streaming before the server rejects it. An
   early header check returns 413 before the connection transmits any body bytes,
   saving bandwidth and goroutine time.

Together these gaps mean a misconfigured or adversarial OTel Collector can cause
unbounded memory growth on the OTLP receiver, undermining the tool's reliability
as an always-on diagnostic endpoint.

## Proposed Solution

Apply a two-layer defence in each of the three OTLP handlers (`handleMetrics`,
`handleTraces`, `handleLogs`):

| Layer | Mechanism | When |
|---|---|---|
| 1 — Header check | Reject if `Content-Length > 32 MiB` | Before any body read |
| 2 — Streaming cap | `http.MaxBytesReader(w, req.Body, 32<<20)` | Wraps raw body, before gzip reader |

The 32 MiB limit is sufficient for any realistic OTLP batch while ruling out both
large plain payloads and gzip bombs.

**Note:** Layer 2 (`http.MaxBytesReader`) is specified in `receiver-002` of the
`add-http-server-timeouts` change. This change scopes the additional Content-Length
pre-check (Layer 1) and adds the zip-bomb unit test that was missing from the
companion change.

## Why

- **Zip bomb**: the gzip reader expands the stream before `MaxBytesReader` can
  count the bytes. Wrapping the *raw* body first prevents this but a Content-Length
  pre-check is a cheaper, earlier safeguard.
- **Bandwidth and goroutine cost**: rejecting on `Content-Length` avoids accepting
  even the first byte of an oversized payload.
- **Defence in depth**: neither guard alone is sufficient for all attack vectors;
  both together close the gap completely.

## What Changes

- `internal/receiver/http.go` — add `Content-Length` pre-check helper
  `rejectIfBodyTooLarge` called at the top of each handler, before gzip branch.
- New unit tests in `internal/receiver/http_body_limit_test.go` covering:
  - Oversized `Content-Length` header → 413.
  - Body that exceeds limit mid-stream → 413.
  - Gzip bomb (compressed body that expands past limit) → 413.
  - Normal-sized body → 200.
- Spec delta extending `receiver` capability with `receiver-003`.

## Dependencies

- `add-http-server-timeouts` MUST be applied first (it introduces `receiver-002`
  and adds `MaxBytesReader`). This change adds above and around that foundation.

## Scope

**In scope:**
- `Content-Length` header pre-check in all three OTLP HTTP handlers.
- Unit tests for oversized plain body, oversized Content-Length header, and gzip
  bomb scenarios.

**Out of scope:**
- gRPC receiver (separate size controls via `grpc.MaxRecvMsgSize`).
- REST API server body limits (API payloads are small query parameters only).
- TLS or authentication changes.

## Success Criteria

- [ ] `rejectIfBodyTooLarge` (or inline equivalent) rejects requests where
      `Content-Length > 32<<20` with HTTP 413 before reading any body.
- [ ] `http.MaxBytesReader` wraps `req.Body` before the gzip reader is created
      (prerequisite from `receiver-002` confirmed present).
- [ ] Unit test: `Content-Length: 33554433` → HTTP 413.
- [ ] Unit test: streaming body > 32 MiB without Content-Length → HTTP 413.
- [ ] Unit test: gzip-encoded body that expands past 32 MiB → HTTP 413.
- [ ] Unit test: normal 1 MiB body → HTTP 200.
- [ ] `go build ./...` and `go test ./...` pass with no regressions.
