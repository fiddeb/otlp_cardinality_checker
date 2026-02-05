# ui Specification (Delta)

## ADDED Requirements
### Requirement: Metric Details Active Series Breakdown
The system SHALL display both OTLP active series and Prometheus active series in the metric details view.

#### Scenario: Histogram metric details
- **WHEN** a user views metric details for a Histogram metric
- **THEN** the UI shows OTLP active series
- **AND** the UI shows Prometheus active series
- **AND** the UI explains that Prometheus series includes bucket series plus _sum and _count per label combination

#### Scenario: Non-histogram metric details
- **WHEN** a user views metric details for a non-histogram metric
- **THEN** the UI shows OTLP active series
- **AND** the UI shows Prometheus active series
- **AND** Prometheus active series equals OTLP active series
