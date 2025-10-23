# OTLP Cardinality Checker - Architecture

## System Overview

The OTLP Cardinality Checker is designed as a standalone service that acts as an **OTLP endpoint destination**. It receives telemetry data from an OpenTelemetry Collector (which handles various input sources), extracts metadata structure, and provides a REST API for querying the collected metadata.

```
┌─────────────────────────────────────────────────────┐
│        OpenTelemetry Collector                      │
│  (Handles: Kafka, Redis, Prometheus, etc.)         │
│                                                     │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐      │
│  │ Receivers│   │Processors│   │ Exporters│      │
│  │ (Various)│──▶│          │──▶│   OTLP   │      │
│  └──────────┘   └──────────┘   └─────┬────┘      │
└───────────────────────────────────────┼────────────┘
                                        │ OTLP (gRPC/HTTP)
                                        ▼
┌─────────────────────────────────────────────────────┐
│         OTLP Cardinality Checker                    │
│         (OTLP Endpoint - Metadata Analysis)         │
│                                                     │
│  ┌──────────────┐    ┌──────────────┐            │
│  │OTLP Endpoints│───▶│   Analyzer   │            │
│  │ gRPC / HTTP  │    │   (Extract   │            │
│  │  Servers     │    │   Metadata)  │            │
│  └──────────────┘    └───────┬──────┘            │
│                              │                    │
│                      ┌───────▼──────┐            │
│                      │    Storage    │            │
│                      │  (In-Memory / │            │
│                      │   PostgreSQL) │            │
│                      └───────┬──────┘            │
│                              │                    │
│                      ┌───────▼──────┐            │
│                      │   REST API   │            │
│                      └──────────────┘            │
└─────────────────────────────────────────────────────┘
         │
         ▼
   ┌──────────┐
   │  Users   │
   │  (Query) │
   └──────────┘
```

## High-Level Architecture

```
┌─────────────────┐
│  Applications   │
│  with OTel SDK  │
└────────┬────────┘
         │ OTLP
         ↓
┌─────────────────────────────────────────┐
│  OTLP Cardinality Checker               │
│                                         │
│  ┌─────────────────────────────────┐  │
│  │  OTLP Receiver Layer            │  │
│  │  └─ HTTP Server (port 4318)     │  │
│  │     (gRPC planned for Phase 2)  │  │
│  └──────────────┬──────────────────┘  │
│                 │                      │
│  ┌──────────────▼──────────────────┐  │
│  │  Protocol Decoder               │  │
│  │  (OTLP Protobuf → Structs)      │  │
│  └──────────────┬──────────────────┘  │
│                 │                      │
│  ┌──────────────▼──────────────────┐  │
│  │  Metadata Extractor             │  │
│  │  ├─ Metrics Analyzer            │  │
│  │  ├─ Traces Analyzer             │  │
│  │  └─ Logs Analyzer               │  │
│  └──────────────┬──────────────────┘  │
│                 │                      │
│  ┌──────────────▼──────────────────┐  │
│  │  Storage Layer                  │  │
│  │  ├─ In-Memory Store (Primary)   │  │
│  │  └─ PostgreSQL (Optional)       │  │
│  └──────────────┬──────────────────┘  │
│                 │                      │
│  ┌──────────────▼──────────────────┐  │
│  │  Query API                      │  │
│  │  └─ REST API (port 8080)        │  │
│  └─────────────────────────────────┘  │
└─────────────────────────────────────────┘
         │
         ↓
┌─────────────────┐
│  Users/Tools    │
│  (curl, UI)     │
└─────────────────┘
```

## Component Design

### 1. OTLP Endpoint Layer

**Ansvar**: Ta emot OTLP data från OpenTelemetry Collector via HTTP (gRPC kommer i Phase 2)

**Viktigt**: Vi implementerar INTE en full OpenTelemetry Collector receiver. Vi bygger en enkel HTTP server som förstår OTLP protobuf-format. OpenTelemetry Collector använder sina OTLP HTTP exporter för att skicka data till oss.

**Implementation**:
```go
package receiver

// Uses official opentelemetry-go libraries
// - go.opentelemetry.io/collector/receiver/otlpreceiver
// Alternative: Custom implementation using protobuf definitions
```

**Beslut**: **Använd OpenTelemetry Collector SDK components**
- **Pro**: Proven, maintained, spec-compliant
- **Pro**: Automatic updates när OTLP spec ändras
- **Pro**: Battle-tested i production
- **Con**: Något större dependency footprint

**Configuration**:
```yaml
receiver:
  grpc:
    endpoint: "0.0.0.0:4317"
    max_recv_msg_size_mib: 32
  http:
    endpoint: "0.0.0.0:4318"
    max_request_body_size: 33554432  # 32MB
```

