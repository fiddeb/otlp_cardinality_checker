#!/bin/bash
set -e

echo "=== ClickHouse Performance Test Suite ==="
echo ""

# Check prerequisites
if ! command -v k6 &> /dev/null; then
    echo "❌ k6 is not installed"
    echo "Install with: brew install k6"
    exit 1
fi

if ! clickhouse-client --query "SELECT 1" >/dev/null 2>&1; then
    echo "❌ ClickHouse is not running"
    echo "Start it with: ./scripts/start-clickhouse.sh"
    exit 1
fi

echo "✓ Prerequisites checked"

# Start server in background
echo ""
echo "Starting server with ClickHouse backend..."
export STORAGE_BACKEND="clickhouse"
export CLICKHOUSE_ADDR="localhost:9000"
export USE_AUTOTEMPLATE="false"

# Kill any existing server
pkill -f "bin/occ" 2>/dev/null || true
sleep 1

# Build and start
go build -o bin/occ ./cmd/server
./bin/occ &
SERVER_PID=$!
echo "✓ Server started (PID: $SERVER_PID)"

# Cleanup function
cleanup() {
    echo ""
    echo "Stopping server..."
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
    echo "✓ Server stopped"
}

trap cleanup EXIT

# Wait for server to start
echo ""
echo "Waiting for server to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:8080/health >/dev/null 2>&1; then
        echo "✓ Server is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ Server did not start in time"
        exit 1
    fi
    sleep 1
done

# Run write test
echo ""
echo "=== Running Write Load Test (2 minutes) ==="
echo "Target: 100 metrics/sec, 50 spans/sec, 30 logs/sec"
echo ""

k6 run scripts/k6-clickhouse-write.js

# Wait for buffer flush
echo ""
echo "Waiting for buffer flush (10 seconds)..."
sleep 10

# Check ClickHouse data
echo ""
echo "=== ClickHouse Data Summary ==="
echo ""
echo "Metrics count:"
clickhouse-client --query "SELECT count() FROM metrics FINAL"

echo ""
echo "Spans count:"
clickhouse-client --query "SELECT count() FROM spans FINAL"

echo ""
echo "Logs count:"
clickhouse-client --query "SELECT count() FROM logs FINAL"

echo ""
echo "Attribute values count:"
clickhouse-client --query "SELECT count() FROM attribute_values FINAL"

echo ""
echo "Unique services:"
clickhouse-client --query "SELECT DISTINCT arrayJoin(services) as service FROM metrics FINAL UNION ALL SELECT DISTINCT arrayJoin(services) FROM spans FINAL UNION ALL SELECT DISTINCT arrayJoin(services) FROM logs FINAL"

# Run read test
echo ""
echo "=== Running Read Load Test (1 minute) ==="
echo "Target: 50 requests/sec"
echo ""

k6 run scripts/k6-clickhouse-read.js

# Print results summary
echo ""
echo "=== Test Results ==="
echo ""

if [ -f k6-clickhouse-write-results.json ]; then
    echo "Write test results saved to: k6-clickhouse-write-results.json"
    
    # Extract key metrics
    WRITE_SUCCESS=$(jq -r '.metrics.write_success_rate.values.rate * 100' k6-clickhouse-write-results.json 2>/dev/null || echo "N/A")
    WRITE_P95=$(jq -r '.metrics.write_duration.values["p(95)"]' k6-clickhouse-write-results.json 2>/dev/null || echo "N/A")
    METRICS_WRITTEN=$(jq -r '.metrics.metrics_written.values.count' k6-clickhouse-write-results.json 2>/dev/null || echo "N/A")
    
    echo "  Write Success Rate: ${WRITE_SUCCESS}%"
    echo "  Write Duration p95: ${WRITE_P95}ms"
    echo "  Metrics Written: ${METRICS_WRITTEN}"
fi

if [ -f k6-clickhouse-read-results.json ]; then
    echo ""
    echo "Read test results saved to: k6-clickhouse-read-results.json"
    
    # Extract key metrics
    READ_SUCCESS=$(jq -r '.metrics.read_success_rate.values.rate * 100' k6-clickhouse-read-results.json 2>/dev/null || echo "N/A")
    READ_P95=$(jq -r '.metrics.read_duration.values["p(95)"]' k6-clickhouse-read-results.json 2>/dev/null || echo "N/A")
    
    echo "  Read Success Rate: ${READ_SUCCESS}%"
    echo "  Read Duration p95: ${READ_P95}ms"
fi

echo ""
echo "=== Performance Test Complete ==="
