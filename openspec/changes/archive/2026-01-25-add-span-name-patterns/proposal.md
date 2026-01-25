# Change: Add Span Name Pattern Analysis

## Why

Currently, span metadata groups by exact span name, making it difficult to identify instrumentation patterns across services. When developers use dynamic values in span names (e.g., `GET /users/123` vs `GET /users/456`), each unique name creates a separate metadata entry, obscuring the underlying pattern and making it hard to understand span design at a glance.

Log analysis already uses template extraction (via Drain algorithm) to identify patterns like `"User <NUM> logged in"` from `"User 123 logged in"`. We need the same capability for span names to quickly visualize how services structure their traces.

## What Changes

- Add pattern extraction for span names using regex-based template detection (similar to log body templates)
- Store example span names for each discovered pattern
- Track pattern count and percentage distribution
- Focus exclusively on span names (not events, links, or other span properties)
- Reuse existing pattern matching infrastructure (`config.CompiledPattern`)

**Benefits:**
- Quick overview of span naming conventions across services
- Identify over-instrumented spans (high cardinality span names)
- Compare span design patterns between different services
- Detect common anti-patterns (IDs, timestamps in span names)

**Non-Goals (out of scope):**
- Event name pattern analysis (future work)
- Link pattern analysis (future work)
- Attribute value pattern detection (already handled by cardinality tracking)

## Impact

**Affected specs:**
- New capability: `span-analysis` (no existing spec, creating new one)

**Affected code:**
- `pkg/models/metadata.go` - Add `NamePatterns []*SpanNamePattern` to `SpanMetadata`
- `internal/analyzer/traces.go` - Add pattern extraction during span analysis
- `internal/analyzer/spantemplate.go` - New file for span name template logic (reuses log template patterns)
- API responses will include pattern data in span metadata JSON

**Storage impact:**
- Minimal memory increase (~100 bytes per unique pattern)
- Patterns stored alongside existing span metadata

**Breaking changes:**
- None (additive change, backward compatible)
