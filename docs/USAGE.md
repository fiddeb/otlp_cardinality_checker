# Usage Guide

A practical guide for using the OTLP Cardinality Checker to analyze your telemetry metadata.

## Quick Start

### 1. Start the Server

```bash
# Build
# Build
make build

# Run
./bin/occ

# Server starts on:
#   OTLP gRPC: localhost:4317
#   OTLP HTTP: http://localhost:4318
#   REST API:  http://localhost:8080
```

### 2. Send Telemetry Data

Point your OpenTelemetry Collector or SDK to the OTLP endpoint:

```yaml
# OpenTelemetry Collector config (gRPC)
exporters:
  otlp/cardinality:
    endpoint: localhost:4317
    tls:
      insecure: true
    
service:
  pipelines:
    metrics:
      exporters: [otlp/cardinality]
    traces:
      exporters: [otlp/cardinality]
    logs:
      exporters: [otlp/cardinality]
```

Or use HTTP protocol:

```yaml
# OpenTelemetry Collector config (HTTP)
exporters:
  otlphttp/cardinality:
    endpoint: http://localhost:4318
    compression: gzip
    
service:
  pipelines:
    metrics:
      exporters: [otlphttp/cardinality]
    traces:
      exporters: [otlphttp/cardinality]
    logs:
      exporters: [otlphttp/cardinality]
```

Or from your application:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
```

### 3. Query Metadata

```bash
# Check if data is arriving
curl http://localhost:8080/api/v1/health

# List all metrics
curl http://localhost:8080/api/v1/metrics?limit=5

# Get details for a specific metric
curl http://localhost:8080/api/v1/metrics/your_metric_name
```

### 4. Quick Test - Check Your Metric

```bash
# Step 1: Find a metric name
curl -s "http://localhost:8080/api/v1/metrics?limit=5" | jq -r '.data[].name'

# Step 2: Check its labels and cardinality
curl -s "http://localhost:8080/api/v1/metrics/YOUR_METRIC_NAME" | \
  jq '.label_keys | to_entries[] | {
    label: .key,
    cardinality: .value.estimated_cardinality,
    sample_values: .value.value_samples[0:3]
  }'
```

**Example output:**
```json
{
  "label": "user_id",
  "cardinality": 20,
  "sample_values": ["user_1", "user_10", "user_11"]
}
{
  "label": "method",
  "cardinality": 1,
  "sample_values": ["GET"]
}
```

**Interpretation:**
- `user_id` has 20 unique values → creates 20+ time series
- `method` has 1 unique value → not contributing to cardinality

---

## Common Use Cases

This section covers real-world scenarios for analyzing **Metrics**, **Traces**, and **Logs**. Each example includes the API endpoint and expected output.

---

## 📊 Working with Metrics

### 🔍 Find Metrics for a Service

**Question:** What metrics does `my-service` produce?

```bash
curl -s "http://localhost:8080/api/v1/metrics?service=my-service" | \
  jq -r '.data[] | .name'
```

**Output:**
```
http_requests_total
http_request_duration_seconds
cache_hits_total
database_queries_total
```

---

### 📊 Check Metric Labels

**Question:** What labels does `http_requests_total` have?

```bash
curl -s "http://localhost:8080/api/v1/metrics/http_requests_total" | \
  jq '.label_keys | keys'
```

**Output:**
```json
[
  "endpoint",
  "method",
  "status"
]
```

**With cardinality info:**

```bash
curl -s "http://localhost:8080/api/v1/metrics/http_requests_total" | \
  jq -r '.label_keys | to_entries[] | "\(.key): \(.value.estimated_cardinality) unique values"'
```

**Output:**
```
endpoint: 45 unique values
method: 4 unique values
status: 8 unique values
```

---

### ⚠️ Identify High Cardinality Labels in Metrics

**Question:** Which metric labels have too many unique values?

```bash
# Find labels with >20 unique values (adjust threshold as needed)
curl -s "http://localhost:8080/api/v1/metrics/http_requests_total" | \
  jq '.label_keys | to_entries[] | select(.value.estimated_cardinality > 20) | {
    label: .key,
    cardinality: .value.estimated_cardinality,
    samples: .value.value_samples[0:5]
  }'
```

**Output:**
```json
{
  "label": "user_id",
  "cardinality": 1247,
  "samples": [
    "user_001",
    "user_002",
    "user_003",
    "user_004",
    "user_005"
  ]
}
```

**Interpretation:** ⚠️ `user_id` has 1247 unique values - this will create 1247+ time series!

---

### 📊 List All Metrics with Sample Counts

**Question:** Which metrics receive the most samples?

```bash
curl -s "http://localhost:8080/api/v1/metrics?limit=100" | \
  jq -r '.data[] | "\(.name): \(.sample_count) samples"' | \
  sort -t: -k2 -nr | head -10
