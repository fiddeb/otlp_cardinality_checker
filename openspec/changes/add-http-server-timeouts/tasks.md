# Tasks: add-http-server-timeouts

## Ordered Work Items

- [x] **T-01** Add `ReadTimeout`, `WriteTimeout`, `IdleTimeout` to the `http.Server` literal in `internal/receiver/http.go`.
  - Values: `ReadTimeout: 30*time.Second`, `WriteTimeout: 30*time.Second`, `IdleTimeout: 120*time.Second`.
  - Verify: `grep -n "ReadTimeout" internal/receiver/http.go` shows all three fields.

- [x] **T-02** Add `http.MaxBytesReader(w, req.Body, 32<<20)` at the top of `handleMetrics`, `handleTraces`, and `handleLogs` in `internal/receiver/http.go`.
  - Apply before any body read (including gzip decompression check).
  - Verify: `grep -n "MaxBytesReader" internal/receiver/http.go` shows three call sites.

- [x] **T-03** Add `ReadTimeout`, `WriteTimeout`, `IdleTimeout` to the `http.Server` literal in `internal/api/server.go`.
  - Same values as T-01.
  - Verify: `grep -n "ReadTimeout" internal/api/server.go` shows all three fields.

- [x] **T-04** Write integration test for OTLP receiver timeout in `internal/receiver/http_timeout_test.go`.
  - Start the receiver with `httptest.NewServer` or a real listener.
  - Send a partial request body (write headers, then hang).
  - Assert connection is closed by the server within `ReadTimeout + 1s`.

- [x] **T-05** Write integration test for API server timeout in `internal/api/server_timeout_test.go`.
  - Same approach as T-04 for the REST API server.

- [x] **T-06** Run `go build ./...` — confirm no compilation errors.

- [x] **T-07** Run `go test ./...` — confirm all tests pass, no regressions.

## Dependencies

- T-02 depends on T-01 (server must exist before handlers are adjusted).
- T-04 depends on T-01 and T-02.
- T-05 depends on T-03.
- T-06 and T-07 run after T-01 through T-05.

## Parallelisable Work

- T-01 + T-02 (receiver changes) and T-03 (API server change) are independent of each other and can be implemented in parallel.
- T-04 and T-05 can be written in parallel once their respective server changes are complete.
