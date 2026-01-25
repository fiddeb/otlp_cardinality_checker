# sessions Specification

## Purpose
TBD - created by archiving change add-snapshot-sessions. Update Purpose after archive.
## Requirements
### Requirement: Session Persistence
The system SHALL provide file-based persistence for analysis sessions.

#### Scenario: Save current state as session
- **GIVEN** the system has collected telemetry data
- **WHEN** `POST /api/v1/sessions` is called with `{"name": "my-session"}`
- **THEN** current state is serialized to `data/sessions/my-session.json.gz`
- **AND** response includes session metadata (size, counts, created timestamp)

#### Scenario: Save with service filter
- **GIVEN** the system has data from multiple services
- **WHEN** `POST /api/v1/sessions` is called with `{"name": "payment-only", "services": ["payment-service"]}`
- **THEN** only data from payment-service is saved
- **AND** other service data is excluded from the session

#### Scenario: Save with signal filter
- **GIVEN** the system has metrics, traces, and logs
- **WHEN** `POST /api/v1/sessions` is called with `{"name": "metrics-only", "signals": ["metrics"]}`
- **THEN** only metrics are saved
- **AND** traces and logs are excluded from the session

#### Scenario: Duplicate session name
- **WHEN** `POST /api/v1/sessions` is called with name of existing session
- **THEN** HTTP 409 Conflict is returned
- **AND** error message suggests using a different name

### Requirement: Session Loading
The system SHALL support loading previously saved sessions.

#### Scenario: Load session replaces current state
- **GIVEN** the system has current data and a saved session exists
- **WHEN** `POST /api/v1/sessions/{name}/load` is called
- **THEN** current state is replaced with session data
- **AND** response includes loaded counts (metrics, spans, logs)

#### Scenario: Load with signal filter
- **GIVEN** a session contains metrics, traces, and logs
- **WHEN** `POST /api/v1/sessions/{name}/load` is called with `{"signals": ["metrics", "traces"]}`
- **THEN** only metrics and traces are loaded
- **AND** logs remain empty

#### Scenario: Load non-existent session
- **WHEN** `POST /api/v1/sessions/nonexistent/load` is called
- **THEN** HTTP 404 Not Found is returned

### Requirement: Session Merging
The system SHALL support merging sessions into current state.

#### Scenario: Merge adds to current state
- **GIVEN** current state has metrics A, B and session has metrics B, C
- **WHEN** `POST /api/v1/sessions/{name}/merge` is called
- **THEN** current state has metrics A, B, C
- **AND** metric B counts are summed
- **AND** metric B cardinality uses HLL union

#### Scenario: Merge timestamp handling
- **GIVEN** current state has metric X with FirstSeen=T1, LastSeen=T2
- **AND** session has metric X with FirstSeen=T0, LastSeen=T3
- **WHEN** session is merged
- **THEN** merged metric X has FirstSeen=T0 (earliest)
- **AND** merged metric X has LastSeen=T3 (latest)

#### Scenario: Merge HLL cardinality
- **GIVEN** current state has attribute with cardinality estimate 1000
- **AND** session has same attribute with cardinality estimate 2000
- **WHEN** session is merged
- **THEN** merged cardinality reflects union (not sum)
- **AND** cardinality is approximately 2500 (assuming 500 overlap)

### Requirement: Session Listing
The system SHALL provide listing of saved sessions.

#### Scenario: List all sessions
- **WHEN** `GET /api/v1/sessions` is called
- **THEN** array of session metadata is returned
- **AND** each entry includes: id, created, signals, size_bytes, description

#### Scenario: Empty session list
- **GIVEN** no sessions have been saved
- **WHEN** `GET /api/v1/sessions` is called
- **THEN** empty array is returned

### Requirement: Session Deletion
The system SHALL support deleting saved sessions.

#### Scenario: Delete existing session
- **GIVEN** a session named "old-session" exists
- **WHEN** `DELETE /api/v1/sessions/old-session` is called
- **THEN** HTTP 200 OK is returned
- **AND** session file is removed from disk

