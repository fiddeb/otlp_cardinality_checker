## ADDED Requirements

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