```

**Output:**
```
http_requests_total: 15000 samples
cpu_usage_percent: 12000 samples
memory_bytes: 10000 samples
```

---

### 📈 Check Resource Attributes for Metrics

**Question:** What resource attributes (like `service.name`, `host.name`) are attached to metrics?

```bash
curl -s "http://localhost:8080/api/v1/metrics/http_requests_total" | \
  jq '.resource_keys | to_entries[] | {
    key: .key,
    cardinality: .value.estimated_cardinality,
    samples: .value.value_samples[0:3]
  }'
```

**Output:**
```json
{
  "key": "service.name",
  "cardinality": 3,
  "samples": ["api-server", "worker", "cache"]
}
{
  "key": "host.name",
  "cardinality": 12,
  "samples": ["host-1", "host-2", "host-3"]
}
```

---

## 🔍 Working with Traces (Spans)

### 📋 List All Span Operations

**Question:** What span operations are being traced?

```bash
curl -s "http://localhost:8080/api/v1/spans?limit=100" | \
  jq -r '.data[] | .name'
```

**Output:**
```
HTTP GET /api/users
HTTP POST /api/orders
database_query
cache_lookup
external_api_call
```

---

### 🔍 Get Details for a Specific Span

**Question:** What attributes does the `HTTP GET /api/users` span have?

```bash
curl -s "http://localhost:8080/api/v1/spans/HTTP%20GET%20%2Fapi%2Fusers" | \
  jq '{
    name,
    kind,
    attribute_keys: (.attribute_keys | keys),
    sample_count
  }'
```

**Output:**
```json
{
  "name": "HTTP GET /api/users",
  "kind": "Server",
  "attribute_keys": [
    "http.method",
    "http.route",
    "http.status_code",
    "http.target"
  ],
  "sample_count": 5420
}
```

---

### ⚠️ Identify High Cardinality Span Attributes

**Question:** Which span attributes have too many unique values?

```bash
curl -s "http://localhost:8080/api/v1/spans/HTTP%20GET%20%2Fapi%2Fusers" | \
  jq '.attribute_keys | to_entries[] | select(.value.estimated_cardinality > 50) | {
    attribute: .key,
    cardinality: .value.estimated_cardinality,
    samples: .value.value_samples[0:5]
  }'
```

**Output:**
```json
{
  "attribute": "http.target",
  "cardinality": 342,
  "samples": [
    "/api/users/123",
    "/api/users/456",
    "/api/users/789",
    "/api/users/101112",
    "/api/users/131415"
  ]
}
```

**Interpretation:** ⚠️ `http.target` includes user IDs in the path, creating high cardinality. Consider using `http.route` instead (e.g., `/api/users/:id`).

---

### 🔍 Find Traces by Service

**Question:** What operations does `api-server` trace?

```bash
curl -s "http://localhost:8080/api/v1/spans?service=api-server" | \
  jq -r '.data[] | .name'
```

**Output:**
```
HTTP GET /api/users
HTTP POST /api/orders
HTTP GET /api/products
database_query
```

---

### 📊 Check Span Resource Attributes

**Question:** What resource attributes are attached to spans?

```bash
curl -s "http://localhost:8080/api/v1/spans/database_query" | \
  jq '.resource_keys | to_entries[] | {
    key: .key,
    cardinality: .value.estimated_cardinality,
    samples: .value.value_samples[0:3]
  }'
```

**Output:**
```json
{
  "key": "service.name",
  "cardinality": 2,
  "samples": ["api-server", "worker"]
}
{
  "key": "service.version",
  "cardinality": 3,
  "samples": ["v1.2.3", "v1.2.4", "v1.3.0"]
}
```

---

### 📈 List Spans by Sample Count

**Question:** Which span operations are most frequently traced?

```bash
curl -s "http://localhost:8080/api/v1/spans?limit=100" | \
  jq -r '.data[] | "\(.name): \(.sample_count) samples"' | \
  sort -t: -k2 -nr | head -10
```

**Output:**
```
HTTP GET /api/users: 8500 samples
database_query: 6200 samples
cache_lookup: 4800 samples
```

---

### 🔍 Analyze Span Name Patterns

**Question:** What patterns exist in span names? Are there dynamic values like IDs or timestamps?

```bash
curl -s "http://localhost:8080/api/v1/spans/HTTP%20GET%20%2Fapi%2Fusers" | \
  jq '.name_patterns[] | {
    template,
    count,
    percentage: "\(.percentage)%",
    examples: .examples[0:2]
  }'
```

**Output:**
```json
{
  "template": "HTTP GET <URL>",
  "count": 4500,
  "percentage": "90%",
  "examples": ["HTTP GET /api/users/123", "HTTP GET /api/users/456"]
}
{
  "template": "HTTP GET /api/users",
  "count": 500,
  "percentage": "10%",
  "examples": ["HTTP GET /api/users"]
}
```

**Interpretation:** 90% of spans have dynamic URL paths (user IDs in the path). Consider using `http.route` attribute instead of embedding IDs in span names.

---

### ⚠️ Identify High Cardinality Span Names

**Question:** Which spans have dynamic values in their names causing cardinality issues?

```bash
curl -s "http://localhost:8080/api/v1/spans?limit=100" | \
  jq '.data[] | select(.name_patterns != null and (.name_patterns | length) > 1) | {
    name,
    pattern_count: (.name_patterns | length),
    top_pattern: .name_patterns[0].template,
    examples: .name_patterns[0].examples
  }'
