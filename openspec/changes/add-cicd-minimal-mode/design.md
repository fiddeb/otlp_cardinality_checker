# Design: CI/CD Minimal Mode

## Overview
This document describes the architectural design for OCC's minimal operating mode optimized for CI/CD pipelines.

## Architecture

### Component States by Mode

| Component | Normal Mode | Minimal Mode |
|-----------|-------------|---------------|
| OTLP Receiver (gRPC) | ✅ Active | ✅ Active |
| OTLP Receiver (HTTP) | ✅ Active | ✅ Active |
| API Server | ✅ Active | ✅ Active |
| UI Server | ✅ Active | ❌ Disabled |
| Storage Backend | Memory + Session save/load | Memory-only |
| Session Management | ✅ Can save/load | ❌ Disabled |
| Metrics Exporter | ✅ Active | ⚠️ Optional |
| Auto-report | ❌ Disabled | ✅ Optional |

### System Flow

```
┌─────────────────┐
│   CI Pipeline   │
└────────┬────────┘
         │ starts
         ▼
┌─────────────────────────────────────┐
│  OCC (--minimal --duration 5m)     │
│                                     │
│  ┌──────────────┐  ┌─────────────┐│
│  │     OTLP     │  │     API     ││
│  │   Receiver   │  │   Server    ││
│  │  :4317,:4318 │  │    :8080    ││
│  └──────┬───────┘  └──────▲──────┘│
│         │                  │       │
│         ▼                  │       │
│  ┌─────────────────────────┴────┐ │
│  │    Memory Storage             │ │
│  │  (bounded, auto-evict)        │ │
│  └───────────┬───────────────────┘ │
│              │                      │
│              ▼                      │
│  ┌───────────────────────────────┐ │
│  │   Report Generator            │ │
│  │   (timer-triggered)           │ │
│  └───────────┬───────────────────┘ │
└──────────────┼─────────────────────┘
               │
               ▼
      ┌────────────────┐
      │  report.json   │
      └────────────────┘
```

## Startup Modes

### Normal Mode (Default)
```bash
# Start in normal mode - no flags needed
occ start

# All components active:
# ✅ OTLP receivers
# ✅ API server  
# ✅ UI server (web interface at http://localhost:3000)
# ✅ Memory storage with session save/load capability
# ✅ Runs indefinitely until stopped
# ✅ Can save snapshots: occ session save my-session.json
# ✅ Can load sessions: occ session load my-session.json
```

### Minimal Mode (CI/CD)
```bash
# Start in minimal mode
occ start --minimal

# Only essential components:
# ✅ OTLP receivers
# ✅ API server
# ❌ UI server (disabled)
# ⚠️  Memory-only storage
# ⚠️  Optional auto-shutdown
```

## Configuration

### CLI Flags

```bash
occ start [flags]

Mode Selection:
  (none)                  Normal mode (default) - full features
  --minimal               Enable minimal mode (OTLP + API only)
  --cicd                  Alias for --minimal

Minimal Mode Flags (only applicable with --minimal):
  --duration DURATION     Auto-shutdown and report after duration (e.g., "5m", "1h")
  --report-output PATH    Path to write report (default: stdout)
  --report-format FORMAT  Report format: json|yaml|text (default: text)
  --report-verbosity LEVEL Report detail level: basic|verbose (default: basic)
  --report-max-items N    Max items per section in report (default: 20, 0=unlimited)
  --session-export PATH   Export session file for loading in full UI (optional)
  --api-only              Keep running after duration (no auto-shutdown)
  --max-memory MB         Max memory usage before eviction (default: 512)
  --exit-on-threshold     Exit code 1 if cardinality warnings, 2 if critical

Standard Flags (still applicable):
  --otlp-grpc-port PORT   OTLP gRPC port (default: 4317)
  --otlp-http-port PORT   OTLP HTTP port (default: 4318)
  --api-port PORT         API port (default: 8080)
  --log-level LEVEL       Log level (default: info)
```

### Environment Variables

```bash
OCC_MINIMAL=true
OCC_DURATION=5m
OCC_REPORT_OUTPUT=/tmp/report.txt
OCC_REPORT_FORMAT=text
OCC_REPORT_VERBOSITY=basic  # or verbose
OCC_REPORT_MAX_ITEMS=20     # 0 for unlimited
OCC_SESSION_EXPORT=/tmp/session.json
OCC_MAX_MEMORY=512
OCC_EXIT_ON_THRESHOLD=true
```

