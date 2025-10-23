# Contributing to OTLP Cardinality Checker

Thank you for your interest in contributing! This document provides guidelines for contributing to the project.

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow

## Getting Started

### Prerequisites

- **Go 1.21+** installed
- **Docker** (optional, for PostgreSQL testing)
- **Make** (optional, for convenience commands)

### Development Setup

1. **Clone the repository**
```bash
git clone https://github.com/yourusername/otlp_cardinality_checker.git
cd otlp_cardinality_checker
```

2. **Install dependencies**
```bash
go mod download
```

3. **Run tests**
```bash
go test ./...
```

4. **Run locally**
```bash
go run cmd/server/main.go
```

5. **Verify it's working**
```bash
# In another terminal
curl http://localhost:8080/api/v1/health
```

### Project Structure

```
otlp_cardinality_checker/
├── cmd/
│   └── server/              # Main application entry point
│       └── main.go
├── internal/
│   ├── receiver/           # OTLP receivers (gRPC/HTTP)
│   ├── analyzer/           # Metadata extraction logic
│   │   ├── metrics/
│   │   ├── traces/
│   │   └── logs/
│   ├── storage/            # Storage implementations
│   │   ├── memory/
│   │   └── postgres/
│   ├── api/                # REST API handlers
│   └── config/             # Configuration management
├── pkg/
│   └── models/             # Shared data models
├── tests/
│   ├── integration/        # Integration tests
│   └── loadgen/            # Load testing tools
├── docs/                   # Additional documentation
├── config/                 # Configuration examples
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
├── PRODUCT.md
├── ARCHITECTURE.md
└── CONTRIBUTING.md
```

## Development Workflow

### 1. Create an Issue

Before starting work, create or find an issue describing:
- What problem you're solving
- Proposed approach (for larger changes)
- Any questions or concerns

### 2. Branch Naming

Create a branch with a descriptive name:
```bash
git checkout -b feature/add-metric-filtering
git checkout -b fix/span-analyzer-panic
git checkout -b docs/update-api-examples
```

Prefixes:
- `feature/` - New features
- `fix/` - Bug fixes
- `refactor/` - Code refactoring
- `docs/` - Documentation updates
- `test/` - Test additions/fixes

### 3. Make Changes

Follow the coding guidelines (see below) and write tests.

### 4. Commit Messages

Write clear, descriptive commit messages:

```
feat: add filtering by service name in metrics API

- Add service query parameter to /api/v1/metrics
- Update storage layer to support service filtering
- Add tests for new filtering logic

Closes #123
```

Format:
```
<type>: <subject>

<body>

<footer>
```

Types:
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation
- `test` - Tests
- `refactor` - Code refactoring
- `perf` - Performance improvements
- `chore` - Maintenance tasks

### 5. Push and Create PR

```bash
git push origin feature/add-metric-filtering
```

Create a Pull Request with:
- Clear description of changes
- Link to related issue
- Screenshots (if UI changes)
- Test results

## Coding Guidelines

### Go Style

