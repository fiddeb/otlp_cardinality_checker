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
          "value_samples": ["200", "201", "400", "404", "500"]
        }
      },
      "resource_keys": {...},
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

Response example (with span name patterns):
```json
{
  "name": "HTTP GET /api/users",
  "kind": 2,
  "kind_name": "Server",
  "attribute_keys": {...},
  "resource_keys": {...},
  "name_patterns": [
    {
      "template": "HTTP GET <URL>",
      "count": 4500,
      "percentage": 90.0,
      "examples": ["HTTP GET /api/users/123", "HTTP GET /api/users/456", "HTTP GET /api/users/789"]
    },
    {
      "template": "HTTP GET /api/users",
      "count": 500,
      "percentage": 10.0,
      "examples": ["HTTP GET /api/users"]
    }
  ],
  "sample_count": 5000,
  "services": {"my-service": 5000}
}
```

The `name_patterns` field shows patterns extracted from span names:
- `template`: The pattern with dynamic values replaced by placeholders (`<NUM>`, `<UUID>`, `<URL>`, etc.)
- `count`: Number of spans matching this pattern
- `percentage`: Percentage of total spans matching this pattern
- `examples`: Up to 3 example span names that matched this pattern

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

## Attribute Catalog

The attribute catalog provides a **global view** of all attribute keys across metrics, spans, and logs. This helps identify high-cardinality attributes that may be causing performance or cost issues in your observability system.

### List All Attributes

```
GET /api/v1/attributes
```

Query Parameters:
- `signal_type` (optional): Filter by signal type (`metric`, `span`, `log`)
- `scope` (optional): Filter by scope (`resource`, `attribute`, `both`)
- `min_cardinality` (optional): Minimum estimated cardinality (e.g., `1000` for high-cardinality only)
- `sort_by` (optional): Sort field (`cardinality`, `count`, `first_seen`, `last_seen`, `key`) (default: `cardinality`)
- `sort_direction` (optional): Sort direction (`asc`, `desc`) (default: `desc`)
- `page` (optional): Page number (1-indexed, default: 1)
- `page_size` (optional): Items per page (default: 50, max: 100)

Response:
```json
{
  "attributes": [
    {
      "key": "user_id",
      "count": 1234567,
      "estimated_cardinality": 10523,
      "value_samples": ["user_1", "user_42", "user_999", ...],
      "signal_types": ["metric", "span", "log"],
      "scope": "attribute",
      "first_seen": "2025-11-01T10:00:00Z",
      "last_seen": "2025-11-09T15:30:00Z"
    }
  ],
  "total": 342,
  "page": 1,
  "page_size": 50
}
```

#### Examples

**List all high-cardinality attributes (>1000 unique values):**
```bash
curl "http://localhost:8080/api/v1/attributes?min_cardinality=1000&sort_by=cardinality&sort_direction=desc"
```

**Find attributes only used in metrics:**
```bash
curl "http://localhost:8080/api/v1/attributes?signal_type=metric"
```

**Find resource attributes with high cardinality:**
```bash
curl "http://localhost:8080/api/v1/attributes?scope=resource&min_cardinality=100"
```

**Get second page of attributes sorted by count:**
```bash
curl "http://localhost:8080/api/v1/attributes?sort_by=count&sort_direction=desc&page=2&page_size=20"
```

**Identify cross-signal attributes (used in all signal types):**
```bash
curl -s "http://localhost:8080/api/v1/attributes" | \
  jq '.attributes[] | select(.signal_types | length == 3) | {key, cardinality: .estimated_cardinality, signals: .signal_types}'
```

### Get Specific Attribute

```
GET /api/v1/attributes/{key}
```

Response:
```json
{
  "key": "http.method",
  "count": 987654,
  "estimated_cardinality": 9,
  "value_samples": ["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "TRACE", "CONNECT"],
  "signal_types": ["metric", "span"],
  "scope": "attribute",
  "first_seen": "2025-11-01T10:00:00Z",
  "last_seen": "2025-11-09T15:30:00Z"
}
```

#### Examples

**Get details for a specific attribute:**
```bash
curl "http://localhost:8080/api/v1/attributes/http.method"
```

**Check if an attribute exists:**
```bash
curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/api/v1/attributes/user_id"
# Returns 200 if exists, 404 if not found
```

