## ADDED Requirements

### Requirement: Attribute Storage Interface
The system SHALL provide storage operations for attribute catalog.

#### Scenario: Store attribute value
- **WHEN** `StoreAttributeValue(ctx, "user_id", "12345", "metric", "attribute")` is called
- **THEN** the attribute metadata is updated in storage
- **AND** HyperLogLog sketch includes the value
- **AND** the operation is thread-safe

#### Scenario: Retrieve attribute
- **WHEN** `GetAttribute(ctx, "user_id")` is called
- **THEN** the complete AttributeMetadata is returned
- **AND** includes estimated cardinality from HyperLogLog

#### Scenario: List attributes with filter
- **WHEN** `ListAttributes(ctx, filter)` is called with signal_type filter
- **THEN** only attributes used in specified signal type are returned

### Requirement: In-Memory Storage
The system SHALL implement in-memory storage for attribute catalog with concurrent access support.

#### Scenario: Concurrent writes
- **WHEN** multiple goroutines call StoreAttributeValue for same key
- **THEN** all operations complete without data races
- **AND** final state reflects all observations

#### Scenario: Memory efficiency
- **WHEN** tracking 100,000 attribute keys
- **THEN** memory usage is approximately 2GB (16KB HLL + metadata per key)

### Requirement: SQLite Persistence
The system SHALL implement SQLite persistence for attribute catalog.

#### Scenario: HLL serialization
- **WHEN** storing AttributeMetadata in SQLite
- **THEN** HyperLogLog sketch is serialized to BLOB
- **AND** deserialized correctly on retrieval

#### Scenario: Migration
- **WHEN** server starts with SQLite backend
- **THEN** migration 007 creates attribute_catalog table
- **AND** creates 5 indexes for query optimization

#### Scenario: JSON array storage
- **WHEN** storing value_samples and signal_types
- **THEN** arrays are serialized as JSON
- **AND** deserialized correctly on retrieval

### Requirement: Filtering Support
The system SHALL support filtering attributes by signal type, scope, and cardinality.

#### Scenario: Filter by signal type
- **WHEN** filter specifies signal_type="metric"
- **THEN** only attributes used in metrics are returned

#### Scenario: Filter by scope
- **WHEN** filter specifies scope="resource"
- **THEN** only resource attributes are returned

#### Scenario: Filter by minimum cardinality
- **WHEN** filter specifies min_cardinality=1000
- **THEN** only attributes with estimated_cardinality >= 1000 are returned

#### Scenario: Combined filters
- **WHEN** filter specifies signal_type="metric" AND min_cardinality=1000
- **THEN** only high-cardinality metric attributes are returned

### Requirement: Sorting Support
The system SHALL support sorting attributes by multiple fields.

#### Scenario: Sort by cardinality descending
- **WHEN** sort_by="cardinality" and sort_direction="desc"
- **THEN** attributes are returned in descending cardinality order

#### Scenario: Sort by count
- **WHEN** sort_by="count"
- **THEN** attributes are sorted by observation count

#### Scenario: Sort by timestamp
- **WHEN** sort_by="first_seen" or "last_seen"
- **THEN** attributes are sorted by specified timestamp

#### Scenario: Sort by key name
- **WHEN** sort_by="key"
- **THEN** attributes are sorted alphabetically by key name

### Requirement: Pagination Support
The system SHALL support pagination for attribute listing.

#### Scenario: Page size
- **WHEN** requesting page_size=50
- **THEN** at most 50 attributes are returned

#### Scenario: Page number
- **WHEN** requesting page=2 with page_size=50
- **THEN** attributes 51-100 are returned

#### Scenario: Total count
- **WHEN** listing attributes with pagination
- **THEN** response includes total count of matching attributes

### Requirement: Dual Storage Mode
The system SHALL support dual storage mode (in-memory + SQLite) for attribute catalog.

#### Scenario: Dual writes
- **WHEN** storage mode is "dual"
- **THEN** StoreAttributeValue writes to both in-memory and SQLite
- **AND** both storages are kept in sync

#### Scenario: Read preference
- **WHEN** storage mode is "dual"
- **THEN** GetAttribute reads from in-memory storage
- **AND** SQLite is used only for persistence
