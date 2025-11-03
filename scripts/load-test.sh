#!/bin/bash
# Load test script for OTLP Cardinality Checker
# Tests memory usage and performance under realistic load

set -e

ENDPOINT="${OTLP_ENDPOINT:-http://localhost:4318}"
NUM_METRICS="${NUM_METRICS:-1000}"
NUM_SERVICES="${NUM_SERVICES:-10}"
CARDINALITY="${CARDINALITY:-50}"
DURATION="${DURATION:-60}"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

function log() {
    echo -e "${GREEN}[$(date +%T)]${NC} $1"
}

function log_metric() {
    echo -e "${BLUE}[METRIC]${NC} $1"
}

# Get timestamp in nanoseconds
function get_timestamp() {
    echo "$(($(date +%s) * 1000000000))"
}

# Send a batch of metrics
function send_metric_batch() {
    local service=$1
    local batch_size=$2
    
    local timestamp=$(get_timestamp)
    local json='{"resource_metrics": [{"resource": {"attributes": [{"key": "service.name", "value": {"string_value": "'$service'"}}]}, "scope_metrics": [{"metrics": ['
    
    for i in $(seq 1 $batch_size); do
        local metric_num=$((RANDOM % NUM_METRICS))
        local label_value=$((RANDOM % CARDINALITY))
        
        if [ $i -gt 1 ]; then
            json+=','
        fi
        
        json+='{"name": "test_metric_'$metric_num'", "sum": {"aggregation_temporality": 2, "is_monotonic": true, "data_points": [{"attributes": [{"key": "label1", "value": {"string_value": "value_'$label_value'"}}, {"key": "label2", "value": {"string_value": "value_'$((RANDOM % 10))'"}}, {"key": "method", "value": {"string_value": "GET"}}], "as_int": '$RANDOM', "time_unix_nano": '$timestamp'}]}}'
    done
    
    json+=']}]}]}'
    
    curl -s -X POST "$ENDPOINT/v1/metrics" \
        -H "Content-Type: application/json" \
        -d "$json" > /dev/null
}

# Monitor memory usage
function monitor_memory() {
    while true; do
        local mem=$(ps aux | grep occ | grep -v grep | awk '{print $6}')
        if [ ! -z "$mem" ]; then
            log_metric "Memory: ${mem}KB ($(echo "scale=2; $mem/1024" | bc)MB)"
        fi
        sleep 5
    done
}

# Check if server is running
if ! curl -s "$ENDPOINT/../health" > /dev/null 2>&1; then
    echo "Error: Server not running at $ENDPOINT"
    exit 1
fi

echo "=========================================="
echo "  OTLP Cardinality Checker - Load Test"
echo "=========================================="
echo ""
echo "Configuration:"
echo "  Endpoint:       $ENDPOINT"
echo "  Unique Metrics: $NUM_METRICS"
echo "  Services:       $NUM_SERVICES"
echo "  Cardinality:    $CARDINALITY values per label"
echo "  Duration:       ${DURATION}s"
echo "  Batch Size:     10 metrics per request"
echo ""

# Get baseline metrics
log "Getting baseline metrics..."
baseline_metrics=$(curl -s "http://localhost:8080/api/v1/metrics" | jq '.total')
baseline_mem=$(ps aux | grep occ | grep -v grep | awk '{print $6}')

echo "Baseline:"
echo "  Metrics:   $baseline_metrics"
echo "  Memory:    ${baseline_mem}KB ($(echo "scale=2; $baseline_mem/1024" | bc)MB)"
echo ""

# Start memory monitor in background
monitor_memory &
MONITOR_PID=$!

# Cleanup on exit
trap "kill $MONITOR_PID 2>/dev/null || true" EXIT

log "Starting load test..."
start_time=$(date +%s)
request_count=0
error_count=0

while true; do
    current_time=$(date +%s)
    elapsed=$((current_time - start_time))
    
    if [ $elapsed -ge $DURATION ]; then
        break
    fi
    
    # Send batch from random service
    service_num=$((RANDOM % NUM_SERVICES))
    if send_metric_batch "service_$service_num" 10 2>/dev/null; then
        request_count=$((request_count + 1))
    else
        error_count=$((error_count + 1))
    fi
    
    # Progress every 10 requests
    if [ $((request_count % 10)) -eq 0 ]; then
        log "Sent $request_count batches ($(($request_count * 10)) data points), elapsed: ${elapsed}s"
    fi
done

# Stop memory monitor
kill $MONITOR_PID 2>/dev/null || true

log "Load test completed!"
echo ""

# Get final metrics
log "Collecting final statistics..."
sleep 2

final_metrics=$(curl -s "http://localhost:8080/api/v1/metrics" | jq '.total')
final_mem=$(ps aux | grep occ | grep -v grep | awk '{print $6}')
services=$(curl -s "http://localhost:8080/api/v1/services" | jq '. | length')

# Calculate some sample metrics
sample_metric=$(curl -s "http://localhost:8080/api/v1/metrics?limit=1" | jq -r '.data[0].name')
if [ ! -z "$sample_metric" ]; then
    cardinality=$(curl -s "http://localhost:8080/api/v1/metrics/$sample_metric" | jq -r '.label_keys.label1.estimated_cardinality')
    sample_count=$(curl -s "http://localhost:8080/api/v1/metrics/$sample_metric" | jq -r '.sample_count')
fi

echo "=========================================="
echo "  LOAD TEST RESULTS"
echo "=========================================="
echo ""
echo "Requests:"
echo "  Total batches:    $request_count"
echo "  Total datapoints: $(($request_count * 10))"
echo "  Errors:           $error_count"
echo "  Duration:         ${DURATION}s"
echo "  Rate:             $(echo "scale=2; $request_count / $DURATION" | bc) req/s"
echo "  Throughput:       $(echo "scale=2; ($request_count * 10) / $DURATION" | bc) datapoints/s"
echo ""
echo "Metrics:"
echo "  Baseline:         $baseline_metrics"
echo "  Final:            $final_metrics"
echo "  New:              $((final_metrics - baseline_metrics))"
echo "  Services tracked: $services"
echo ""
if [ ! -z "$cardinality" ]; then
echo "Sample Metric: $sample_metric"
echo "  Sample count:     $sample_count"
echo "  Label1 card:      $cardinality"
echo ""
fi
echo "Memory Usage:"
echo "  Baseline:         ${baseline_mem}KB ($(echo "scale=2; $baseline_mem/1024" | bc)MB)"
echo "  Final:            ${final_mem}KB ($(echo "scale=2; $final_mem/1024" | bc)MB)"
echo "  Growth:           $((final_mem - baseline_mem))KB ($(echo "scale=2; ($final_mem - $baseline_mem)/1024" | bc)MB)"
echo "  Per metric:       $(echo "scale=2; ($final_mem - $baseline_mem) / ($final_metrics - $baseline_metrics)" | bc)KB"
echo ""
echo "=========================================="

# Check for high cardinality
log "Checking for high cardinality labels..."
high_card=$(curl -s "http://localhost:8080/api/v1/metrics" | \
    jq -r '.data[] | select(.label_keys | to_entries[] | .value.estimated_cardinality > 20) | .name' | \
    head -5)

if [ ! -z "$high_card" ]; then
    echo ""
    echo "⚠️  High cardinality metrics detected:"
    echo "$high_card" | while read metric; do
        echo "  - $metric"
    done
fi

echo ""
log "Test complete! ✓"
