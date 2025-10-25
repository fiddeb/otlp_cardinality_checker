#!/bin/bash
# Examples of sending histogram metrics to OTLP endpoint

ENDPOINT="http://127.0.0.1:4318"

# 1. HTTP Request Duration Histogram
echo "Sending HTTP request duration histogram..."
curl -X POST ${ENDPOINT}/v1/metrics \
  -H "Content-Type: application/json" \
  -d '{
    "resourceMetrics": [{
      "resource": {
        "attributes": [
          {"key": "service.name", "value": {"stringValue": "api-gateway"}},
          {"key": "host.name", "value": {"stringValue": "web-01"}}
        ]
      },
      "scopeMetrics": [{
        "metrics": [{
          "name": "http_request_duration_seconds",
          "description": "HTTP request duration in seconds",
          "unit": "s",
          "histogram": {
            "dataPoints": [{
              "timeUnixNano": "'$(date +%s)'000000000",
              "count": "1000",
              "sum": 125.5,
              "bucketCounts": ["50", "200", "350", "250", "100", "40", "10"],
              "explicitBounds": [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5],
              "attributes": [
                {"key": "method", "value": {"stringValue": "GET"}},
                {"key": "endpoint", "value": {"stringValue": "/api/v1/users"}},
                {"key": "status_code", "value": {"stringValue": "200"}}
              ]
            }],
            "aggregationTemporality": 2
          }
        }]
      }]
    }]
  }'

sleep 1

# 2. Database Query Duration Histogram
echo "Sending database query duration histogram..."
curl -X POST ${ENDPOINT}/v1/metrics \
  -H "Content-Type: application/json" \
  -d '{
    "resourceMetrics": [{
      "resource": {
        "attributes": [
          {"key": "service.name", "value": {"stringValue": "order-service"}},
          {"key": "host.name", "value": {"stringValue": "db-client-01"}}
        ]
      },
      "scopeMetrics": [{
        "metrics": [{
          "name": "db_query_duration_seconds",
          "description": "Database query duration in seconds",
          "unit": "s",
          "histogram": {
            "dataPoints": [{
              "timeUnixNano": "'$(date +%s)'000000000",
              "count": "500",
              "sum": 75.3,
              "bucketCounts": ["100", "150", "120", "80", "40", "8", "2"],
              "explicitBounds": [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0],
              "attributes": [
                {"key": "db.system", "value": {"stringValue": "postgresql"}},
                {"key": "db.operation", "value": {"stringValue": "SELECT"}},
                {"key": "db.table", "value": {"stringValue": "orders"}}
              ]
            }],
            "aggregationTemporality": 2
          }
        }]
      }]
    }]
  }'

sleep 1

# 3. Response Size Histogram (in bytes)
echo "Sending response size histogram..."
curl -X POST ${ENDPOINT}/v1/metrics \
  -H "Content-Type: application/json" \
  -d '{
    "resourceMetrics": [{
      "resource": {
        "attributes": [
          {"key": "service.name", "value": {"stringValue": "api-gateway"}},
          {"key": "host.name", "value": {"stringValue": "web-01"}}
        ]
      },
      "scopeMetrics": [{
        "metrics": [{
          "name": "http_response_size_bytes",
          "description": "HTTP response size in bytes",
          "unit": "By",
          "histogram": {
            "dataPoints": [{
              "timeUnixNano": "'$(date +%s)'000000000",
              "count": "800",
              "sum": 12500000,
              "bucketCounts": ["100", "200", "250", "150", "70", "25", "5"],
              "explicitBounds": [1024, 4096, 16384, 65536, 262144, 1048576, 4194304],
              "attributes": [
                {"key": "method", "value": {"stringValue": "GET"}},
                {"key": "endpoint", "value": {"stringValue": "/api/v1/products"}},
                {"key": "status_code", "value": {"stringValue": "200"}}
              ]
            }],
            "aggregationTemporality": 2
          }
        }]
      }]
    }]
  }'

sleep 1

# 4. Cache Hit Latency Histogram
echo "Sending cache hit latency histogram..."
curl -X POST ${ENDPOINT}/v1/metrics \
  -H "Content-Type: application/json" \
  -d '{
    "resourceMetrics": [{
      "resource": {
        "attributes": [
          {"key": "service.name", "value": {"stringValue": "product-service"}},
          {"key": "host.name", "value": {"stringValue": "cache-client-01"}}
        ]
      },
      "scopeMetrics": [{
        "metrics": [{
          "name": "cache_operation_duration_seconds",
          "description": "Cache operation duration in seconds",
          "unit": "s",
          "histogram": {
            "dataPoints": [{
              "timeUnixNano": "'$(date +%s)'000000000",
              "count": "2000",
              "sum": 3.5,
              "bucketCounts": ["500", "800", "400", "200", "80", "15", "5"],
              "explicitBounds": [0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1],
              "attributes": [
                {"key": "cache.operation", "value": {"stringValue": "get"}},
                {"key": "cache.hit", "value": {"stringValue": "true"}},
                {"key": "cache.key_prefix", "value": {"stringValue": "product:"}}
              ]
            }],
            "aggregationTemporality": 2
          }
        }]
      }]
    }]
  }'

sleep 1

# 5. Message Processing Time Histogram
echo "Sending message processing time histogram..."
curl -X POST ${ENDPOINT}/v1/metrics \
  -H "Content-Type: application/json" \
  -d '{
    "resourceMetrics": [{
      "resource": {
        "attributes": [
          {"key": "service.name", "value": {"stringValue": "payment-processor"}},
          {"key": "host.name", "value": {"stringValue": "worker-03"}}
        ]
      },
      "scopeMetrics": [{
        "metrics": [{
          "name": "message_processing_duration_seconds",
          "description": "Message processing duration in seconds",
          "unit": "s",
          "histogram": {
            "dataPoints": [{
              "timeUnixNano": "'$(date +%s)'000000000",
              "count": "350",
              "sum": 892.5,
              "bucketCounts": ["20", "50", "100", "80", "60", "30", "10"],
              "explicitBounds": [0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0],
              "attributes": [
                {"key": "messaging.system", "value": {"stringValue": "kafka"}},
                {"key": "messaging.operation", "value": {"stringValue": "process"}},
                {"key": "message.type", "value": {"stringValue": "payment-event"}}
              ]
            }],
            "aggregationTemporality": 2
          }
        }]
      }]
    }]
  }'

echo ""
echo "✅ All histogram examples sent successfully!"
echo "Check the UI at http://localhost:8080"