### 2. Protocol Decoder

**Ansvar**: Konvertera OTLP protobuf till interna datastrukturer

**Key Types**:
```go
package decoder

import (
    "go.opentelemetry.io/collector/pdata/pmetric"
    "go.opentelemetry.io/collector/pdata/ptrace"
    "go.opentelemetry.io/collector/pdata/plog"
)

// Använder officiella pdata packages
// Ger direktaccess till OTLP datastrukturer
```

### 3. Metadata Extractor

**Ansvar**: Extrahera metadata-nycklar från telemetri

**Design Philosophy**: 
- Ingen värde-lagring (för att undvika kardinalitetsproblem i verktyget)
- Endast unika nycklar per signal type
- Gruppera per resource (service.name, deployment.environment, etc.)

#### 3.1 Metrics Analyzer

**Extraherar**:
```go
type MetricMetadata struct {
    Name              string            // Metric name
    Type              string            // Gauge, Counter, Histogram, etc.
    Unit              string            // Optional unit
    Description       string            // Metric description
    LabelKeys         []string          // Unika label keys
    ResourceKeys      []string          // Resource attribute keys
    ScopeInfo         ScopeMetadata     // Instrumentation scope
    FirstSeen         time.Time
    LastSeen          time.Time
    SampleCount       int64             // How many datapoints seen
}

type ScopeMetadata struct {
    Name    string
    Version string
}
```

**Exempel**:
```
Metric: http_server_duration
Type: Histogram
Unit: ms
LabelKeys: [method, status_code, route]
ResourceKeys: [service.name, service.version, deployment.environment]
```

#### 3.2 Traces Analyzer

**Extraherar**:
```go
type SpanMetadata struct {
    Name              string            // Span name
    Kind              string            // Client, Server, Internal, etc.
    AttributeKeys     []string          // Span attribute keys
    EventNames        []string          // Unique event names
    EventAttrKeys     map[string][]string  // Event -> attribute keys
    LinkAttrKeys      []string          // Link attribute keys
    ResourceKeys      []string          // Resource attribute keys
    ScopeInfo         ScopeMetadata
    FirstSeen         time.Time
    LastSeen          time.Time
    SpanCount         int64
}
```

#### 3.3 Logs Analyzer

**Extraherar**:
```go
type LogMetadata struct {
    SeverityText      string            // INFO, ERROR, etc.
    AttributeKeys     []string          // Log record attribute keys
    ResourceKeys      []string          // Resource attribute keys
    ScopeInfo         ScopeMetadata
    FirstSeen         time.Time
    LastSeen          time.Time
    RecordCount       int64
}
```

**Design Note**: Body ignoreras helt eftersom det kan innehålla arbiträr data

### 4. Storage Layer

**Design Decision**: **Primary in-memory, Optional PostgreSQL**

#### 4.1 In-Memory Store

**Struktur**:
```go
package storage

type MetadataStore struct {
    mu sync.RWMutex
    
    // Metrics: metricName -> MetricMetadata
    metrics map[string]*MetricMetadata
    
    // Spans: spanName -> SpanMetadata
    spans map[string]*SpanMetadata
    
    // Logs: severity -> LogMetadata (grouped by severity)
    logs map[string]*LogMetadata
    
    // Resource-based indexing
    serviceIndex map[string][]string  // service.name -> [metric/span names]
}

func (s *MetadataStore) AddMetric(m *MetricMetadata) error
func (s *MetadataStore) GetMetric(name string) (*MetricMetadata, bool)
func (s *MetadataStore) ListMetrics(filter Filter) []*MetricMetadata
```

**Fördelar**:
- Extrem snabb läsning/skrivning
- Ingen external dependency
- Perfekt för utveckling och CI/CD
- Låg latency för queries

**Memory Estimation**:
- 1000 unique metrics × ~500 bytes = ~500KB
- 1000 unique spans × ~500 bytes = ~500KB
- Total: < 5MB för typisk microservice

#### 4.2 PostgreSQL Store (Optional)

