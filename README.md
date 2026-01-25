# OTLP Cardinality Checker

<p align="center">
  <img src="docs/logo.png" alt="OTLP Cardinality Checker Logo" width="400">
</p>

> A lightweight metadata analysis tool for OpenTelemetry Protocol (OTLP) telemetry

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)


## What is this?

OTLP Cardinality Checker is a tool that helps you understand the **metadata structure** of your OpenTelemetry telemetry before it explodes your observability costs. It acts as an **OTLP endpoint** that receives telemetry data over HTTP or gRPC and analyzes incoming metrics, traces, and logs to identify:

- Which metric names and attribute keys are being used
- Which span names and attribute keys exist in traces  
- Which log attribute keys appear
- Resource attributes across all signals
- Potential cardinality issues in your instrumentation

**Key principle**: We only track metadata keys, never actual values, to avoid creating cardinality problems in the tool itself.

You can send data directly from applications using the OpenTelemetry SDK, or route data through an OpenTelemetry Collector for more complex scenarios.

## Problem Statement

High cardinality in observability data leads to:
- Unexpected costs in backends (Prometheus, Datadog, etc.)
- Performance problems in collectors and storage
- Difficulty identifying the source of cardinality explosions
- Reactive firefighting instead of proactive monitoring

Most teams discover cardinality issues **after** they hit production and the bill arrives.

## Solution

OTLP Cardinality Checker gives you visibility into your telemetry metadata structure:

```
┌──────────────────────────────────┐
│  Applications with OTel SDK      │
│  (Direct OTLP export)            │
└──────────┬───────────────────────┘
           │ OTLP
           │
           ↓
      ┌────────────────────────┐
      │                        │
      │  OTLP Cardinality      │◄──────────┐
      │  Checker               │           │
      │                        │           │ OTLP
      │  • gRPC (4317)         │           │
      │  • HTTP (4318)         │           │
      │  • Metadata Analysis   │           │
      │  • Cardinality Est.    │           │
      │                        │           │
      └────────┬───────────────┘           │
               │                           │
               │                ┌──────────┴──────────────┐
               │                │  OpenTelemetry          │
               │                │  Collector (optional)   │
               │                │                         │
               │                │  For complex routing,   │
               │                │  processing, or         │
               │                │  multiple sources       │
               │                └─────────▲───────────────┘
               │                          │
               ↓                          │
      ┌────────────────────┐    ┌─────────────────────────┐
      │  REST API (8080)   │    │  Data Sources           │
      │  Web UI            │    │  (Kafka/Redis/Prom...)  │
      └────────────────────┘    └─────────────────────────┘
```

## Features

- OTLP HTTP (4318) and gRPC (4317) endpoints
- Works with any OTel Collector receiver (Kafka, Redis, Prometheus, etc.)
- Analyzes metrics, traces, and logs metadata
- Log template extraction using Drain algorithm (53k-1.6M events/sec)
- Span name pattern detection for high-cardinality naming
- Cardinality estimation with HyperLogLog
- Global attribute catalog across all signals
- In-memory storage (ephemeral by design)
- REST API with pagination and filtering
- Embedded React Web UI (built into the binary)
- Docker and Kubernetes deployment

## Quick Start

### Prerequisites

- Go 1.24+ (for building from source)
- Docker (optional, for building container images)
- Kubernetes (optional, for deployment)

### Installation

#### Option 1: Build and Run Locally

```bash
# Clone repository
git clone https://github.com/fiddeb/otlp_cardinality_checker.git
cd otlp_cardinality_checker

# Build using Makefile (recommended)
make build

# Run
./bin/occ
```

#### Option 2: Deploy to Kubernetes

```bash
# Build Docker image (on a machine with Docker)
docker build -t occ:latest .

# Tag and push to your registry
docker tag occ:latest your-registry/occ:latest
docker push your-registry/occ:latest

# Deploy to Kubernetes
kubectl apply -f k8s/

# Port-forward to access locally
kubectl port-forward svc/occ 8080:8080 4317:4317 4318:4318
```

