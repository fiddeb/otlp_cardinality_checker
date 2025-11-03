# K6 Load Testing for OTLP Cardinality Checker

## Mixed Signal Load Test

This test simulates realistic production traffic with a mix of:
- **Metrics** (40%): Counters, Gauges, and Histograms
- **Logs** (30%): Various severity levels with structured attributes
- **Traces** (30%): Distributed tracing spans

### Cardinality Mix
- **70% Low Cardinality**: Standard attributes (method, status, environment)
- **30% High Cardinality**: Unique IDs (user_id, session_id, request_id)

## Prerequisites

1. **Install K6**:
   ```bash
   # macOS
   brew install k6
   
   # Or download from: https://k6.io/docs/getting-started/installation/
   ```

2. **Start the OTLP Cardinality Checker**:
   ```bash
   ./bin/otlp-cardinality-checker
   # Should be listening on http://localhost:4318
   ```

## Running the Test

### Quick Test (30 seconds)
```bash
k6 run --duration 30s --vus 10 scripts/k6-mixed-load-test.js
```

### Full Load Test (11 minutes with ramp-up/down)
```bash
k6 run scripts/k6-mixed-load-test.js
```

### Custom Test
```bash
# 100 VUs for 5 minutes
k6 run --duration 5m --vus 100 scripts/k6-mixed-load-test.js

# Spike test: 500 VUs for 2 minutes
k6 run --duration 2m --vus 500 scripts/k6-mixed-load-test.js
```

## Test Stages

The default test runs through these stages:

1. **Warm up** (30s): 10 VUs - System initialization
2. **Ramp up** (1m): 10 → 50 VUs - Gradual load increase
3. **Steady load** (2m): 50 → 100 VUs - Normal operation
4. **Sustain** (3m): 100 VUs - Peak sustained load
5. **Spike** (1m): 100 → 200 VUs - Traffic spike
6. **Spike sustain** (2m): 200 VUs - Handle spike
7. **Ramp down** (1m): 200 → 50 VUs - Load decrease
8. **Cool down** (30s): 50 → 0 VUs - Graceful shutdown

**Total Duration**: ~11 minutes

## Expected Performance

### Target Metrics
- **Throughput**: 1,000+ requests/second
- **P95 Latency**: < 500ms
- **Error Rate**: < 5%

### Signal Distribution
- Metrics: ~40% of traffic
- Logs: ~30% of traffic  
- Traces: ~30% of traffic

### Cardinality
- Low cardinality attributes: ~70%
- High cardinality attributes: ~30%

## Monitoring During Test

### Terminal 1: Run K6 Test
```bash
k6 run scripts/k6-mixed-load-test.js
```

### Terminal 2: Watch Server Logs
```bash
./bin/otlp-cardinality-checker
```

### Terminal 3: Monitor System Resources
```bash
# CPU and Memory
top -pid $(pgrep -f otlp-cardinality-checker)

# Or use htop
htop -p $(pgrep -f otlp-cardinality-checker)
```

### Terminal 4: Query API During Load
```bash
# Check metrics count
watch -n 5 'curl -s http://localhost:3000/api/v1/metrics | jq ".data | length"'

# Check memory usage
watch -n 5 'curl -s http://localhost:3000/api/v1/memory'

# Check metadata complexity
watch -n 10 'curl -s http://localhost:3000/api/v1/metadata/complexity?limit=10'
```

## Results Interpretation

### Good Results
```
✓ Duration: 660s
✓ Requests: 660,000
✓ Errors: 0
✓ Error Rate: 0.00%
✓ Avg Duration: 150ms
✓ P95 Duration: 450ms
✓ Requests/sec: 1000
```

### Warning Signs
- ⚠️ Error rate > 1%
- ⚠️ P95 latency > 500ms
- ⚠️ Requests/sec drops significantly
- ⚠️ Memory usage grows continuously

### Critical Issues
- ❌ Error rate > 5%
- ❌ P95 latency > 1000ms
- ❌ Server crashes or restarts
- ❌ Memory leak detected

## Troubleshooting

### High Latency
- Check if database operations are slow
- Look for lock contention
- Verify disk I/O isn't saturated

### High Error Rate
- Check server logs for errors
- Verify OTLP endpoint is responding
- Look for JSON parsing errors

### Memory Issues
- Monitor memory growth over time
- Check for memory leaks
- Consider reducing cardinality

### Low Throughput
- Increase VUs if server can handle more
- Check network bandwidth
- Look for bottlenecks in processing pipeline

## Advanced Options

### Testing Specific Scenarios

**High Cardinality Test** (90% high cardinality):
```javascript
// Modify line in k6-mixed-load-test.js:
const cardinalityType = Math.random() < 0.1 ? 'low' : 'high'; // Changed from 0.7
```

**Metrics Only**:
```javascript
// Modify signal selection:
sendMetrics(service, cardinalityType); // Always send metrics
```

**Stress Test** (find breaking point):
```bash
k6 run --duration 5m --vus 500 scripts/k6-mixed-load-test.js
```

## Cleanup After Test

```bash
# Clear data
curl -X POST http://localhost:3000/api/v1/admin/clear

# Or use the UI: click "🗑️ Clear Data" button
```
