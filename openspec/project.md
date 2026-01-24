# Project Context

## Purpose

OTLP Cardinality Checker is a **metadata analysis tool** for OpenTelemetry Protocol (OTLP) telemetry. It acts as an **OTLP endpoint destination** that receives telemetry from an OpenTelemetry Collector and inspects structural characteristics of incoming metrics, traces, and logs — **without persisting any raw telemetry payloads**.

It serves as a diagnostic tool to:
- Identify high-cardinality attributes before full observability ingestion
- Analyze attribute structure and usage patterns across all signal types
- Detect log patterns using Drain algorithm for message grouping
- Explore service metadata, span coverage, and complexity scores interactively
- Estimate cardinality using HyperLogLog for memory-efficient tracking

**Key Principle**: We store ONLY metadata structure (keys, not values) to avoid cardinality explosion in the tool itself.

### Architecture Flow
```
Sources (Kafka/Redis/Prometheus) → OpenTelemetry Collector → This Tool (OTLP Endpoints) → REST API
```

**Implementation Note**: We implement simple OTLP HTTP/gRPC servers (not full Collector receivers). The Collector sends us data via its OTLP exporters.

## Tech Stack

### Backend (Go 1.24+)
- **Language**: Go 1.24.0 (toolchain 1.24.9)
- **HTTP Router**: chi v5 (`github.com/go-chi/chi/v5`)
- **OTLP Protocol**: `go.opentelemetry.io/proto/otlp` v1.8.0
- **gRPC**: `google.golang.org/grpc` v1.76.0
- **Protobuf**: `google.golang.org/protobuf` v1.36.10
- **ClickHouse**: `github.com/ClickHouse/clickhouse-go/v2` v2.40.3 (columnar database)
- **Cardinality Estimation**: HyperLogLog algorithm (custom implementation) + ClickHouse `uniqExact()`
- **Log Templating**: Drain algorithm for pattern extraction (20-30k+ EPS)

### Frontend (React + Vite)
- **Framework**: React 18.2.0
- **Build Tool**: Vite 5.0.0
- **Bundler**: Vite with React plugin (`@vitejs/plugin-react`)
- **Language**: JavaScript (JSX)
- **Styling**: Vanilla CSS with index.css

### Storage
- **Primary**: ClickHouse (default, production-grade columnar database)
- **Alternative**: In-memory storage (fast, 500k+ metrics, no persistence)
- **Storage Modes**: `clickhouse` (default) or `memory`
- **Batch Writes**: Automatic buffering with configurable batch size and flush interval

### Deployment
- **Container**: Docker (Dockerfile in root)
- **Orchestration**: Kubernetes (manifests in `k8s/`)
- **Build System**: Makefile + Go build

### Testing & Load Testing
- **Load Testing**: k6 (JavaScript-based HTTP load testing)
- **Test Scripts**: `scripts/` directory with k6 scripts for metrics, traces, logs

## Project Conventions

### Code Style

**Naming:**
- Packages: lowercase, single word (e.g., `analyzer`, `storage`)
- Interfaces: `-er` suffix for single method (e.g., `Analyzer`, `Storage`)
- Structs: PascalCase (e.g., `MetricMetadata`, `AttributeMetadata`)
- Functions: camelCase for private, PascalCase for exported (e.g., `extractKeys`, `ExtractMetricKeys`)

**Formatting:**
- Use `gofmt` for all Go code
- **No emojis** in code, comments, or commit messages unless specifically requested
- Clear, descriptive variable and function names
- Prefer explicit over clever code

**Error Handling:**
- Always wrap errors with context: `fmt.Errorf("processing metric %s: %w", name, err)`
- Never ignore errors (no `_, _` patterns)
- Return errors, never panic (except in init/startup validation)

**Context Usage:**
- Context is always the first parameter: `func (s *Store) Get(ctx context.Context, key string)`
- Check `ctx.Done()` in loops and long-running operations

### Architecture Patterns

**Layered Architecture:**
```
OTLP Endpoint Layer → Analyzer Layer → Storage Layer → API Layer
```

1. **OTLP Endpoints** (`internal/receiver/`): Simple HTTP/gRPC servers accepting OTLP protobuf
2. **Analyzer** (`internal/analyzer/`): Extracts metadata keys from telemetry signals
3. **Storage** (`internal/storage/`): ClickHouse primary (with batch buffering), optional in-memory mode
4. **API** (`internal/api/`): REST endpoints for querying metadata

**Concurrency Model:**
- Use **channel-based pipelines** for data flow
- Pattern: `inputChan → Process() → outputChan`
- Check `ctx.Done()` in select statements for cancellation

**Thread Safety:**
- Use `sync.RWMutex` for read-heavy workloads
- Minimize critical sections
- Prefer channels over shared memory for goroutine communication