**Schema**:
```sql
-- Metrics metadata
CREATE TABLE metric_metadata (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    unit VARCHAR(50),
    description TEXT,
    label_keys TEXT[] NOT NULL,
    resource_keys TEXT[] NOT NULL,
    scope_name VARCHAR(255),
    scope_version VARCHAR(50),
    first_seen TIMESTAMPTZ NOT NULL,
    last_seen TIMESTAMPTZ NOT NULL,
    sample_count BIGINT DEFAULT 0,
    UNIQUE(name)
);

CREATE INDEX idx_metric_name ON metric_metadata(name);
CREATE INDEX idx_metric_last_seen ON metric_metadata(last_seen);

-- Span metadata
CREATE TABLE span_metadata (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    kind VARCHAR(50) NOT NULL,
    attribute_keys TEXT[] NOT NULL,
    event_names TEXT[],
    resource_keys TEXT[] NOT NULL,
    scope_name VARCHAR(255),
    scope_version VARCHAR(50),
    first_seen TIMESTAMPTZ NOT NULL,
    last_seen TIMESTAMPTZ NOT NULL,
    span_count BIGINT DEFAULT 0,
    UNIQUE(name)
);

CREATE INDEX idx_span_name ON span_metadata(name);

-- Log metadata
CREATE TABLE log_metadata (
    id BIGSERIAL PRIMARY KEY,
    severity_text VARCHAR(50) NOT NULL,
    attribute_keys TEXT[] NOT NULL,
    resource_keys TEXT[] NOT NULL,
    scope_name VARCHAR(255),
    scope_version VARCHAR(50),
    first_seen TIMESTAMPTZ NOT NULL,
    last_seen TIMESTAMPTZ NOT NULL,
    record_count BIGINT DEFAULT 0,
    UNIQUE(severity_text)
);
```

**Sync Strategy**:
```go
// Periodisk batch-write från in-memory till PostgreSQL
type PersistenceManager struct {
    store    *MetadataStore
    db       *sql.DB
    interval time.Duration  // Default: 30s
}

func (pm *PersistenceManager) SyncLoop(ctx context.Context) {
    ticker := time.NewTicker(pm.interval)
    for {
        select {
        case <-ticker.C:
            pm.syncToDatabase()
        case <-ctx.Done():
            return
        }
    }
}
```

### 5. Query API

**Design**: Simple REST API

**Endpoints**:
```
GET  /api/v1/metrics              List all metrics metadata
GET  /api/v1/metrics/{name}       Get specific metric metadata
GET  /api/v1/spans                List all spans metadata
GET  /api/v1/spans/{name}         Get specific span metadata
GET  /api/v1/logs                 List all logs metadata
GET  /api/v1/services             List all services
GET  /api/v1/services/{name}      Get service metadata
GET  /api/v1/summary              Get overall summary
GET  /api/v1/health               Health check
```

**Query Parameters**:
```
?service=my-service           Filter by service name
?environment=production       Filter by environment
?since=2024-01-01            Filter by time
?limit=100                    Limit results
```

**Response Format**:
```json
{
  "metadata": {
    "total_count": 156,
    "filtered_count": 10,
    "timestamp": "2024-01-15T10:30:00Z"
  },
  "metrics": [
    {
      "name": "http_server_duration",
      "type": "Histogram",
      "unit": "ms",
      "label_keys": ["method", "status_code", "route"],
      "resource_keys": ["service.name", "service.version"],
      "scope": {
        "name": "go.opentelemetry.io/contrib/instrumentation/net/http",
        "version": "0.45.0"
      },
      "first_seen": "2024-01-15T10:00:00Z",
      "last_seen": "2024-01-15T10:30:00Z",
      "sample_count": 15420
    }
  ]
}
```

## Data Flow

### Metrics Path
```
1. OTLP gRPC/HTTP request arrives
2. Receiver decodes protobuf → pmetric.Metrics
3. For each ResourceMetrics:
   - Extract resource attributes keys
   - For each ScopeMetrics:
     - Extract scope info
     - For each Metric:
       - Extract metric name, type, unit
       - Extract all data point label keys
       - Merge into MetricMetadata
4. Store updates in-memory store
5. (Optional) Batch-sync to PostgreSQL
```

### Spans Path
```
1. OTLP request with traces
2. Decode to ptrace.Traces
3. For each ResourceSpans:
   - Extract resource keys
   - For each ScopeSpans:
     - For each Span:
       - Extract span name, kind
       - Extract attribute keys
       - Extract event names & keys
       - Merge into SpanMetadata
4. Store in memory
```

## Configuration

**File Format**: YAML

```yaml
# config.yaml
server:
  grpc:
    enabled: true
    address: "0.0.0.0:4317"
    max_message_size: 33554432  # 32MB
  
  http:
    enabled: true
    address: "0.0.0.0:4318"
    max_body_size: 33554432

api:
  address: "0.0.0.0:8080"
  read_timeout: 30s
  write_timeout: 30s

storage:
  type: "memory"  # or "postgres"
  
  postgres:
    enabled: false
    host: "localhost"
    port: 5432
    database: "otlp_cardinality"
    user: "otlp"
    password: "${DB_PASSWORD}"
    sync_interval: 30s
    
logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # json or console

retention:
  max_age: 720h  # 30 days
  cleanup_interval: 1h
```

