# OTLP Cardinality Checker - Product Document

## Problem Statement

OpenTelemetry telemetry kan snabbt bli ohållbar när instrumentering introducerar höga kardinalitetsattribut. Detta leder till:

- **Oväntade kostnader** i observability-backends (Prometheus, Datadog, etc.)
- **Performance-problem** i collectors och backends
- **Svårigheter att identifiera källan** till kardinalitetsproblem
- **Reaktiv problemhantering** istället för proaktiv övervakning

Det saknas verktyg som enkelt kan inspektera och förstå **vilken metadata-struktur** som faktiskt skickas via OTLP innan data når produktionssystem.

## Solution

OTLP Cardinality Checker är ett lightweight analysverktyg skrivet i Go som:

1. **Tar emot OTLP-telemetri** via gRPC och HTTP (som en standard OTLP-endpoint)
2. **Extraherar metadata-struktur** från all inkommande telemetri
3. **Lagrar och visualiserar** vilka nycklar och attribut som används
4. **Ger insikter** om potentiella kardinalitetsproblem

### Vad analyseras

#### Metrics
- Metric name
- Alla unika label-nycklar (inte värden)
- Resource attributes (nycklar)
- Scope/instrumentation library info

#### Logs  
- Unika nycklar i resource attributes
- Unika nycklar i log attributes
- Body struktur ignoreras (för att undvika höga volymer)

#### Traces
- Unika span names
- Unika nycklar i span attributes
- Unika nycklar i resource attributes
- Event och link attribute nycklar

### Vad analyseras INTE

- **Inga värden** lagras (för att undvika kardinalitetsexplosion i verktyget själv)
- **Ingen time-series data** - fokus är på struktur, inte mätvärden
- **Ingen full trace reconstruction** - bara metadata

## Target Audience

### Primary Users
- **Platform Engineers** som ansvarar för observability-infrastruktur
- **SRE Teams** som behöver förstå telemetrikostnader
- **DevOps Engineers** som troubleshooting instrumentering

### Use Cases
1. **Pre-production validation** - Verifiera metadata innan deploy
2. **Cost analysis** - Förstå vad som driver observability-kostnader
3. **Instrumentation debugging** - Identifiera over-instrumented services
4. **Metadata governance** - Säkerställ att teams följer attribute conventions

## Key Features

### 1. OTLP-Kompatibel Receiver
- Stödjer både **gRPC** (port 4317) och **HTTP** (port 4318)
- Fullt kompatibel med OpenTelemetry SDK:er
- Fungerar som drop-in replacement för testing

### 2. Metadata Extraction
- Real-time parsing av OTLP protobuf
- Automatisk upptäckt av nya nycklar
- Ingen sampling - all metadata analyseras

### 3. Efficient Storage
- In-memory datastrukturer för snabb access
- Minimal memory footprint (endast unika nycklar)
- Optional persistence till PostgreSQL för historik

### 4. Query & Analysis API
- REST API för att hämta metadata
- Filter per signal type, service, miljö
- Export till JSON/YAML för vidare analys

### 5. Web UI (Future)
- Visualisera metadata-struktur
- Upptäck kardinalitetsproblem
- Jämför över tid

## Non-Goals

- **Inte en full observability backend** - använd Prometheus/Grafana för det
- **Inte en metrics aggregator** - aggregering sker i vanliga backends
- **Inte för production metrics collection** - verktyget är för analys

## Success Metrics

1. **Adoption**: Teams använder verktyget i CI/CD pipelines
2. **Cost reduction**: Teams identifierar och fixar kardinalitetsproblem
3. **Developer experience**: < 5 minuter från start till första insikt
4. **Performance**: Kan hantera 10K spans/sec på standard laptop

## Alternatives Considered

### Prometheus cardinality explorer
- **Pro**: Etablerat verktyg
- **Con**: Endast för metrics, kräver full Prometheus stack

### Full observability backend
- **Pro**: Komplett lösning
- **Con**: Overkill för enbart metadata-analys, dyr att köra

### Custom collector processor
- **Pro**: Kan integreras i befintlig pipeline
- **Con**: Svårt att query och analysera historisk data

## Technical Constraints

- **Go 1.21+** för modern standardbibliotek
- **PostgreSQL 14+** för optional persistence
- **Linux/macOS/Windows** support
- **Low resource footprint** - ska kunna köras på developer laptops

## Timeline

### Phase 1: MVP (4-6 veckor)
- OTLP receiver (gRPC + HTTP)
- Basic metadata extraction
- In-memory storage
- Simple REST API

### Phase 2: Production Ready (2-3 veckor)
- PostgreSQL persistence
- Comprehensive testing
- Documentation
- Docker support

### Phase 3: Enhanced Features (Future)
- Web UI
- Alerting för nya höga-kardinalitetsattribut
- Integration med CI/CD
- Comparison tools

## Open Questions

1. **Retention policy**: Hur länge ska metadata lagras?
   - Förslag: 30 dagar default, konfigurerbar

2. **Multi-tenancy**: Stödja flera miljöer/teams i samma instans?
   - Förslag: Phase 2 feature, filter på resource attributes

3. **Export format**: Vilka format behövs för integration?
   - Förslag: JSON, YAML, CSV för Excel-analys

## References

- [OpenTelemetry Protocol Specification](https://opentelemetry.io/docs/specs/otlp/)
- [Cardinality in Prometheus](https://prometheus.io/docs/practices/naming/#labels)
- [OpenTelemetry Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/)