**Metadata Structures:**
- All metadata includes `*Count int64` (number of times seen)
- Timestamps: FirstSeen and LastSeen (time.Time) for tracking lifecycle
- Always sort keys for consistent output
- Merge keys using set union when updating

**Complexity Score Calculation:**
```
complexity_score = total_keys × max_cardinality
```
Where:
- `total_keys` = attribute_keys + resource_keys + event_keys + link_keys
- `max_cardinality` = highest cardinality among all keys in the signal

This score helps identify over-instrumented signals that create storage/query burden.

**Key Extraction Pattern:**
```go
func extractKeys(attrs map[string]interface{}) []string {
    keys := make([]string, 0, len(attrs))
    for k := range attrs {
        keys = append(keys, k)
    }
    sort.Strings(keys) // Always sort
    return keys
}
```

### Testing Strategy

**Table-Driven Tests:**
```go
tests := []struct {
    name    string
    input   Type
    want    Result
    wantErr bool
}{
    {name: "case1", input: ..., want: ...},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) { ... })
}
```

**Test Helpers:**
- Create helpers in `testutil/` for test data generation (e.g., `CreateTestMetric`)

**Mocking:**
- Define interfaces for dependencies to enable mocking
- Example: `Storage` interface with `mockStorage` implementation

**Load Testing:**
- Use k6 scripts in `scripts/` directory
- Test realistic workloads (50k+ observations, 1000+ unique metrics)
- Profile with pprof during load tests to identify bottlenecks

### Git Workflow

**Feature Branch Workflow:**
1. Always create feature branch from main: `git checkout -b feature/descriptive-name`
2. Work and commit frequently on feature branch
3. Create Pull Request when ready (never merge directly to main)
4. Merge via PR, then delete feature branch

**Commit Frequently:**
- Commit immediately after: tests pass, build succeeds, feature works, bug fixed
- Small, frequent commits > large, infrequent ones

**Semantic Commit Messages:**
```bash
# Format: <type>: <description>
# Types: feat, fix, refactor, test, docs, chore, perf, style

feat: add log body template extraction
fix: correct percentage calculation in logs analyzer
refactor: extract pattern matching to separate analyzer
test: add benchmark for template extraction
docs: update API documentation
chore: update K6 tests to write-only mode
perf: optimize regex pattern compilation
```

**Branch naming:**
- `feature/description` - New features
- `fix/description` - Bug fixes
- `refactor/description` - Code refactoring
- `docs/description` - Documentation updates

## Domain Context

### OpenTelemetry Protocol (OTLP)

**OTLP Signals:**
- **Metrics**: Gauges, counters, histograms, summaries with labels/attributes
- **Traces**: Spans with attributes, events, and links
- **Logs**: Log records with attributes, severity, body

**OTLP Endpoints:**
- gRPC: port 4317 (binary protobuf)
- HTTP: port 4318 (JSON or binary protobuf)

**Attribute Structure:**
- **Resource Attributes**: Identify the source (service.name, host.name, etc.)
- **Scope Attributes**: Identify the instrumentation library
- **Data Point Attributes**: Signal-specific metadata (metric labels, span attributes, log fields)

### Cardinality

**High Cardinality**: Attribute keys with many unique values (e.g., user_id, request_id)
- Causes: Storage explosion, query slowdown, high costs
- Thresholds: >1000 unique values = high, >100 = medium, ≤100 = low

**Cardinality Estimation**: Use HyperLogLog algorithm for memory-efficient estimation
- Accurate to ~2% error with minimal memory (<2KB per metric)

### Log Templating (Drain Algorithm)

**Purpose**: Extract patterns from log bodies to identify unique log types without storing full text
- Example: `"User 123 logged in"` + `"User 456 logged in"` → `"User <*> logged in"`
- Performance: 20-30k+ events per second (EPS), up to 53k-1.6M EPS in benchmarks

**Drain Algorithm Implementation:**
1. **First pass**: Logs are grouped by severity (info, error, etc.) and emitting service
2. **Second pass**: Drain-style structure mining:
   - Log lines are tokenized (space-delimited by default)
   - Common structure is extracted into templates
   - Variable fields are isolated and replaced with wildcards (e.g., `<*>`)
   - Each pattern is tracked with:
     - Count of matching messages
     - Extracted field keys (attribute + resource)
     - Service and severity grouping
     - Example log body

**Pre-Masking Patterns**: Before Drain extraction, specific patterns are masked:
- Apache timestamps: `[Weekday Month DD HH:MM:SS YYYY]`
- Syslog timestamps: `MMM DD HH:MM:SS`
- ISO timestamps: `YYYY/MM/DD HH:MM:SS`
- UUIDs, emails, URLs, IPs, durations, hex strings, numbers

**Pattern Configuration**: Patterns defined in `config/patterns.yaml`, loaded at startup

