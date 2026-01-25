# Capability: Span Analysis

## ADDED Requirements

### Requirement: Span Name Pattern Extraction

The system SHALL extract templates from span names by replacing dynamic values (numbers, UUIDs, timestamps, IPs, hex strings) with standardized placeholders to identify instrumentation patterns.

#### Scenario: HTTP path with numeric ID

- **GIVEN** a span with name `GET /users/123`
- **WHEN** the span is analyzed
- **THEN** the extracted pattern is `GET /users/<NUM>`
- **AND** the pattern count increments
- **AND** the original span name is stored as an example

#### Scenario: gRPC method call

- **GIVEN** a span with name `v1.UserService/GetUser`
- **WHEN** the span is analyzed
- **THEN** the pattern is stored as-is (no dynamic values detected)
- **AND** the span name is recorded as an example

#### Scenario: Multiple placeholders in one name

- **GIVEN** a span with name `POST /orders/550e8400-e29b-41d4-a716-446655440000/items/42`
- **WHEN** the span is analyzed
- **THEN** the pattern is `POST /orders/<UUID>/items/<NUM>`
- **AND** both placeholders are applied correctly

#### Scenario: Timestamp in span name

- **GIVEN** a span with name `operation-2024-01-22T10:30:00`
- **WHEN** the span is analyzed
- **THEN** the pattern is `operation-<TIMESTAMP>`

### Requirement: Pattern Metadata Storage

The system SHALL store pattern metadata including template string, occurrence count, percentage of total spans, and example span names for each discovered pattern.

#### Scenario: Pattern count tracking

- **GIVEN** 100 spans have been analyzed for a service
- **AND** 70 spans matched pattern `GET /api/v1/<NUM>`
- **AND** 30 spans matched pattern `POST /api/v1/users`
- **WHEN** pattern metadata is retrieved
- **THEN** the first pattern has count=70 and percentage=70.0
- **AND** the second pattern has count=30 and percentage=30.0

#### Scenario: Example span names

- **GIVEN** a pattern `GET /users/<NUM>` has matched 10 different spans
- **AND** the first 3 unique span names were: `GET /users/123`, `GET /users/456`, `GET /users/789`
- **WHEN** pattern metadata is retrieved
- **THEN** the examples field contains those 3 span names
- **AND** no more than 3 examples are stored (memory bounded)

#### Scenario: Pattern included in span metadata

- **GIVEN** a span name has associated patterns
- **WHEN** the span metadata is serialized to JSON for API response
- **THEN** the `name_patterns` field contains the pattern list
- **AND** each pattern includes template, count, percentage, and examples

### Requirement: Pattern Placeholder Types

The system SHALL support the following placeholder types for span name pattern extraction, applied in the specified order:

1. `<TIMESTAMP>` - ISO 8601 timestamps, Unix timestamps, common date formats
2. `<UUID>` - Standard UUID format (8-4-4-4-12 hex digits)
3. `<NUM>` - Integer and floating-point numbers
4. `<IP>` - IPv4 and IPv6 addresses
5. `<HEX>` - Hexadecimal strings (with or without `0x` prefix)

#### Scenario: Placeholder priority order

- **GIVEN** a span name `request-2024-01-22-192.168.1.1`
- **WHEN** patterns are applied in order (timestamp before IP)
- **THEN** the result is `request-<TIMESTAMP>-<IP>`
- **AND** the timestamp pattern matched the date portion
- **AND** the IP pattern matched the address portion

#### Scenario: No pattern match

- **GIVEN** a span name `myOperation`
- **WHEN** pattern extraction is performed
- **THEN** the pattern equals the original span name
- **AND** no placeholders are inserted
- **AND** the pattern is still tracked with examples

### Requirement: Pattern Analysis Integration

The system SHALL integrate pattern extraction into the existing span analysis workflow without breaking existing metadata structure or API compatibility.

#### Scenario: Existing span metadata preserved

- **GIVEN** a span is analyzed with pattern extraction enabled
- **WHEN** the span metadata is generated
- **THEN** all existing fields (Name, Kind, AttributeKeys, etc.) remain unchanged
- **AND** the new `NamePatterns` field is added
- **AND** existing API clients can ignore the new field

#### Scenario: Backward compatible JSON

- **GIVEN** a client expects the old span metadata JSON format
- **WHEN** the API returns span metadata with patterns
- **THEN** the response includes all original fields
- **AND** the `name_patterns` field is present but can be ignored
- **AND** no existing fields are removed or renamed

#### Scenario: Pattern extraction per span name

- **GIVEN** multiple spans with different names are analyzed
- **WHEN** metadata is aggregated by span name
- **THEN** each unique span name has its own pattern analysis
- **AND** patterns are calculated per span name group (not globally)
