## ADDED Requirements

### Requirement: REST API server SHALL enforce connection timeouts

The REST API server SHALL configure `ReadTimeout`, `WriteTimeout`, and `IdleTimeout` on its `http.Server` to prevent indefinite goroutine and file-descriptor retention caused by stalled or misbehaving clients.

**ID:** `api-timeout-001`
**Priority:** MUST
**Rationale:** Zero-value timeouts in Go's net/http mean no deadline is applied. The REST API server is equally exposed to slow or stalled clients as the OTLP receiver; both servers must apply the same minimum protection.

#### Scenario: Server uses explicit ReadTimeout
**Given** the REST API server is running
**When** a client opens a connection and never finishes sending the request
**Then** the server SHALL close the connection after `ReadTimeout` (30 s) has elapsed
**And** no goroutine for that connection SHALL remain after the timeout fires

#### Scenario: Server uses explicit WriteTimeout
**Given** the REST API server is processing a request
**When** the handler produces a response but the client does not read it
**Then** the server SHALL abort the write after `WriteTimeout` (30 s)

#### Scenario: Server uses explicit IdleTimeout for keep-alive connections
**Given** the REST API server has served a request on a keep-alive connection
**When** no new request arrives within `IdleTimeout` (120 s)
**Then** the server SHALL close the idle connection
