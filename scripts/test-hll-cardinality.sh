#!/bin/bash
# Test HyperLogLog cardinality estimation with real OTLP data

ENDPOINT="http://localhost:4318/v1/metrics"

echo "Testing HyperLogLog cardinality estimation..."
echo "Sending 1000 data points with 500 unique label values to metric 'hll_test_metric'"

for i in {1..1000}; do
    # Generate value that cycles through 0-499 (500 unique values)
    VALUE=$((i % 500))
    
    # Create OTLP metrics JSON
    JSON=$(cat <<EOF
{
  "resourceMetrics": [{
    "resource": {
      "attributes": [
        {"key": "service.name", "value": {"stringValue": "hll-test-service"}}
      ]
    },
    "scopeMetrics": [{
      "scope": {"name": "test"},
      "metrics": [{
        "name": "hll_test_metric",
        "description": "Test metric for HyperLogLog validation",
        "unit": "1",
        "gauge": {
          "dataPoints": [{
            "timeUnixNano": "$(date +%s)000000000",
            "asInt": "$i",
            "attributes": [
              {"key": "high_cardinality_label", "value": {"stringValue": "value_$VALUE"}},
              {"key": "method", "value": {"stringValue": "GET"}}
            ]
          }]
        }
      }]
    }]
  }]
}
EOF
)
    
    # Send to OTLP endpoint
    curl -s -X POST "$ENDPOINT" \
         -H "Content-Type: application/json" \
         -d "$JSON" > /dev/null
    
    # Progress indicator
    if [ $((i % 100)) -eq 0 ]; then
        echo "Sent $i data points..."
    fi
done

echo ""
echo "✅ Sent 1000 data points with 500 unique values"
echo ""
echo "Querying metric to see HLL estimation..."
sleep 1

# Query the metric
curl -s "http://localhost:8080/api/v1/metrics/hll_test_metric" | jq '{
  metric_name: .name,
  sample_count: .sample_count,
  high_cardinality_label: {
    count: .label_keys.high_cardinality_label.count,
    estimated_cardinality: .label_keys.high_cardinality_label.estimated_cardinality,
    sample_values: .label_keys.high_cardinality_label.value_samples,
    percentage: .label_keys.high_cardinality_label.percentage
  }
}'

echo ""
echo "Expected cardinality: 500"
echo "HLL estimation should be within ~2-5% of 500"
