# Tasks: add-otlp-body-size-limit

## Ordered Work Items

- [ ] **T-01** Confirm `http.MaxBytesReader` is present (from `add-http-server-timeouts`
  T-02). If the companion change has not been applied, apply it first.
  - Verify: `grep -n "MaxBytesReader" internal/receiver/http.go` returns three
    call sites (one per handler).

- [ ] **T-02** Add `Content-Length` pre-check at the top of `handleMetrics`,
  `handleTraces`, and `handleLogs` in `internal/receiver/http.go`.
  - Reject with HTTP 413 when `req.ContentLength > 32<<20`.
  - Must execute before the gzip branch and before `req.Body` is read.
  - Verify: `grep -n "ContentLength" internal/receiver/http.go` shows three
    call sites.

- [ ] **T-03** Write unit tests in `internal/receiver/http_body_limit_test.go`
  using `httptest.NewRecorder` and a real `HTTPReceiver` (or handler function
  directly):
  - `TestOversizedContentLengthRejected`: POST with `Content-Length: 33554433`
    and empty body → assert HTTP 413.
  - `TestOversizedStreamingBodyRejected`: POST without Content-Length, stream
    body > 32 MiB → assert HTTP 413.
  - `TestGzipBombRejected`: POST with `Content-Encoding: gzip`, compressed body
    that expands to > 32 MiB → assert HTTP 413.
  - `TestNormalBodyAccepted`: POST with a valid 1 MiB protobuf body → assert
    HTTP 200.

- [ ] **T-04** Run `go build ./...` — confirm no compilation errors.

- [ ] **T-05** Run `go test ./internal/receiver/...` — confirm all new tests pass
  and no existing tests regress.

## Dependencies

- T-01 is a prerequisite verification step; T-02 and T-03 depend on it being
  satisfied.
- T-03 depends on T-02 (tests exercise the Content-Length check).
- T-04 and T-05 run after T-02 and T-03.

## Parallelisable Work

- T-02 (implementation) and T-03 (tests) can be drafted in parallel; final test
  verification requires T-02 to be complete.