```

**Output:**
```json
{
  "name": "process-batch-123",
  "pattern_count": 5,
  "top_pattern": "process-batch-<NUM>",
  "examples": ["process-batch-1", "process-batch-42", "process-batch-999"]
}
```

**Interpretation:** ⚠️ Span names contain batch IDs. This creates many unique span names which fragments your trace analysis.

---

## 📝 Working with Logs

### 📋 List All Log Severities

**Question:** What log severity levels are being collected?

```bash
curl -s "http://localhost:8080/api/v1/logs" | \
  jq -r '.data[] | .severity'
```

**Output:**
```
INFO
WARN
ERROR
DEBUG
```

---

### 📝 Get Details for a Specific Severity

**Question:** What attributes do ERROR logs have?

```bash
curl -s "http://localhost:8080/api/v1/logs/ERROR" | \
  jq '{
    severity,
    attribute_keys: (.attribute_keys | keys),
    sample_count
  }'
```

**Output:**
```json
{
  "severity": "ERROR",
  "attribute_keys": [
    "error.message",
    "error.type",
    "module",
    "trace_id"
  ],
  "sample_count": 1250
}
```

---

### ⚠️ Identify High Cardinality Log Attributes

**Question:** Which log attributes have too many unique values?

```bash
curl -s "http://localhost:8080/api/v1/logs/ERROR" | \
  jq '.attribute_keys | to_entries[] | select(.value.estimated_cardinality > 30) | {
    attribute: .key,
    cardinality: .value.estimated_cardinality,
    samples: .value.value_samples[0:5]
  }'
```

**Output:**
```json
{
  "attribute": "error.message",
  "cardinality": 487,
  "samples": [
    "Connection timeout to database",
    "Invalid user input: email format",
    "Rate limit exceeded for user 123",
    "Failed to parse JSON response",
    "Null pointer exception in handler"
  ]
}
```

**Interpretation:** ⚠️ `error.message` has 487 unique values. This is expected for error messages, but consider if you need to store all variations.

---

### 📝 Find Logs by Service

**Question:** What log severities does `api-server` produce?

```bash
curl -s "http://localhost:8080/api/v1/logs?service=api-server" | \
  jq -r '.data[] | .severity'
```

**Output:**
```
INFO
WARN
ERROR
```

---

### 📊 Check Log Resource Attributes

**Question:** What resource attributes are attached to logs?

```bash
curl -s "http://localhost:8080/api/v1/logs/ERROR" | \
  jq '.resource_keys | to_entries[] | {
    key: .key,
    cardinality: .value.estimated_cardinality,
    samples: .value.value_samples[0:3]
  }'
```

**Output:**
```json
{
  "key": "service.name",
  "cardinality": 4,
  "samples": ["api-server", "worker", "scheduler", "cache"]
}
{
  "key": "deployment.environment",
  "cardinality": 3,
  "samples": ["production", "staging", "development"]
}
```

---

### 📈 Compare Log Volumes by Severity

**Question:** Which log severities have the most samples?

```bash
curl -s "http://localhost:8080/api/v1/logs?limit=100" | \
  jq -r '.data[] | "\(.severity): \(.sample_count) samples"' | \
  sort -t: -k2 -nr
```

**Output:**
```
INFO: 45000 samples
WARN: 5200 samples
ERROR: 1250 samples
DEBUG: 800 samples
```

---

## 🔄 Cross-Signal Analysis

### 📊 Compare All Signal Types for a Service

**Question:** What telemetry does `api-server` produce across all signal types?

```bash
# Get overview for a service
curl -s "http://localhost:8080/api/v1/services/api-server/overview" | \
  jq '{
    metrics: [.metrics[] | .name],
    spans: [.spans[] | .name],
    logs: [.logs[] | .severity]
  }'
```

**Output:**
```json
{
  "metrics": [
    "http_requests_total",
    "http_request_duration_seconds",
    "cache_hits_total"
  ],
  "spans": [
    "HTTP GET /api/users",
    "HTTP POST /api/orders",
    "database_query"
  ],
  "logs": [
    "INFO",
    "WARN",
    "ERROR"
  ]
}
```

---

### 📈 Find Services with High Cardinality Across All Signals

**Question:** Which services have cardinality issues in any signal type?

```bash
# Use the noisy neighbor detection script
./scripts/find-noisy-neighbors.sh
```

**Output:**
```
═══════════════════════════════════════════════════════════════
1️⃣  Services by Total Sample Volume
═══════════════════════════════════════════════════════════════
  📊 api-server:
     Total: 15000 samples
     Metrics: 8000 | Traces: 5000 | Logs: 2000
     Signal types: metrics, traces, logs