### Configuration Precedence
1. CLI flags (highest)
2. Environment variables
3. Config file (if specified)
4. Defaults (lowest)

## Data Flow

### Startup Sequence

```go
1. Parse flags/env → Minimal mode detected
2. Initialize memory-only storage
3. Start OTLP receivers (gRPC + HTTP)
4. Start API server
5. Skip UI server initialization
6. If --duration set:
   - Start shutdown timer
   - Register report generator callback
7. Register signal handlers (SIGTERM, SIGINT)
8. Log "OCC running in minimal mode"
```

### Shutdown Sequence

```go
1. Trigger: Timer expires OR signal received
2. Stop accepting new OTLP data
3. Drain in-flight requests (5s grace)
4. Generate report:
   - Query storage for all metrics
   - Calculate cardinality stats
   - Format output (text/JSON/YAML)
   - Write to file/stdout
5. If --session-export specified:
   - Export full session data
   - Write to session file
6. If --exit-on-threshold:
   - Check thresholds
   - Set exit code
7. Close API server
8. Close storage
9. Exit with code
```

## Storage Strategy

### Memory-Only Backend

```go
type MinimalStorage struct {
    metrics       map[string]*MetricMetadata // key: metric name
    attributes    map[string]StringSet       // key: attribute key
    maxMemoryMB   int
    evictionPolicy EvictionPolicy // LRU or oldest-first
    mu            sync.RWMutex
}

type MetricMetadata struct {
    Name           string
    LabelKeys      []string
    SampleCount    int64
    LastSeen       time.Time
    Cardinality    int // Estimated unique label value combinations
}
```

### Memory Management

- **Max memory limit**: Configurable (default 512MB)
- **Eviction policy**: Remove oldest metrics by `LastSeen` when limit reached
- **Monitoring**: Track current memory usage via runtime metrics
- **Warning threshold**: Log warning at 80% memory usage

## Report Generation

### Report Purpose

The **report** is a human-readable summary of telemetry discovered during the run. It answers:
- What metrics does my application send?
- What traces/spans are being generated?
- What logs are being emitted?
- What attribute keys are being used across all signals?
- Are there any high-cardinality issues?
- Basic statistics about the telemetry

This is different from **session export** which contains full data for loading in the UI.

### Report Verbosity Levels

**Basic Mode** (default):
- Summary statistics
- Top N items by cardinality (default 20)
- High-cardinality warnings
- Recommendations
- **Use case**: Quick overview, CI pipelines, alerts

**Verbose Mode** (`--report-verbosity verbose`):
- Complete list of all discovered metrics, spans, logs
- Full label/attribute keys for each item
- All cardinality details
- Sorted by cardinality (highest first)
- **Use case**: Detailed analysis, documentation, auditing

**Max Items** (`--report-max-items N`):
- Limits items shown per section (metrics, spans, logs)
- Default: 20 items
- Set to 0 for unlimited (show all)
- Applies to both basic and verbose modes
- Summary always shows total counts

### Report Schema (JSON)