See [k8s/README.md](k8s/README.md) for detailed Kubernetes deployment instructions.

The tool will start listening on:
- **gRPC**: `localhost:4317` (OTLP gRPC endpoint)
- **HTTP**: `localhost:4318` (OTLP HTTP endpoint)
- **API**: `localhost:8080` (REST API)
- **Web UI**: `http://localhost:8080` (Embedded React interface)

### Web UI

![Dashboard](docs/img/dashboard.png)

 The UI is built into the binary and provides:

- **Dashboard** - Service overview with telemetry summary
- **Metadata Complexity** - Identify metrics/spans/logs with high-cardinality attributes
- **Metrics Overview** - Aggregated view of all metrics with type and unit breakdowns
- **Active Series** - Real-time cardinality tracking with HyperLogLog estimation
- **Metrics Details** - Filterable table of all metrics with attribute analysis
- **Traces** - Span name analysis with attribute keys and sample counts
- **Trace Patterns** - Aggregated span name patterns (e.g., `GET <URL>`)
- **Logs** - Log template extraction by severity level with Drain algorithm
- **Attributes** - Global attribute catalog across all signals
- **Noisy Neighbors** - Identify services generating excessive telemetry volume
- **Memory** - Runtime memory usage statistics
- **Service Explorer** - Deep dive into all telemetry for a specific service
- **Dark Mode** - Toggle between light and dark themes

The UI is statically compiled into the binary, so no external files are needed.

### Using with OpenTelemetry SDK

Point your application's OTLP exporter to the checker:

```bash
# Environment variables
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf

# Run your application
./your-app
```

### Query Metadata

```bash
# List all metrics
curl http://localhost:8080/api/v1/metrics

# Get specific metric
curl http://localhost:8080/api/v1/metrics/http_server_duration

# List spans
curl http://localhost:8080/api/v1/spans

# Get log templates by severity (with autotemplate enabled)
curl http://localhost:8080/api/v1/logs/INFO

# Get span patterns (aggregated pattern analysis)
curl http://localhost:8080/api/v1/span-patterns

# Get summary
curl http://localhost:8080/api/v1/summary
```

Example response:

```json
{
  "data": [
    {
      "name": "http_server_duration",
      "type": "Histogram",
      "unit": "ms",
      "attribute_keys": {
        "method": {
          "count": 15420,
          "estimated_cardinality": 7,
          "percentage": 100.0
        },
        "status_code": {
          "count": 15420,
          "estimated_cardinality": 10,
          "percentage": 100.0
        },
        "route": {
          "count": 15420,
          "estimated_cardinality": 25,
          "percentage": 100.0
        }
      },
      "resource_keys": {
        "service.name": {
          "count": 15420,
          "estimated_cardinality": 1,
          "percentage": 100.0
        },
        "service.version": {
          "count": 15420,
          "estimated_cardinality": 1,
          "percentage": 100.0
        }
      },
      "sample_count": 15420,
      "services": {
        "my-service": 15420
      }
    }
  ],
  "total": 1,
  "limit": 100,
  "offset": 0,
  "has_more": false
}
```

## Use Cases

### 1. Pre-Production Validation

Run the checker in your CI/CD pipeline to validate instrumentation:

```bash
# Start checker
./bin/occ &

# Run integration tests
go test ./...

# Query metadata
curl 'http://localhost:8080/api/v1/metrics' | jq '.' > metadata-report.json

# Check for high cardinality (example)
curl -s 'http://localhost:8080/api/v1/metrics' | \
  jq -r '.data[] | select(.attribute_keys | to_entries[] | .value.estimated_cardinality > 100) | .name'
```

### 2. Cost Analysis

Understand what's driving your observability costs:

