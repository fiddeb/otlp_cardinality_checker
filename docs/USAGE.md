# Usage Guide

A practical guide for using the OTLP Cardinality Checker to analyze your telemetry metadata.

## Quick Start

### 1. Start the Server

```bash
# Build
go build -o otlp-cardinality-checker ./cmd/server

# Run
./otlp-cardinality-checker

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

### ⚠️ Identify High Cardinality Labels

**Question:** Which labels have too many unique values?

```bash
# Find labels with >20 unique values (adjust threshold as needed)
curl -s "http://localhost:8080/api/v1/metrics/http_requests_total" | \
  jq '.label_keys | to_entries[] | select(.value.estimated_cardinality > 20) | {
    label: .key,
    cardinality: .value.estimated_cardinality,
    samples: .value.value_samples[0:5]
  }'
```

**Note:** Adjust the threshold (20, 50, 100) based on your needs. Higher cardinality = more unique values = higher cost.

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

### 📉 Find Underutilized Labels

**Question:** Which labels are rarely used?

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
      logs: [.logs[] | .severity_text]
    }'
done
```

---

### 📈 Monitor Cardinality Growth

**Question:** Is cardinality increasing over time?

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

## API Reference

### Metrics

#### List All Metrics

```bash
GET /api/v1/metrics?service={name}&limit={N}&offset={M}
```

**Parameters:**
- `service` (optional): Filter by service name
- `limit` (optional): Items per page (default: 100, max: 1000)
- `offset` (optional): Skip N items (default: 0)

**Example:**
```bash
curl "http://localhost:8080/api/v1/metrics?limit=10&offset=0"
```

**Response:**
```json
{
  "data": [...],
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
  "unit": "1",
  "description": "Total HTTP requests",
  "label_keys": {
    "method": {
      "count": 1000,
      "percentage": 100,
      "estimated_cardinality": 4,
      "value_samples": ["GET", "POST", "PUT", "DELETE"],
      "first_seen": "2025-10-23T20:00:00Z",
      "last_seen": "2025-10-23T22:00:00Z"
    }
  },
  "resource_keys": {...},
  "sample_count": 1000,
  "services": {
    "api-server": 800,
    "proxy": 200
  }
}
```

### Spans (Traces)

#### List All Spans

```bash
GET /api/v1/spans?service={name}&limit={N}&offset={M}
```

**Example:**
```bash
curl "http://localhost:8080/api/v1/spans?service=api-server"
```

#### Get Specific Span

```bash
GET /api/v1/spans/{name}
```

**Example:**
```bash
curl "http://localhost:8080/api/v1/spans/HTTP%20GET%20/api/users"
```

### Logs

#### List All Log Metadata

```bash
GET /api/v1/logs?service={name}&limit={N}&offset={M}
```

**Example:**
```bash
curl "http://localhost:8080/api/v1/logs"
```

#### Get Specific Log Level

```bash
GET /api/v1/logs/{severity}
```

**Example:**
```bash
curl "http://localhost:8080/api/v1/logs/ERROR"
```

### Services

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
[
  "api-server",
  "database",
  "cache",
  "proxy"
]
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
  "metrics": [...],
  "spans": [...],
  "logs": [...]
}
```

---

## Useful jq Patterns

### Extract Specific Fields

```bash
# Just metric names
curl -s "http://localhost:8080/api/v1/metrics" | jq -r '.data[] | .name'

# Metric names with sample counts
curl -s "http://localhost:8080/api/v1/metrics" | \
  jq -r '.data[] | "\(.name): \(.sample_count) samples"'

# Metrics grouped by type
curl -s "http://localhost:8080/api/v1/metrics" | \
  jq 'group_by(.type) | map({type: .[0].type, count: length})'
```

### Filter and Sort

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

### Create Reports

```bash
# Cardinality report
curl -s "http://localhost:8080/api/v1/metrics?limit=1000" | \
  jq -r '.data[] | .name as $metric | .label_keys | to_entries[] | 
    select(.value.estimated_cardinality > 20) | 
    "\($metric).\(.key): \(.value.estimated_cardinality)"' | \
  sort -t: -k2 -nr
```

**Output:**
```
api_calls.user_id: 1247
http_requests.request_id: 982
database_queries.query_hash: 456
cache_operations.cache_key: 234
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
# Find metrics missing 'service.name'
curl -s "http://localhost:8080/api/v1/metrics?limit=1000" | \
  jq -r '.data[] | select(.resource_keys["service.name"].count == 0) | .name'
```

### Calculate Total Cardinality

```bash
# Total cardinality across all labels for a metric
curl -s "http://localhost:8080/api/v1/metrics/http_requests_total" | \
  jq '[.label_keys[] | .estimated_cardinality] | reduce .[] as $item (1; . * $item)'
```

### Export to CSV

```bash
# Export metrics to CSV
echo "metric,label,cardinality,percentage" > cardinality.csv
curl -s "http://localhost:8080/api/v1/metrics?limit=10000" | \
  jq -r '.data[] | .name as $m | .label_keys | to_entries[] | 
    "\($m),\(.key),\(.value.estimated_cardinality),\(.value.percentage)"' \
  >> cardinality.csv
```

---

## Load Testing

See [scripts/README.md](scripts/README.md) for K6 load testing examples.

Quick test:
```bash
k6 run --vus 10 --duration 30s scripts/load-test.js
```

---

## Next Steps

- **Production Deployment**: See [../k8s/README.md](../k8s/README.md) for Kubernetes deployment guide
- **API Documentation**: See [API.md](API.md)
- **Scalability**: See [SCALABILITY.md](SCALABILITY.md)