```

---

## 📉 Optimization Use Cases

### 📉 Find Underutilized Labels (Metrics)

### 📉 Find Underutilized Labels (Metrics)

**Question:** Which metric labels are rarely used?

```bash
# Find labels present in <50% of samples
curl -s "http://localhost:8080/api/v1/metrics/http_requests_total" | \
  jq '.label_keys | to_entries[] | select(.value.percentage < 50) | {
    label: .key,
    usage: "\(.value.percentage)%",
    count: .value.count
  }'
```

**Output:**
```json
{
  "label": "cache_key",
  "usage": "23.5%",
  "count": 235
}
```

**Interpretation:** `cache_key` only appears in 23.5% of samples - maybe it's optional or conditionally added.

---

### 📉 Find Underutilized Attributes (Spans)

**Question:** Which span attributes are rarely used?

```bash
curl -s "http://localhost:8080/api/v1/spans/HTTP%20GET%20%2Fapi%2Fusers" | \
  jq '.attribute_keys | to_entries[] | select(.value.percentage < 50) | {
    attribute: .key,
    usage: "\(.value.percentage)%"
  }'
```

**Output:**
```json
{
  "attribute": "http.user_agent",
  "usage": "15.2%"
}
```

---

### 📉 Find Underutilized Attributes (Logs)

**Question:** Which log attributes are rarely populated?

```bash
curl -s "http://localhost:8080/api/v1/logs/ERROR" | \
  jq '.attribute_keys | to_entries[] | select(.value.percentage < 50) | {
    attribute: .key,
    usage: "\(.value.percentage)%"
  }'
```

---

### 🔄 Compare Services

**Question:** How does telemetry differ between services?

```bash
# Get overview for each service
for service in service-a service-b; do
  echo "=== $service ==="
  curl -s "http://localhost:8080/api/v1/services/$service/overview" | \
    jq '{
      metrics: [.metrics[] | .name],
      spans: [.spans[] | .name],
      logs: [.logs[] | .severity]
    }'
done
```

---

### 📈 Monitor Cardinality Growth (Metrics)

**Question:** Is metric cardinality increasing over time?

```bash
# Take snapshot
curl -s "http://localhost:8080/api/v1/metrics/http_requests_total" | \
  jq '.label_keys.user_id.estimated_cardinality' > snapshot1.txt

# Wait some time...
sleep 3600

# Compare
curl -s "http://localhost:8080/api/v1/metrics/http_requests_total" | \
  jq '.label_keys.user_id.estimated_cardinality' > snapshot2.txt

# Show growth
echo "Growth: $(($(cat snapshot2.txt) - $(cat snapshot1.txt))) new unique values"
```

---

### 📈 Monitor Cardinality Growth (Spans)

**Question:** Is span attribute cardinality increasing?

```bash
# Initial snapshot
curl -s "http://localhost:8080/api/v1/spans/HTTP%20GET%20%2Fapi%2Fusers" | \
  jq '.attribute_keys."http.target".estimated_cardinality' > span_snapshot1.txt

# Wait and compare
sleep 3600
curl -s "http://localhost:8080/api/v1/spans/HTTP%20GET%20%2Fapi%2Fusers" | \
  jq '.attribute_keys."http.target".estimated_cardinality' > span_snapshot2.txt

