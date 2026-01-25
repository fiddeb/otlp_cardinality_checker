# Implementation Tasks

## 1. Data Model

- [x] 1.1 Add `SpanNamePattern` struct to `pkg/models/metadata.go` with fields: Template, Count, Percentage, Examples
- [x] 1.2 Add `NamePatterns []*SpanNamePattern` field to `SpanMetadata`
- [x] 1.3 Add JSON marshaling tags to ensure proper API serialization

## 2. Template Extraction

- [x] 2.1 Create `internal/analyzer/spantemplate.go` with `SpanNameAnalyzer` struct
- [x] 2.2 Implement `ExtractPattern(spanName string) string` using `config.CompiledPattern`
- [x] 2.3 Implement `AddSpanName(name string)` to track patterns and examples
- [x] 2.4 Implement `GetPatterns() []*SpanNamePattern` returning sorted patterns by count

## 3. Integration

- [x] 3.1 Add `SpanNameAnalyzer` to `TracesAnalyzer` struct
- [x] 3.2 Call `AddSpanName()` during span processing in `AnalyzeWithContext()`
- [x] 3.3 Populate `NamePatterns` field after analysis completes
- [x] 3.4 Calculate percentages based on total span count

## 4. Testing

- [x] 4.1 Unit tests for `SpanNameAnalyzer` pattern extraction (HTTP paths, gRPC methods, generic patterns)
- [x] 4.2 Test cases: `GET /users/{id}`, `POST /orders/{id}/items`, `grpc.Service/Method`, `my-operation-123`
- [x] 4.3 Integration test with full `TracesAnalyzer` to verify patterns appear in metadata
- [x] 4.4 Verify JSON serialization in API responses

## 5. Documentation

- [x] 5.1 Add example to API.md showing span pattern output
- [x] 5.2 Update USAGE.md with span pattern analysis examples
- [x] 5.3 Add inline code comments explaining pattern matching logic

## 6. UI Integration

- [x] 6.1 Add Patterns column to TracesView.jsx showing per-span pattern count
- [x] 6.2 Add `/api/v1/span-patterns` endpoint for global pattern aggregation
- [x] 6.3 Implement `GetSpanPatterns()` in storage interface and memory store
- [x] 6.4 Create `TracePatterns.jsx` component with multi-span pattern highlighting
- [x] 6.5 Add "Trace Patterns" tab to App.jsx navigation

## Dependencies

- Tasks 2.x must complete before 3.x (analyzer needed for integration)
- Tasks 1.x can run in parallel with 2.x
- Task 4.x requires 1.x, 2.x, and 3.x complete

## Validation

Each task completion requires:
- Code compiles without errors
- Existing tests still pass (`go test ./...`)
- New functionality manually verified with test data
