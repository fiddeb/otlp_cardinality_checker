#!/bin/bash
# Verify data is being written to ClickHouse

set -e

echo "=== ClickHouse Data Verification ==="
echo

echo "1. Checking ClickHouse connection..."
if ! clickhouse-client --query "SELECT 1" > /dev/null 2>&1; then
    echo "❌ ClickHouse is not running or not accessible"
    exit 1
fi
echo "✓ ClickHouse is running"
echo

echo "2. Checking table row counts..."
clickhouse-client --query "
SELECT 
    'metrics' as table, 
    count() as total_rows,
    uniq(name) as unique_items
FROM metrics
UNION ALL
SELECT 
    'spans' as table, 
    count() as total_rows,
    uniq(name) as unique_items
FROM spans
UNION ALL
SELECT 
    'logs' as table, 
    count() as total_rows,
    uniq(severity) as unique_items
FROM logs
FORMAT PrettyCompact
"
echo

echo "3. Checking recent metrics (last 10)..."
clickhouse-client --query "
SELECT 
    name,
    service_name,
    sample_count,
    updated_at
FROM metrics FINAL 
ORDER BY updated_at DESC 
LIMIT 10
FORMAT PrettyCompact
"
echo

echo "4. Checking recent spans (last 10)..."
clickhouse-client --query "
SELECT 
    name,
    service_name,
    sample_count,
    updated_at
FROM spans FINAL 
ORDER BY updated_at DESC 
LIMIT 10
FORMAT PrettyCompact
"
echo

echo "5. Checking recent logs (last 10)..."
clickhouse-client --query "
SELECT 
    severity,
    service_name,
    sample_count,
    updated_at
FROM logs FINAL 
ORDER BY updated_at DESC 
LIMIT 10
FORMAT PrettyCompact
"
echo

echo "6. Checking REST API response..."
METRIC_COUNT=$(curl -s "http://localhost:8080/api/v1/metrics?limit=1000" | jq '.data | length')
SPAN_COUNT=$(curl -s "http://localhost:8080/api/v1/spans?limit=1000" | jq '.data | length')
LOG_COUNT=$(curl -s "http://localhost:8080/api/v1/logs?limit=1000" | jq '.data | length')

echo "API reports:"
echo "  - Metrics: $METRIC_COUNT"
echo "  - Spans: $SPAN_COUNT"
echo "  - Logs: $LOG_COUNT"
echo

echo "=== Verification Complete ==="
