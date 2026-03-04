# attribute-tracking Specification

## Purpose
TBD - created by archiving change add-attribute-catalog. Update Purpose after archive.
## Requirements
### Requirement: Global Attribute Tracking
The system SHALL track all attribute keys observed across metrics, spans, and logs in a unified catalog.

#### Scenario: Attribute observed in metric
- **WHEN** a metric with label `http.method=GET` is received
- **THEN** the attribute catalog records key `http.method` with value `GET` and signal type `metric`

#### Scenario: Attribute observed in multiple signals
- **WHEN** attribute `user_id` appears in both metrics and spans
- **THEN** the attribute catalog records one entry with signal types `[metric, span]`

#### Scenario: Resource vs data-point attributes
- **WHEN** attribute `service.name` appears in resource attributes
- **THEN** the attribute catalog records scope as `resource`
- **WHEN** same key appears in data-point attributes
- **THEN** the attribute catalog updates scope to `both`

### Requirement: Cardinality Estimation
The system SHALL estimate unique value count for each attribute key using HyperLogLog algorithm.

#### Scenario: Cardinality estimation
- **WHEN** attribute `user_id` has 10,000 unique values
- **THEN** the HyperLogLog sketch estimates cardinality within 2% error (9,800-10,200)

#### Scenario: Memory efficiency
- **WHEN** tracking cardinality for any attribute
- **THEN** the HyperLogLog sketch uses approximately 16KB memory regardless of cardinality

### Requirement: Value Sampling
The system SHALL store up to 10 sample values for each attribute key.

#### Scenario: Sample collection
- **WHEN** attribute `http.status_code` has values `200`, `404`, `500`
- **THEN** the attribute catalog stores all three as samples
- **WHEN** more than 10 unique values are observed
- **THEN** only the first 10 unique values are kept as samples

### Requirement: Signal Type Tracking
The system SHALL track which signal types (metric, span, log) use each attribute key.

#### Scenario: Single signal type
- **WHEN** attribute `metric.aggregation` only appears in metrics
- **THEN** signal types array contains only `[metric]`

#### Scenario: Multiple signal types
- **WHEN** attribute `http.method` appears in metrics, spans, and logs
- **THEN** signal types array contains `[metric, span, log]`

### Requirement: Attribute Scope Tracking
The system SHALL distinguish between resource attributes and data-point attributes.

#### Scenario: Resource attribute only
- **WHEN** attribute `host.name` only appears in resource attributes
- **THEN** scope is `resource`

#### Scenario: Data-point attribute only
- **WHEN** attribute `http.route` only appears in data-point attributes
- **THEN** scope is `attribute`

#### Scenario: Both scopes
- **WHEN** attribute `environment` appears in both resource and data-point attributes
- **THEN** scope is `both`

### Requirement: Observation Counting
The system SHALL count the total number of times each attribute key is observed.

#### Scenario: Count tracking
- **WHEN** attribute `user_id` appears in 10,000 data points
- **THEN** the count field is 10,000

#### Scenario: Count vs cardinality
- **WHEN** attribute `user_id` appears 10,000 times with 1,000 unique values
- **THEN** count is 10,000 and estimated cardinality is approximately 1,000

### Requirement: Timestamp Tracking
The system SHALL record first_seen and last_seen timestamps for each attribute key.

#### Scenario: First observation
- **WHEN** attribute `new_key` is observed for the first time
- **THEN** first_seen and last_seen are set to current timestamp

#### Scenario: Subsequent observations
- **WHEN** attribute is observed again
- **THEN** only last_seen is updated to current timestamp
- **AND** first_seen remains unchanged

### Requirement: Deep Watch Data Model
The system SHALL provide a `WatchedAttribute` data structure that stores all unique values and their occurrence counts for a watched attribute key.

#### Description
When Deep Watch is active for a key, the system MUST collect all distinct values observed for that key after activation. The structure MUST remain bounded by a configurable cap to prevent unbounded memory growth.

#### Requirements
1. `WatchedAttribute` MUST store a map of unique values to their occurrence counts (`map[string]int64`)
2. `WatchedAttribute` MUST record the timestamp when watching was activated (`watching_since`)
3. `WatchedAttribute` MUST track `unique_count` and `total_observations` separately
4. `WatchedAttribute` MUST enforce a maximum unique-value cap (default 10,000 per key)
5. When the cap is reached, `WatchedAttribute` MUST set `overflow = true` and stop accepting new unique values
6. `WatchedAttribute` MUST still increment `total_observations` even when overflowed
7. `WatchedAttribute` MUST be safe for concurrent access (RWMutex)
8. `WatchedAttribute` MUST NOT replace or modify `AttributeMetadata`; the two coexist independently

