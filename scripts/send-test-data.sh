#!/bin/bash
# Test script for sending OTLP data to the cardinality checker

set -e

# Configuration
ENDPOINT="${OTLP_ENDPOINT:-http://localhost:4318}"
SERVICE_NAME="${SERVICE_NAME:-test-service}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

function log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

function log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

function log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Generate timestamp
function get_timestamp() {
    echo "$(($(date +%s) * 1000000000))"
}

# Generate random hex
function random_hex() {
    local length=$1
    openssl rand -hex "$length" 2>/dev/null || head -c "$length" /dev/urandom | xxd -p | tr -d '\n'
}

# Send metrics
function send_metrics() {
    local metric_name=$1
    local value=$2
    shift 2
    local attributes="$@"
    
    log_info "Sending metric: $metric_name = $value"
    
    # Build attributes JSON
    local attrs_json=""
    for attr in $attributes; do
        IFS='=' read -r key val <<< "$attr"
        if [ -n "$attrs_json" ]; then
            attrs_json="$attrs_json,"
        fi
        attrs_json="$attrs_json{\"key\": \"$key\", \"value\": {\"stringValue\": \"$val\"}}"
    done
    
    curl -s -X POST "${ENDPOINT}/v1/metrics" \
      -H "Content-Type: application/json" \
      -d '{
        "resourceMetrics": [{
          "resource": {
            "attributes": [
              {"key": "service.name", "value": {"stringValue": "'"$SERVICE_NAME"'"}},
              {"key": "service.version", "value": {"stringValue": "1.0.0"}},
              {"key": "deployment.environment", "value": {"stringValue": "test"}}
            ]
          },
          "scopeMetrics": [{
            "scope": {
              "name": "test-instrumentation",
              "version": "0.1.0"
            },
            "metrics": [{
              "name": "'"$metric_name"'",
              "description": "Test metric",
              "unit": "1",
              "sum": {
                "dataPoints": [{
                  "timeUnixNano": "'"$(get_timestamp)"'",
                  "asInt": "'"$value"'",
                  "attributes": ['"$attrs_json"']
                }],
                "aggregationTemporality": 2,
                "isMonotonic": true
              }
            }]
          }]
        }]
      }' || log_error "Failed to send metric"
}

# Send traces
function send_trace() {
    local span_name=$1
    shift
    local attributes="$@"
    
    log_info "Sending trace: $span_name"
    
    # Build attributes JSON
    local attrs_json=""
    for attr in $attributes; do
        IFS='=' read -r key val <<< "$attr"
        if [ -n "$attrs_json" ]; then
            attrs_json="$attrs_json,"
        fi
        attrs_json="$attrs_json{\"key\": \"$key\", \"value\": {\"stringValue\": \"$val\"}}"
    done
    
    local trace_id=$(random_hex 16)
    local span_id=$(random_hex 8)
    local start_time=$(get_timestamp)
    local end_time=$((start_time + 2000000000)) # +2 seconds
    
    curl -s -X POST "${ENDPOINT}/v1/traces" \
      -H "Content-Type: application/json" \
      -d '{
        "resourceSpans": [{
          "resource": {
            "attributes": [
              {"key": "service.name", "value": {"stringValue": "'"$SERVICE_NAME"'"}},
              {"key": "service.version", "value": {"stringValue": "1.0.0"}}
            ]
          },
          "scopeSpans": [{
            "scope": {
              "name": "test-instrumentation",
              "version": "0.1.0"
            },
            "spans": [{
              "traceId": "'"$trace_id"'",
              "spanId": "'"$span_id"'",
              "name": "'"$span_name"'",
              "kind": 3,
              "startTimeUnixNano": "'"$start_time"'",
              "endTimeUnixNano": "'"$end_time"'",
              "attributes": ['"$attrs_json"'],
              "status": {"code": 1}
            }]
          }]
        }]
      }' || log_error "Failed to send trace"
}

