## AI Slop Detector Report

**Branch:** `slop` — **Risk classification:** Medium  
*(Internal tool, no injection surface, no auth changes, no new dependencies — but correctness and coverage gaps are real.)*

**Slop Risk Score: 34 / 100 — Moderate risk**

| Group | Weight | Raw | Contribution |
|-------|--------|-----|--------------|
| A: Supply chain & security | 45 | 28 | 12.6 |
| B: Correctness & robustness | 30 | 48 | 14.4 |
| C: Maintainability & integration | 15 | 40 | 6.0 |
| D: Stylistic & linguistic slop | 10 | 13 | 1.3 |
| **Total** | | | **34.3** |

---

### Findings

---

#### F-01 — No HTTP server timeouts (Group A — High)

**Location:** http.go, server.go

**Evidence:**
```go
r.server = &http.Server{
    Addr:    addr,
    Handler: mux,
}
```
Both the OTLP receiver and the REST API server are created with zero timeout configuration. `ReadTimeout`, `WriteTimeout`, and `IdleTimeout` are all unset, meaning Go uses no deadline at all.

**Why it matters:** A misbehaving OTel Collector, a slow client, or a crashed connection that doesn't send FIN will hold a goroutine and file descriptor open indefinitely. Under any sustained anomaly, this exhausts the goroutine pool. This is a classic "looks correct" Go slop pattern — the server starts and works fine until it doesn't.

**Fix:** Add timeouts to both server structs. Minimum: `ReadTimeout: 30s`, `WriteTimeout: 30s`, `IdleTimeout: 120s`. For the OTLP receiver, additionally wrap each handler body reader with `http.MaxBytesReader(w, req.Body, 32<<20)` (32 MB cap).

**Required test:** Integration test that sends a partial/slow request and confirms the server closes the connection within the configured timeout.

---

#### F-02 — No request body size limit on OTLP HTTP receiver (Group A — Medium)

**Location:** http.go

**Evidence:** `grep -rn "MaxBytesReader" --include="*.go"` returns empty. The body is read unconditionally after optional gzip decompression.

**Why it matters:** An attacker or a misconfigured collector can stream an arbitrarily large request body. With gzip, a small compressed payload can expand to gigabytes (zip bomb). Since the decompressor runs before any size check, the risk exceeds a plain large payload.

**Fix:** Apply `http.MaxBytesReader` *before* gzip decompression. Check `Content-Length` header against the limit and reject early. A 32 MB limit accommodates normal OTLP batch sizes.

**Required test:** Unit test that sends a body exceeding the limit and asserts a 413 response.

---

#### F-03 — Double `store.Close()` in main (Group B — High)

**Location:** main.go, main.go

**Evidence:**
```go
// Line 38-42
defer func() {
    if err := store.Close(); err != nil {
        log.Printf("Error closing storage: %v", err)
    }
}()
// ...
// Line 131
if err := store.Close(); err != nil {
    log.Printf("Error closing storage: %v", err)
}
```

On the normal signal-handling path, `store.Close()` is called explicitly at line 131, then again by the deferred function at line 39 when `main` returns. If `Close()` is not idempotent (the sessions store opens files), this causes a double-free or spurious error log on every clean shutdown.

**Fix:** Remove the explicit `store.Close()` at line 131. The defer is sufficient. If ordering relative to server shutdown matters, extend the defer chain or use a sync mechanism.

**Required test:** Integration test or unit test that calls `Close()` twice and asserts no error or panic on the second call (and documents idempotency as a contract).

---

#### F-04 — Zero test coverage on critical ingestion and storage paths (Group B — High)

**Evidence (from `go test ./... -coverprofile`):**
```
internal/receiver    coverage: 0.0%
internal/storage     coverage: 0.0%
internal/storage/memory  coverage: 0.0%
```

These three packages implement the entire telemetry ingestion pipeline and in-memory data store — the core of the tool. With 0% coverage, F-03 above went undetected, and correctness of merge logic, concurrency (per-signal `RWMutex`), and cardinality tracking is unverified.

**Fix:** Add table-driven tests for `Store.StoreMetric`/`StoreSpan`/`StoreLog` merge paths; tests for concurrent writes via `t.Parallel` + `go func()`; and at minimum smoke tests for the HTTP receiver handlers using `httptest.NewRecorder`.