echo "Growth: $(($(cat span_snapshot2.txt) - $(cat span_snapshot1.txt))) new unique values"
```

---

## API Reference

### Health Check

```bash
GET /api/v1/health
```

**Example:**
```bash
curl "http://localhost:8080/api/v1/health"
```

**Response:**
```json
{
  "status": "ok",
  "timestamp": "2025-10-24T12:00:00Z",
  "version": "1.0.0",
  "uptime": "2h15m30s",
  "memory": {
    "alloc_mb": 45,
    "total_alloc_mb": 120,
    "sys_mb": 67,
    "num_gc": 15
  }
}
```

---

### Metrics API

#### List All Metrics

```bash
GET /api/v1/metrics?service={name}&limit={N}&offset={M}
```

**Parameters:**
- `service` (optional): Filter by service name
- `limit` (optional): Items per page (default: 100, max: 10000)
- `offset` (optional): Skip N items (default: 0)

**Example:**
```bash
curl "http://localhost:8080/api/v1/metrics?limit=10&offset=0"
```

**Response:**
```json
{
  "data": [
    {
      "name": "http_requests_total",
      "type": "Sum",
      "sample_count": 1000,
      "services": {
        "api-server": 800,
        "proxy": 200
      }
    }
  ],
  "total": 1250,
  "limit": 10,
  "offset": 0,
  "has_more": true
}
```

#### Get Specific Metric

```bash
GET /api/v1/metrics/{name}
```

**Example:**
```bash
curl "http://localhost:8080/api/v1/metrics/http_requests_total"
```

**Response:**
```json
{
  "name": "http_requests_total",
  "type": "Sum",
  "label_keys": {
    "method": {
      "count": 1000,
      "percentage": 100,
      "estimated_cardinality": 4,
      "value_samples": ["GET", "POST", "PUT", "DELETE"]
    },
    "endpoint": {
      "count": 1000,
      "percentage": 100,
      "estimated_cardinality": 45,
      "value_samples": ["/api/users", "/api/orders", "/api/products"]
    }
  },
  "resource_keys": {
    "service.name": {
      "count": 1000,
      "percentage": 100,
      "estimated_cardinality": 3,
      "value_samples": ["api-server", "worker", "cache"]
    }
  },
  "scope_info": {
    "name": "myapp-instrumentation",
    "version": "1.0.0"
  },
  "sample_count": 1000,
  "services": {
    "api-server": 800,
    "proxy": 200
  }
}
```

---

### Spans (Traces) API

#### List All Spans

```bash
GET /api/v1/spans?service={name}&limit={N}&offset={M}
```

**Parameters:**
- `service` (optional): Filter by service name
- `limit` (optional): Items per page (default: 100, max: 10000)
- `offset` (optional): Skip N items (default: 0)

**Example:**
```bash
curl "http://localhost:8080/api/v1/spans?service=api-server&limit=10"
```

**Response:**
```json
{
  "data": [
    {
      "name": "HTTP GET /api/users",
      "kind": "Server",
      "sample_count": 5420,
      "services": {
        "api-server": 5420
      }
    }
  ],
  "total": 50,
  "limit": 10,
  "offset": 0,
  "has_more": true
}
```

#### Get Specific Span

```bash
GET /api/v1/spans/{name}
```

**Example:**
```bash
curl "http://localhost:8080/api/v1/spans/HTTP%20GET%20%2Fapi%2Fusers"
```

**Response:**
```json
{
  "name": "HTTP GET /api/users",
  "kind": "Server",
  "attribute_keys": {
    "http.method": {
      "count": 5420,
      "percentage": 100,
      "estimated_cardinality": 1,
      "value_samples": ["GET"]
    },
    "http.route": {
      "count": 5420,
      "percentage": 100,
      "estimated_cardinality": 1,
      "value_samples": ["/api/users"]
    },
    "http.status_code": {
      "count": 5420,
      "percentage": 100,
      "estimated_cardinality": 5,
      "value_samples": ["200", "400", "404", "500", "503"]
    }
  },
  "resource_keys": {
    "service.name": {
      "count": 5420,
      "percentage": 100,
      "estimated_cardinality": 1,
      "value_samples": ["api-server"]
    }
  },
  "scope_info": {
    "name": "myapp-tracer",
    "version": "1.0.0"
  },
  "sample_count": 5420,
  "services": {
    "api-server": 5420
  }
}
```

---

### Logs API

#### List All Log Severities

```bash
GET /api/v1/logs?service={name}&limit={N}&offset={M}
```

**Parameters:**
- `service` (optional): Filter by service name
- `limit` (optional): Items per page (default: 100, max: 10000)
- `offset` (optional): Skip N items (default: 0)

**Example:**
```bash
curl "http://localhost:8080/api/v1/logs?service=api-server"
```

**Response:**
```json
{
  "data": [
    {
      "severity": "INFO",
      "sample_count": 45000,
      "services": {
        "api-server": 35000,
        "worker": 10000
      }
    },
    {
      "severity": "ERROR",
      "sample_count": 1250,
      "services": {
        "api-server": 800,
        "worker": 450
      }
    }
  ],
  "total": 4,
  "limit": 100,
  "offset": 0,
  "has_more": false
}
```

#### Get Specific Log Severity

```bash
GET /api/v1/logs/{severity}
```

**Example:**
```bash
curl "http://localhost:8080/api/v1/logs/ERROR"
```

**Response:**
```json
{
  "severity": "ERROR",
  "attribute_keys": {
    "error.type": {
      "count": 1250,
      "percentage": 100,
      "estimated_cardinality": 15,
      "value_samples": ["NullPointerException", "TimeoutException", "ValidationError"]
    },
    "module": {
      "count": 1250,
      "percentage": 100,
      "estimated_cardinality": 8,
      "value_samples": ["database", "cache", "api"]
    },
    "trace_id": {
      "count": 1100,
      "percentage": 88,
      "estimated_cardinality": 1050,
      "value_samples": ["abc123...", "def456...", "ghi789..."]
    }
  },
  "resource_keys": {
    "service.name": {
      "count": 1250,
      "percentage": 100,
      "estimated_cardinality": 4,
      "value_samples": ["api-server", "worker", "scheduler", "cache"]
    }
  },
  "scope_info": {
    "name": "myapp-logger",
    "version": "1.0.0"
  },
  "sample_count": 1250,
  "services": {
    "api-server": 800,
    "worker": 450
  }
}
```

---

### Services API

#### List All Services

```bash
GET /api/v1/services
```

**Example:**
```bash
curl "http://localhost:8080/api/v1/services"
```

**Response:**
```json
{
  "data": [
    "api-server",
    "worker",
    "cache",
    "scheduler"
  ],
  "total": 4
}
```

#### Get Service Overview

```bash
GET /api/v1/services/{name}/overview
```

**Example:**
```bash
curl "http://localhost:8080/api/v1/services/api-server/overview"
```

**Response:**
```json
{
  "service_name": "api-server",
  "metrics": [
    {
      "name": "http_requests_total",
      "type": "Sum",
      "sample_count": 800
    },
    {
      "name": "http_request_duration_seconds",
      "type": "Histogram",
      "sample_count": 800
    }
  ],
  "spans": [
    {
      "name": "HTTP GET /api/users",
      "kind": "Server",
      "sample_count": 5420
    },
    {
      "name": "database_query",
      "kind": "Client",
      "sample_count": 3200
    }
  ],
  "logs": [
    {
      "severity": "INFO",
      "sample_count": 35000
    },
    {
      "severity": "ERROR",
      "sample_count": 800
    }
  ]
}
```

---

## Useful jq Patterns

### Extract Specific Fields

**Metrics:**
```bash
# Just metric names
curl -s "http://localhost:8080/api/v1/metrics" | jq -r '.data[] | .name'

