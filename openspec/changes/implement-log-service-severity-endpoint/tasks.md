# Tasks: Implement Log Service Severity Endpoint

## 1. Investigation

- [ ] 1.1 Review LogServiceDetails.jsx to understand expected response format
- [ ] 1.2 Check ClickHouse logs table schema (pattern_template, service_name, severity)
- [ ] 1.3 Test existing /api/v1/logs endpoint to see current data structure
- [ ] 1.4 Verify ListLogs method supports service_name filtering

## 2. Backend Implementation

- [ ] 2.1 Implement getLogByServiceAndSeverity handler in server.go
- [ ] 2.2 Extract service and severity from URL parameters
- [ ] 2.3 Call s.store.ListLogs with service_name filter
- [ ] 2.4 Filter results to matching severity
- [ ] 2.5 Aggregate attribute_keys across filtered logs
- [ ] 2.6 Aggregate resource_keys across filtered logs
- [ ] 2.7 Build body_templates array from pattern_template + sample_count

## 3. Error Handling

- [ ] 3.1 Validate service and severity parameters (not empty)
- [ ] 3.2 Handle storage errors gracefully (return 500)
- [ ] 3.3 Handle no results case (return 200 with empty arrays)
- [ ] 3.4 Add logging for debugging

## 4. Testing

- [ ] 4.1 Build and start OCC with ClickHouse backend
- [ ] 4.2 Test: GET /api/v1/logs/service/inventory-svc/severity/ERROR
- [ ] 4.3 Verify response structure matches LogServiceDetails expectations
- [ ] 4.4 Test with multiple services and severities
- [ ] 4.5 Test with non-existent service (should return empty, not error)
- [ ] 4.6 Open UI and navigate to LogServiceDetails to verify it works

## 5. Integration

- [ ] 5.1 Verify LogServiceDetails component displays data correctly
- [ ] 5.2 Check browser console for errors
- [ ] 5.3 Test drill-down flow: Logs → Service → Severity → Templates
- [ ] 5.4 Verify body_templates render with counts
- [ ] 5.5 Verify attribute/resource keys display properly

## 6. Documentation

- [ ] 6.1 Add API documentation for endpoint in docs/API.md
- [ ] 6.2 Document query parameters (if any)
- [ ] 6.3 Document response schema
- [ ] 6.4 Add example curl command

## Completion Criteria

- ✅ Endpoint returns 200 OK for valid service+severity
- ✅ Response includes body_templates, attribute_keys, resource_keys
- ✅ LogServiceDetails UI component works without 501 errors
- ✅ Empty results handled gracefully
- ✅ No browser console errors
- ✅ Documentation updated
