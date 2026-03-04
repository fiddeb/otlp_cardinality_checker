# Change: Add Attribute Deep Watch

## change-id
`add-attribute-deep-watch`

## Status
Proposed

## Why
The tool stores at most 10 sample values per attribute key by design, making it impossible to debug corrupted or misbehaving fields because the offending values are never fully captured. Teams must resort to verbose collector debug exporters and manual correlation to identify bad values.

## What Changes
- New `WatchedAttribute` data model in `attribute-tracking` spec
- New watch interface methods added to `storage` spec and in-memory implementation
- New watch API endpoints: `POST/DELETE /api/v1/attributes/:key/watch` and `GET /api/v1/attributes/watched`
- Watch toggle column and Value Explorer panel added to Attribute Catalog UI

## Problem

The tool stores a maximum of 10 sample values per attribute key by design — preventing cardinality explosion in the tool itself. This makes it impossible to debug a specific corrupted or misbehaving field, because the actual offending values are never captured.

Today the workaround is to route suspicious telemetry through the OpenTelemetry Collector's debug exporter with `verbosity: detailed`, then manually correlate service, attribute, and value. This works but is cumbersome and requires collector reconfiguration.

When Kafka is in the pipeline, data can be replayed from the offset where the problem started, which eliminates the retroactivity concern that would otherwise make a runtime-only solution insufficient.

## Proposed Solution

Add a **Deep Watch** mode for individual attribute keys. When a key is watched, the tool switches from storing only samples to storing all unique values with their occurrence counts. This is bounded by a per-key cap (default 10,000 unique values) to prevent unbounded memory use.

Deep Watch is activated either:
1. **At startup** via `--watch-fields=key1,key2` (useful for Kafka replay)
2. **At runtime** via a toggle in the Attribute Catalog UI (useful for live investigation)

In the Attribute Catalog, a watched attribute row shows a "watching since HH:MM" indicator. Clicking the attribute key opens a **Value Explorer** panel: a searchable, sortable list of all collected values and their counts.

Deep Watch state is in-memory only and does not persist across restarts. However, when a session is saved, the collected watch data (value-count map) for all currently watched keys is included in the session snapshot. Loading a session restores the collected values as **read-only** — watch is inactive and no new values are collected. The user can reactivate a watch toggle (or use `--watch-fields`) to resume collection, for example after a Kafka replay.

## Scope

| Capability | Change |
|---|---|
| `attribute-tracking` | New `WatchedAttribute` data model |
| `storage` | New watch interface methods and in-memory implementation |
| `api` | New watch endpoints (enable, disable, query values) |
| `ui` | Watch toggle in Attribute Catalog + Value Explorer panel |

## Constraints

- Max 10 watched fields simultaneously (configurable at startup)
- Max 10,000 unique values per watched field before overflow
- Watched values stored as `map[string]int64` (value → count)
- State resets on restart — documented in UI and README
- Sessions include collected watch data; loaded sessions restore values as read-only (watch inactive)
- No changes to the existing 10-sample default for unwatched attributes
- `--watch-fields` startup flag is additive; individual UI toggles add to it at runtime

## Out of Scope

- Persisting active watch configuration across restarts (values are saved in sessions, config is not)
- Deep watch for signals other than the Attribute Catalog (metrics/span/log tabs)
- Streaming/push updates to the Value Explorer (polling is sufficient)
