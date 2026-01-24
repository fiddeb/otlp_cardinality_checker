## ADDED Requirements

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
