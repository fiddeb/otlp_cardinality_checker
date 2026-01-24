# OTLP Cardinality Checker

<p align="center">
  <img src="docs/logo.png" alt="OTLP Cardinality Checker Logo" width="400">
</p>

> A lightweight metadata analysis tool for OpenTelemetry Protocol (OTLP) telemetry

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)


## What is this?

OTLP Cardinality Checker is a tool that helps you understand the **metadata structure** of your OpenTelemetry telemetry before it explodes your observability costs. It acts as an **OTLP endpoint destination** that receives data from an OpenTelemetry Collector and analyzes incoming metrics, traces, and logs to identify:

- Which metric names and label keys are being used
- Which span names and attribute keys exist in traces
- Which attribute keys appear in logs
- Potential cardinality issues in your instrumentation

**Key principle**: We only track metadata keys, never actual values, to avoid creating cardinality problems in the tool itself.

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
┌─────────────────────────────────┐
│ Data Sources                    │
│ (Kafka/Redis/Prometheus/etc.)   │
└──────────┬──────────────────────┘
           │
           ↓
┌──────────────────────────────────┐
│  OpenTelemetry Collector         │
│  • Receivers (various)           │
│  • Processors                    │
│  • OTLP HTTP Exporter            │
└──────────┬───────────────────────┘
           │ OTLP/HTTP
           ↓
┌──────────────────────────────────┐
│  OTLP Cardinality Checker        │
│  • OTLP gRPC Endpoint (4317)     │
│  • OTLP HTTP Endpoint (4318)     │
│  • Metadata Extraction           │
│  • Cardinality Analysis          │
└──────────┬───────────────────────┘
           │
           ↓
┌──────────────────────────────────┐
│  REST API (8080)                 │
│  Query & Explore Metadata        │
└──────────────────────────────────┘
```

## Features

**Current (Phase 1 & 2 - Complete):**
- ✅ **OTLP HTTP Endpoint** - Receives data from OpenTelemetry Collector (port 4318)
- ✅ **OTLP gRPC Endpoint** - Full gRPC support (port 4317)
- ✅ **Source Agnostic** - Works with any Collector receiver (Kafka, Redis, Prometheus, etc.)
- ✅ **Metadata Extraction** - Analyzes metrics, traces, and logs
- ✅ **Automatic Log Template Extraction** - Drain algorithm for pattern detection (20-30k+ EPS)
- ✅ **Cardinality Tracking** - Unique value count estimation using HyperLogLog algorithm
- ✅ **Global Attribute Catalog** - Track all attribute keys across ALL signals (metrics/spans/logs)
- ✅ **Span Name Pattern Analysis** - Detect high-cardinality span naming with pattern extraction
- ✅ **In-Memory Storage** - Fast, handles 500,000+ metrics, ephemeral by design
- ✅ **REST API** - Query metadata with pagination, filtering, and sorting
- ✅ **Web UI** - Visual exploration with interactive dashboards and filtering
- ✅ **Service-Level Filtering** - View telemetry by service.name
- ✅ **Docker & Kubernetes** - Production-ready deployment manifests

**Planned (Phase 3):**
- **Alerting** - Notify on cardinality thresholds  

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
  "success": true,
  "data": {
    "name": "http_server_duration",
    "type": "Histogram",
    "unit": "ms",
    "label_keys": ["method", "status_code", "route"],
    "resource_keys": ["service.name", "service.version"],
    "sample_count": 15420
  },
  "metadata": {
    "timestamp": "2024-01-15T10:30:00Z"
  }
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
  jq -r '.data[] | select(.label_keys | to_entries[] | .value.estimated_cardinality > 100) | .name'
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
  jq -r '.data[] | [.name, .type, (.label_keys | length), .sample_count] | @csv' > metrics.csv

# Analyze in Excel/SQL
```

### 3. Instrumentation Debugging

Identify over-instrumented services:

