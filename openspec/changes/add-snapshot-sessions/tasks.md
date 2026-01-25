# Tasks: Add Snapshot Sessions with Diff Mode

## Phase 1: Core Session Infrastructure

### 1.1 Models and Types
- [x] 1.1.1 Create `pkg/models/session.go` with Session, SessionMetadata types
- [x] 1.1.2 Add SerializedHLL type with base64 encoding/decoding
- [x] 1.1.3 Create `pkg/models/diff.go` with DiffResult, Change, Severity types
- [x] 1.1.4 Add JSON tags and validation for all new types
- [x] 1.1.5 Write unit tests for HLL serialization round-trip

### 1.2 Storage Layer
- [x] 1.2.1 Add session methods to `storage.Storage` interface → Created separate `internal/storage/sessions/` package
- [x] 1.2.2 Implement file-based session storage in `internal/storage/sessions/`
- [x] 1.2.3 Add gzip compression for session files
- [x] 1.2.4 Implement session directory initialization on startup
- [x] 1.2.5 Add configuration for session directory and limits
- [x] 1.2.6 Write integration tests for save/load/delete

### 1.3 Serialization
- [x] 1.3.1 Add `MarshalSession()` method → Created `internal/storage/sessions/serializer.go`
- [x] 1.3.2 Add `UnmarshalSession()` method → Created in serializer.go
- [x] 1.3.3 Implement HLL register serialization (base64)
- [x] 1.3.4 Handle nil HLL gracefully during serialization
- [x] 1.3.5 Write tests for large session serialization (50k+ metrics)

## Phase 2: Session API

### 2.1 CRUD Endpoints
- [x] 2.1.1 Implement `POST /api/v1/sessions` - create session
- [x] 2.1.2 Implement `GET /api/v1/sessions` - list sessions
- [x] 2.1.3 Implement `GET /api/v1/sessions/{name}` - get session metadata
- [x] 2.1.4 Implement `DELETE /api/v1/sessions/{name}` - delete session
- [x] 2.1.5 Add request validation (name format, size limits)
- [x] 2.1.6 Write API tests for all CRUD operations

### 2.2 Load and Merge Endpoints
- [x] 2.2.1 Implement `POST /api/v1/sessions/{name}/load` - load session
- [x] 2.2.2 Implement `POST /api/v1/sessions/{name}/merge` - merge into current
- [ ] 2.2.3 Add `signals` filter parameter (metrics, traces, logs)
- [ ] 2.2.4 Add `services` filter parameter
- [x] 2.2.5 Write tests for filtered load/merge

### 2.3 Export/Import Endpoints
- [x] 2.3.1 Implement `GET /api/v1/sessions/{name}/export` - download session JSON
- [x] 2.3.2 Implement `POST /api/v1/sessions/import` - upload session JSON
- [ ] 2.3.3 Add Content-Type handling (application/json, application/gzip)
- [x] 2.3.4 Write tests for export/import round-trip

## Phase 3: Diff Engine

### 3.1 Core Diff Algorithm
- [x] 3.1.1 Implement metric diff (added/removed/changed)
- [x] 3.1.2 Implement span diff
- [x] 3.1.3 Implement log diff
- [ ] 3.1.4 Implement attribute catalog diff
- [x] 3.1.5 Calculate severity scores for changes
- [ ] 3.1.6 Write unit tests for each signal type diff

### 3.2 Diff API
- [x] 3.2.1 Implement `GET /api/v1/sessions/diff?from=X&to=Y`
- [ ] 3.2.2 Add `signal_type` filter parameter
- [ ] 3.2.3 Add `service` filter parameter
- [x] 3.2.4 Add `min_severity` filter parameter
- [ ] 3.2.5 Write API tests for diff endpoint

### 3.3 Change Detection Logic
- [x] 3.3.1 Detect cardinality changes with thresholds
- [x] 3.3.2 Detect sample rate changes
- [x] 3.3.3 Detect new high-cardinality attributes
- [x] 3.3.4 Detect label/attribute key changes
- [ ] 3.3.5 Detect log template changes
- [ ] 3.3.6 Write tests for each change type detection

## Phase 4: Merge Logic

### 4.1 HLL Merge
- [x] 4.1.1 Implement HLL union operation for cardinality merge → Uses hyperloglog.Merge
- [x] 4.1.2 Handle nil HLL cases during merge
- [ ] 4.1.3 Write tests verifying cardinality accuracy after merge

### 4.2 Metadata Merge
- [x] 4.2.1 Implement metric metadata merge (sum counts, union keys) → Uses existing MergeMetricMetadata
- [x] 4.2.2 Implement span metadata merge → Uses StoreSpan with merge logic
- [x] 4.2.3 Implement log metadata merge → Uses StoreLog with merge logic
- [x] 4.2.4 Implement attribute catalog merge → Added MergeAttribute method
- [x] 4.2.5 Handle timestamp merge (min FirstSeen, max LastSeen)
- [ ] 4.2.6 Write integration tests for multi-session merge

## Phase 5: UI Integration

### 5.1 Sessions Tab
- [x] 5.1.1 Create `SessionsView.jsx` component
- [x] 5.1.2 Implement session list with metadata display
- [x] 5.1.3 Add "Save Current" button/modal
- [x] 5.1.4 Add Load/Merge/Delete actions
- [x] 5.1.5 Add export/import functionality
- [x] 5.1.6 Add session indicator in header (loaded session name)

### 5.2 Diff View
- [x] 5.2.1 Create `DiffView.jsx` component
- [x] 5.2.2 Implement session pair selector
- [x] 5.2.3 Display summary (added/removed/changed counts)
- [x] 5.2.4 Display critical changes with highlighting
- [x] 5.2.5 Implement expandable change details
- [x] 5.2.6 Add signal type filter tabs
- [x] 5.2.7 Add service filter dropdown

### 5.3 Navigation
- [x] 5.3.1 Add Sessions tab to main navigation
- [x] 5.3.2 Add "Compare" button in Sessions list
- [x] 5.3.3 Link from diff results to detailed views

## Phase 6: Documentation and Polish

### 6.1 Documentation
- [ ] 6.1.1 Add sessions section to docs/API.md
- [ ] 6.1.2 Add sessions section to docs/USAGE.md
- [ ] 6.1.3 Document configuration options
- [ ] 6.1.4 Add CI/CD integration examples

### 6.2 Polish
- [ ] 6.2.1 Add progress indicator for large session operations
- [ ] 6.2.2 Add error handling and user feedback
- [ ] 6.2.3 Add session naming validation (kebab-case)
- [ ] 6.2.4 Add disk space warnings

## Dependencies

- Phase 2 depends on Phase 1
- Phase 3 depends on Phase 1 (needs serialized session format)
- Phase 4 depends on Phase 1
- Phase 5 depends on Phases 2, 3, 4

## Parallelizable Work

- 1.1 (Models) and 1.2 (Storage) can start in parallel
- 3.1 (Diff algorithm) and 4.1 (Merge logic) can run in parallel
- 5.1 (Sessions UI) and 5.2 (Diff UI) can run in parallel after API is ready
