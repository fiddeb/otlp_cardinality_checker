# Design: Attribute Deep Watch

## Context

The existing `AttributeMetadata` struct in `pkg/models/attribute.go` stores up to 10 sample values using a bounded slice. This is correct for normal operation. Deep Watch needs a separate, parallel data structure that is only activated for specific keys.

## Data Model

A new struct lives alongside `AttributeMetadata`:

```go
// WatchedAttribute holds full value-frequency data for a deep-watched attribute key.
// It is separate from AttributeMetadata to avoid touching the hot path for all attributes.
type WatchedAttribute struct {
    mu sync.RWMutex

    // Key is the attribute key being watched
    Key string `json:"key"`

    // Values maps unique observed values to their occurrence count
    Values map[string]int64 `json:"values"`

    // UniqueCount is len(Values), cached to avoid lock on read
    UniqueCount int64 `json:"unique_count"`

    // TotalObservations is the total number of AddValue calls since watching started
    TotalObservations int64 `json:"total_observations"`

    // Active is true when the watch is collecting new values.
    // False when restored from a session (read-only view of historical data).
    Active bool `json:"active"`

    // Overflow is true when the unique value cap was reached; collection paused
    Overflow bool `json:"overflow"`

    // WatchingSince is when deep watch was first activated for this key
    WatchingSince time.Time `json:"watching_since"`

    // MaxValues is the unique-value cap (default 10,000)
    MaxValues int `json:"-"`
}
```

## Storage Layer

The `Storage` interface gains three new methods:

```go
// WatchAttribute activates deep watch for an attribute key.
WatchAttribute(ctx context.Context, key string) error

// UnwatchAttribute deactivates deep watch and discards collected values.
UnwatchAttribute(ctx context.Context, key string) error

// GetWatchedAttribute returns the WatchedAttribute for a key, or nil if not watched.
GetWatchedAttribute(ctx context.Context, key string) (*models.WatchedAttribute, error)

// ListWatchedAttributes returns all currently watched attributes.
ListWatchedAttributes(ctx context.Context) ([]*models.WatchedAttribute, error)
```

The in-memory store adds a `watched map[string]*models.WatchedAttribute` field with its own mutex.

## Hot Path Impact

`StoreAttributeValue` in `memory/store.go` must check if the key is watched and, if so, also call `WatchedAttribute.AddValue(value)`. The check is a single map read under RLock — negligible overhead when no keys are watched.

```go
func (s *Store) StoreAttributeValue(ctx context.Context, key, value, signalType, scope string) error {
    // Existing logic ...

    // Deep watch: append to watched values if active
    s.watchedmu.RLock()
    w, watched := s.watched[key]
    s.watchedmu.RUnlock()
    if watched && w.Active {
        w.AddValue(value)
    }

    return nil
}
```

## Startup Flag

`--watch-fields` is a comma-separated list of attribute keys. The server calls `storage.WatchAttribute(ctx, key)` for each during startup after storage is initialized.

```
otlp-cardinality-checker --watch-fields=workflow.folder,service.instance.id
```

## API Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/attributes/:key/watch` | Enable deep watch |
| `DELETE` | `/api/v1/attributes/:key/watch` | Disable deep watch |
| `GET` | `/api/v1/attributes/:key/watch` | Get value explorer data |

`GET /api/v1/attributes/:key/watch` response:

```json
{
  "key": "workflow.folder",
  "watching_since": "2026-03-04T14:32:00Z",
  "unique_count": 847,
  "total_observations": 12430,
  "overflow": false,
  "values": [
    { "value": "reports/q1", "count": 312 },
    { "value": "exports/daily", "count": 198 }
  ]
}
```

`values` array is sorted by `count` descending. Query params `sort_by` (count|value|last_seen) and `page`/`page_size` supported.

## UI

### Attribute Catalog table changes

- New column: **Watch** — toggle button (inactive / active)
- Active row: badge showing "watching since HH:MM"
- Overflow state: warning icon + tooltip "10,000 unique values reached, collection paused"

### Value Explorer panel

Opens as a side panel or full detail view when clicking on a watched attribute key. Shows:
- `WatchingSince`, `UniqueCount`, `TotalObservations`, `Overflow` warning
- Search input (client-side filter)
- Sortable table: Value | Count

## Memory Estimate

10,000 unique values × ~80 bytes per map entry ≈ 800KB per watched field.  
Max 10 watched fields ≈ 8MB worst case. Acceptable alongside the rest of the tool's ~256MB budget.
