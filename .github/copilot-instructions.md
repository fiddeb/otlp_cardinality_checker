# GitHub Copilot Instructions for OTLP Cardinality Checker

## Project Context

This is a **metadata analysis tool** for OpenTelemetry Protocol (OTLP) telemetry. It acts as an **OTLP endpoint destination** that receives telemetry from an OpenTelemetry Collector.

**Architecture Flow:**
```
Sources (Kafka/Redis/Prometheus) → OpenTelemetry Collector → This Tool (OTLP Endpoints)
```

The core purpose is to extract and track metadata structure (keys, not values) from metrics, traces, and logs.

**Key Principle**: We store ONLY metadata keys, never actual values, to avoid cardinality explosion in the tool itself.

**Implementation Note**: We implement simple OTLP HTTP/gRPC servers (not full Collector receivers). The Collector sends us data via its OTLP exporters.

## Architecture Patterns

### 1. Layered Architecture

```
OTLP Endpoint Layer → Analyzer Layer → Storage Layer → API Layer
```

- **OTLP Endpoints**: Simple HTTP/gRPC servers that accept OTLP protobuf from Collector exporters
- **Analyzer**: Extracts metadata keys from telemetry signals
- **Storage**: In-memory primary, optional PostgreSQL persistence
- **API**: REST endpoints for querying metadata

### 2. Concurrency Model

Use **channel-based pipelines** for data flow:

```go
// Example pattern
type Pipeline struct {
    inputChan  chan Data
    outputChan chan Result
}

func (p *Pipeline) Process(ctx context.Context) {
    for {
        select {
        case data := <-p.inputChan:
            result := p.analyze(data)
            p.outputChan <- result
        case <-ctx.Done():
            return
        }
    }
}
```

### 3. Thread Safety

- Use `sync.RWMutex` for read-heavy workloads
- Minimize critical sections
- Prefer channels over shared memory for goroutine communication

## Go Style Preferences

### Naming

```go
// Packages: lowercase, single word
package analyzer

// Interfaces: -er suffix for single method
type Analyzer interface {
    Analyze(ctx context.Context, data interface{}) error
}

// Structs: PascalCase
type MetricMetadata struct {
    Name string
    Keys []string
}

// Functions: camelCase for private, PascalCase for exported
func extractKeys(m map[string]string) []string
func ExtractMetricKeys(metric pmetric.Metric) []string
```

### Error Handling

Always wrap errors with context:

```go
if err := processMetric(m); err != nil {
    return fmt.Errorf("processing metric %s: %w", m.Name, err)
}
```

Never ignore errors:
```go
// BAD
data, _ := fetchData()

// GOOD
data, err := fetchData()
if err != nil {
    return fmt.Errorf("fetching data: %w", err)
}
```

### Context Usage

Context is always the first parameter:

```go
func (s *Store) GetMetric(ctx context.Context, name string) (*Metric, error)
```

Check context cancellation in loops:

```go
for _, item := range items {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        process(item)
    }
}
```

## Data Model Conventions

### Metadata Structures

All metadata structs should include:
- `FirstSeen time.Time` - When first observed
- `LastSeen time.Time` - Most recent observation
- `*Count int64` - Number of times seen

```go
type MetricMetadata struct {
    Name         string    `json:"name"`
    Type         string    `json:"type"`
    LabelKeys    []string  `json:"label_keys"`
    ResourceKeys []string  `json:"resource_keys"`
    FirstSeen    time.Time `json:"first_seen"`
    LastSeen     time.Time `json:"last_seen"`
    SampleCount  int64     `json:"sample_count"`
}
```

### Key Extraction Pattern

When extracting keys from maps/attributes:

```go
// Extract and sort keys for consistent output
func extractKeys(attrs map[string]interface{}) []string {
    keys := make([]string, 0, len(attrs))
    for k := range attrs {
        keys = append(keys, k)
    }
    sort.Strings(keys) // Always sort for consistency
    return keys
}
```

### Merging Metadata

When updating existing metadata, merge keys (set union):