Follow [Effective Go](https://go.dev/doc/effective_go) and these additional guidelines:

#### 1. Formatting
```bash
# Use gofmt
go fmt ./...

# Use goimports (installs missing imports)
goimports -w .
```

#### 2. Naming Conventions

**Packages**: Short, lowercase, single-word names
```go
package analyzer  // Good
package metricAnalyzer  // Bad
```

**Interfaces**: Use `-er` suffix for single-method interfaces
```go
type Analyzer interface {
    Analyze(data interface{}) error
}

type Storage interface {
    Store(key string, value interface{}) error
    Retrieve(key string) (interface{}, error)
}
```

**Structs**: Use PascalCase
```go
type MetricMetadata struct {
    Name      string
    LabelKeys []string
}
```

**Variables**: Use camelCase
```go
var metricCount int
var serviceNames []string
```

#### 3. Error Handling

Always handle errors explicitly:

```go
// Good
result, err := someFunction()
if err != nil {
    return fmt.Errorf("failed to process: %w", err)
}

// Bad
result, _ := someFunction()
```

Wrap errors with context:
```go
if err := validateMetric(m); err != nil {
    return fmt.Errorf("validating metric %s: %w", m.Name, err)
}
```

#### 4. Context Usage

Always pass context as first parameter:

```go
func (s *Store) GetMetric(ctx context.Context, name string) (*Metric, error) {
    // Implementation
}
```

#### 5. Comments

Document all exported functions, types, and constants:

```go
// MetadataStore holds metadata for all observed telemetry signals.
// It provides thread-safe access to metrics, spans, and logs metadata.
type MetadataStore struct {
    mu      sync.RWMutex
    metrics map[string]*MetricMetadata
}

// AddMetric adds or updates metric metadata in the store.
// If a metric with the same name already exists, it merges the metadata.
func (s *MetadataStore) AddMetric(m *MetricMetadata) error {
    // Implementation
}
```

#### 6. Testing

Name test functions clearly:
```go
func TestMetadataStore_AddMetric(t *testing.T) { }
func TestMetadataStore_AddMetric_DuplicateNames(t *testing.T) { }
```

Use table-driven tests:
```go
func TestAnalyzeMetric(t *testing.T) {
    tests := []struct {
        name    string
        input   pmetric.Metric
        want    *MetricMetadata
        wantErr bool
    }{
        {
            name:  "gauge metric",
            input: createGaugeMetric(),
            want:  &MetricMetadata{Type: "Gauge"},
        },
        {
            name:    "invalid metric",
            input:   createInvalidMetric(),
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := AnalyzeMetric(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("AnalyzeMetric() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("AnalyzeMetric() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

Use testify for assertions (optional but recommended):
```go
import "github.com/stretchr/testify/assert"

func TestSomething(t *testing.T) {
    result := someFunction()
    assert.NotNil(t, result)
    assert.Equal(t, "expected", result.Value)
}
```

#### 7. Concurrency

Use channels and goroutines idiomatically:

```go
// Good: Use select for multiple channels
func (p *Pipeline) Process(ctx context.Context) error {
    for {
        select {
        case data := <-p.inputChan:
            p.handle(data)
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}

// Use sync.WaitGroup for coordinating goroutines
var wg sync.WaitGroup
for _, item := range items {
    wg.Add(1)
    go func(i Item) {
        defer wg.Done()
        process(i)
    }(item)
}
wg.Wait()
```

#### 8. Logging

Use structured logging with levels:

```go
import "log/slog"

logger.Info("processing metric",
    "name", metricName,
    "labels", labelCount,
)

logger.Error("failed to store metric",
    "error", err,
    "metric", metricName,
)
```

### API Design

#### REST Endpoints

- Use nouns, not verbs
- Use plural names
- Use kebab-case for multi-word resources

```
GET    /api/v1/metrics           # List
GET    /api/v1/metrics/{name}    # Get
GET    /api/v1/metric-groups     # kebab-case for multi-word
```

#### Response Format

Always return consistent JSON structure:

```json
{
  "success": true,
  "data": { },
  "error": null,
  "metadata": {
    "timestamp": "2024-01-15T10:30:00Z",
    "request_id": "abc-123"
  }
}
```

Error responses:
```json
{
  "success": false,
  "data": null,
  "error": {
    "code": "METRIC_NOT_FOUND",
    "message": "Metric 'http_requests_total' not found",
    "details": {}
  },
  "metadata": {
    "timestamp": "2024-01-15T10:30:00Z",
    "request_id": "abc-123"
  }
}
```

### Performance Guidelines

1. **Avoid allocations in hot paths**
```go
// Bad: Creates new slice on every call
func getKeys(m map[string]string) []string {
    keys := []string{}
    for k := range m {
        keys = append(keys, k)
    }
    return keys
}

// Good: Pre-allocate
func getKeys(m map[string]string) []string {
    keys := make([]string, 0, len(m))
    for k := range m {
        keys = append(keys, k)
    }
    return keys
}
```

2. **Use sync.Pool for frequently allocated objects**
```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func process() {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer bufferPool.Put(buf)
    buf.Reset()
    // Use buffer
}
```

3. **Minimize lock contention**
```go
// Use RWMutex when reads >> writes
type Store struct {
    mu   sync.RWMutex
    data map[string]interface{}
}

func (s *Store) Get(key string) interface{} {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.data[key]
}
```

## Testing

### Unit Tests

Test all public functions and edge cases:

```bash
go test ./internal/analyzer/...
```

### Integration Tests

Test end-to-end flows:

```bash
go test ./tests/integration/...
```

### Coverage

Aim for >80% coverage:

```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Benchmarks

Write benchmarks for performance-critical code:

```go
func BenchmarkMetadataStore_AddMetric(b *testing.B) {
    store := NewMetadataStore()
    metric := createTestMetric()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        store.AddMetric(metric)
    }
}
```

Run benchmarks:
```bash
go test -bench=. -benchmem ./...
```

## Pull Request Process

1. **Ensure all tests pass**
```bash
make test  # or go test ./...
```

2. **Run linters**
```bash
golangci-lint run
```

3. **Update documentation** if needed

4. **Request review** from maintainers

5. **Address feedback** promptly

6. **Squash commits** if requested before merge

### PR Checklist

Before submitting, ensure:
- [ ] Tests added/updated and passing
- [ ] Documentation updated
- [ ] Code follows style guidelines
- [ ] Commit messages are clear
- [ ] No unnecessary dependencies added
- [ ] Backwards compatibility maintained (or breaking change documented)

## Release Process

(For maintainers)

1. Update version in `version.go`
2. Update CHANGELOG.md
3. Create git tag: `git tag v0.1.0`
4. Push tag: `git push origin v0.1.0`
5. GitHub Actions builds and publishes release

## Questions?

- Open an issue with the `question` label
- Join discussions in GitHub Discussions
- Check existing documentation in `/docs`

## License

By contributing, you agree that your contributions will be licensed under the project's license.

## Recognition

Contributors will be recognized in:
- README.md contributors section
- Release notes
- GitHub contributors page

Thank you for contributing! 🎉
