# api Specification Changes

## MODIFIED Requirements

### Requirement: Logs By Service Endpoint
The system SHALL provide an endpoint to list logs grouped by service name for ClickHouse backend.

#### Scenario: List logs by service
- **GIVEN** ClickHouse storage backend is active
- **WHEN** client calls GET /api/v1/logs/by-service
- **THEN** endpoint returns 200 OK (not 501 Not Implemented)
- **AND** response contains array of services with log data
- **AND** each service includes severity breakdown

#### Scenario: Query with limit parameter
- **GIVEN** multiple services have logs
- **WHEN** client calls GET /api/v1/logs/by-service?limit=10
- **THEN** response contains max 10 services
- **AND** services are ordered by sample count (descending)

#### Scenario: Empty result
- **GIVEN** no logs are stored
- **WHEN** client calls GET /api/v1/logs/by-service
- **THEN** response returns 200 OK
- **AND** data array is empty

### Requirement: Metadata Complexity Endpoint Reliability
The system SHALL ensure GetMetadataComplexity endpoint returns valid responses without runtime errors.

#### Scenario: Successful complexity query
- **GIVEN** ClickHouse contains metrics, spans, and logs
- **WHEN** client calls GET /api/v1/cardinality/complexity?threshold=10&limit=50
- **THEN** endpoint returns 200 OK (not 500 Internal Server Error)
- **AND** response contains signals array
- **AND** each signal has signal_type, signal_name, complexity_score, total_keys

#### Scenario: Query with no results
- **GIVEN** no signals exceed threshold
- **WHEN** client calls GET /api/v1/cardinality/complexity?threshold=100
- **THEN** endpoint returns 200 OK
- **AND** signals array is empty
- **AND** no error is returned

#### Scenario: Invalid threshold
- **GIVEN** invalid threshold parameter
- **WHEN** client calls GET /api/v1/cardinality/complexity?threshold=-1
- **THEN** endpoint returns 400 Bad Request
- **AND** error message explains invalid parameter

### Requirement: Consistent Error Responses
The system SHALL return consistent error response format across all API endpoints.

#### Scenario: Error response structure
- **GIVEN** any API endpoint encounters an error
- **WHEN** error response is returned
- **THEN** response has fields: success (false), error (string), metadata (timestamp)
- **AND** HTTP status code matches error category

#### Scenario: 500 error logging
- **GIVEN** internal server error occurs
- **WHEN** 500 response is returned
- **THEN** full error details are logged server-side
- **AND** generic error message is returned to client (no stack traces)
