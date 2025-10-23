# API Documentation

## Pagination

All list endpoints support pagination to handle large datasets efficiently.

### Query Parameters

- `limit` (optional): Number of items per page (default: 100, max: 1000)
- `offset` (optional): Number of items to skip (default: 0)
- `service` (optional): Filter by service name

### Response Format

```json
{
  "data": [...],          // Array of items for current page
  "total": 10000,         // Total number of items
  "limit": 100,           // Items per page
  "offset": 0,            // Current offset
  "has_more": true        // Whether there are more pages
}
```

### Examples

#### Get first page (default 100 items)
```bash
curl http://localhost:8080/api/v1/metrics
```

#### Get specific page
```bash
curl "http://localhost:8080/api/v1/metrics?limit=50&offset=100"
```

#### Filter by service with pagination
```bash
curl "http://localhost:8080/api/v1/metrics?service=my-service&limit=100&offset=0"
```

#### Navigate pages programmatically
```bash
#!/bin/bash
offset=0
limit=100

while true; do
  response=$(curl -s "http://localhost:8080/api/v1/metrics?limit=$limit&offset=$offset")
  echo "$response" | jq '.data[].name'
  
  has_more=$(echo "$response" | jq -r '.has_more')
  if [ "$has_more" != "true" ]; then
    break
  fi
  
  offset=$((offset + limit))
done
```

## Endpoints

### Metrics

#### List all metrics
```
GET /api/v1/metrics?limit=N&offset=M&service=NAME
```

Response:
```json
{
  "data": [
    {
      "name": "http_requests_total",
      "type": "Sum",
      "unit": "1",
      "description": "Total HTTP requests",
      "label_keys": {
        "status": {
          "count": 100,
          "percentage": 100,
          "estimated_cardinality": 5,
          "value_samples": ["200", "201", "400", "404", "500"],
          "first_seen": "2025-10-23T20:00:00Z",
          "last_seen": "2025-10-23T21:00:00Z"
        }
      },
      "resource_keys": {...},
      "first_seen": "2025-10-23T20:00:00Z",
      "last_seen": "2025-10-23T21:00:00Z",
      "sample_count": 100,
      "services": {
        "my-service": 100
      }
    }
  ],
  "total": 3,
  "limit": 100,
  "offset": 0,
  "has_more": false
}
```

#### Get specific metric
```
GET /api/v1/metrics/{name}
```

Response: Single metric object (not paginated)

### Spans

#### List all spans
```
GET /api/v1/spans?limit=N&offset=M&service=NAME
```

Response format: Same as metrics, but with span metadata

#### Get specific span
```
GET /api/v1/spans/{name}
```

### Logs

#### List all log metadata
```
GET /api/v1/logs?limit=N&offset=M&service=NAME
```

Response format: Same as metrics, but with log metadata grouped by severity

#### Get specific log by severity
```
GET /api/v1/logs/{severity}
```

### Services

#### List all services
```
GET /api/v1/services
```

Response: Array of service names (not paginated, typically small)

#### Get service overview
```
GET /api/v1/services/{name}/overview
```

Response: Complete telemetry footprint for a service
```json
{
  "service_name": "my-service",
  "metrics": [...],   // All metrics from this service
  "spans": [...],     // All spans from this service
  "logs": [...]       // All logs from this service
}
```

Note: Service overview is **not paginated** as it's meant to show the complete footprint. For large services with many metrics, consider using the filtered list endpoints instead:
```bash
curl "http://localhost:8080/api/v1/metrics?service=my-service&limit=100"
```

### Health

#### Health check
```
GET /health
```

Response:
```json
{
  "status": "ok"
}
```

## Performance Considerations

### Optimal Page Sizes

- **Default (100)**: Good balance for web UI pagination
- **Small (10-50)**: Better for real-time updates, lower latency
- **Large (500-1000)**: Fewer requests, but higher memory usage per request

### Response Time Expectations

With in-memory storage:
- List 100 items: <10ms
- List 1,000 items: <50ms
- Get single item: <1ms

### Best Practices

1. **Always use pagination** when listing resources
2. **Start with default limit** (100) and adjust based on needs
3. **Filter by service** when possible to reduce dataset size
4. **Cache responses** if data doesn't change frequently
5. **Use offset-based pagination** for simple navigation
6. **Consider cursor-based pagination** for very large datasets (future enhancement)

## Cardinality Metadata

### Understanding the Response Fields

#### `estimated_cardinality`
- Number of unique values observed for this key
- Accurate up to 100 unique values (MaxSamples)
- Beyond 100 values, continues counting but stops storing samples

#### `value_samples`
- Up to 100 example values
- **Always sorted** for consistency
- Useful for spotting cardinality issues

#### `percentage`
- Percentage of samples that include this key
- 100% = always present
- <100% = optional or conditionally added

#### `count`
- Number of times this key was observed
- Increases even if cardinality is maxed out

### Identifying High Cardinality

```bash
# Find metrics with high cardinality labels
curl -s "http://localhost:8080/api/v1/metrics" | \
  jq '.data[] | select(.label_keys | to_entries[] | .value.estimated_cardinality > 100) | {name, high_card_labels: [.label_keys | to_entries[] | select(.value.estimated_cardinality > 100) | .key]}'
```

### Spotting Optional Labels

```bash
# Find labels that are not always present
curl -s "http://localhost:8080/api/v1/metrics/http_requests_total" | \
  jq '.label_keys | to_entries[] | select(.value.percentage < 100) | {key: .key, percentage: .value.percentage}'
```
