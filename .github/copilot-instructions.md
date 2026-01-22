# OTLP Cardinality Checker

OTLP metadata analysis tool that receives telemetry from OpenTelemetry Collector.

## Core Principle

**Store ONLY metadata keys, never actual values** to prevent cardinality explosion.

## Architecture

```
Sources → OpenTelemetry Collector → This Tool (OTLP HTTP/gRPC) → Metadata Storage
```

## Skills Reference

For detailed patterns, use these skills:
- **occ-project**: Project-specific Go patterns, data models, API design
- **go-development**: Build, test, dependency management
- **git-workflow**: Branching, commits, remote operations

## Quick Rules

### Commit Workflow

Commit immediately when: tests pass, build succeeds, feature works, bug fixed.

```bash
git commit -m "<type>: <description>"
# Types: feat, fix, refactor, test, docs, chore, perf
```

Always use feature branches: `feature/`, `fix/`, `refactor/`
Never merge directly to main - use Pull Requests.

### Code Style

- Context as first parameter
- Wrap errors with `fmt.Errorf("context: %w", err)`
- Pre-allocate slices when size known
- Table-driven tests
- Structured logging with slog
- No emojis unless requested

### Data Model

```go
// GOOD: Keys only
type MetricMetadata struct {
    LabelKeys []string
    SampleCount int64
}

// BAD: Never store values
Labels map[string]string  // Creates cardinality explosion
```

### What to Avoid

1. Storing attribute values (only keys)
2. Heavy reflection (prefer explicit types)
3. Global variables (use DI)
4. Panic/recover (use error returns)
5. Unbounded goroutines (use worker pools)
