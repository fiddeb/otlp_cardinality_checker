# Design: Span Name Pattern Analysis

## Context

The system currently tracks exact span names as keys in metadata storage. Services using dynamic values in span names (e.g., REST paths with IDs, operation names with timestamps) create high-cardinality metadata that's hard to analyze. Log analysis already solves this with template extraction using regex patterns.

**Stakeholders:** Platform engineers analyzing instrumentation design, service owners reviewing trace structure

**Constraints:**
- Must not break existing span metadata structure
- Reuse existing pattern matching infrastructure from log analysis
- Minimal memory overhead (target: <1% increase for typical workloads)
- No external dependencies

## Goals / Non-Goals

**Goals:**
- Extract patterns from span names (e.g., `GET /users/{id}` from `GET /users/123`)
- Track pattern distribution (count and percentage)
- Store example span names for each pattern
- Enable quick visualization of span naming conventions

**Non-Goals:**
- Real-time pattern learning (use fixed regex patterns, not Drain algorithm)
- Event/link pattern analysis (future scope)
- Automatic remediation or alerting on bad patterns
- Cross-service pattern correlation (each service analyzed independently)

## Decisions

### Decision: Use Regex Patterns Instead of Drain

**What:** Reuse `config.CompiledPattern` regex approach from log template extraction, not the Drain clustering algorithm.

**Why:**
- Span names are typically structured (URLs, gRPC methods) vs. free-form log messages
- Regex patterns work well for known formats: `/users/{id}`, `v1.Service/Method`
- Drain adds complexity (tree clustering, similarity matching) not needed for structured names
- Reusing existing patterns (`<NUM>`, `<UUID>`, `<TIMESTAMP>`) is sufficient

**Alternatives considered:**
1. **Drain algorithm** - Overkill for structured span names; adds memory/CPU overhead
2. **Manual pattern list** - Too rigid; regex provides better coverage with less configuration
3. **No patterns** - Current state; doesn't solve the core problem

### Decision: Store Patterns Per SpanMetadata

**What:** Add `NamePatterns []*SpanNamePattern` field directly to `SpanMetadata` struct.

**Why:**
- Keeps pattern data with its source (span name metadata)
- Simplifies API responses (patterns included automatically)
- No additional storage layer needed
- Consistent with how log templates are stored in `LogMetadata.BodyTemplates`

**Alternatives considered:**
1. **Separate patterns storage** - Adds complexity; requires joins for API queries
2. **Global pattern registry** - Loses service-specific context; harder to analyze per-service

### Decision: Track Example Span Names (Max 3)

**What:** Store up to 3 example span names per pattern in `SpanNamePattern.Examples []string`.

**Why:**
- Helps users verify pattern accuracy ("does this pattern match what I expect?")
- Minimal memory impact (~200 bytes per pattern with 3 examples)
- Similar to `KeyMetadata.ValueSamples` approach for attribute values

**Alternatives considered:**
1. **No examples** - Users can't validate pattern correctness
2. **Single example** - Not enough to show pattern variety
3. **Unlimited examples** - Memory risk for high-cardinality spans

## Technical Approach

### Pattern Extraction Flow

```
Span Name → Apply Patterns → Template → Track Metadata
"GET /users/123" → (replace <NUM>) → "GET /users/<NUM>" → Count++, Store Example
```

### Pattern Application Order

1. Timestamps: `2024-01-22` → `<TIMESTAMP>`
2. UUIDs: `550e8400-e29b-41d4-a716-446655440000` → `<UUID>`
3. Numbers: `123`, `456.78` → `<NUM>`
4. IP addresses: `192.168.1.1` → `<IP>`
5. Hex strings: `0x1a2b3c`, `deadbeef` → `<HEX>`

### Data Structure

```go
type SpanNamePattern struct {
    Template   string   `json:"template"`    // Pattern: "GET /users/<NUM>"
    Count      int64    `json:"count"`       // How many spans matched
    Percentage float64  `json:"percentage"`  // % of total spans
    Examples   []string `json:"examples"`    // First 3 unique examples
}
```

### Integration Points

1. **TracesAnalyzer.AnalyzeWithContext()** - Call pattern analyzer for each span.Name
2. **SpanMetadata marshaling** - Patterns included in JSON automatically
3. **API GET /spans/{name}** - Returns patterns in response

## Risks / Trade-offs

### Risk: Pattern explosion from poorly named spans

**Scenario:** Service uses completely random span names (`operation_a1b2c3d4`)

**Mitigation:**
- Patterns still reduce cardinality vs. exact names (hex → `<HEX>`)
- Limit examples to 3 per pattern (memory bounded)
- If >100 patterns per span name, likely indicates bad instrumentation (surfacing the problem is valuable)

### Risk: Pattern regex overhead

**Scenario:** Pattern matching adds latency to span processing

**Mitigation:**
- Pre-compiled regex patterns (done at startup, not per span)
- Simple patterns (anchored, non-backtracking)
- Benchmark target: <1% overhead on existing trace processing
- Pattern matching only on span.Name (small string), not full span data

### Trade-off: Regex patterns vs. learned templates

**Chosen:** Fixed regex patterns (like logs)

**Pros:**
- Predictable behavior
- No training phase needed
- Reuses existing infrastructure

**Cons:**
- Won't catch novel patterns not covered by regex
- May over-mask in some cases (e.g., legitimate numeric constants)

**Decision:** Acceptable trade-off. Most span names follow conventions (HTTP, gRPC, method names). If users need custom patterns, they can add regex to `config/patterns.yaml`.

## Migration Plan

**No migration needed** - this is an additive feature.

**Rollout:**
1. Deploy new version with pattern analysis
2. Existing span metadata unchanged (patterns empty/null)
3. New spans automatically get pattern analysis
4. No breaking changes to API responses (new field added)

**Rollback:**
- If performance issues detected, patterns remain stored but can be ignored by clients
- Feature can be disabled with config flag (future: add `enable_span_patterns: false`)

## Open Questions

1. **Should we apply patterns to event names later?**
   - **Decision:** Not in this change. Focus on span names only. Event patterns can be separate change if needed.

2. **Should patterns be configurable per-service?**
   - **Decision:** No. Use global patterns for consistency. Can revisit if users request service-specific patterns.

3. **Should we track pattern cardinality (unique values per placeholder)?**
   - **Decision:** Not yet. Current scope is pattern identification. Cardinality tracking can be added later if valuable.