```go
func (m *MetricMetadata) Merge(other *MetricMetadata) {
    // Merge label keys (union)
    keySet := make(map[string]bool)
    for _, k := range m.LabelKeys {
        keySet[k] = true
    }
    for _, k := range other.LabelKeys {
        keySet[k] = true
    }
    
    m.LabelKeys = make([]string, 0, len(keySet))
    for k := range keySet {
        m.LabelKeys = append(m.LabelKeys, k)
    }
    sort.Strings(m.LabelKeys)
    
    // Update timestamps
    if other.LastSeen.After(m.LastSeen) {
        m.LastSeen = other.LastSeen
    }
    m.SampleCount += other.SampleCount
}
```

## Testing Patterns

### Table-Driven Tests

Use table-driven tests for comprehensive coverage:

```go
func TestExtractMetricKeys(t *testing.T) {
    tests := []struct {
        name    string
        input   pmetric.Metric
        want    []string
        wantErr bool
    }{
        {
            name:  "gauge with labels",
            input: createGaugeMetric(map[string]string{"method": "", "status": ""}),
            want:  []string{"method", "status"},
        },
        {
            name:    "nil metric",
            input:   pmetric.Metric{},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ExtractMetricKeys(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Test Helpers

Create helper functions for test data:

```go
// testutil/otlp.go
func CreateTestMetric(name string, labels map[string]string) pmetric.Metric {
    metrics := pmetric.NewMetrics()
    rm := metrics.ResourceMetrics().AppendEmpty()
    sm := rm.ScopeMetrics().AppendEmpty()
    m := sm.Metrics().AppendEmpty()
    m.SetName(name)
    
    gauge := m.SetEmptyGauge()
    dp := gauge.DataPoints().AppendEmpty()
    for k := range labels {
        dp.Attributes().PutStr(k, "")
    }
    
    return m
}
```

### Mocking

Use interfaces for dependencies to enable mocking:

```go
// Define interface
type Storage interface {
    Store(ctx context.Context, key string, value interface{}) error
    Retrieve(ctx context.Context, key string) (interface{}, error)
}

// Test with mock
type mockStorage struct {
    stored map[string]interface{}
}

func (m *mockStorage) Store(ctx context.Context, key string, value interface{}) error {
    m.stored[key] = value
    return nil
}
```

## API Design

### Handler Pattern

Use this pattern for API handlers:

```go
func (h *Handler) GetMetric(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // Extract parameters
    name := chi.URLParam(r, "name")
    if name == "" {
        h.respondError(w, http.StatusBadRequest, "metric name required")
        return
    }
    
    // Business logic
    metric, err := h.store.GetMetric(ctx, name)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            h.respondError(w, http.StatusNotFound, "metric not found")
            return
        }
        h.respondError(w, http.StatusInternalServerError, "internal error")
        return
    }
    
    // Success response
    h.respondJSON(w, http.StatusOK, metric)
}
```

### Response Format

Always use consistent JSON structure:

```go
type APIResponse struct {
    Success  bool        `json:"success"`
    Data     interface{} `json:"data,omitempty"`
    Error    *APIError   `json:"error,omitempty"`
    Metadata Metadata    `json:"metadata"`
}

type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

type Metadata struct {
    Timestamp time.Time `json:"timestamp"`
    RequestID string    `json:"request_id,omitempty"`
}
```

## Performance Guidelines

### Memory Efficiency

Pre-allocate slices when size is known:

```go
// BAD
var keys []string
for k := range m {
    keys = append(keys, k)
}

// GOOD
keys := make([]string, 0, len(m))
for k := range m {
    keys = append(keys, k)
}
```

### String Operations

Use `strings.Builder` for concatenation:

```go
var sb strings.Builder
for _, s := range parts {
    sb.WriteString(s)
}
result := sb.String()
```

### Sync.Pool

Use sync.Pool for frequently allocated objects:

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func process() {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()
    // Use buffer
}
```

## Logging

Use structured logging (slog in Go 1.21+):

```go
import "log/slog"

logger.Info("processing metric",
    "metric_name", name,
    "label_count", len(labels),
    "duration_ms", duration.Milliseconds(),
)

logger.Error("failed to store metric",
    "error", err,
    "metric_name", name,
)
```

## Common Patterns

### Graceful Shutdown

```go
func (s *Server) Run(ctx context.Context) error {
    errChan := make(chan error, 1)
    
    // Start servers
    go func() {
        errChan <- s.grpcServer.Serve(s.grpcListener)
    }()
    
    go func() {
        errChan <- s.httpServer.ListenAndServe()
    }()
    
    // Wait for shutdown or error
    select {
    case err := <-errChan:
        return err
    case <-ctx.Done():
        return s.Shutdown(context.Background())
    }
}

func (s *Server) Shutdown(ctx context.Context) error {
    s.grpcServer.GracefulStop()
    return s.httpServer.Shutdown(ctx)
}
```

