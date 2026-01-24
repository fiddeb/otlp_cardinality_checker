# Change: Add Global Attribute Catalog

## Why

High-cardinality attributes are a major cost driver in observability systems. Before this feature, users could see cardinality metrics per signal type (metrics, traces, logs), but had no way to answer:
- "Which attribute keys are high-cardinality across ALL signals?"
- "Is `user_id` only used in metrics, or also in traces and logs?"
- "What are the top 10 attributes driving cardinality in my entire system?"

Without global visibility, teams waste time correlating cardinality across different signal views and miss cross-signal cardinality patterns.

## What Changes

Add a global attribute catalog that tracks ALL attribute keys across metrics, spans, and logs:

1. **Data Model**: New `AttributeMetadata` type with HyperLogLog-based cardinality estimation
2. **Storage Layer**: In-memory and SQLite storage for attribute catalog
3. **Analyzer Integration**: Extract attributes from all signals and feed catalog
4. **API Endpoints**: REST API for querying attribute catalog
5. **UI Component**: New "Attributes" tab with filtering, sorting, and pagination

### Key Capabilities
- Track which signals (metric/span/log) use each attribute
- Distinguish resource vs data-point attributes
- Estimate cardinality using HyperLogLog (memory-efficient)
- Store sample values (up to 10) for each attribute
- Filter by signal type, scope, or cardinality threshold
- Sort by cardinality, count, or timestamp

## Impact

### Affected Capabilities (New)
- `attribute-tracking`: Global attribute metadata collection
- `storage`: Extended storage interface with attribute catalog methods
- `api`: New REST endpoints for attribute queries
- `ui`: New Attributes tab in web interface

### Affected Code
- `pkg/models/attribute.go` (NEW): AttributeMetadata model with HLL
- `internal/storage/interface.go`: Added 3 methods for attribute catalog
- `internal/storage/memory/store.go`: In-memory attribute catalog storage
- `internal/storage/sqlite/store.go`: SQLite persistence with migration 007
- `internal/analyzer/common.go` (NEW): Shared attribute extraction logic
- `internal/analyzer/metrics.go`: Extract metric label attributes
- `internal/analyzer/traces.go`: Extract span attributes
- `internal/analyzer/logs.go`: Extract log attributes
- `internal/receiver/*.go`: Pass AttributeCatalog to analyzers
- `internal/api/server.go`: New GET /api/v1/attributes endpoints
- `web/src/components/AttributesView.jsx` (NEW): UI component

### Database Changes
- **BREAKING**: Requires migration 007 for SQLite users
- New table: `attribute_catalog` with HLL blob storage
- 5 indexes for efficient filtering and sorting

### Performance Considerations
- HyperLogLog uses ~16KB per attribute (precision 14)
- In-memory catalog can handle 500k+ unique attribute keys
- SQLite writes should be batched (future optimization needed)

## Migration Path

For existing deployments:
1. Update binary (includes migration 007)
2. Restart server (migration runs automatically on SQLite mode)
3. Attribute catalog starts empty
4. Populates as new telemetry arrives

No data loss - existing metrics/spans/logs data unaffected.
