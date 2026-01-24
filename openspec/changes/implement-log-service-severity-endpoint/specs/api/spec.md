# API Specification Changes

## ADDED Requirements

### Requirement: Log Service Severity Detail Endpoint

**Description**: The API SHALL provide an endpoint to retrieve detailed log analysis for a specific service and severity combination.

**Requirements**:
- The endpoint SHALL be accessible at `GET /api/v1/logs/service/{service}/severity/{severity}`
- The endpoint SHALL accept URL parameters: `service` (service name) and `severity` (log severity level)
- The endpoint SHALL return aggregated log pattern data including body templates, attribute keys, and resource keys
- The endpoint SHALL filter logs by both service_name and severity
- The endpoint SHALL return 200 OK with empty arrays for non-existent service/severity combinations
- The endpoint SHALL return 400 Bad Request if service or severity parameters are missing

**Rationale**: Enables UI drill-down from Logs overview to service-specific severity analysis, allowing users to debug log issues at granular level.

#### Scenario: Retrieve log details for service and severity

**GIVEN**: ClickHouse has logs for service "inventory-svc" with severity "ERROR"  
**WHEN**: Client requests `GET /api/v1/logs/service/inventory-svc/severity/ERROR`  
**THEN**: 
- ✅ Response status is 200 OK
- ✅ Response includes `body_templates` array with pattern and count
- ✅ Response includes `attribute_keys` map with key metadata
- ✅ Response includes `resource_keys` map with key metadata
- ✅ Response includes `service_name` and `severity` fields

#### Scenario: Handle non-existent service

**GIVEN**: ClickHouse has no logs for service "nonexistent-svc"  
**WHEN**: Client requests `GET /api/v1/logs/service/nonexistent-svc/severity/ERROR`  
**THEN**:
- ✅ Response status is 200 OK (not 404)
- ✅ Response includes empty `body_templates` array
- ✅ Response includes empty `attribute_keys` object
- ✅ Response includes empty `resource_keys` object

#### Scenario: Handle missing parameters

**GIVEN**: API receives request with empty service or severity  
**WHEN**: Client requests `GET /api/v1/logs/service//severity/ERROR`  
**THEN**:
- ✅ Response status is 400 Bad Request
- ✅ Error message indicates "service and severity are required"

#### Scenario: Aggregate keys across multiple log entries

**GIVEN**: Service "api-svc" has 3 log entries with ERROR severity  
**AND**: Each entry has different attribute_keys and resource_keys  
**WHEN**: Client requests `GET /api/v1/logs/service/api-svc/severity/ERROR`  
**THEN**:
- ✅ Response `attribute_keys` contains union of all attribute keys
- ✅ Response `resource_keys` contains union of all resource keys
- ✅ Key metadata counts are aggregated across entries

### Requirement: Body Templates in Response

**Description**: The endpoint SHALL return body templates (log patterns) extracted from the pattern_template field in ClickHouse logs table.

**Requirements**:
- Each body template SHALL include `template` (the pattern string) and `count` (sample count)
- Body templates SHALL be sorted by count descending (most common first)
- The system SHALL use the `pattern_template` field from ClickHouse logs table
- Empty pattern_template values SHALL be skipped (not included in response)

**Rationale**: Body templates allow users to identify common log message patterns and their frequency for debugging.

#### Scenario: Return body templates with counts

**GIVEN**: Logs table has pattern_template "Failed to connect: <*>" with sample_count 1250  
**WHEN**: Client requests endpoint for matching service+severity  
**THEN**:
- ✅ Response `body_templates` includes `{"template": "Failed to connect: <*>", "count": 1250}`
- ✅ Templates are sorted by count descending

#### Scenario: Handle logs without patterns

**GIVEN**: Logs table has entries where pattern_template is empty or null  
**WHEN**: Client requests endpoint  
**THEN**:
- ✅ Empty patterns are excluded from `body_templates` array
- ✅ Response does not crash or return null

## MODIFIED Requirements

None - this is a new endpoint without modifications to existing functionality.

## REMOVED Requirements

None