```json
{
  "version": "1.0",
  "generated_at": "2026-01-25T10:30:00Z",
  "duration": "5m",
  "occ_version": "0.2.0",
  "summary": {
    "total_metrics": 42,
    "total_span_names": 15,
    "total_log_types": 8,
    "total_attributes": 23,
    "total_samples": {
      "metrics": 1500000,
      "spans": 850000,
      "logs": 320000
    },
    "high_cardinality_count": 5
  },
  "metrics": [
    {
      "name": "http_requests_total",
      "type": "counter",
      "label_keys": ["method", "path", "status"],
      "sample_count": 50000,
      "estimated_cardinality": 1200,
      "severity": "ok"
    },
    {
      "name": "user_events",
      "type": "counter",
      "label_keys": ["user_id", "event_type", "platform"],
      "sample_count": 150000,
      "estimated_cardinality": 12000,
      "severity": "critical"
    }
  ],
  "spans": [
    {
      "name": "HTTP GET /api/users/{id}",
      "attribute_keys": ["http.method", "http.route", "user.id", "http.status_code"],
      "span_count": 125000,
      "estimated_cardinality": 8500,
      "severity": "warning"
    },
    {
      "name": "database.query",
      "attribute_keys": ["db.system", "db.name", "db.statement", "db.user"],
      "span_count": 200000,
      "estimated_cardinality": 15000,
      "severity": "critical"
    }
  ],
  "logs": [
    {
      "body_pattern": "User login successful",
      "attribute_keys": ["user.id", "source.ip", "timestamp"],
      "log_count": 45000,
      "estimated_cardinality": 2000,
      "severity": "ok"
    },
    {
      "body_pattern": "Error: {error_message}",
      "attribute_keys": ["error.type", "error.message", "stack_trace"],
      "log_count": 1200,
      "estimated_cardinality": 500,
      "severity": "ok"
    }
  ],
  "attributes": [
    {
      "key": "user_id",
      "signal_types": ["metric", "span", "log"],
      "used_by": {
        "metrics": ["user_events", "user_sessions"],
        "spans": ["HTTP GET /api/users/{id}"],
        "logs": ["User login successful"]
      },
      "estimated_unique_values": 10000,
      "severity": "warning"
    },
    {
      "key": "http.method",
      "signal_types": ["metric", "span"],
      "used_by": {
        "metrics": ["http_requests_total"],
        "spans": ["HTTP GET /api/users/{id}", "HTTP POST /api/orders"]
      },
      "estimated_unique_values": 7,
      "severity": "ok"
    }
  ],
  "recommendations": [
    "Consider removing 'user_id' from metrics - high cardinality risk",
    "Span 'database.query' has 15K cardinality - review attribute usage",
    "Metric 'user_events' has 12K cardinality - review label usage",
    "Use semantic conventions for HTTP attributes (http.method, http.route)"
  ]
}
```

### Report Text Format

```
OCC Telemetry Analysis Report
==============================
Generated: 2026-01-25 10:30:00
Duration: 5m
OCC Version: 0.2.0

Summary
-------
Metrics: 42
Span names: 15
Log types: 8
Unique attributes: 23
High cardinality issues: 5 ⚠️

Samples received:
  Metrics: 1,500,000
  Spans: 850,000
  Logs: 320,000

Metrics
-------
✓ http_requests_total (counter)
  Labels: method, path, status
  Cardinality: 1,200
  Samples: 50,000

✗ user_events (counter) [CRITICAL]
  Labels: user_id, event_type, platform
  Cardinality: 12,000 ⚠️
  Samples: 150,000

Spans
-----
⚠️  HTTP GET /api/users/{id} [WARNING]
  Attributes: http.method, http.route, user.id, http.status_code
  Cardinality: 8,500
  Spans: 125,000

✗ database.query [CRITICAL]
  Attributes: db.system, db.name, db.statement, db.user
  Cardinality: 15,000 ⚠️
  Spans: 200,000

Logs
----
✓ User login successful
  Attributes: user.id, source.ip, timestamp
  Cardinality: 2,000
  Logs: 45,000

✓ Error: {error_message}
  Attributes: error.type, error.message, stack_trace
  Cardinality: 500
  Logs: 1,200

Attributes (Cross-Signal)
-------------------------
⚠️  user_id [HIGH CARDINALITY]
  Used in: metrics (user_events), spans (HTTP GET), logs (login)
  Unique values: ~10,000
  Impact: HIGH

✓ http.method
  Used in: metrics (http_requests_total), spans (HTTP GET, HTTP POST)
  Unique values: 7
  Impact: Low

Recommendations
---------------
• Consider removing 'user_id' from metrics and spans
• Review 'database.query' span - high cardinality from db.statement
• Review label usage in 'user_events' metric
• Use OpenTelemetry semantic conventions for consistency
• High cardinality detected - use session export for detailed analysis

Session Export
--------------
For detailed analysis, export session with:
  --session-export output.json
Then load in full OCC UI:
  occ session load output.json
```

### Session Export Format

If `--session-export` is specified, OCC exports full session data in standard session format:
  "summary": {
    "total_metrics": 150,
    "total_attributes": 45,
    "high_cardinality_metrics": 5,
    "total_samples_received": 1500000
  },
  "metrics": [
    {
      "name": "http_requests_total",
      "cardinality": 12000,
      "label_keys": ["method", "path", "status"],
      "sample_count": 50000,
      "severity": "warning"
    }
  ],
  "attributes": [
    {
      "key": "user_id",
      "usage_count": 8,
      "estimated_cardinality": 10000,
      "severity": "critical"
    }
  ],
  "thresholds": {
    "cardinality_warning": 1000,
    "cardinality_critical": 10000,
    "status": "warning"
  }
}
```

### Severity Levels

| Severity | Cardinality | Exit Code |
|----------|-------------|------------|
| ok | < 1000 | 0 |
| warning | 1000-9999 | 1 (if --exit-on-threshold) |
| critical | ≥ 10000 | 2 (if --exit-on-threshold) |

## API Endpoints

All standard API endpoints remain available:

- `GET /api/v1/metrics` - List all metrics
- `GET /api/v1/metrics/{name}` - Get metric details
- `GET /api/v1/attributes` - List all attributes
- `GET /api/v1/attributes/top-cardinality` - High-cardinality attributes
- `GET /api/v1/health` - Health check
- `GET /api/v1/report` - Generate report on-demand

### New Endpoint

```
GET /api/v1/report?format=json|yaml|text