```bash
# Run in staging for 24 hours
./bin/occ

# Export metadata to JSON
curl -s 'http://localhost:8080/api/v1/metrics' | jq '.' > metrics.json

# Or use jq to create CSV format
curl -s 'http://localhost:8080/api/v1/metrics' | \
  jq -r '.data[] | [.name, .type, (.attribute_keys | length), .sample_count] | @csv' > metrics.csv

# Analyze in Excel/SQL
```

### 3. Instrumentation Debugging

Identify over-instrumented services:

```bash
# Filter by service
curl "http://localhost:8080/api/v1/metrics?service=payment-service"

# Check for unusual attribute keys
# Example: user_id in attributes = BAD (high cardinality)
```

## Configuration

The server uses environment variables for configuration:

```bash
# OTLP gRPC endpoint address (default: 0.0.0.0:4317)
export OTLP_GRPC_ADDR="0.0.0.0:4317"

# OTLP HTTP endpoint address (default: 0.0.0.0:4318)
export OTLP_HTTP_ADDR="0.0.0.0:4318"

# REST API address (default: 0.0.0.0:8080)
export API_ADDR="0.0.0.0:8080"

# Enable automatic log template extraction with Drain algorithm (default: true)
export USE_AUTOTEMPLATE=true

# Run the server
./bin/occ
```

**Note**: OTLP Cardinality Checker uses **in-memory storage only**. Data is ephemeral and lost on restart. This is by design - the tool is meant for diagnostic analysis, not long-term data retention. Simply restart and re-analyze from your data sources as needed

### Automatic Log Template Extraction

When `USE_AUTOTEMPLATE=true`, the tool uses the **Drain algorithm** to automatically detect patterns in log bodies:

```bash
# Enable autotemplate mode
USE_AUTOTEMPLATE=true ./bin/occ
```