**Get cardinality for specific attribute:**
```bash
curl -s "http://localhost:8080/api/v1/attributes/user_id" | jq '.estimated_cardinality'
```

### Understanding Attribute Catalog Fields

#### `key`
The attribute key name (e.g., `user_id`, `http.method`, `service.name`)

#### `count`
Total number of times this attribute key was observed across all signals

#### `estimated_cardinality`
Estimated number of unique values for this attribute key using HyperLogLog algorithm:
- **Precision**: 14 (±0.81% error)
- **Memory**: ~16KB per attribute
- **Accuracy**: Very accurate even for millions of unique values

#### `value_samples`
Array of up to 10 sample values observed for this attribute:
- First 10 unique values encountered
- Useful for understanding what type of data the attribute contains
- **Not** a statistical sample - just the first few unique values

#### `signal_types`
Array of signal types where this attribute was observed:
- `metric`: Used in metric labels or resource attributes
- `span`: Used in span attributes or resource attributes
- `log`: Used in log attributes or resource attributes

#### `scope`
Where the attribute appears:
- `resource`: Only in resource attributes (e.g., `service.name`, `host.name`)
- `attribute`: Only in data-point attributes (e.g., `http.status_code`, `user_id`)
- `both`: Appears in both resource and data-point attributes

#### `first_seen` / `last_seen`
Timestamps for when this attribute key was first/last observed

### Use Cases

#### 1. Identify High-Cardinality Attributes
Find attributes that may be causing storage or performance issues:
```bash
curl -s "http://localhost:8080/api/v1/attributes?min_cardinality=10000&sort_by=cardinality&sort_direction=desc" | \
  jq '.attributes[] | {key, cardinality: .estimated_cardinality, signals: .signal_types}'
```

#### 2. Find Unused Attributes
Identify attributes that are rarely used (low count):
```bash
curl -s "http://localhost:8080/api/v1/attributes?sort_by=count&sort_direction=asc&page_size=20" | \
  jq '.attributes[] | {key, count, cardinality: .estimated_cardinality}'
```

#### 3. Audit Attribute Naming Conventions
Find attributes that don't follow semantic conventions:
```bash
curl -s "http://localhost:8080/api/v1/attributes" | \
  jq '.attributes[] | select(.key | test("^[a-z]") | not) | .key'
```

#### 4. Identify Resource vs Data Attributes
See which attributes are mixed between resource and data scopes:
```bash
curl -s "http://localhost:8080/api/v1/attributes?scope=both" | \
  jq '.attributes[] | {key, scope, signals: .signal_types}'
```

#### 5. Cross-Signal Attribute Analysis
Find attributes used across all three signal types:
```bash
curl -s "http://localhost:8080/api/v1/attributes" | \
  jq '[.attributes[] | select(.signal_types | length == 3)] | length'
```

### Performance Considerations

- **In-memory mode**: Sub-millisecond response times for queries
- **SQLite mode**: <100ms for most queries (uses indexes)
- **Pagination**: Always use pagination for large result sets
- **Filtering**: Apply filters (`signal_type`, `scope`, `min_cardinality`) to reduce result size

### Known Limitations

- **Sample values**: Only first 10 unique values are stored
- **SQLite performance**: Under high load, synchronous writes can be slow (future optimization planned)
- **No historical data**: Only tracks attributes from telemetry received while running

---

## Sessions API

Sessions allow you to save, load, and compare snapshots of metadata state.

### Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/sessions` | List all saved sessions |
| POST | `/api/v1/sessions` | Create a new session |
| GET | `/api/v1/sessions/{name}` | Get session metadata |
| DELETE | `/api/v1/sessions/{name}` | Delete a session |
| POST | `/api/v1/sessions/{name}/load` | Load session (replace current data) |
| POST | `/api/v1/sessions/{name}/merge` | Merge session into current data |
| GET | `/api/v1/sessions/{name}/export` | Download session as JSON |
| POST | `/api/v1/sessions/import` | Upload session from JSON |
| GET | `/api/v1/sessions/diff` | Compare two sessions |

### Create Session

```bash
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "name": "baseline-2026-01-25",
    "description": "Pre-deploy baseline",
    "signals": ["metrics", "traces"],
    "services": ["payment-service"]
  }'
```

