# K6 Load Tests for OTLP Cardinality Checker

This directory contains K6 load test scripts for testing the OTLP Cardinality Checker with **metrics**, **traces**, and **logs**.

## Installation

```bash
# macOS
brew install k6

# Linux (Debian/Ubuntu)
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Docker
docker pull grafana/k6
```

## Test Scripts

### 1. Metrics Load Test (`load-test-metrics.js`)
Tests metric ingestion with configurable cardinality and metric count.

**Basic usage:**
```bash
k6 run scripts/load-test-metrics.js
```

**Custom configuration:**
```bash
k6 run --vus 50 --duration 120s \
  -e NUM_METRICS=500000 \
  -e CARDINALITY=100 \
  scripts/load-test-metrics.js
```

**Environment variables:**
- `OTLP_ENDPOINT` - OTLP endpoint (default: `http://localhost:4218`)
- `API_ENDPOINT` - REST API endpoint (default: `http://localhost:8080`)
- `NUM_METRICS` - Number of unique metric names (default: 1000)
- `CARDINALITY` - Number of unique values per label (default: 50)

**Features:**
- Generates realistic metric data with multiple labels
- Tracks high cardinality labels (>40 unique values)
- Hybrid metric ID generation for better coverage
- Reports memory usage and metrics created

### 2. Traces Load Test (`load-test-traces.js`)
Tests trace/span ingestion with configurable span operations and cardinality.

**Basic usage:**
```bash
k6 run scripts/load-test-traces.js
```

**Custom configuration:**
```bash
k6 run --vus 50 --duration 120s \
  -e NUM_SPANS=10000 \
  -e CARDINALITY=100 \
  scripts/load-test-traces.js
```

**Environment variables:**
- `OTLP_ENDPOINT` - OTLP endpoint (default: `http://localhost:4318`)
- `API_ENDPOINT` - REST API endpoint (default: `http://localhost:8080`)
- `NUM_SPANS` - Number of unique span operations (default: 100)
- `CARDINALITY` - Number of unique values per attribute (default: 50)

**Features:**
- Generates spans with HTTP attributes
- Multiple span kinds (Internal, Server, Client, etc.)
- Resource attributes tracking
- Tracks high cardinality span attributes

### 3. Logs Load Test (`load-test-logs.js`)
Tests log ingestion with multiple severity levels and attributes.

**Basic usage:**
```bash
k6 run scripts/load-test-logs.js
```

**Custom configuration:**
```bash
k6 run --vus 50 --duration 120s \
  -e NUM_MODULES=1000 \
  -e CARDINALITY=100 \
  scripts/load-test-logs.js
```

**Environment variables:**
- `OTLP_ENDPOINT` - OTLP endpoint (default: `http://localhost:4318`)
- `API_ENDPOINT` - REST API endpoint (default: `http://localhost:8080`)
- `NUM_MODULES` - Number of unique modules (default: 100)
- `CARDINALITY` - Number of unique values per attribute (default: 50)

**Features:**
- Generates logs with INFO, WARN, ERROR, DEBUG severities
- Realistic log attributes (module, trace_id, user_id)
- Resource attributes for service tracking
- Reports severity breakdown

### 4. Stress Test (`stress-test.js`)
Ramps up load to find breaking points.

**Run:**
```bash
k6 run scripts/stress-test.js
```

Stages:
- 30s ramp to 10 VUs
- 1m ramp to 50 VUs
- 1m ramp to 100 VUs
- 30s ramp down to 0

### 3. Shell Script (`load-test.sh`)
Bash-based load test with memory monitoring (legacy, k6 preferred).

**Run:**
```bash
./scripts/load-test.sh
```

## Typical Test Scenarios

### Scenario 1: Realistic Production Load
Simulates a medium-sized deployment with multiple services.

```bash
k6 run --vus 10 --duration 60s \
  -e NUM_METRICS=1000 \
  -e CARDINALITY=50 \
  scripts/load-test-metrics.js
```

**Expected results:**
- ~6000 metric updates/min
- <10ms p95 latency
- <100MB memory growth

### Scenario 2: High Cardinality Test
Tests behavior with high cardinality labels.

```bash
k6 run --vus 20 --duration 120s \
  -e NUM_METRICS=500 \
  -e CARDINALITY=1000 \
  scripts/load-test-metrics.js
```

**Watch for:**
- Memory growth (should cap at MaxSamples=100)
- API response times
- High cardinality warnings

### Scenario 3: Large Deployment
Tests with many unique metrics (10k+).

```bash
k6 run --vus 50 --duration 180s \
  -e NUM_METRICS=10000 \
  -e CARDINALITY=100 \
  scripts/load-test-metrics.js
```

**Expected behavior:**
- ~500MB memory usage
- Slower API responses due to large result sets
- Pagination becomes critical

### Scenario 4: Stress Test
Find the breaking point.

```bash
k6 run scripts/stress-test.js
```

**Watch for:**
- When error rate increases
- When API latency degrades
- Memory usage patterns

## Monitoring During Tests

### Memory Usage
```bash
# Terminal 1: Run test
k6 run scripts/load-test-metrics.js

# Terminal 2: Watch memory
watch -n 1 'ps aux | grep otlp-cardinality-checker | grep -v grep | awk "{print \$6/1024 \" MB\"}"'
```

### API Performance
```bash
# Check API response time
time curl -s "http://localhost:8080/api/v1/metrics?limit=100" | jq '.total'
```

### Current State
```bash
# Get metrics count
curl -s "http://localhost:8080/api/v1/metrics" | jq '{total, limit, has_more}'

# Get high cardinality metrics
curl -s "http://localhost:8080/api/v1/metrics?limit=100" | \
  jq '.data[] | select(.label_keys | to_entries[] | .value.estimated_cardinality > 50) | .name'
```

## Interpreting Results

### K6 Output Metrics

```
✓ status is 200
✓ response time < 500ms

checks.........................: 100.00% ✓ 12000 ✗ 0
data_received..................: 2.4 MB  40 kB/s
data_sent......................: 12 MB   200 kB/s
http_req_duration..............: avg=45ms min=5ms med=40ms max=150ms p(95)=80ms
http_reqs......................: 12000   200/s
iteration_duration.............: avg=150ms
vus............................: 10
vus_max........................: 10
```

**Good indicators:**
- ✓ checks: 100% pass rate
- http_req_duration p(95): <500ms
- No errors
- Steady memory growth, then plateau

**Warning signs:**
- ✗ Failed checks
- p(95) > 1000ms
- Increasing error rate
- Linear memory growth without plateau

## Expected Performance

Based on hardware and configuration:

| Scenario | Memory | p95 Latency | Throughput |
|----------|--------|-------------|------------|
| 1k metrics, 10 VUs | ~100MB | <50ms | 200 req/s |
| 5k metrics, 20 VUs | ~200MB | <100ms | 400 req/s |
| 10k metrics, 50 VUs | ~500MB | <200ms | 800 req/s |

## Troubleshooting

### High Memory Usage
- Check `MaxSamples` setting (default: 100)
- Look for high cardinality labels
- Verify value samples are capped

### Slow API Responses
- Use pagination (`?limit=100`)
- Filter by service (`?service=X`)
- Check if returning too much data

### Connection Errors
- Verify server is running
- Check ports (4318 for OTLP, 8080 for API)
- Increase file descriptors: `ulimit -n 10000`

## Next Steps

After load testing:

1. **If memory usage is high** → Consider PostgreSQL persistence
2. **If cardinality is problematic** → Implement HyperLogLog
3. **If latency is high** → Add sharded maps
4. **If it works well** → Deploy to Kubernetes!
