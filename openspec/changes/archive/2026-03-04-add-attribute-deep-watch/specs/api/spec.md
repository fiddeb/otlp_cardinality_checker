# Spec Delta: api

## ADDED Requirements

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
