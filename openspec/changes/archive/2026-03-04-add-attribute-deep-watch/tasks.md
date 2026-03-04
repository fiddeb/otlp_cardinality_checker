# Tasks: Add Attribute Deep Watch

## Phase 1: Data Model

### Task 1.1: WatchedAttribute struct
- [x] Create `WatchedAttribute` struct in `pkg/models/attribute.go`
- [x] Fields: `Key`, `Values map[string]int64`, `UniqueCount`, `TotalObservations`, `Overflow`, `WatchingSince`, `MaxValues`, `mu sync.RWMutex`
- [x] Implement `NewWatchedAttribute(key string, maxValues int) *WatchedAttribute`
- [x] Implement `AddValue(value string)` (increments count, respects overflow)
- [x] **Tests**: unit tests for AddValue, overflow, concurrency

---

## Phase 2: Storage Layer

### Task 2.1: Extend Storage interface
- [x] Add `WatchAttribute(ctx, key string) error` to `internal/storage/interface.go`
- [x] Add `UnwatchAttribute(ctx, key string) error`
- [x] Add `GetWatchedAttribute(ctx, key string) (*models.WatchedAttribute, error)`
- [x] Add `ListWatchedAttributes(ctx) ([]*models.WatchedAttribute, error)`

### Task 2.2: In-memory implementation
- [x] Add `watched map[string]*models.WatchedAttribute` and `watchedmu sync.RWMutex` to `memory.Store`
- [x] Implement all four new interface methods in `internal/storage/memory/store.go`
- [x] Update `StoreAttributeValue` to call `w.AddValue(value)` when key is watched (RLock check, no blocking)
- [x] **Tests**: table-driven tests for all four methods, idempotent activate, limit enforcement, hot path overhead

### Task 2.3: Max watched limit config
- [x] Add `MaxWatchedFields int` to `storage.Config` (default 10)
- [x] Pass through to `memory.Store`

---

## Phase 3: Startup Flag

### Task 3.1: `--watch-fields` flag
- [x] Add `WatchFields []string` to server config in `cmd/server/main.go`
- [x] Parse comma-separated `--watch-fields` flag
- [x] After storage init, call `storage.WatchAttribute` for each key
- [x] If count exceeds `MaxWatchedFields`, log error and exit non-zero
- [x] **Tests**: verify startup with valid and overflowing `--watch-fields`

---

## Phase 4: API Layer

### Task 4.1: Watch management endpoints
- [x] Register routes: `POST /api/v1/attributes/{key}/watch` and `DELETE /api/v1/attributes/{key}/watch`
- [x] Implement `handleWatchAttribute` (POST): call storage, return 200/409/429
- [x] Implement `handleUnwatchAttribute` (DELETE): call storage, return 204/404
- [x] URL-decode `:key` path param for dot-separated keys
- [x] **Tests**: handler tests for all status codes

### Task 4.2: Value Explorer endpoint
- [x] Register route: `GET /api/v1/attributes/{key}/watch`
- [x] Implement `handleGetWatchedAttribute`: fetch from storage, sort and paginate values
- [x] Support query params: `sort_by` (count|value), `sort_direction` (asc|desc), `page`, `page_size`, `q` (prefix filter)
- [x] Return 404 if key not watched
- [x] **Tests**: handler tests for sorting, pagination, prefix filter, not-found

### Task 4.3: Extend existing attribute endpoints
- [x] Add `watched bool` field to `AttributeMetadata` JSON response in list and detail endpoints
- [x] **Tests**: verify `watched` field is correct in list response

---

## Phase 5: UI

### Task 5.1: Watch toggle in Attribute Catalog
- [x] Add **Watch** column to Attribute Catalog table in `web/src/components/AttributeCatalog.jsx` (or equivalent)
- [x] Render toggle button per row, calling POST/DELETE watch endpoints
- [x] Show `"watching since HH:MM"` badge on active rows
- [x] Disable inactive toggles when limit is reached (tooltip)
- [x] Show overflow warning icon + tooltip

### Task 5.2: Value Explorer panel
- [x] Create `web/src/components/ValueExplorer.jsx`
- [x] Fetch from `GET /api/v1/attributes/:key/watch` on open and on Refresh
- [x] Render header: key, watching since, unique_count, total_observations
- [x] Render overflow warning banner when `overflow = true`
- [x] Render searchable, sortable table (Value | Count)
- [x] Empty state message when `values` is empty
- [x] Connect: clicking watched attribute key in catalog opens Value Explorer

---

---

## Phase 6: Session Integration

### Task 6.1: Serialize watch data in sessions
- [x] Include all `WatchedAttribute` entries (active and inactive) in session save payload
- [x] Deserialize watch data on session load, restoring each as `Active = false`
- [x] Ensure Value Explorer API works for inactive (restored) watch entries
- [x] **Tests**: round-trip save+load preserves values, counts, overflow; `StoreAttributeValue` does not mutate inactive entries

---

## Dependencies

- Phase 2 depends on Phase 1
- Phase 3 depends on Phase 2
- Phase 4 depends on Phase 2
- Phase 5 depends on Phase 4
- Phase 6 depends on Phase 2 (storage) and the existing sessions implementation
