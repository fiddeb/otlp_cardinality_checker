# Persistence Design

OTLP Cardinality Checker uses **in-memory storage only** by design. This is an intentional architectural decision, not a limitation.

## Why Ephemeral?

The tool is designed as a **diagnostic analyzer**, not a telemetry database:

| Diagnostic Tool | Telemetry Database |
|-----------------|-------------------|
| Shows current state | Stores historical data |
| Answers "what's happening now?" | Answers "what happened over time?" |
| Ephemeral, restart-friendly | Persistent, durable |
| Like `htop` or `top` | Like Prometheus or ClickHouse |

**Core insight**: Metadata analysis is diagnostic. You analyze current instrumentation, identify issues, fix them, and move on. Historical storage adds complexity without clear value for this use case.

## Memory Characteristics

### Capacity

- **500,000+ metrics** with moderate cardinality
- **<256MB memory** under typical load
- **~8-10 KB per metric** including attribute tracking

### Cardinality Estimation

Uses **HyperLogLog** algorithm for memory-efficient cardinality estimation:
- ~16KB per tracked attribute
- ~0.81% estimation error
- Allows tracking millions of unique values without storing them

### What's Stored

| Data Type | What We Store | What We Don't Store |
|-----------|---------------|---------------------|
| Metrics | Name, type, unit, label keys | Actual label values (only samples) |
| Spans | Span name, attribute keys | Full traces |
| Logs | Severity, attribute keys, patterns | Full log bodies |
| Attributes | Key names, cardinality estimate | All values (only HLL sketch + samples) |

## Restart Behavior

When the application restarts:
1. All in-memory metadata is lost
2. Application starts fresh with empty storage
3. New data flows in from OpenTelemetry Collector
4. Metadata rebuilds as telemetry arrives

**This is expected behavior**, not data loss.

## Use Cases

### Development & Debugging
```bash
# Start analyzer
./bin/occ

# Run tests or exercise your application
go test ./...

# Query metadata to understand instrumentation
curl http://localhost:8080/api/v1/metrics | jq '.data[] | .name'

# Stop when done - data not needed after analysis
```

### CI/CD Validation
```bash
# Spin up analyzer in pipeline
./bin/occ &

# Run integration tests
./run-tests.sh

# Capture metadata report
curl http://localhost:8080/api/v1/metrics > instrumentation-report.json

# Check for cardinality issues
./check-cardinality.sh

# No cleanup needed - container destroyed after pipeline
```

### Staging Analysis
```bash
# Point staging collector at analyzer for a few hours
# Query results
curl http://localhost:8080/api/v1/cardinality/high

# Make fixes to instrumentation
# Restart analyzer with fresh state
# Re-analyze to confirm fixes
```

## Memory Limits

If you need to handle very large deployments:

1. **Increase container memory**: Default 512MB, can increase to 1-2GB
2. **Use sampling**: Configure OpenTelemetry Collector to sample data
3. **Filter at source**: Don't send all services to analyzer
4. **Multiple analyzers**: Run separate instances per environment/team

## Future: Export/Import (If Needed)

If you need to preserve analysis results:

### Manual Export
```bash
# Export current state to JSON
curl http://localhost:8080/api/v1/metrics > metrics.json
curl http://localhost:8080/api/v1/spans > spans.json
curl http://localhost:8080/api/v1/logs > logs.json
```

### Scripted Export
```bash
#!/bin/bash
# export-metadata.sh
DATE=$(date +%Y%m%d-%H%M%S)
curl -s http://localhost:8080/api/v1/metrics > "metadata-${DATE}-metrics.json"
curl -s http://localhost:8080/api/v1/spans > "metadata-${DATE}-spans.json"
curl -s http://localhost:8080/api/v1/logs > "metadata-${DATE}-logs.json"
echo "Exported to metadata-${DATE}-*.json"
```

**Note**: Import functionality is not yet implemented. Re-analyze from source if needed.

## Comparison with Alternatives

| Approach | Pros | Cons |
|----------|------|------|
| **In-memory (current)** | Simple, fast, no dependencies | Data lost on restart |
| **SQLite** | Persistent, embedded | Slower writes, WAL complexity |
| **PostgreSQL** | Scalable, proven | External dependency, operational overhead |
| **ClickHouse** | High performance | Heavy external dependency |

We chose in-memory because the benefits of persistence don't outweigh the complexity for a diagnostic tool.

## When Persistence Might Be Needed

Consider adding persistence if:
- You need to track cardinality trends over days/weeks
- You're building dashboards on historical metadata
- Multiple teams need shared access to analysis results
- Compliance requires audit trails of instrumentation

**Current recommendation**: Export JSON if you need to save results. If there's strong demand for built-in persistence, we can revisit.
