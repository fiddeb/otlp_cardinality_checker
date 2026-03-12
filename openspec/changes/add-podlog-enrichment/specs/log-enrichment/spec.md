# log-enrichment Specification (Delta)

## ADDED Requirements

### Requirement: Pod Log Service Name Discovery

When pod log enrichment is enabled, the system SHALL resolve a service name from an ordered list of resource attribute keys when `service.name` is absent.

**ID:** `log-enrichment-001`
**Priority:** MUST
**Rationale:** Pod logs from `filelog/podlogs` carry Kubernetes resource attributes but no `service.name`. Without enrichment, every record falls into `"unknown"`, making per-service analysis impossible.

#### Scenario: Service name resolved from k8s.container.name
- **GIVEN** pod log enrichment is enabled
- **AND** a log export has resource attribute `k8s.container.name=nova-powerplay-app` but no `service.name`
- **WHEN** the log record is processed
- **THEN** the service name SHALL be `nova-powerplay-app`

#### Scenario: Priority list is respected
- **GIVEN** pod log enrichment is enabled
- **AND** resource attributes contain both `app=frontend` and `k8s.container.name=frontend-container`
- **WHEN** the default label priority list places `app` before `k8s.container.name`
- **THEN** the service name SHALL be `frontend`

#### Scenario: No matching label falls back to unknown_service
- **GIVEN** pod log enrichment is enabled
- **AND** no resource attribute matches any label in the priority list
- **WHEN** the log record is processed
- **THEN** the service name SHALL be `"unknown_service"`

#### Scenario: Enrichment disabled preserves existing behaviour
- **GIVEN** pod log enrichment is disabled (default)
- **AND** a log export has no `service.name` resource attribute
- **WHEN** the log record is processed
- **THEN** the service name SHALL be `"unknown"` as before

### Requirement: Configurable Service Label Priority List

When pod log enrichment is enabled, the system SHALL use a configurable ordered list of resource attribute keys for service name discovery.

**ID:** `log-enrichment-002`
**Priority:** MUST
**Rationale:** Different Kubernetes environments use different labelling conventions. A fixed list would not cover all deployments.

#### Scenario: Default priority list is used when no override is set
- **GIVEN** pod log enrichment is enabled
- **AND** `POD_LOG_SERVICE_LABELS` env var is not set
- **WHEN** the system starts
- **THEN** service name discovery SHALL use the default ordered label list including `service_name`, `app`, `k8s.container.name`, and others

#### Scenario: Custom priority list overrides the default
- **GIVEN** `POD_LOG_SERVICE_LABELS=my_app,service` is set
- **WHEN** the system starts and processes a log with resource attribute `my_app=payments`
- **THEN** service name SHALL be `payments`

### Requirement: Severity Inference from Log Body

When pod log enrichment is enabled, the system SHALL infer severity from the log body text when `SeverityText` is empty and `SeverityNumber` is 0.

**ID:** `log-enrichment-003`
**Priority:** MUST
**Rationale:** Many application runtimes (Node.js, Python stdlib, etc.) embed level keywords in the log body but do not populate the OTLP `severity_text` field. Without inference, all such records collapse into `UNSET`.

#### Scenario: ERROR keyword in body
- **GIVEN** pod log enrichment is enabled
- **AND** a log record has empty `SeverityText`, `SeverityNumber` 0, and body `"[RedisCacheHandler] Redis connection error: ECONNREFUSED"`
- **WHEN** the record is processed
- **THEN** the inferred severity SHALL be `"ERROR"`

#### Scenario: WARN keyword in body
- **GIVEN** pod log enrichment is enabled
- **AND** a log record has empty `SeverityText` and body containing `"warn"`
- **WHEN** the record is processed
- **THEN** the inferred severity SHALL be `"WARN"`

#### Scenario: Case-insensitive matching
- **GIVEN** pod log enrichment is enabled
- **AND** a log record body contains `"Warning: disk usage high"`
- **WHEN** the record is processed
- **THEN** the inferred severity SHALL be `"WARN"`

#### Scenario: No keyword match
- **GIVEN** pod log enrichment is enabled
- **AND** a log record has empty `SeverityText` and a body with no recognised level keyword
- **WHEN** the record is processed
- **THEN** the severity SHALL remain `"UNSET"`

#### Scenario: Existing severity is not overridden
- **GIVEN** pod log enrichment is enabled
- **AND** a log record already has `SeverityText=INFO`
- **WHEN** the record is processed
- **THEN** the severity SHALL remain `"INFO"` and body inference SHALL NOT run

### Requirement: Opt-In Feature Flag

The system SHALL expose pod log enrichment as an opt-in capability controlled by the `POD_LOG_ENRICHMENT` environment variable.

**ID:** `log-enrichment-004`
**Priority:** MUST
**Rationale:** Enrichment applies heuristics that may not be appropriate for all log sources. Existing deployments with structured logs MUST NOT be affected by default.

#### Scenario: Enrichment disabled by default
- **GIVEN** `POD_LOG_ENRICHMENT` env var is not set
- **WHEN** the server starts
- **THEN** pod log enrichment SHALL be inactive
- **AND** log processing behaviour SHALL be identical to the pre-feature state

#### Scenario: Enrichment enabled explicitly
- **GIVEN** `POD_LOG_ENRICHMENT=true` is set
- **WHEN** the server starts
- **THEN** pod log enrichment SHALL be active for all incoming log records