**Required test:** All three packages must reach ≥ 50% statement coverage before the next merge to main.

---

#### F-05 — Dead `analyzeMetric()` wrapper with misleading "backward compatibility" comment (Group C — Medium)

**Location:** metrics.go

**Evidence:** The function is declared and documented but has zero call-sites outside of the comment references to its name:
```go
// analyzeMetric extracts metadata from a single metric (backward compatibility).
func (a *MetricsAnalyzer) analyzeMetric(...) *models.MetricMetadata {
    return a.analyzeMetricWithContext(context.Background(), ...)
}
```
`grep -rn "\.analyzeMetric("` returns no results in non-test code.

**Why it matters:** A dead wrapper with a "backward compatibility" label signals speculative generality — an AI pattern of preserving legacy seams that don't exist. It also means the `context.Background()` call-path is untested because nothing exercises it.

**Fix:** Delete `analyzeMetric`. If it is needed in a future public API, reintroduce it then. Update the comments that reference it.

---

#### F-06 — 5× copy-paste of `extract*Keys` body (Group C — High)

**Location:** metrics.go — `extractGaugeKeys`, `extractSumKeys`, `extractHistogramKeys`, `extractExponentialHistogramKeys`, `extractSummaryKeys`

**Evidence:** All five functions have an identical inner loop (attribute extraction, fingerprint, catalog feed, `AddValue`, percentage update). Only the concrete proto type of the first argument differs. This is ~200 lines of duplicated logic.

**Why it matters:** A bug fix or behavior change (e.g., a new cardinality tracking strategy) must be applied in five places. F-03 style bugs become 5× more expensive to find and fix.

**Fix:** Extract a generic helper:
```go
func (a *MetricsAnalyzer) extractDataPointKeys(
    ctx context.Context,
    attrs map[string]string,
    metadata *models.MetricMetadata,
    serviceName string,
) { ... }
```
Each `extract*Keys` function then iterates its data points, calls `extractAttributes`, and delegates to the shared helper. This reduces the duplicated lines by ~80%.

---

#### F-07 — Manual bubble sort instead of `sort.Float64s` (Group C/D — Low)

**Location:** metrics.go

**Evidence:**
```go
for i := 0; i < len(bounds); i++ {
    for j := i + 1; j < len(bounds); j++ {
        if bounds[j] < bounds[i] {
            bounds[i], bounds[j] = bounds[j], bounds[i]
        }
    }
}
```
`sort.Float64s` exists in the standard library. The custom bubble sort is O(n²) and a textbook AI slop pattern — it "works" but signals the code was generated without awareness of the standard library.

**Fix:** Replace with `sort.Float64s(bounds)`.

---

#### F-08 — `time.Sleep(100ms)` as startup readiness mechanism (Group B — Low)

**Location:** main.go

**Evidence:**
```go
time.Sleep(100 * time.Millisecond)
log.Println("All servers started successfully")
```
The log line is misleading: it prints before any server has confirmed it is listening. Under a loaded CI runner or slow machine the receiver may not be ready and the "All servers started" message is incorrect.

**Fix:** If readiness confirmation is needed, use a channel signal from each server goroutine after `net.Listen` succeeds. If the log is purely cosmetic, remove it or document that it is not a real health check.

---

### Summary table

| ID | Group | Severity | Title |
|----|-------|----------|-------|
| F-01 | A | High | No HTTP server timeouts |
| F-02 | A | Medium | No request body size limit (zip bomb risk) |
| F-03 | B | High | Double `store.Close()` on clean shutdown |
| F-04 | B | High | 0% coverage on receiver and memory store |
| F-05 | C | Medium | Dead `analyzeMetric` wrapper |
| F-06 | C | High | 5× copy-paste of extract*Keys logic |
| F-07 | C/D | Low | Bubble sort instead of `sort.Float64s` |
| F-08 | B | Low | `time.Sleep` as startup readiness |

### Human gate

Risk: **Medium** → Senior review required if Slop Risk Score ≥ 40. Current score is 34, so standard review is sufficient, but F-03 (double close) and F-04 (zero coverage on core paths) should be treated as blocking before shipping.