# Metric names with sample counts
curl -s "http://localhost:8080/api/v1/metrics" | \
  jq -r '.data[] | "\(.name): \(.sample_count) samples"'

# Metrics grouped by type
curl -s "http://localhost:8080/api/v1/metrics" | \
  jq '[.data[] | {type, name}] | group_by(.type) | map({type: .[0].type, count: length, metrics: [.[].name]})'
```

**Spans:**
```bash
# Just span names
curl -s "http://localhost:8080/api/v1/spans" | jq -r '.data[] | .name'

# Spans with their kinds
curl -s "http://localhost:8080/api/v1/spans" | \
  jq -r '.data[] | "\(.name) (\(.kind))"'

# Spans grouped by kind
curl -s "http://localhost:8080/api/v1/spans" | \
  jq '[.data[] | {kind, name}] | group_by(.kind) | map({kind: .[0].kind, count: length})'
```

**Logs:**
```bash
# Just severities
curl -s "http://localhost:8080/api/v1/logs" | jq -r '.data[] | .severity'

# Severities with sample counts
curl -s "http://localhost:8080/api/v1/logs" | \
  jq -r '.data[] | "\(.severity): \(.sample_count) samples"'
```

---

### Filter and Sort

**Metrics:**
```bash
# Metrics with >1000 samples
curl -s "http://localhost:8080/api/v1/metrics" | \
  jq '.data[] | select(.sample_count > 1000)'

# Top 10 metrics by sample count
curl -s "http://localhost:8080/api/v1/metrics?limit=1000" | \
  jq '.data | sort_by(.sample_count) | reverse | .[0:10] | .[] | {name, sample_count}'

# Metrics with high cardinality labels
curl -s "http://localhost:8080/api/v1/metrics?limit=1000" | \
  jq '.data[] | select(.label_keys | to_entries[] | .value.estimated_cardinality > 50) | .name'
```

**Spans:**
```bash
# Spans with >5000 samples
curl -s "http://localhost:8080/api/v1/spans" | \
  jq '.data[] | select(.sample_count > 5000)'

# Top 10 spans by sample count
curl -s "http://localhost:8080/api/v1/spans?limit=1000" | \
  jq '.data | sort_by(.sample_count) | reverse | .[0:10] | .[] | {name, kind, sample_count}'

# Spans with high cardinality attributes
curl -s "http://localhost:8080/api/v1/spans?limit=1000" | \
  jq '.data[] | select(.attribute_keys | to_entries[] | .value.estimated_cardinality > 100) | .name'
```

**Logs:**
```bash
# Logs with >10000 samples
curl -s "http://localhost:8080/api/v1/logs" | \
  jq '.data[] | select(.sample_count > 10000)'

# Severities sorted by volume
curl -s "http://localhost:8080/api/v1/logs" | \
  jq '.data | sort_by(.sample_count) | reverse | .[] | {severity, sample_count}'
```

---

### Create Reports

**Metrics Cardinality Report:**
```bash
curl -s "http://localhost:8080/api/v1/metrics?limit=1000" | \
  jq -r '.data[] | .name as $metric | .label_keys | to_entries[] | 
    select(.value.estimated_cardinality > 20) | 
    "\($metric).\(.key): \(.value.estimated_cardinality)"' | \
  sort -t: -k2 -nr
```

**Output:**
```
http_requests.user_id: 1247
api_calls.session_id: 982
database_queries.query_hash: 456
cache_operations.cache_key: 234
```

**Spans Cardinality Report:**
```bash
curl -s "http://localhost:8080/api/v1/spans?limit=1000" | \
  jq -r '.data[] | .name as $span | .attribute_keys | to_entries[] | 
    select(.value.estimated_cardinality > 50) | 
    "\($span).\(.key): \(.value.estimated_cardinality)"' | \
  sort -t: -k2 -nr