## Concurrency Model

**Approach**: Actor-like pattern med channels

```go
// Receiver goroutines → analyzer channels → storage goroutines
type Pipeline struct {
    metricsChan chan pmetric.Metrics
    tracesChan  chan ptrace.Traces
    logsChan    chan plog.Logs
    
    analyzers struct {
        metrics *MetricsAnalyzer
        traces  *TracesAnalyzer
        logs    *LogsAnalyzer
    }
    
    store *MetadataStore
}

func (p *Pipeline) Start(ctx context.Context) error {
    // Start analyzer workers
    go p.analyzers.metrics.Process(ctx, p.metricsChan, p.store)
    go p.analyzers.traces.Process(ctx, p.tracesChan, p.store)
    go p.analyzers.logs.Process(ctx, p.logsChan, p.store)
    
    // Start OTLP receivers
    // ...
}
```

## Error Handling

**Philosophy**: Never lose metadata due to errors

```go
// Graceful degradation
type Analyzer interface {
    Process(ctx context.Context, data interface{}, store *MetadataStore) error
}

// If a single metric fails to parse, log error but continue
// If storage write fails, retry with backoff
// If PostgreSQL unavailable, continue with in-memory only
```

## Performance Considerations

### Bottlenecks
1. **OTLP parsing**: Use protobuf directly, avoid reflection
2. **Map locking**: Use sync.RWMutex, minimize critical sections
3. **Memory growth**: Periodic cleanup of old metadata

### Optimizations
1. **Batch processing**: Process multiple datapoints before lock
2. **String interning**: Reuse common strings (label keys)
3. **Bloom filters**: Quick existence checks before full lookup

### Benchmarks Target
```
Throughput: 10,000 spans/sec on 2-core machine
Latency: p99 < 10ms for metadata extraction
Memory: < 100MB for 10k unique metadata entries
```

## Security

**Considerations**:
1. **No authentication in MVP** - deploy in trusted networks
2. **No PII storage** - only metadata keys, never values
3. **Rate limiting** - prevent DoS
4. **Input validation** - validate OTLP messages

**Future**: TLS, API keys, mTLS

## Deployment

### Docker
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o otlp-cardinality-checker ./cmd/server

FROM alpine:3.18
COPY --from=builder /app/otlp-cardinality-checker /usr/local/bin/
EXPOSE 4317 4318 8080
CMD ["otlp-cardinality-checker"]
```

### Kubernetes
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: otlp-cardinality-checker
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: checker
        image: otlp-cardinality-checker:latest
        ports:
        - containerPort: 4317  # gRPC
        - containerPort: 4318  # HTTP
        - containerPort: 8080  # API
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

## Testing Strategy

### Unit Tests
- Metadata extraction logic
- Storage operations
- API handlers

### Integration Tests
- End-to-end OTLP → storage → API
- PostgreSQL persistence
- Concurrent requests

### Load Tests
```bash
# Generate OTLP traffic
go run ./tests/loadgen --rate 1000 --duration 60s
```

## Alternatives & Trade-offs

### Why not use OpenTelemetry Collector?
**Considered**: Extend OTel Collector with custom processor

**Decision**: Build standalone tool
- **Pro**: Simpler for users - single binary
- **Pro**: Optimized for metadata extraction only
- **Pro**: Easier to develop UI/API
- **Con**: More code to maintain

### Why not store actual values?
**Decision**: Only keys, never values
- **Pro**: Prevents tool itself from having cardinality issues
- **Pro**: Lower memory/storage requirements
- **Pro**: Faster queries
- **Con**: Can't detect actual high-cardinality values

This is acceptable because the goal is to understand **structure**, not analyze values.

### Why in-memory primary?
**Decision**: Memory-first, DB-optional
- **Pro**: Zero dependencies for basic usage
- **Pro**: Fast development iteration
- **Pro**: Low latency queries
- **Con**: Lost on restart (mitigated by optional persistence)

## Future Enhancements

1. **Cardinality Estimation**
   - Add HyperLogLog for approximate unique counts
   - Track label value cardinality per metric

2. **Change Detection**
   - Alert when new labels appear
   - Track label additions/removals over time

3. **UI Dashboard**
   - Visual exploration of metadata
   - Drill-down by service

4. **Sampling**
   - Sample high-volume telemetry
   - Configurable sampling rates

## References

- [OpenTelemetry Collector Architecture](https://opentelemetry.io/docs/collector/architecture/)
- [OTLP Specification](https://opentelemetry.io/docs/specs/otlp/)
- [Go Concurrency Patterns](https://go.dev/blog/pipelines)