**Similarity Threshold**: 0.7 (increased from 0.5 for more specific templates)

This allows pattern-based aggregation and analysis **without full-text log storage**.

## Web Interface (SPA)

The frontend is a single-page React application with multiple views for exploring telemetry metadata:

### Main Tabs

1. **Dashboard (Overview)**
   - High-level counters: total metrics, logs, traces, services
   - Quick system-wide insight
   - Entry point for navigation

2. **Metrics View**
   - Metric name, type (gauge/counter/histogram/etc.), unit, sample count
   - Complexity score: `total_keys × max_cardinality`
   - Color-coded complexity badges: Green (<50), Orange (50-200), Red (>200)
   - Click metric → Details view with attribute breakdown

3. **Metrics Overview**
   - Similar to Metrics View but with different layout/filtering
   - Sortable columns (click headers)

4. **Traces View**
   - Span name listing
   - Number of spans, attribute keys, resource-level attributes
   - Service-span mapping
   - Similar architecture to Metrics tab

5. **Logs View**
   - Grouped by severity (info, error, critical, etc.)
   - Shows Drain-extracted patterns
   - Click pattern → TemplateDetails view with service breakdown

6. **Attributes Tab** (NEW)
   - Global attribute catalog across ALL signals
   - Shows cardinality, sample values, signal types (metric/span/log)
   - Filters: signal type, scope (resource/attribute), min cardinality, search
   - Sortable by cardinality, count, key name
   - Color-coded badges: High (>1000), Medium (100-1000), Low (≤100)

7. **Service Explorer**
   - List of services (by `service.name`)
   - Per-service metrics, spans, logs breakdown

8. **High Cardinality**
   - Cross-signal cardinality analysis
   - Shows attribute keys with high unique value counts
   - Threshold-based filtering

9. **Noisy Neighbors**
   - Identifies services/signals generating high data volumes
   - Helps pinpoint over-instrumentation

10. **Metadata Complexity**
    - Displays signals sorted by complexity score
    - Includes signal type, name, total keys, max cardinality
    - Useful for identifying over-instrumented signals

11. **Memory View**
    - Backend runtime metrics: RAM usage, CPU, ingestion rate
    - Useful for performance tuning and debugging

### Component Structure
- All views are functional React components (JSX)
- State management via `useState` and `useEffect` hooks
- Fetches data from REST API (`/api/v1/*`)
- Styling: Vanilla CSS (no external UI frameworks)

## REST API Endpoints

The API layer exposes metadata via HTTP endpoints on port 8080:

### Metrics
- `GET /api/v1/metrics` - List all metrics (paginated)
- `GET /api/v1/metrics/{name}` - Metric details with attribute breakdown

### Traces
- `GET /api/v1/spans` - List all span names
- `GET /api/v1/spans/{name}` - Span details with attributes

### Logs
- `GET /api/v1/logs` - List log patterns by severity
- `GET /api/v1/logs/{severity}` - Get log pattern for specific severity
- `GET /api/v1/logs/patterns` - Get all log patterns
- `GET /api/v1/logs/patterns/{severity}/{template}` - Pattern details with service breakdown
- `GET /api/v1/logs/by-service` - Service-based log navigation
- `GET /api/v1/logs/service/{service}/severity/{severity}` - Logs for specific service+severity

### Attributes (Global Catalog)
- `GET /api/v1/attributes` - List all attributes with filters (signal_type, scope, min_cardinality, sort_by, pagination)
- `GET /api/v1/attributes/{key}` - Details for specific attribute key

### Services
- `GET /api/v1/services` - List all known services (`service.name`)
- `GET /api/v1/services/{name}/overview` - Service overview with metrics/spans/logs counts

### Analysis
- `GET /api/v1/cardinality/high` - High-cardinality keys across signals (threshold, limit)
- `GET /api/v1/complexity` - Metadata complexity analysis (threshold, limit)

### Admin
- `POST /api/v1/admin/clear` - Clear all stored metadata (dangerous, use with caution)

### Health
- `GET /api/v1/health` - Health check endpoint

All endpoints support query parameters for filtering, sorting, and pagination where applicable.

## Important Constraints

### Performance
- Target: 30k-50k signals/second ingestion rate
- ClickHouse optimized for high-throughput batch writes and analytical queries
- In-memory mode available for 500k+ metrics without persistence
- Memory target: <256MB under normal load (memory mode)

### Memory Efficiency
- Store only metadata keys, never values (except samples: max 10 per attribute)
- Use HyperLogLog for cardinality estimation (~16KB per attribute, 0.81% error)
- ClickHouse `uniqExact()` for precise cardinality when needed
- Timestamps: FirstSeen and LastSeen per metadata object

### Data Safety
- Never store PII or sensitive data
- Only track attribute key names, not their values (except samples)
- Log pattern extraction replaces variable fields with wildcards