```

**Output:**
```
HTTP GET /api/users.http.target: 342
database_query.query_text: 156
external_api_call.url: 89
```

**Logs Cardinality Report:**
```bash
curl -s "http://localhost:8080/api/v1/logs?limit=1000" | \
  jq -r '.data[] | .severity as $sev | .attribute_keys | to_entries[] | 
    select(.value.estimated_cardinality > 30) | 
    "\($sev).\(.key): \(.value.estimated_cardinality)"' | \
  sort -t: -k2 -nr
```

**Output:**
```
ERROR.error.message: 487
WARN.module: 45
INFO.trace_id: 10523
```

---

### Cross-Signal Analysis

**Services with all three signal types:**
```bash
curl -s "http://localhost:8080/api/v1/services" | jq -r '.data[]' | while read service; do
  metrics=$(curl -s "http://localhost:8080/api/v1/metrics?service=$service" | jq -r '.total')
  spans=$(curl -s "http://localhost:8080/api/v1/spans?service=$service" | jq -r '.total')
  logs=$(curl -s "http://localhost:8080/api/v1/logs?service=$service" | jq -r '.total')
  echo "$service: metrics=$metrics spans=$spans logs=$logs"
done
```

**Output:**
```
api-server: metrics=45 spans=12 logs=3
worker: metrics=23 spans=8 logs=3
cache: metrics=15 spans=5 logs=2
```

---

## Troubleshooting

### No Data Showing Up

**Check OTLP endpoint:**
```bash
# Verify server is running
curl http://localhost:8080/health

# Check OTLP endpoint
curl -X POST http://localhost:4318/v1/metrics \
  -H "Content-Type: application/json" \
  -d '{"resource_metrics":[]}'
```

**Verify Collector config:**
```yaml
exporters:
  otlp:
    endpoint: http://localhost:4318
    # NOT localhost:4317 (that's gRPC, not yet implemented)
```

### High Memory Usage

**Check metrics count:**
```bash
curl -s "http://localhost:8080/api/v1/metrics" | jq '.total'
```

**Expected memory:**
- 1,000 metrics: ~30 MB
- 10,000 metrics: ~150 MB
- 50,000 metrics: ~420 MB

**If higher than expected:**
- Check for high cardinality labels (>100 unique values)
- Check if many metrics have 4+ labels
- Consider cleaning up unused metrics

### Slow API Responses

**Use pagination:**
```bash
# BAD: Returns all 50,000 metrics
curl "http://localhost:8080/api/v1/metrics"

# GOOD: Returns 100 at a time
curl "http://localhost:8080/api/v1/metrics?limit=100"
```

**Filter by service:**
```bash
# Instead of getting all and filtering client-side
curl "http://localhost:8080/api/v1/metrics?service=my-service"
```

---

## Integration Examples

### CI/CD Pipeline Check

```bash
#!/bin/bash
# Check for high cardinality before deploying

MAX_CARDINALITY=100

high_card=$(curl -s "http://localhost:8080/api/v1/metrics?limit=1000" | \
  jq "[.data[] | .label_keys | to_entries[] | select(.value.estimated_cardinality > $MAX_CARDINALITY)] | length")

if [ "$high_card" -gt 0 ]; then
  echo "❌ Found $high_card labels with cardinality > $MAX_CARDINALITY"
  exit 1
else
  echo "✅ All labels within cardinality limits"
fi
```

### Prometheus Alert

```yaml
# Alert on high cardinality (if you export metrics from this tool)
- alert: HighCardinalityDetected
  expr: otlp_label_cardinality > 1000
  annotations:
    summary: "High cardinality detected in {{ $labels.metric }}.{{ $labels.label }}"
```

### Grafana Dashboard Query

```bash
# Export data for Grafana
curl -s "http://localhost:8080/api/v1/metrics?service=my-service" | \
  jq '.data[] | {
    metric: .name,
    cardinality: ([.label_keys[] | .estimated_cardinality] | add),
    samples: .sample_count
  }'
```

---

## Best Practices

### 1. Regular Monitoring

Run cardinality checks daily:
```bash
# Save to file with timestamp
curl -s "http://localhost:8080/api/v1/metrics?limit=10000" > \
  "metrics_$(date +%Y%m%d).json"

