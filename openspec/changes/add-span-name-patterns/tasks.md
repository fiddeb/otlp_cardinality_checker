# Implementation Tasks

## 1. Data Model

- [ ] 1.1 Add `SpanNamePattern` struct to `pkg/models/metadata.go` with fields: Template, Count, Percentage, Examples
- [ ] 1.2 Add `NamePatterns []*SpanNamePattern` field to `SpanMetadata`
- [ ] 1.3 Add JSON marshaling tags to ensure proper API serialization

## 2. Template Extraction

- [ ] 2.1 Create `internal/analyzer/spantemplate.go` with `SpanNameAnalyzer` struct
- [ ] 2.2 Implement `ExtractPattern(spanName string) string` using `config.CompiledPattern`
- [ ] 2.3 Implement `AddSpanName(name string)` to track patterns and examples
- [ ] 2.4 Implement `GetPatterns() []*SpanNamePattern` returning sorted patterns by count

## 3. Integration

- [ ] 3.1 Add `SpanNameAnalyzer` to `TracesAnalyzer` struct
- [ ] 3.2 Call `AddSpanName()` during span processing in `AnalyzeWithContext()`
- [ ] 3.3 Populate `NamePatterns` field after analysis completes
- [ ] 3.4 Calculate percentages based on total span count

## 4. Testing

- [ ] 4.1 Unit tests for `SpanNameAnalyzer` pattern extraction (HTTP paths, gRPC methods, generic patterns)
- [ ] 4.2 Test cases: `GET /users/{id}`, `POST /orders/{id}/items`, `grpc.Service/Method`, `my-operation-123`
- [ ] 4.3 Integration test with full `TracesAnalyzer` to verify patterns appear in metadata
- [ ] 4.4 Verify JSON serialization in API responses

## 5. Documentation

- [ ] 5.1 Add example to API.md showing span pattern output
- [ ] 5.2 Update USAGE.md with span pattern analysis examples
- [ ] 5.3 Add inline code comments explaining pattern matching logic

## Dependencies

- Tasks 2.x must complete before 3.x (analyzer needed for integration)
- Tasks 1.x can run in parallel with 2.x
- Task 4.x requires 1.x, 2.x, and 3.x complete

## Validation

Each task completion requires:
- Code compiles without errors
- Existing tests still pass (`go test ./...`)
- New functionality manually verified with test data
