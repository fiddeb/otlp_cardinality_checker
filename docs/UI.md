# User Interface

This page showcases the OTLP Cardinality Checker web interface.

## Dashboard

![Dashboard](img/dashboard.png)

Main dashboard overview showing cardinality analysis across all signal types.

## Sessions Management

![Sessions List](img/Sessions.png)

View and manage analysis sessions.

## Metrics Analysis

![Metrics Overview](img/metric_overview.png)

High-level overview of metrics cardinality patterns.

![Metrics Details](img/metric_detail.png)

Detailed metrics cardinality analysis with breakdowns by attribute keys.

![Single Metric View](img/metric_deep.png)

Deep dive into a specific metric's cardinality.

![Active Series](img/active_series.png)

Active time series monitoring and cardinality tracking.

## Logs Analysis

![Logs](img/logs.png)

Log cardinality patterns and attribute key analysis.

![Log Patterns](img/log_patterns.png)

Detected patterns in log attribute cardinality.

## Traces Analysis

![Traces](img/traces.png)

Trace cardinality overview and span attribute analysis.

![Trace Patterns](img/trace_patterns.png)

Pattern detection in trace attribute cardinality.

## Advanced Analysis

![Attributes](img/attributes.png)

Cross-signal attribute key cardinality analysis.

![Metadata Complexity](img/metadata_complexity.png)

Metadata complexity visualization showing attribute key combinations.

![Noicy Neigbour](img/noicy_neigbour.png)

Shows problematic signals

## Memory

![Memory Usage](img/memory_usage.png)

Runtime memory usage statistics — heap size, goroutine counts, and per-signal storage breakdown.

## Attribute Deep Watch

![Deep Watch](img/deep_watch.png)

Captures every distinct value seen for a watched attribute key. Activate from the Attributes view or via the API (`POST /api/v1/attributes/{key}/watch`).

## Load Testing

![K6 Load Test](img/k6_load.png)

Cardinality behavior under load testing with k6.

## Binary

![occ](img/binary.png)

When starting the binary, API addresses, feature flags, and version info are shown.