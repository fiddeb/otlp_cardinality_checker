## ADDED Requirements

### Requirement: OTLP HTTP Receiver SHALL enforce connection timeouts

The OTLP HTTP receiver SHALL configure `ReadTimeout`, `WriteTimeout`, and `IdleTimeout` on its `http.Server` to prevent indefinite goroutine and file-descriptor retention caused by stalled or misbehaving clients.

**ID:** `receiver-001`
**Priority:** MUST
**Rationale:** Zero-value timeouts in Go's net/http mean no deadline is applied. A single stalled TCP connection holds a goroutine and a file descriptor open permanently, exhausting OS and process limits under sustained anomalies.

#### Scenario: Server uses explicit ReadTimeout
**Given** the OTLP HTTP receiver is constructed
**When** a client opens a connection and never finishes sending the request headers
**Then** the server SHALL close the connection after `ReadTimeout` (30 s) has elapsed
**And** no goroutine for that connection SHALL remain after the timeout fires

#### Scenario: Server uses explicit WriteTimeout
**Given** the OTLP HTTP receiver is processing a request
**When** the handler has produced a response but the client is not reading it
**Then** the server SHALL abort the write after `WriteTimeout` (30 s)

#### Scenario: Server uses explicit IdleTimeout for keep-alive connections
**Given** the OTLP HTTP receiver has served a request on a keep-alive connection
**When** no new request arrives within `IdleTimeout` (120 s)
**Then** the server SHALL close the idle connection

### Requirement: OTLP HTTP Receiver SHALL limit request body size

The OTLP HTTP receiver SHALL apply `http.MaxBytesReader` to each handler's request body before any read or decompression occurs, capping ingestion at 32 MiB per request.

**ID:** `receiver-002`
**Priority:** MUST
**Rationale:** Without a body size cap an attacker or misconfigured collector can stream arbitrarily large payloads. With gzip enabled, a small compressed body can expand to gigabytes in memory before the read completes.

#### Scenario: Oversized body is rejected with 413
**Given** the OTLP HTTP receiver is running
**When** a client sends a POST /v1/metrics request whose body exceeds 32 MiB
**Then** the server SHALL respond with HTTP 413 Request Entity Too Large
**And** the request SHALL be terminated before the full body is read into memory

#### Scenario: Normal batch is accepted
**Given** the OTLP HTTP receiver is running
**When** a client sends a POST /v1/metrics request whose body is under 32 MiB
**Then** the server SHALL process the request normally
**And** return HTTP 200 OK