# Send logs
function send_log() {
    local severity=$1
    local message=$2
    shift 2
    local attributes="$@"
    
    log_info "Sending log: [$severity] $message"
    
    # Build attributes JSON
    local attrs_json=""
    for attr in $attributes; do
        IFS='=' read -r key val <<< "$attr"
        if [ -n "$attrs_json" ]; then
            attrs_json="$attrs_json,"
        fi
        attrs_json="$attrs_json{\"key\": \"$key\", \"value\": {\"stringValue\": \"$val\"}}"
    done
    
    curl -s -X POST "${ENDPOINT}/v1/logs" \
      -H "Content-Type: application/json" \
      -d '{
        "resourceLogs": [{
          "resource": {
            "attributes": [
              {"key": "service.name", "value": {"stringValue": "'"$SERVICE_NAME"'"}},
              {"key": "service.version", "value": {"stringValue": "1.0.0"}}
            ]
          },
          "scopeLogs": [{
            "scope": {
              "name": "test-instrumentation",
              "version": "0.1.0"
            },
            "logRecords": [{
              "timeUnixNano": "'"$(get_timestamp)"'",
              "body": {"stringValue": "'"$message"'"},
              "severityText": "'"$severity"'",
              "attributes": ['"$attrs_json"']
            }]
          }]
        }]
      }' || log_error "Failed to send log"
}

# Test scenarios
function test_good_cardinality() {
    log_info "=== Testing GOOD cardinality patterns ==="
    
    # Metric with low cardinality labels
    send_metrics "http_requests_total" 100 "method=GET" "status=200"
    send_metrics "http_requests_total" 50 "method=POST" "status=201"
    send_metrics "http_requests_total" 10 "method=GET" "status=404"
    send_metrics "http_requests_total" 5 "method=GET" "status=500"
    
    # Trace with reasonable attributes
    send_trace "HTTP GET /api/users" "http.method=GET" "http.status_code=200"
    send_trace "HTTP POST /api/users" "http.method=POST" "http.status_code=201"
    
    # Logs with structured attributes
    send_log "INFO" "User logged in" "action=login" "result=success"
    send_log "WARN" "Rate limit exceeded" "action=api_call" "result=throttled"
}

function test_bad_cardinality() {
    log_info "=== Testing BAD cardinality patterns (high cardinality) ==="
    
    # Metric with HIGH cardinality label (user_id - BAD!)
    for i in {1..20}; do
        send_metrics "api_calls_total" $i "method=GET" "user_id=user_$i"
    done
    
    # Trace with high cardinality attribute (request_id - OK in traces, but shows pattern)
    for i in {1..10}; do
        local req_id=$(random_hex 8)
        send_trace "HTTP GET /api/resource" "http.method=GET" "request.id=$req_id"
    done
    
    # Log with high cardinality (user_id in logs - potentially problematic)
    for i in {1..15}; do
        send_log "INFO" "API request processed" "user_id=user_$i" "action=fetch"
    done
}

function test_missing_labels() {
    log_info "=== Testing optional/missing labels ==="
    
    # Same metric with different label sets
    send_metrics "cache_hits_total" 100 "cache_name=redis" "region=us-east-1"
    send_metrics "cache_hits_total" 50 "cache_name=memcached"  # Missing region
    send_metrics "cache_hits_total" 75 "cache_name=redis" "region=eu-west-1"
}

# Main execution
function main() {
    echo "========================================"
    echo "OTLP Cardinality Checker - Test Script"
    echo "========================================"
    echo ""
    echo "Endpoint: $ENDPOINT"
    echo "Service:  $SERVICE_NAME"
    echo ""
    
    case "${1:-all}" in
        good)
            test_good_cardinality
            ;;
        bad)
            test_bad_cardinality
            ;;
        missing)
            test_missing_labels
            ;;
        all)
            test_good_cardinality
            sleep 1
            test_bad_cardinality
            sleep 1
            test_missing_labels
            ;;
        *)
            echo "Usage: $0 [good|bad|missing|all]"
            echo ""
            echo "Scenarios:"
            echo "  good    - Send data with good cardinality patterns"
            echo "  bad     - Send data with high cardinality (anti-patterns)"
            echo "  missing - Send data with optional/missing labels"
            echo "  all     - Run all scenarios (default)"
            exit 1
            ;;
    esac
    
    echo ""
    log_info "✓ Test data sent successfully!"
    echo ""
    echo "Query the API:"
    echo "  curl http://localhost:8080/api/v1/metrics"
    echo "  curl http://localhost:8080/api/v1/services/$SERVICE_NAME/overview"
    echo "  curl http://localhost:8080/api/v1/metrics/http_requests_total"
}

main "$@"