### Configuration Management

Use functional options pattern:

```go
type Config struct {
    grpcPort int
    httpPort int
}

type Option func(*Config)

func WithGRPCPort(port int) Option {
    return func(c *Config) {
        c.grpcPort = port
    }
}

func NewServer(opts ...Option) *Server {
    config := &Config{
        grpcPort: 4317,
        httpPort: 4318,
    }
    for _, opt := range opts {
        opt(config)
    }
    return &Server{config: config}
}
```

## Dependencies

### Prefer Standard Library

Use Go standard library when possible:
- `net/http` for HTTP servers
- `encoding/json` for JSON
- `context` for cancellation
- `sync` for concurrency

### External Dependencies

Only add dependencies when necessary:
- **OpenTelemetry Collector SDK** - For OTLP receiver
- **pgx** - For PostgreSQL (if enabled)
- **chi** or **gorilla/mux** - For HTTP routing
- **testify** - For testing assertions (optional)

## Code Organization

### Package Structure

```
internal/
├── receiver/          # OTLP protocol handling
│   ├── grpc.go
│   └── http.go
├── analyzer/          # Metadata extraction
│   ├── metrics.go
│   ├── traces.go
│   └── logs.go
├── storage/           # Data persistence
│   ├── memory/
│   │   └── store.go
│   └── postgres/
│       └── store.go
└── api/              # REST API
    ├── handlers.go
    └── middleware.go
```

Each package should have:
- Clear single responsibility
- Exported interfaces for testability
- Internal implementation details

## Documentation

### Package Documentation

Every package should have a doc comment:

```go
// Package analyzer extracts metadata structure from OTLP telemetry.
//
// It processes metrics, traces, and logs to identify unique attribute keys,
// metric names, span names, etc. without storing actual values.
package analyzer
```

### Function Documentation

Document all exported functions:

```go
// ExtractMetricKeys extracts all unique label keys from a metric.
// It returns a sorted slice of keys for consistent output.
// The metric type (gauge, counter, etc.) does not affect the extraction.
func ExtractMetricKeys(metric pmetric.Metric) ([]string, error) {
    // Implementation
}
```

## Security Considerations

### Input Validation

Always validate OTLP input:

```go
func validateMetric(m *MetricMetadata) error {
    if m.Name == "" {
        return errors.New("metric name cannot be empty")
    }
    if len(m.Name) > 255 {
        return errors.New("metric name too long")
    }
    return nil
}
```

### No Value Storage

Never store attribute values, only keys:

```go
// GOOD: Store only keys
type MetricMetadata struct {
    LabelKeys []string
}

// BAD: Never store values
type MetricMetadata struct {
    Labels map[string]string  // NO! This creates cardinality explosion
}
```

## Code Style Guidelines

### Formatting and Style

- **No emojis** in code, comments, or commit messages unless specifically requested
- Use clear, descriptive variable and function names
- Prefer explicit over clever code
- Keep functions small and focused
- Use consistent formatting (gofmt)

### Comments

Write comments that explain **why**, not **what**:

```go
// GOOD: Explains reasoning
// Use sync.Pool to reduce GC pressure under high load
var bufferPool = sync.Pool{...}

// BAD: States the obvious
// Create a buffer pool
var bufferPool = sync.Pool{...}
```

Avoid decorative elements:
```go
// GOOD
// Process handles incoming OTLP requests

// BAD
// ✨ Process handles incoming OTLP requests ✨
```

## When to Ask for Help

If Copilot suggests:
1. **Storing attribute values** - Remind it we only store keys
2. **Using reflection heavily** - Prefer explicit type handling
3. **Global variables** - Use dependency injection instead
4. **Panic/recover** - Use error returns instead
5. **Unbounded goroutines** - Use worker pools or bounded channels
6. **Emojis in code** - Remove them unless specifically requested

## Summary

**Key Takeaways**:
- Metadata keys only, never values
- Idiomatic Go: simple, explicit, testable
- Channel-based concurrency
- Consistent error handling
- Table-driven tests
- In-memory first, optional persistence
- REST API with consistent responses
- Clean code without unnecessary decorations