Response:
```json
{
  "message": "Session created successfully",
  "session": {
    "name": "baseline-2026-01-25",
    "description": "Pre-deploy baseline",
    "created_at": "2026-01-25T10:30:00Z",
    "size_bytes": 1245678,
    "signals": ["metrics", "traces"],
    "stats": {
      "metric_count": 150,
      "span_count": 45,
      "log_count": 0,
      "attribute_count": 320
    }
  }
}
```

### List Sessions

```bash
curl http://localhost:8080/api/v1/sessions
```

Response:
```json
{
  "sessions": [
    {
      "name": "baseline-2026-01-25",
      "created_at": "2026-01-25T10:30:00Z",
      "size_bytes": 1245678,
      "stats": { ... }
    },
    {
      "name": "post-deploy-v2",
      "created_at": "2026-01-25T14:15:00Z",
      "size_bytes": 1567890,
      "stats": { ... }
    }
  ],
  "total": 2
}
```

### Compare Sessions (Diff)

```bash
curl "http://localhost:8080/api/v1/sessions/diff?from=baseline-2026-01-25&to=post-deploy-v2"
```

Response:
```json
{
  "from_session": "baseline-2026-01-25",
  "to_session": "post-deploy-v2",
  "summary": {
    "total_changes": 15,
    "critical": 2,
    "warning": 5,
    "info": 8
  },
  "changes": {
    "metrics": {
      "added": [
        {
          "name": "http_request_duration_new",
          "severity": "warning",
          "metadata": {
            "active_series": 5000,
            "label_count": 8
          }
        }
      ],
      "removed": [],
      "changed": [
        {
          "name": "user_sessions",
          "severity": "critical",
          "details": [
            {
              "field": "labels.user_id.cardinality",
              "from": 100,
              "to": 50000,
              "change_pct": 49900.0,
              "severity": "critical"
            }
          ]
        }
      ]
    },
    "spans": { ... },
    "logs": { ... }
  }
}
```

### Filter Diff by Severity

```bash
curl "http://localhost:8080/api/v1/sessions/diff?from=A&to=B&min_severity=warning"
```

### Load Session

Load a session, replacing the current in-memory data:

```bash
curl -X POST http://localhost:8080/api/v1/sessions/baseline-2026-01-25/load
```

### Merge Session

Merge a session into the current data (additive):

```bash
curl -X POST http://localhost:8080/api/v1/sessions/baseline-2026-01-25/merge
```

### Export/Import Sessions

Export for backup or sharing:
```bash
curl http://localhost:8080/api/v1/sessions/baseline-2026-01-25/export > backup.json
```

Import from file:
```bash
curl -X POST http://localhost:8080/api/v1/sessions/import \
  -H "Content-Type: application/json" \
  -d @backup.json
```

### Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `OCC_SESSION_DIR` | `./data/sessions` | Directory for session storage |
| `OCC_MAX_SESSION_SIZE` | `104857600` (100MB) | Maximum session file size |
| `OCC_MAX_SESSIONS` | `50` | Maximum number of saved sessions |

### Use Cases

#### Pre/Post Deploy Comparison

```bash
# Before deploy
curl -X POST http://localhost:8080/api/v1/sessions \
  -d '{"name": "pre-deploy-v2.1"}'

# ... deploy happens ...

# After deploy (wait for new telemetry)
curl -X POST http://localhost:8080/api/v1/sessions \
  -d '{"name": "post-deploy-v2.1"}'

# Compare
curl "http://localhost:8080/api/v1/sessions/diff?from=pre-deploy-v2.1&to=post-deploy-v2.1"
```

#### Multi-Signal Analysis

Save different signals from separate test runs and merge them:

```bash
# Day 1: Collect metrics
curl -X POST -d '{"name": "metrics-jan25", "signals": ["metrics"]}' \
  http://localhost:8080/api/v1/sessions

# Day 2: Collect traces
curl -X POST -d '{"name": "traces-jan26", "signals": ["traces"]}' \
  http://localhost:8080/api/v1/sessions

# Merge traces into metrics session view
curl -X POST http://localhost:8080/api/v1/sessions/metrics-jan25/load
curl -X POST http://localhost:8080/api/v1/sessions/traces-jan26/merge

# Save combined view
curl -X POST -d '{"name": "combined-jan25-26"}' \
  http://localhost:8080/api/v1/sessions
```