**Features:**
- **Algorithm**: Drain (ICWS'17) - Fixed-depth tree with token similarity clustering
- **Performance**: 53k-1.6M events/sec (exceeds 20-30k target by 2-80x)
- **Pattern detection**: Automatically groups similar log messages into templates
- **Pre-masking**: Recognizes timestamps, UUIDs, IPs, URLs, emails, and more
- **Example bodies**: Each template stores an example log for reference

**Example patterns detected:**
```
Syslog:
  Dec  4 10:30:15 host sshd[1234]: Accepted publickey for user from 1.2.3.4
  → <TIMESTAMP> host sshd[<NUM>]: Accepted publickey for user from <IP>

Apache:
  [Sun Dec 04 04:51:08 2005] [notice] jk2_init() Found child 6725
  → <TIMESTAMP> [notice] jk2_init() Found child <NUM>
```

**Query templates:**
```bash
# Get all INFO-level log templates
curl http://localhost:8080/api/v1/logs/INFO | jq '.body_templates'

# Get ERROR-level templates with sample counts
curl http://localhost:8080/api/v1/logs/ERROR | jq '{severity, sample_count, body_templates}'
```

## Documentation

- **[docs/USAGE.md](docs/USAGE.md)** - **Start here!** Practical usage guide with examples
- **[docs/API.md](docs/API.md)** - Complete REST API documentation with pagination examples
- **[k8s/README.md](k8s/README.md)** - Kubernetes deployment guide
- **[scripts/README.md](scripts/README.md)** - Load testing guide with K6
- **[.github/copilot-instructions.md](.github/copilot-instructions.md)** - GitHub Copilot coding guidelines

## Project Status

**Phase 1 & 2 complete. Phase 3 in progress.**

### Phase 1: MVP **COMPLETE**
- [x] OTLP HTTP receiver implementation (port 4318)
- [x] Metadata extraction for metrics, traces, and logs  
- [x] In-memory storage with cardinality tracking
- [x] REST API with pagination support
- [x] Docker and Kubernetes deployment
- [x] Load testing (validated with 50,000 metrics)
- [x] Comprehensive documentation

**Performance:**
- 50,000 metrics in 421 MB memory (~8.4 KB per metric)
- 450 req/s, 4,455 datapoints/s sustained
- P95 latency 45ms under load

### Phase 2: Production Hardening
- [x] OTLP gRPC receiver (port 4317)
- [x] Automatic log template extraction with Drain algorithm
- [x] Pattern pre-masking (timestamps, UUIDs, IPs, URLs, etc.)
- [x] Configurable similarity threshold for template specificity
- [x] Comprehensive pattern validation tests
- [x] Performance benchmarks (53k-1.6M EPS)
- [ ] Configuration file support (YAML)
- [ ] Helm charts for easier Kubernetes deployment

### Phase 3: Enhanced Features (Future)
- [x] Web UI for visualization (embedded React UI)
- [ ] CI/CD integrations
- [ ] Comparison tools


## Technology Stack

- **Language**: Go 1.24+
- **OTLP**: gRPC (4317) and HTTP (4318) receivers using OpenTelemetry SDK
- **Storage**: In-memory (ephemeral by design)
- **API**: REST API with chi router
- **UI**: React with Vite (embedded in binary)
- **Algorithms**: Drain (log templates), HyperLogLog (cardinality estimation)
- **Testing**: Go testing, testify, K6 load tests

## Contributing

We welcome contributions!

### Quick Contribution Guide

```bash
# Fork and clone
git clone https://github.com/yourusername/otlp_cardinality_checker.git

# Create branch
git checkout -b feature/your-feature

# Make changes and test
go test ./...

# Commit and push
git commit -m "feat: add your feature"
git push origin feature/your-feature

# Create PR
```

## Alternatives

| Tool | When to use instead |
|------|--------------------|
| Prometheus /api/v1/status/tsdb or VictoriaMetrics Cardinality Explorer | If you only need metrics cardinality analysis and already run Prometheus/VictoriaMetrics |
| Full observability backend (Grafana Cloud, Datadog, etc.) | If you need long-term storage, alerting, and dashboards |
| OpenTelemetry Collector processor | If you want inline telemetry transformation without a separate analysis tool |

This tool is useful when you want a quick, standalone analysis of your OTLP telemetry metadata across metrics, traces, and logs - without committing to a full observability backend.

## FAQ

**Does this replace my observability backend?**  
No. This is a diagnostic tool for understanding your telemetry structure, not for storing or querying production data.

**How much memory does it use?**  
About 8-9 KB per metric. 10k metrics ≈ 85 MB, 50k metrics ≈ 425 MB.

**What about persistence?**  
Data lives in memory and is lost on restart. This is intentional - restart and re-analyze as needed.

**What is span name pattern analysis?**  
Groups similar span names into patterns. `GET /users/123` and `GET /users/456` become `GET <URL>`. See the "Trace Patterns" tab or `/api/v1/span-patterns`.

**What is log template extraction?**  
The Drain algorithm groups similar log messages. "User 123 logged in" and "User 456 logged in" become "User <NUM> logged in". Runs at 53k-1.6M events/sec.

## Deployment

### Kubernetes

See [k8s/README.md](k8s/README.md) for complete deployment instructions including:
- Building Docker images
- Deploying to Kubernetes
- Configuring OpenTelemetry Collector
- Ingress setup
- Monitoring and troubleshooting

### Docker

```bash
# Build image
docker build -t occ:latest .

# Run container
docker run -p 4317:4317 -p 4318:4318 -p 8080:8080 occ:latest
```

## License

Apache License 2.0 - See [LICENSE](LICENSE) and [NOTICE](NOTICE) for details.

## Contact

- **Issues**: [GitHub Issues](https://github.com/yourusername/otlp_cardinality_checker/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/otlp_cardinality_checker/discussions)

## Roadmap

See [Project Board](https://github.com/yourusername/otlp_cardinality_checker/projects) for current work and future plans.

---

**Star ⭐ this repo if you find it useful!**

