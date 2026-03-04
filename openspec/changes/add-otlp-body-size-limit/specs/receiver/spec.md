## ADDED Requirements

### Requirement: OTLP HTTP Receiver SHALL reject oversized Content-Length before reading body

The OTLP HTTP receiver SHALL inspect the `Content-Length` request header and
immediately return HTTP 413 when the declared size exceeds 32 MiB, before any
body bytes are read or decompressed.

**ID:** `receiver-003`
**Priority:** MUST
**Related:** `receiver-002` (MaxBytesReader streaming cap)
**Rationale:** A `MaxBytesReader` cap (receiver-002) prevents unbounded reads
during streaming but does not avoid the cost of accepting the connection and
beginning body transfer. An early `Content-Length` check returns 413 before a
single body byte is read, saving bandwidth and reducing goroutine exposure for
clearly oversized requests.

#### Scenario: Oversized Content-Length is rejected before body is read

**Given** the OTLP HTTP receiver is running
**When** a client sends a POST /v1/metrics request with `Content-Length: 33554433`
  (one byte over 32 MiB)
**Then** the server SHALL respond with HTTP 413 Request Entity Too Large
**And** the response SHALL be sent before any body bytes are consumed

#### Scenario: Normal Content-Length is not rejected

**Given** the OTLP HTTP receiver is running
**When** a client sends a POST /v1/logs request with `Content-Length` below 32 MiB
**Then** the server SHALL not reject on Content-Length alone
**And** SHALL proceed to read and process the body normally

#### Scenario: Gzip bomb is rejected before memory expansion occurs

**Given** the OTLP HTTP receiver is running
**And** a client sends a POST /v1/traces request with `Content-Encoding: gzip`
**When** the compressed body would expand to more than 32 MiB when decompressed
**Then** the server SHALL respond with HTTP 413 Request Entity Too Large
**And** SHALL NOT load the fully expanded payload into memory
