#!/bin/bash
set -e

echo "=== ClickHouse Integration Test ==="
echo ""

# Check if ClickHouse is running
if ! clickhouse-client --query "SELECT 1" >/dev/null 2>&1; then
    echo "❌ ClickHouse is not running"
    echo "Start it with: ./scripts/start-clickhouse.sh"
    exit 1
fi

echo "✓ ClickHouse is running"

# Build the application
echo ""
echo "Building application..."
go build -o bin/occ ./cmd/server
echo "✓ Application built"

# Start the server in background
echo ""
echo "Starting server with ClickHouse backend..."
export STORAGE_BACKEND="clickhouse"
export CLICKHOUSE_ADDR="localhost:9000"
export USE_AUTOTEMPLATE="false"

./bin/occ &
SERVER_PID=$!
echo "✓ Server started (PID: $SERVER_PID)"

# Wait for server to start
sleep 3

# Function to cleanup
cleanup() {
    echo ""
    echo "Cleaning up..."
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
    echo "✓ Server stopped"
}

trap cleanup EXIT

# Test health endpoint
echo ""
echo "Testing health endpoint..."
HEALTH=$(curl -s http://localhost:8080/api/v1/health)
if echo "$HEALTH" | grep -q "ok"; then
    echo "✓ Health check passed"
else
    echo "❌ Health check failed"
    echo "Response: $HEALTH"
    exit 1
fi

# Send test metric via OTLP HTTP endpoint
echo ""
echo "Sending test metric via OTLP..."

# Create test OTLP metric payload
cat > /tmp/test_metric.json << 'EOF'
{
  "resourceMetrics": [{
    "resource": {
      "attributes": [{
        "key": "service.name",
        "value": {"stringValue": "test-service"}
      }]
    },
    "scopeMetrics": [{
      "metrics": [{
        "name": "http_requests_total",
        "description": "Total HTTP requests",
        "unit": "1",
        "sum": {
          "dataPoints": [{
            "asInt": 42,
            "attributes": [
              {"key": "method", "value": {"stringValue": "GET"}},
              {"key": "status", "value": {"stringValue": "200"}}
            ],
            "timeUnixNano": "1699545600000000000"
          }],
          "aggregationTemporality": 2,
          "isMonotonic": true
        }
      }]
    }]
  }]
}
EOF

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST http://localhost:4318/v1/metrics \
  -H "Content-Type: application/json" \
  -d @/tmp/test_metric.json)

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "200" ]; then
    echo "✓ Metric sent successfully"
else
    echo "❌ Failed to send metric (HTTP $HTTP_CODE)"
    exit 1
fi

# Wait for buffer flush (6 seconds for 5s timer + margin)
echo ""
echo "Waiting for buffer flush (6 seconds)..."
sleep 6

# Query metric via REST API
echo ""
echo "Querying metric via REST API..."
METRIC=$(curl -s http://localhost:8080/api/v1/metrics/http_requests_total)

if echo "$METRIC" | grep -q "http_requests_total"; then
    echo "✓ Metric retrieved successfully"
    echo ""
    echo "Metric details:"
    echo "$METRIC" | jq '.' || echo "$METRIC"
else
    echo "❌ Failed to retrieve metric"
    echo "Response: $METRIC"
    exit 1
fi

# Check ClickHouse directly
echo ""
echo "Verifying data in ClickHouse..."
CLICKHOUSE_DATA=$(clickhouse-client --query "SELECT name, label_keys, sample_count FROM metrics FINAL WHERE name = 'http_requests_total' FORMAT JSONCompact")

if echo "$CLICKHOUSE_DATA" | grep -q "http_requests_total"; then
    echo "✓ Data verified in ClickHouse"
    echo ""
    echo "ClickHouse row:"
    echo "$CLICKHOUSE_DATA" | jq '.' || echo "$CLICKHOUSE_DATA"
else
    echo "❌ No data found in ClickHouse"
    echo "Response: $CLICKHOUSE_DATA"
    exit 1
fi

echo ""
echo "=== All tests passed! ==="
echo ""
echo "Summary:"
echo "  ✓ ClickHouse connection"
echo "  ✓ Application startup"
echo "  ✓ Health check"
echo "  ✓ OTLP endpoint (write)"
echo "  ✓ Buffer flush"
echo "  ✓ REST API (read)"
echo "  ✓ ClickHouse data persistence"
