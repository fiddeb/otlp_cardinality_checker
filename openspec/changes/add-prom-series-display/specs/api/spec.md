# api Specification (Delta)

## ADDED Requirements
### Requirement: Metric Active Series Fields
The system SHALL return both OTLP and Prometheus active series fields for metric responses.

#### Scenario: Metrics list response
- **WHEN** GET /api/v1/metrics is called
- **THEN** each metric includes:
  - `active_series_otlp`
  - `active_series_prometheus`

#### Scenario: Metric detail response
- **WHEN** GET /api/v1/metrics/:name is called
- **THEN** the response includes:
  - `active_series_otlp`
  - `active_series_prometheus`

#### Scenario: Prometheus series estimation for histograms
- **WHEN** a metric is Histogram
- **THEN** `active_series_prometheus = active_series_otlp * (bucket_count + 2)`
- **AND** `bucket_count = explicit_bounds + 1` for classic histograms

#### Scenario: Prometheus series estimation for exponential histograms
- **WHEN** a metric is ExponentialHistogram
- **THEN** `active_series_prometheus = active_series_otlp * (bucket_count + 2)`
- **AND** `bucket_count = scales * 10` as an approximate bucket count per scale

#### Scenario: Prometheus series estimation for non-histograms
- **WHEN** a metric is not Histogram
- **THEN** `active_series_prometheus = active_series_otlp`