Response: Same structure as file report
Status: 200 OK, 503 if shutting down
```

## Implementation Considerations

### Graceful Shutdown

```go
type MinimalServer struct {
    otlpReceiver *OTLPReceiver
    apiServer    *APIServer
    storage      Storage
    shutdownCh   chan struct{}
    reportGen    *ReportGenerator
}

func (s *MinimalServer) Shutdown(ctx context.Context) error {
    // 1. Stop OTLP receiver
    s.otlpReceiver.GracefulStop()
    
    // 2. Wait for in-flight requests (with timeout)
    select {
    case <-s.otlpReceiver.Done():
    case <-time.After(5 * time.Second):
        log.Warn("OTLP receiver drain timeout")
    }
    
    // 3. Generate report
    if err := s.reportGen.Generate(); err != nil {
        return fmt.Errorf("report generation failed: %w", err)
    }
    
    // 4. Shutdown API
    return s.apiServer.Shutdown(ctx)
}
```

### Signal Handling

```go
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

go func() {
    <-sigCh
    log.Info("Received shutdown signal")
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        log.Error("Shutdown error", "error", err)
        os.Exit(1)
    }
    os.Exit(server.ExitCode())
}()
```

## Testing Strategy

### Unit Tests
- Report generation with various data sets
- Memory eviction policy
- Shutdown sequence
- Exit code calculation

### Integration Tests
- Start in minimal mode, send OTLP data, verify report
- Duration timeout triggers shutdown
- Signal handling (SIGTERM)
- API availability during minimal mode
- Memory limit enforcement

### E2E Tests
- GitHub Actions workflow example
- Docker Compose CI setup
- Kubernetes Job example

## Performance Targets

- **Startup time**: < 2 seconds
- **Memory baseline**: < 50MB empty
- **Memory with load**: < 100MB for 1000 metrics
- **Shutdown time**: < 5 seconds
- **Report generation**: < 1 second for 10k metrics

## Report Examples

See [REPORT-EXAMPLES.md](REPORT-EXAMPLES.md) for full realistic examples of:
- Basic mode report (e-commerce app)
- Verbose mode report (partial)
- Comparison table

## Security Considerations

- No authentication required in minimal mode (CI environment assumed secure)
- Optional: `--api-auth-token` flag for added security
- Report may contain sensitive metric names (document security implications)
- Ensure OTLP endpoints only bind to localhost by default in CI mode

## Monitoring & Observability

### Logs
```
INFO: Starting OCC in minimal mode (duration: 5m)
INFO: OTLP gRPC receiver listening on :4317
INFO: OTLP HTTP receiver listening on :4318
INFO: API server listening on :8080
WARN: Memory usage 82% (420MB / 512MB)
INFO: Duration timeout reached, initiating shutdown
INFO: Report generated: /tmp/report.json
INFO: Shutdown complete, exit code: 1
```

### Metrics (if enabled)
- `occ_minimal_mode` (gauge: 1=minimal, 0=normal)
- `occ_memory_usage_bytes` (gauge)
- `occ_metrics_tracked` (gauge)
- `occ_samples_received_total` (counter)

## Documentation Requirements

1. **README update**: Add minimal mode section
2. **CI/CD Guide**: Examples for:
   - GitHub Actions
   - Docker Compose
3. **Session Workflow**: Document CI → report → load in UI workflow
4. **API Documentation**: Update with `/api/v1/report` endpoint
5. **Report Schema**: JSON schema definition (session-compatible)
6. **Troubleshooting**: Common issues in CI environments

## Future Enhancements

- **Report comparison**: Compare report against baseline
- **Plugin support**: Custom report generators
- **Alert webhooks**: POST report to URL on completion
- **Incremental reports**: Generate report every N minutes
- **Remote storage**: Optional cloud upload for reports
