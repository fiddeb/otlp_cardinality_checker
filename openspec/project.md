# Project Context

## Purpose

OTLP Cardinality Checker is a lightweight metadata analysis tool for OpenTelemetry Protocol (OTLP) telemetry. It helps teams understand the metadata structure of their observability data before it explodes costs in production backends.

**Core Goals:**
- Analyze metrics, traces, and logs metadata (keys only, not values)
- Identify potential cardinality issues in instrumentation
- Provide a development/analysis tool for pre-production validation
- Prevent cost surprises from high-cardinality observability data

**Key Principle:** Track metadata keys only, never actual values, to avoid creating cardinality problems in the tool itself.

## Tech Stack

### Backend
- **Language:** Go 1.24+
- **OTLP Protocol:** OpenTelemetry Protocol (gRPC & HTTP)
- **Protobuf:** `go.opentelemetry.io/proto/otlp` v1.8.0
- **gRPC:** `google.golang.org/grpc` v1.76.0
- **HTTP Router:** `github.com/go-chi/chi/v5` v5.2.3
- **Storage:** 
  - Primary: In-memory (custom implementation)
  - Optional: SQLite (`modernc.org/sqlite`)
  - Dual-mode storage factory pattern
- **Cardinality Estimation:** HyperLogLog (custom implementation)
- **Log Pattern Mining:** Drain algorithm (custom implementation)

### Frontend
- **Framework:** React 18.2
- **Build Tool:** Vite 5.0
- **Language:** JavaScript (JSX)

### DevOps
- **Container:** Docker (multi-stage builds)
- **Orchestration:** Kubernetes (deployment, service, ingress manifests)
- **Testing:** Native Go testing, load testing with K6/JavaScript
- **Profiling:** Go pprof (exposed on port 6060)

### Build System
- **Makefile:** Unified build system for backend + frontend
- **Go Modules:** Dependency management
- **npm:** Frontend dependency management

## Project Conventions

### Code Style

**Go Backend:**
- Standard Go formatting (`gofmt`)
- Package-level comments required
- Exported functions/types must have doc comments
- Error handling: explicit error returns, wrapped errors with context
- Naming: 
  - Packages: lowercase, single word when possible
  - Interfaces: `-er` suffix when appropriate (e.g., `Analyzer`)
  - Constants: CamelCase for exported, camelCase for unexported
- File organization: `filename.go` for implementation, `filename_test.go` for tests

**Frontend:**
- React functional components preferred
- JSX for component files (`.jsx`)
- Component names: PascalCase (e.g., `MetricsView.jsx`)
- Styling: CSS-in-JS or separate CSS files

**File Structure:**
```
cmd/server/           - Entry point
internal/             - Private application code
  analyzer/           - Telemetry analysis logic
  api/                - REST API handlers
  config/             - Configuration management
  receiver/           - OTLP receivers (gRPC/HTTP)
  storage/            - Storage implementations
pkg/                  - Public libraries (reusable)
  hyperloglog/        - HyperLogLog cardinality estimation
  models/             - Data models
```

### Architecture Patterns

**Layered Architecture:**
1. **Receiver Layer:** OTLP HTTP (4318) & gRPC (4317) endpoints
2. **Analyzer Layer:** Metadata extraction for metrics/traces/logs
3. **Storage Layer:** Pluggable storage (memory, SQLite, dual-mode)
4. **API Layer:** REST API (8080) for querying metadata

**Key Patterns:**
- **Factory Pattern:** Storage backend selection (`storage.NewStorage()`)
- **Interface-based Design:** Storage interface for multiple implementations
- **Dependency Injection:** Store passed to receivers and API server
- **Graceful Shutdown:** Context-based shutdown with timeouts
- **Template Mining:** Drain algorithm with fixed-depth tree clustering

**Concurrency:**
- Goroutines for parallel server startup
- Mutex-based synchronization in storage layer
- Channel-based error handling and shutdown signaling

**Performance Optimizations:**
- HyperLogLog for memory-efficient cardinality estimation (~12KB per metric)
- Template pre-masking for log pattern detection (timestamps, UUIDs, IPs, etc.)
- Drain algorithm: 53k-1.6M events/second processing

### Testing Strategy

