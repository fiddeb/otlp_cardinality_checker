# Change: Add Snapshot Sessions with Diff Mode

## Why

OCC is ephemeral by design - all data is lost on restart. This creates friction for common workflows:

1. **Incremental signal collection**: Users analyze metrics one day, traces another. Currently impossible to combine results.
2. **Pre/post deploy comparison**: No way to detect "what changed after deploy" without manual JSON exports.
3. **Service-focused analysis**: Cannot view complete telemetry picture (metrics + traces + logs) for a service when collected separately.

Users need a way to persist, merge, and compare analysis results across sessions.

## What Changes

- **NEW**: Sessions capability for saving/loading/merging telemetry state
- **NEW**: Diff API for comparing two sessions and detecting changes
- **NEW**: File-based session storage under `data/sessions/`
- **MODIFIED**: Storage interface to support serialization/deserialization
- **MODIFIED**: UI to include Sessions tab with load/merge/diff functionality

## Impact

- Affected specs: `storage`, `api`, `ui`
- Affected code:
  - `internal/storage/` - session persistence layer
  - `internal/api/` - new session endpoints
  - `pkg/models/` - session and diff types
  - `web/src/components/` - Sessions UI components
- New configuration: `OCC_SESSION_DIR` environment variable
- Backward compatible: existing APIs unchanged
