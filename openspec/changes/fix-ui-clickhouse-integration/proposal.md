# Proposal: Fix UI ClickHouse Integration

## Summary

Fix React UI components to work correctly with ClickHouse backend. Currently, several UI tabs throw errors because they expect SQLite-style responses or call unimplemented endpoints. This proposal addresses: (1) Details.jsx TypeError when reading attribute metadata, (2) MetadataComplexity.jsx 500 error, and (3) LogsView.jsx 501 error for `/api/v1/logs/by-service`.

## Why

After migrating to ClickHouse (PR #13 merged), the React UI is broken in multiple places:

**TypeError in Details.jsx (line 188):**
```javascript
metadata.value_samples.slice(0, 5) // TypeError: Cannot read properties of undefined (reading 'slice')
```
The ClickHouse backend returns different attribute metadata structure - `value_samples` may not exist or be `null`. This affects all signal types: metrics, spans (traces), and logs when viewing details.

**500 Error in MetadataComplexity:**
```
GET /api/v1/cardinality/complexity?threshold=10&limit=50 500 (Internal Server Error)
```
The endpoint exists and is implemented, but likely returns malformed data or encounters runtime error.

**501 Error in LogsView:**
```
GET /api/v1/logs/by-service?limit=1000 501 (Not Implemented)
```
The endpoint `listLogsByService` explicitly returns 501 with message "operation not yet implemented for ClickHouse storage".

These errors break user experience and prevent users from exploring their telemetry data through the UI.

## What Changes

### Frontend Fixes (React Components)

**1. Details.jsx - Add null safety for all signal types:**
```jsx
// Line 188: Add fallback for missing value_samples
// Affects metrics, spans (traces), and logs
<td className="samples">
  {(metadata.value_samples || []).slice(0, 5).join(', ')}
</td>
```

**Note**: The same fix applies to both label keys (line 188) and resource keys tables in Details view, affecting all three signal types: metrics (http_request_duration_ms), spans ("DELETE /api/v1/users"), and logs.

**2. MetadataComplexity.jsx - Better error handling:**
- Add error boundary or null checks
- Log full API response for debugging
- Display user-friendly error message

**3. LogsView.jsx - Implement or replace endpoint:**
- **Option A**: Implement `listLogsByService` in ClickHouse store
- **Option B**: Change UI to use existing `/api/v1/logs` endpoint instead

### Backend Fixes (Go API)

**4. Verify GetMetadataComplexity implementation:**
- Add debug logging to identify 500 error root cause
- Check ClickHouse query validity
- Test with actual ClickHouse data

**5. Implement or remove listLogsByService:**
- If keeping: Implement ClickHouse query for service-grouped logs
- If removing: Update UI to use alternative approach

### Testing

**6. Manual UI testing:**
- Test all UI tabs with ClickHouse backend
- Verify no console errors
- Validate data display correctness

**7. Integration test:**
- Add test that loads UI and checks for API errors
- Validate API responses match UI expectations

## Scope

**In scope:**
- Fix Details.jsx null safety
- Debug and fix MetadataComplexity 500 error
- Implement or replace logs-by-service endpoint
- Update UI to handle ClickHouse response formats

**Out of scope:**
- New UI features
- UI redesign
- Performance optimization
- API v2 endpoints (deferred to Phase 3)