### Compatibility
- Must work as OTLP exporter destination (not Collector receiver)
- Accept data from OpenTelemetry Collector via OTLP exporters
- Support both HTTP (4318) and gRPC (4317) protocols
- Support compressed and uncompressed payloads

## External Dependencies

### OpenTelemetry Collector
- **Role**: Receives data from sources (Kafka, Redis, Prometheus, etc.)
- **Integration**: Sends data to this tool via OTLP HTTP/gRPC exporters
- **Configuration**: Collector must be configured with OTLP exporter pointing to this tool's endpoints

### Example Collector Config:
```yaml
exporters:
  otlphttp:
    endpoint: http://localhost:4318
    compression: gzip  # Optional: compressed payloads
    
service:
  pipelines:
    metrics:
      exporters: [otlphttp]
    traces:
      exporters: [otlphttp]
    logs:
      exporters: [otlphttp]
```

### Data Sources (Upstream of Collector)
- Kafka (logs, metrics, traces)
- Redis (metrics, traces)
- Prometheus (metrics scraping via Prometheus receiver)
- Application instrumentation (OTLP SDK direct)
- Filelog receiver (for log files)
- OTLP receiver (from SDKs)

### Observability Backends (Not Required)
- This tool is **standalone** and does not require a backend
- Can be used alongside Prometheus, Datadog, Grafana, etc. (parallel path)
- Acts as a "metadata tee" to inspect telemetry structure before ingestion

### Development Tools
- **k6** - Load testing framework (JavaScript-based HTTP testing)
- **pprof** - Go profiling tool (CPU, memory, goroutine profiling)
- **Browser** - Required to render React frontend (Chrome/Firefox/Safari)

## Known Issues and Future Improvements

### Batch Write Performance
**Implemented**: ClickHouse batch buffer with automatic flushing
- Configurable batch size (default: 1000 rows)
- Configurable flush interval (default: 5 seconds)
- Reduces write operations significantly compared to synchronous writes
- ClickHouse columnar storage optimized for analytical queries

**Performance**: 10x write throughput compared to previous SQLite implementation

### Log Pattern False Positives
**Issue**: Drain algorithm may occasionally group dissimilar logs if they share token structure

**Mitigation**: Similarity threshold set to 0.7 (high precision) and pre-masking patterns help

### Memory Growth
**Issue**: In-memory storage grows unbounded with unique metric/span/log names

**Planned**: Add optional TTL/eviction policy for stale metadata

### UI State Management
**Issue**: No global state management (Redux/Context) - each component manages its own state

**Impact**: Some data refetched unnecessarily when switching tabs

**Planned**: Consider React Context or lightweight state management for shared data

## Development Setup

### Building
```bash
# Build binary
make build
# Or manually:
go build -o bin/occ ./cmd/server
```

### Running with ClickHouse (Default)
```bash
# Run with ClickHouse (default backend)
./bin/occ

# Custom ClickHouse address
CLICKHOUSE_ADDR=clickhouse:9000 ./bin/occ

# Server listens on:
# - OTLP gRPC: :4317
# - OTLP HTTP: :4318
# - REST API: :8080
# - pprof: :6060
```

### Running with In-Memory Storage
```bash
# Memory-only mode (no persistence)
STORAGE_BACKEND=memory ./bin/occ
```

### Running with Drain Log Templating
```bash
# Enable automatic log template extraction (default: enabled)
USE_AUTOTEMPLATE=true ./bin/occ

# Disable (falls back to basic log metadata)
USE_AUTOTEMPLATE=false ./bin/occ
```

### Frontend Development
```bash
cd web
npm install
npm run dev      # Vite dev server on :5173
npm run build    # Build to web/dist (static files served by Go server)
npm run preview  # Preview production build
```

### Load Testing
```bash
# Metrics load test
K6_VUS=50 K6_DURATION=1m k6 run scripts/load-test-metrics.js

# Traces load test
K6_VUS=50 K6_DURATION=1m k6 run scripts/load-test-traces.js

# Logs load test (with realistic data from file)
K6_VUS=50 K6_DURATION=30s BATCH=10 OTEL_COLLECTOR_URL=http://localhost:4318/v1/logs k6 run scripts/k6-send-logs.js

# Mixed load (all signal types)
k6 run scripts/k6-mixed-load-test.js
```

### Profiling
```bash
# Start server with pprof enabled (default port 6060)
./bin/occ

# In another terminal:
# CPU profile (30 seconds)
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof

# Heap profile
curl http://localhost:6060/debug/pprof/heap > heap.prof

# Analyze with pprof
go tool pprof -http=:8081 cpu.prof
go tool pprof -http=:8081 heap.prof

# Or view flame graph directly
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/profile?seconds=30
```

### Testing
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/analyzer/...

# Run benchmarks
go test -bench=. ./internal/analyzer/autotemplate/
```
