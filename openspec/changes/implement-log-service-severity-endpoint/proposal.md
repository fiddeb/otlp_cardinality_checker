# Proposal: Implement Log Service Severity Endpoint

## Summary

Implement the `/api/v1/logs/service/{service}/severity/{severity}` endpoint for ClickHouse backend to enable `LogServiceDetails` component to display log pattern analysis for a specific service and severity combination.

## Why

### Problem

The `LogServiceDetails.jsx` component attempts to fetch log details via:
```
GET /api/v1/logs/service/{service}/severity/{severity}
```

This endpoint currently returns **501 Not Implemented** for ClickHouse backend, breaking the UI drill-down flow from the Logs view.

**Error from browser console:**
```
GET http://localhost:3000/api/v1/logs/service/inventory-svc/severity/undefined 501 (Not Implemented)
```

### User Impact

Users cannot:
- View log body templates (patterns) for a specific service+severity
- Analyze attribute and resource keys for filtered logs
- Navigate from Logs overview → Service details
- Debug log issues at the service+severity level

### Root Cause

```go
// internal/api/server.go:384
func (s *Server) getLogByServiceAndSeverity(w http.ResponseWriter, r *http.Request) {
    // TODO: Reimplement with ClickHouse storage or remove
    s.respondError(w, http.StatusNotImplemented, "operation not yet implemented for ClickHouse storage")
}
```

This was a TODO left from the ClickHouse migration (PR #13). The endpoint exists and is routed, but has no implementation.

## What Changes

### API Implementation

Implement `getLogByServiceAndSeverity` in ClickHouse backend to:
1. Query `logs` table filtering by `service_name` and `severity`
2. Aggregate `attribute_keys` and `resource_keys` across matching logs
3. Return `body_templates` (pattern_template field) with counts
4. Match the response format expected by `LogServiceDetails.jsx`

**Query Strategy:**
```sql
SELECT 
    pattern_template,
    attribute_keys,
    resource_keys,
    sample_count
FROM logs FINAL
WHERE service_name = ? AND severity = ?
```

### Response Format

```json
{
  "service_name": "inventory-svc",
  "severity": "ERROR",
  "body_templates": [
    {
      "template": "Failed to connect to database: <*>",
      "count": 1250
    }
  ],
  "attribute_keys": {
    "log.level": {"count": 1250},
    "user.id": {"count": 1250}
  },
  "resource_keys": {
    "service.name": {"count": 1250},
    "deployment.environment": {"count": 1250}
  }
}
```

## Scope

### In Scope

- Implement `getLogByServiceAndSeverity` for ClickHouse storage
- Query logs filtering by service_name + severity
- Aggregate attribute_keys and resource_keys
- Return body_templates from pattern_template field
- Handle empty results gracefully (empty arrays, not errors)
- Add error handling for invalid service/severity
- Test with existing ClickHouse data

### Out of Scope

- Changes to `LogServiceDetails.jsx` UI (already working)
- Pagination (use existing logs data, typically <100 patterns per service+severity)
- Implementing for SQLite/Memory backends (future work)
- Advanced filtering (min_count, etc.) - component handles this client-side
- New ClickHouse schema changes
- Performance optimization (start simple, optimize if needed)

### Success Criteria

✅ GET `/api/v1/logs/service/inventory-svc/severity/ERROR` returns 200 OK  
✅ Response includes `body_templates` array with pattern_template and count  
✅ Response includes aggregated `attribute_keys` and `resource_keys`  
✅ `LogServiceDetails` component displays data without errors  
✅ Empty results return `{"body_templates": [], "attribute_keys": {}, "resource_keys": {}}`  
✅ Invalid service/severity returns 404 or empty data (not 500)  

## Dependencies

- ClickHouse `logs` table (already exists)
- Existing `ListLogs` method in storage interface
- `LogServiceDetails.jsx` component (already exists and functional)

## Timeline Estimate

- Implementation: 1-2 hours
- Testing: 30 minutes
- Documentation: 15 minutes
- **Total: ~3 hours**

## References

- UI Component: `web/src/components/LogServiceDetails.jsx`
- Backend Handler: `internal/api/server.go:384` (getLogByServiceAndSeverity)
- ClickHouse Schema: `internal/storage/clickhouse/schema.go` (logs table)
- Related PR: #13 (ClickHouse migration)