```bash
# Filter by service
curl "http://localhost:8080/api/v1/metrics?service=payment-service"

# Check for unusual label keys
# Example: user_id in labels = BAD (high cardinality)
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

See [docs/research/log-templating/](docs/research/log-templating/) for algorithm details and performance benchmarks.

## Documentation

- **[docs/USAGE.md](docs/USAGE.md)** - 📘 **Start here!** Practical usage guide with examples
- **[docs/API.md](docs/API.md)** - Complete REST API documentation with pagination examples
- **[docs/SCALABILITY.md](docs/SCALABILITY.md)** - Performance optimizations and scalability limits
- **[docs/research/log-templating/](docs/research/log-templating/)** - Automatic log template extraction with Drain algorithm
  - **[README.md](docs/research/log-templating/README.md)** - Algorithm research and comparison
  - **[STATUS.md](docs/research/log-templating/STATUS.md)** - Implementation details and benchmarks
- **[k8s/README.md](k8s/README.md)** - Kubernetes deployment guide
- **[scripts/README.md](scripts/README.md)** - Load testing guide with K6
- **[PRODUCT.md](PRODUCT.md)** - Product overview and requirements
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Technical architecture and design decisions
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - How to contribute to the project
- **[.github/copilot-instructions.md](.github/copilot-instructions.md)** - GitHub Copilot coding guidelines

## Project Status

� **Status**: Phase 1 Complete - Production Ready

### Phase 1: MVP ✅ **COMPLETE**
- [x] OTLP HTTP receiver implementation (port 4318)
- [x] Metadata extraction for metrics, traces, and logs  
- [x] In-memory storage with cardinality tracking
- [x] REST API with pagination support
- [x] Docker and Kubernetes deployment
- [x] Load testing (validated with 50,000 metrics)
- [x] Comprehensive documentation

**Performance validated:**
- ✅ Handles 50,000 metrics using 421 MB memory (~8.4 KB per metric)
- ✅ Throughput: 450 req/s, 4,455 datapoints/s sustained
- ✅ Latency: P95 45ms under load, median 1.45ms
- ✅ Success rate: 99.95%
- ✅ See [docs/SCALABILITY.md](docs/SCALABILITY.md) for details

### Phase 2: Production Hardening (Planned)
- Handles 10,000 metrics comfortably (~150MB memory)
- Sub-millisecond metadata updates
- <10ms API responses for 100 items

### Phase 2: Production Hardening ✅ **COMPLETE**
- [x] OTLP gRPC receiver (port 4317)
- [x] Automatic log template extraction with Drain algorithm
- [x] Pattern pre-masking (timestamps, UUIDs, IPs, URLs, etc.)
- [x] Configurable similarity threshold for template specificity
- [x] Comprehensive pattern validation tests
- [x] Performance benchmarks (53k-1.6M EPS)
- [ ] Configuration file support (YAML)
- [ ] Helm charts for easier Kubernetes deployment

### Phase 3: Enhanced Features (Future)
- [x] Web UI for visualization
- [ ] Alerting on cardinality thresholds
- [ ] CI/CD integrations
- [ ] Time-series cardinality trends
- [ ] Comparison tools
- [ ] Multi-replica support with shared storage

## Architecture Overview

```
┌─────────────────────────────────────────┐
│  OTLP Cardinality Checker               │
│                                         │
│  ┌─────────────────────────────────┐  │
│  │  OTLP Receiver Layer            │  │
│  │  └─ HTTP Server (port 4318)     │  │
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
│  │  └─ In-Memory Store             │  │
│  └──────────────┬──────────────────┘  │
│                 │                      │
│  ┌──────────────▼──────────────────┐  │
│  │  REST API (port 8080)           │  │
│  └─────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed design.

## Technology Stack

- **Language**: Go 1.24+
- **OTLP**: OpenTelemetry Collector SDK
- **Storage**: In-memory only (ephemeral by design)
- **API**: net/http with chi router
- **Testing**: Go testing, testify

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for:

- Development setup
- Coding guidelines
- Testing requirements
- PR process

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

## Alternatives Considered

| Tool | Pros | Cons |
|------|------|------|
| **Prometheus Cardinality Explorer** | Established, metrics-focused | Only metrics, requires Prometheus |
| **Full Observability Backend** | Complete solution | Overkill, expensive to run |
| **Custom Collector Processor** | Integrates with existing pipeline | Hard to query historical data |

We chose a standalone tool for simplicity and focus on metadata analysis.

## FAQ

**Q: Does this replace my observability backend?**  
A: No, this is a development/analysis tool, not a production backend.

**Q: Will this work with my existing OTLP setup?**  
A: Yes! It's a standard OTLP receiver. Just point your exporter to it.

**Q: What about cardinality estimation?**  
A: MVP focuses on metadata keys only. Cardinality estimation (HyperLogLog) is planned for Phase 3.

**Q: Is this production-ready?**  
A: Yes! Phase 1 is complete and tested with 50,000 metrics using 421 MB memory. See [docs/SCALABILITY.md](docs/SCALABILITY.md) for performance details.

**Q: How much memory does it use?**  
A: Approximately 8-9 KB per metric. For 10,000 metrics: ~85 MB, for 50,000 metrics: ~425 MB.

**Q: Can I run multiple replicas in Kubernetes?**  
A: Each replica has independent in-memory storage. Data is not shared between replicas by design.

**Q: What is span name pattern analysis?**  
A: The Trace Patterns feature groups similar span names into patterns. For example, `GET /api/v1/users/123` and `GET /api/v1/users/456` become `GET <URL>`. This helps identify high-cardinality span naming issues. Access it via the "Trace Patterns" tab in the UI or `/api/v1/span-patterns` API.

**Q: What is automatic log template extraction?**  
A: When enabled with `USE_AUTOTEMPLATE=true`, the tool uses the Drain algorithm to automatically group similar log messages into patterns. For example, "User 123 logged in" and "User 456 logged in" become one template: "User <NUM> logged in". This helps identify log cardinality issues without manual configuration.

**Q: How fast is the Drain template extraction?**  
A: Performance ranges from 53k to 1.6M events per second depending on concurrency, exceeding the 20-30k EPS target by 2-80x. See [docs/research/log-templating/STATUS.md](docs/research/log-templating/STATUS.md) for benchmarks.

**Q: Can I adjust how specific the templates are?**  
A: Yes, the similarity threshold is configurable (default 0.7). Higher values create more specific templates, lower values create more generic ones.

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

Apache License 2.0 - See [LICENSE](LICENSE) for details.

## Acknowledgments

- [OpenTelemetry](https://opentelemetry.io/) for the OTLP specification
- [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) for receiver components
- The Go community for excellent tooling

## Contact

- **Issues**: [GitHub Issues](https://github.com/yourusername/otlp_cardinality_checker/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/otlp_cardinality_checker/discussions)

## Roadmap

See our [Project Board](https://github.com/yourusername/otlp_cardinality_checker/projects) for current work and future plans.

---

**Star ⭐ this repo if you find it useful!**

