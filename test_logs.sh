#!/bin/bash

# Send some test logs with different services and severities

# Service A - Error logs
curl -X POST http://localhost:4318/v1/logs \
  -H "Content-Type: application/json" \
  -d '{
    "resourceLogs": [{
      "resource": {
        "attributes": [{
          "key": "service.name",
          "value": {"stringValue": "payment-service"}
        }]
      },
      "scopeLogs": [{
        "logRecords": [
          {
            "timeUnixNano": "1609459200000000000",
            "severityNumber": 17,
            "severityText": "ERROR",
            "body": {"stringValue": "Payment failed for order 12345"},
            "attributes": [
              {"key": "order_id", "value": {"stringValue": "12345"}},
              {"key": "error_code", "value": {"stringValue": "INSUFFICIENT_FUNDS"}}
            ]
          },
          {
            "timeUnixNano": "1609459201000000000",
            "severityNumber": 17,
            "severityText": "ERROR",
            "body": {"stringValue": "Payment failed for order 67890"},
            "attributes": [
              {"key": "order_id", "value": {"stringValue": "67890"}},
              {"key": "error_code", "value": {"stringValue": "CARD_DECLINED"}}
            ]
          }
        ]
      }]
    }]
  }'

# Service A - Info logs
curl -X POST http://localhost:4318/v1/logs \
  -H "Content-Type: application/json" \
  -d '{
    "resourceLogs": [{
      "resource": {
        "attributes": [{
          "key": "service.name",
          "value": {"stringValue": "payment-service"}
        }]
      },
      "scopeLogs": [{
        "logRecords": [
          {
            "timeUnixNano": "1609459202000000000",
            "severityNumber": 9,
            "severityText": "INFO",
            "body": {"stringValue": "Payment processed successfully"},
            "attributes": [
              {"key": "order_id", "value": {"stringValue": "11111"}},
              {"key": "amount", "value": {"doubleValue": 99.99}}
            ]
          }
        ]
      }]
    }]
  }'

# Service B - Error logs
curl -X POST http://localhost:4318/v1/logs \
  -H "Content-Type: application/json" \
  -d '{
    "resourceLogs": [{
      "resource": {
        "attributes": [{
          "key": "service.name",
          "value": {"stringValue": "inventory-service"}
        }]
      },
      "scopeLogs": [{
        "logRecords": [
          {
            "timeUnixNano": "1609459203000000000",
            "severityNumber": 17,
            "severityText": "ERROR",
            "body": {"stringValue": "Item out of stock: SKU-12345"},
            "attributes": [
              {"key": "sku", "value": {"stringValue": "SKU-12345"}},
              {"key": "warehouse", "value": {"stringValue": "WEST"}}
            ]
          }
        ]
      }]
    }]
  }'

# Service B - Warn logs
curl -X POST http://localhost:4318/v1/logs \
  -H "Content-Type: application/json" \
  -d '{
    "resourceLogs": [{
      "resource": {
        "attributes": [{
          "key": "service.name",
          "value": {"stringValue": "inventory-service"}
        }]
      },
      "scopeLogs": [{
        "logRecords": [
          {
            "timeUnixNano": "1609459204000000000",
            "severityNumber": 13,
            "severityText": "WARN",
            "body": {"stringValue": "Low stock warning for item"},
            "attributes": [
              {"key": "sku", "value": {"stringValue": "SKU-99999"}},
              {"key": "current_stock", "value": {"intValue": "5"}}
            ]
          }
        ]
      }]
    }]
  }'

echo ""
echo "Test data sent! Check http://localhost:8080"