#### Scenario: Delete non-existent session
- **WHEN** `DELETE /api/v1/sessions/nonexistent` is called
- **THEN** HTTP 404 Not Found is returned

### Requirement: Session Export and Import
The system SHALL support exporting and importing sessions as portable JSON.

#### Scenario: Export session
- **GIVEN** a session named "my-session" exists
- **WHEN** `GET /api/v1/sessions/my-session/export` is called
- **THEN** complete session JSON is returned
- **AND** Content-Type is application/json
- **AND** Content-Disposition suggests filename

#### Scenario: Import session
- **WHEN** `POST /api/v1/sessions/import` is called with session JSON body
- **THEN** session is saved with name from JSON
- **AND** HTTP 201 Created is returned

### Requirement: Session Comparison (Diff)
The system SHALL support comparing two sessions to detect changes.

#### Scenario: Basic diff
- **GIVEN** sessions "before" and "after" exist
- **WHEN** `GET /api/v1/sessions/diff?from=before&to=after` is called
- **THEN** response includes summary (added, removed, changed counts)
- **AND** response includes detailed changes array

#### Scenario: Diff detects added metrics
- **GIVEN** "before" has metrics [A, B] and "after" has metrics [A, B, C]
- **WHEN** diff is computed
- **THEN** metric C appears in `changes.metrics.added`

#### Scenario: Diff detects removed metrics
- **GIVEN** "before" has metrics [A, B, C] and "after" has metrics [A, B]
- **WHEN** diff is computed
- **THEN** metric C appears in `changes.metrics.removed`

#### Scenario: Diff detects cardinality increase
- **GIVEN** "before" has attribute X with cardinality 100
- **AND** "after" has attribute X with cardinality 50000
- **WHEN** diff is computed
- **THEN** attribute X appears in `changes.metrics.changed`
- **AND** change severity is "critical" (>10x increase)

#### Scenario: Diff with service filter
- **GIVEN** sessions contain data from multiple services
- **WHEN** `GET /api/v1/sessions/diff?from=X&to=Y&service=payment-service` is called
- **THEN** only changes affecting payment-service are returned

#### Scenario: Diff with signal filter
- **WHEN** `GET /api/v1/sessions/diff?from=X&to=Y&signal_type=metrics` is called
- **THEN** only metric changes are returned
- **AND** traces and logs sections are omitted

### Requirement: HyperLogLog Serialization
The system SHALL serialize HyperLogLog state for session persistence.

#### Scenario: HLL round-trip
- **GIVEN** an attribute has HLL with cardinality estimate 12345
- **WHEN** session is saved and loaded
- **THEN** loaded attribute has same cardinality estimate (within HLL error margin)

#### Scenario: Nil HLL handling
- **GIVEN** some attributes have nil HLL (legacy or edge case)
- **WHEN** session is saved
- **THEN** nil HLL is serialized as null
- **AND** loaded with cardinality 0

### Requirement: Session Size Limits
The system SHALL enforce configurable size limits on sessions.

#### Scenario: Session exceeds size limit
- **GIVEN** OCC_MAX_SESSION_SIZE is set to 100MB
- **WHEN** attempting to save a session larger than 100MB
- **THEN** HTTP 413 Payload Too Large is returned
- **AND** error message indicates the limit

#### Scenario: Session count limit
- **GIVEN** OCC_MAX_SESSIONS is set to 50 and 50 sessions exist
- **WHEN** attempting to save a new session
- **THEN** HTTP 507 Insufficient Storage is returned
- **AND** error message suggests deleting old sessions

### Requirement: Session Naming
The system SHALL validate session names.

#### Scenario: Valid session name
- **WHEN** session name is "my-session-2026-01-24"
- **THEN** session is created successfully

#### Scenario: Invalid session name with spaces
- **WHEN** session name is "my session"
- **THEN** HTTP 400 Bad Request is returned
- **AND** error message explains naming requirements

#### Scenario: Invalid session name with special chars
- **WHEN** session name is "my/session"
- **THEN** HTTP 400 Bad Request is returned