**Unit Tests:**
- Test files alongside implementation: `*_test.go`
- Table-driven tests preferred
- Use `testing` package, `testify` for assertions
- Coverage focus: storage, analyzers, HyperLogLog

**Benchmark Tests:**
- Performance-critical paths: `*_bench_test.go`
- HyperLogLog serialization benchmarks
- Storage operations benchmarks
- Log template mining benchmarks

**Load Tests:**
- K6 scripts in `scripts/` directory
- `load-test-metrics.js`, `load-test-traces.js`, `load-test-logs.js`
- Target: 50,000 metrics, 450 req/s sustained

**Test Execution:**
```bash
go test ./...              # All unit tests
go test -bench=.           # All benchmarks
make test                  # Backend + frontend tests
scripts/run-all-tests.sh   # Load tests
```

### Git Workflow

**Branching Strategy:**
- `main` branch: production-ready code
- Feature branches: `feature/description`
- Bug fixes: `fix/description`
- No direct commits to `main`

**Commit Conventions:**
- Conventional Commits style
- Format: `type(scope): description`
- Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`
- Examples:
  - `feat(analyzer): add log template extraction`
  - `fix(storage): resolve race condition in metrics update`
  - `docs(readme): update performance benchmarks`

**Pull Requests:**
- Descriptive titles and summaries
- Link related issues
- Require tests for new features
- CI checks must pass before merge

## Domain Context

**OpenTelemetry (OTLP) Concepts:**
- **Metrics:** Time-series data (counters, gauges, histograms) with labels/attributes
- **Traces:** Distributed traces with spans, parent-child relationships
- **Logs:** Structured log records with severity levels and bodies
- **Resource Attributes:** Service-level metadata (service.name, service.version)
- **Cardinality:** Number of unique combinations of label values

**Cardinality Problems:**
- High-cardinality labels (user IDs, request IDs) → cost explosions
- Unbounded label values → storage/query performance degradation
- Common culprits: dynamic labels, timestamps in labels, session IDs

**Drain Algorithm (Log Template Mining):**
- Fixed-depth tree structure for log clustering
- Token similarity matching with configurable threshold (default 0.7)
- Pre-masking patterns: timestamps, UUIDs, IPs, URLs, emails, hex strings
- Templates use placeholders: `<TIMESTAMP>`, `<NUM>`, `<IP>`, `<*>`

**HyperLogLog:**
- Probabilistic cardinality estimation algorithm
- ~1.5% error rate, uses ~12KB memory per metric
- Enables tracking unique value counts without storing values

## Important Constraints

**Technical Constraints:**
- **No Value Storage:** Only track metadata keys, never actual values
- **Memory Limits:** ~8-9 KB per metric in memory mode
- **Single-Node State:** Each instance has independent in-memory storage (no distributed state)
- **Protobuf-only:** OTLP protocol via gRPC/HTTP, no other formats
- **Go 1.24+ Required:** Uses latest Go toolchain features

**Performance Targets:**
- 10,000 metrics: ~85 MB memory
- 50,000 metrics: ~425 MB memory
- API response latency: <10ms for 100 items
- Log template mining: 20,000+ events/second

**Operational Constraints:**
- Not a production observability backend (development/analysis tool only)
- No data persistence guarantees in memory mode
- No authentication/authorization (internal tool assumption)
- No multi-tenancy support

## External Dependencies

**OpenTelemetry:**
- OTLP Protocol Specification (v1.8.0)
- Compatible with any OpenTelemetry Collector exporter
- Follows standard OTLP endpoint paths (`/v1/metrics`, `/v1/traces`, `/v1/logs`)

**Data Sources:**
- Designed to work downstream of OpenTelemetry Collector
- Collector receives from: Kafka, Redis, Prometheus, Jaeger, Zipkin, etc.
- Collector exports to this tool via OTLP HTTP/gRPC exporter

**Monitoring Integration:**
- Optional: pprof for Go runtime profiling (port 6060)
- No built-in metrics export (tool analyzes, doesn't emit telemetry)

**Deployment:**
- Kubernetes: standard deployment/service/ingress resources
- Docker: multi-stage builds, Alpine-based final image
- No external service dependencies (self-contained)