# Compare with previous day
jq '.data[].name' metrics_20251023.json > today.txt
jq '.data[].name' metrics_20251022.json > yesterday.txt
diff yesterday.txt today.txt
```

### 2. Set Cardinality Budgets

Define limits per team/service:
- **Low cardinality** (<10): status codes, methods, regions
- **Medium cardinality** (10-100): endpoints, services, hosts  
- **High cardinality** (>100): ⚠️ Requires approval (user IDs, request IDs)

### 3. Use Service Filtering

Always filter by service to reduce noise:
```bash
curl "http://localhost:8080/api/v1/metrics?service=my-service&limit=100"
```

### 4. Document Your Metrics

When you find a metric with high cardinality:
1. Check if it's intentional (e.g., `http.target` for URLs)
2. Consider if it can be reduced (e.g., group by endpoint pattern)
3. Document the decision

---

## Advanced Queries

### Find Metrics Without a Specific Label

```bash
# Find metrics missing 'service.name' resource attribute
curl -s "http://localhost:8080/api/v1/metrics?limit=1000" | \
  jq -r '.data[] | select(.resource_keys["service.name"] == null or .resource_keys["service.name"].count == 0) | .name'
```

---

### Find Spans Without Expected Attributes

```bash
# Find spans missing 'http.status_code' attribute
curl -s "http://localhost:8080/api/v1/spans?limit=1000" | \
  jq -r '.data[] | select(.attribute_keys["http.status_code"] == null) | .name'
```

---

### Find Logs Without Trace Context

```bash
# Find log severities where trace_id is rarely present
curl -s "http://localhost:8080/api/v1/logs?limit=1000" | \
  jq -r '.data[] | select(.attribute_keys.trace_id.percentage < 10) | .severity'
```

---

### Calculate Total Cardinality

**For Metrics:**
```bash
# Total cardinality across all labels for a metric (multiplicative)
curl -s "http://localhost:8080/api/v1/metrics/http_requests_total" | \
  jq '[.label_keys[] | .estimated_cardinality] | reduce .[] as $item (1; . * $item)'
```

**Output:** `1440` (e.g., 45 endpoints × 4 methods × 8 statuses)

**For Spans:**
```bash
# Total cardinality across all attributes for a span
curl -s "http://localhost:8080/api/v1/spans/HTTP%20GET%20%2Fapi%2Fusers" | \
  jq '[.attribute_keys[] | .estimated_cardinality] | reduce .[] as $item (1; . * $item)'
```

---

### Export to CSV

**Metrics to CSV:**
```bash
echo "metric,label,cardinality,percentage" > metrics_cardinality.csv
curl -s "http://localhost:8080/api/v1/metrics?limit=10000" | \
  jq -r '.data[] | .name as $m | .label_keys | to_entries[] | 
    "\($m),\(.key),\(.value.estimated_cardinality),\(.value.percentage)"' \
  >> metrics_cardinality.csv
```

**Spans to CSV:**
```bash
echo "span,attribute,cardinality,percentage" > spans_cardinality.csv
curl -s "http://localhost:8080/api/v1/spans?limit=10000" | \
  jq -r '.data[] | .name as $s | .attribute_keys | to_entries[] | 
    "\($s),\(.key),\(.value.estimated_cardinality),\(.value.percentage)"' \
  >> spans_cardinality.csv
```

**Logs to CSV:**
```bash
echo "severity,attribute,cardinality,percentage" > logs_cardinality.csv
curl -s "http://localhost:8080/api/v1/logs?limit=10000" | \
  jq -r '.data[] | .severity as $s | .attribute_keys | to_entries[] | 
    "\($s),\(.key),\(.value.estimated_cardinality),\(.value.percentage)"' \
  >> logs_cardinality.csv
```

---

### Compare Signal Types by Cardinality

```bash
# Get highest cardinality attribute from each signal type
echo "=== Highest Cardinality Metrics ==="
curl -s "http://localhost:8080/api/v1/metrics?limit=1000" | \
  jq -r '[.data[] | .name as $m | .label_keys | to_entries[] | {metric: $m, label: .key, card: .value.estimated_cardinality}] | 
    sort_by(.card) | reverse | .[0:5] | .[] | "\(.metric).\(.label): \(.card)"'

echo ""
echo "=== Highest Cardinality Spans ==="
curl -s "http://localhost:8080/api/v1/spans?limit=1000" | \
  jq -r '[.data[] | .name as $s | .attribute_keys | to_entries[] | {span: $s, attr: .key, card: .value.estimated_cardinality}] | 
    sort_by(.card) | reverse | .[0:5] | .[] | "\(.span).\(.attr): \(.card)"'

echo ""
echo "=== Highest Cardinality Logs ==="
curl -s "http://localhost:8080/api/v1/logs?limit=1000" | \
  jq -r '[.data[] | .severity as $s | .attribute_keys | to_entries[] | {severity: $s, attr: .key, card: .value.estimated_cardinality}] | 
    sort_by(.card) | reverse | .[0:5] | .[] | "\(.severity).\(.attr): \(.card)"'
```

---

## Load Testing

Quick test with all signal types:
```bash
./scripts/run-all-tests.sh quick
```

For comprehensive K6 load testing examples, see [scripts/README.md](scripts/README.md).

---

## Next Steps

- **Production Deployment**: See [../k8s/README.md](../k8s/README.md) for Kubernetes deployment guide
- **API Documentation**: See [API.md](API.md)
- **Scalability**: See [SCALABILITY.md](SCALABILITY.md)
