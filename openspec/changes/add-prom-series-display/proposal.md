# Change: Add OTLP vs Prometheus active series display

## Why
Users need to compare OTLP active series (label combinations) with an estimated Prometheus series count for the same metric to understand downstream impact, especially for histograms where buckets expand series count.

## What Changes
- Add Prometheus-active-series estimation to metric responses alongside the existing OTLP active series.
- Display both values in the Metric Details view (detailed metrics cardinality analysis).
- Document the Prometheus series estimation rules (including histogram buckets and _sum/_count).

## Impact
- Affected specs: ui, api
- Affected code: metric models, metrics API response shaping, Metric Details UI
