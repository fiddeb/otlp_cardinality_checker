# api Specification

## Purpose
Provides REST API endpoints for querying and retrieving attribute metadata collected from OTLP telemetry. Enables filtering, sorting, and pagination of attributes by signal type, scope, and cardinality metrics.
## Requirements
### Requirement: List Attributes Endpoint
The system SHALL provide GET /api/v1/attributes endpoint for listing attributes.

#### Scenario: List all attributes
- **WHEN** GET /api/v1/attributes is called
- **THEN** returns JSON array of all AttributeMetadata
- **AND** HTTP status 200

#### Scenario: Filter by signal type
- **WHEN** GET /api/v1/attributes?signal_type=metric is called
- **THEN** returns only attributes used in metrics

#### Scenario: Filter by scope
- **WHEN** GET /api/v1/attributes?scope=resource is called
- **THEN** returns only resource attributes

#### Scenario: Filter by minimum cardinality
- **WHEN** GET /api/v1/attributes?min_cardinality=1000 is called
- **THEN** returns only attributes with estimated_cardinality >= 1000

#### Scenario: Sort by cardinality
- **WHEN** GET /api/v1/attributes?sort_by=cardinality&sort_direction=desc is called
- **THEN** returns attributes sorted by cardinality descending

#### Scenario: Pagination
- **WHEN** GET /api/v1/attributes?page=2&page_size=50 is called
- **THEN** returns attributes 51-100
- **AND** includes pagination metadata (total, page, page_size)

#### Scenario: Combined query parameters
- **WHEN** GET /api/v1/attributes?signal_type=metric&min_cardinality=1000&sort_by=cardinality&page=1&page_size=20
- **THEN** returns top 20 high-cardinality metric attributes

### Requirement: Get Attribute Details Endpoint
The system SHALL provide GET /api/v1/attributes/:key endpoint for retrieving specific attribute.

#### Scenario: Get existing attribute
- **WHEN** GET /api/v1/attributes/user_id is called
- **THEN** returns complete AttributeMetadata for user_id
- **AND** HTTP status 200

#### Scenario: Get non-existent attribute
- **WHEN** GET /api/v1/attributes/nonexistent_key is called
- **THEN** returns HTTP status 404
- **AND** error message "attribute not found"

#### Scenario: URL encoding
- **WHEN** GET /api/v1/attributes/http.method is called (dot in key)
- **THEN** correctly retrieves attribute with key "http.method"

### Requirement: Response Format
The system SHALL return attribute data in consistent JSON format.

#### Scenario: Attribute metadata response
- **WHEN** any attribute endpoint returns data
- **THEN** response includes:
  - `key`: attribute key name
  - `count`: observation count
  - `estimated_cardinality`: HLL-estimated unique value count
  - `value_samples`: array of sample values (max 10)
  - `signal_types`: array of signal types using this attribute
  - `scope`: "resource", "attribute", or "both"
  - `first_seen`: ISO 8601 timestamp
  - `last_seen`: ISO 8601 timestamp

#### Scenario: List response format
- **WHEN** GET /api/v1/attributes returns multiple attributes
- **THEN** response includes:
  - `attributes`: array of AttributeMetadata
  - `total`: total count of matching attributes
  - `page`: current page number
  - `page_size`: number of items per page

### Requirement: Error Handling
The system SHALL return appropriate HTTP status codes and error messages.

#### Scenario: Invalid query parameters
- **WHEN** GET /api/v1/attributes?sort_by=invalid_field is called
- **THEN** returns HTTP status 400
- **AND** error message describing the invalid parameter

#### Scenario: Storage error
- **WHEN** storage backend fails during request
- **THEN** returns HTTP status 500
- **AND** error message "internal server error"

### Requirement: Performance
The system SHALL respond to attribute queries efficiently.

#### Scenario: Large result set
- **WHEN** query returns 10,000+ attributes
- **THEN** pagination limits response to page_size
- **AND** response time is under 1 second

#### Scenario: Complex filters
- **WHEN** query combines multiple filters and sorting
- **THEN** storage layer uses indexes efficiently
- **AND** response time is under 500ms