#### Scenario: Values collected after activation
**GIVEN** Deep Watch is activated for key `workflow.folder` at T0  
**WHEN** values `reports/q1` (312 times) and `exports/daily` (198 times) are observed after T0  
**THEN** `WatchedAttribute.Values["reports/q1"]` equals 312  
**AND** `WatchedAttribute.Values["exports/daily"]` equals 198  
**AND** `WatchedAttribute.UniqueCount` equals 2  
**AND** `WatchedAttribute.TotalObservations` equals 510

#### Scenario: Overflow cap reached
**GIVEN** Deep Watch is active with `MaxValues = 10000`  
**WHEN** 10,001 unique values are observed  
**THEN** `WatchedAttribute.Overflow` equals true  
**AND** the 50,001st unique value is NOT added to the map  
**AND** `TotalObservations` is still incremented for each call

#### Scenario: Data not retroactive
**GIVEN** attribute `workflow.folder` was observed before Deep Watch was activated  
**WHEN** Deep Watch is activated at T0  
**THEN** `WatchedAttribute.Values` contains only values observed at or after T0  
**AND** `WatchingSince` equals T0

### Requirement: Session Serialization of Watch Data
The system SHALL include collected Deep Watch data in session snapshots and restore it as read-only on session load.

#### Description
When a session is saved, the `WatchedAttribute` value maps for all currently watched keys MUST be serialized as part of the session. When a session is loaded, the watch data MUST be restored into `WatchedAttribute` structs with `Active = false`, making the data queryable via the Value Explorer without resuming collection.

#### Requirements
1. Session save MUST serialize all active `WatchedAttribute` entries including `Values`, `UniqueCount`, `TotalObservations`, `Overflow`, and `WatchingSince`
2. Session load MUST deserialize watch data into `WatchedAttribute` structs
3. Restored `WatchedAttribute` entries MUST have watch marked as inactive (no new values collected)
4. The Value Explorer API and UI MUST work identically for restored (inactive) watch data as for active watch data
5. Restored watch data MUST be clearable by the user via the existing deactivate toggle

#### Scenario: Save includes watch data
**GIVEN** `workflow.folder` is watched with 847 unique values  
**WHEN** `POST /api/v1/sessions` is called  
**THEN** the session file contains the value-count map for `workflow.folder`  
**AND** `unique_count` and `total_observations` are preserved

#### Scenario: Load restores values as read-only
**GIVEN** a session was saved with watch data for `workflow.folder`  
**WHEN** `POST /api/v1/sessions/{name}/load` is called  
**THEN** `GetWatchedAttribute(ctx, "workflow.folder")` returns the restored data  
**AND** `StoreAttributeValue` does NOT append new values to `workflow.folder`  
**AND** the Value Explorer displays the restored values

#### Scenario: Reactivate after load
**GIVEN** a session was loaded and `workflow.folder` watch is inactive  
**WHEN** user clicks the watch toggle in the UI  
**THEN** watch becomes active and begins collecting new values  
**AND** existing restored values are retained and new values are added to them

### Requirement: Startup Watch Fields
The system SHALL support activating Deep Watch for one or more attribute keys at startup via configuration.

#### Description
The server MUST accept a `--watch-fields` startup flag as a comma-separated list of attribute keys. Deep Watch SHALL be activated for each listed key during server initialization, before any telemetry is received.

#### Requirements
1. `--watch-fields` MUST accept a comma-separated list of attribute key names
2. Each key in the list MUST be activated via `WatchAttribute` during startup
3. The number of keys MUST NOT exceed the configured maximum watched fields limit (default 10)
4. If `--watch-fields` exceeds the limit, the server MUST log an error and reject startup

#### Scenario: Startup activation
**GIVEN** server is started with `--watch-fields=workflow.folder,service.instance.id`  
**WHEN** server initialization completes  
**THEN** Deep Watch is active for `workflow.folder`  
**AND** Deep Watch is active for `service.instance.id`  
**AND** both fields begin collecting values from the first received telemetry

#### Scenario: Startup limit exceeded
**GIVEN** the max watched fields limit is 10  
**WHEN** server is started with 11 keys in `--watch-fields`  
**THEN** server logs an error and exits with non-zero status