### Requirement: Watch Management Endpoints
The system SHALL provide endpoints to enable and disable Deep Watch for a specific attribute key.

#### Description
`POST /api/v1/attributes/:key/watch` MUST activate Deep Watch for the given key. `DELETE /api/v1/attributes/:key/watch` MUST deactivate it. Both endpoints MUST delegate to the storage layer and return appropriate HTTP status codes.

#### Requirements
1. `POST /api/v1/attributes/:key/watch` MUST activate Deep Watch and return HTTP 200 with the initial `WatchedAttribute` JSON
2. `POST` MUST return HTTP 409 if the key is already being watched
3. `POST` MUST return HTTP 429 if the maximum watched fields limit is reached
4. `DELETE /api/v1/attributes/:key/watch` MUST deactivate Deep Watch and return HTTP 204
5. `DELETE` MUST return HTTP 404 if the key is not currently watched
6. Both endpoints MUST URL-decode the `:key` path parameter (to support dots, slashes)

#### Scenario: Activate watch
**WHEN** `POST /api/v1/attributes/workflow.folder/watch` is called  
**THEN** HTTP 200 is returned  
**AND** response body contains `watching_since`, `key`, `unique_count: 0`, `overflow: false`

#### Scenario: Activate already-watched key
**GIVEN** `workflow.folder` is already watched  
**WHEN** `POST /api/v1/attributes/workflow.folder/watch` is called  
**THEN** HTTP 409 is returned  
**AND** response body contains `error: "already watched"`

#### Scenario: Limit reached
**GIVEN** 10 keys are currently watched  
**WHEN** `POST /api/v1/attributes/new.key/watch` is called  
**THEN** HTTP 429 is returned  
**AND** response body contains `error: "maximum watched fields limit reached"`

#### Scenario: Deactivate watch
**GIVEN** `workflow.folder` is being watched  
**WHEN** `DELETE /api/v1/attributes/workflow.folder/watch` is called  
**THEN** HTTP 204 is returned  
**AND** subsequent `GET .../watch` returns HTTP 404

### Requirement: Value Explorer Endpoint
The system SHALL provide a `GET /api/v1/attributes/:key/watch` endpoint that returns collected Deep Watch values with sorting and pagination.

#### Description
The endpoint MUST return all collected value-count pairs for a watched key. It MUST support sorting and pagination. The response MUST include watch metadata.

#### Requirements
1. `GET /api/v1/attributes/:key/watch` MUST return HTTP 200 with the `WatchedAttribute` data
2. Response MUST include `key`, `watching_since`, `unique_count`, `total_observations`, `overflow`, and `values` array
3. Each entry in `values` MUST include `value` (string) and `count` (int64)
4. The endpoint MUST support `sort_by` query param: `count` (default) or `value`
5. The endpoint MUST support `sort_direction` query param: `desc` (default) or `asc`
6. The endpoint MUST support `page` and `page_size` query params (default page_size 100)
7. The endpoint MUST support `q` query param for server-side prefix search on the `value` field
8. `GET` MUST return HTTP 404 if the key is not currently watched
9. The existing `GET /api/v1/attributes/:key` endpoint MUST include a boolean `watched` field indicating whether Deep Watch is active for the key

#### Scenario: Get value explorer data
**GIVEN** `workflow.folder` is watched with 847 unique values  
**WHEN** `GET /api/v1/attributes/workflow.folder/watch` is called  
**THEN** HTTP 200 is returned  
**AND** `unique_count` equals 847  
**AND** `values` is sorted by count descending  
**AND** `values` contains at most 100 entries (default page_size)

#### Scenario: Query filter
**WHEN** `GET /api/v1/attributes/workflow.folder/watch?q=reports` is called  
**THEN** only values containing `reports` as prefix are returned

#### Scenario: Get on unwatched key
**WHEN** `GET /api/v1/attributes/service.name/watch` is called and `service.name` is not watched  
**THEN** HTTP 404 is returned

#### Scenario: watched field in list response
**GIVEN** `workflow.folder` is being watched  
**WHEN** `GET /api/v1/attributes` is called  
**THEN** the entry for `workflow.folder` includes `"watched": true`  
**AND** all other entries include `"watched": false`

