## 1. Data Model and Storage Interface

- [x] 1.1 Create `pkg/models/attribute.go` with AttributeMetadata struct
- [x] 1.2 Implement HyperLogLog integration for cardinality estimation
- [x] 1.3 Add thread-safe AddValue() method with mutex protection
- [x] 1.4 Add MarshalHLL() and UnmarshalHLL() for SQLite serialization
- [x] 1.5 Create AttributeFilter struct for query parameters
- [x] 1.6 Extend Storage interface with 3 attribute catalog methods

## 2. In-Memory Storage Implementation

- [x] 2.1 Add attributeCatalog map to memory store
- [x] 2.2 Implement StoreAttributeValue with concurrent access support
- [x] 2.3 Implement GetAttribute with RLock for thread-safe reads
- [x] 2.4 Implement ListAttributes with filtering by signal type, scope, cardinality
- [x] 2.5 Add sorting support (cardinality, count, first_seen, last_seen, key)
- [x] 2.6 Add pagination support (page, page_size)

## 3. SQLite Storage Implementation

- [x] 3.1 Create migration 007_attribute_catalog.up.sql
- [x] 3.2 Define attribute_catalog table schema with BLOB for HLL
- [x] 3.3 Create 5 indexes (signal_types, scope, cardinality, count, last_seen)
- [x] 3.4 Implement StoreAttributeValue with UPSERT and HLL serialization
- [x] 3.5 Implement GetAttribute with HLL deserialization
- [x] 3.6 Implement ListAttributes with dynamic SQL query building
- [x] 3.7 Add filter support using WHERE clauses
- [x] 3.8 Add sorting and pagination to SQL queries
- [x] 3.9 Integrate migration 007 into migrations array

## 4. Analyzer Integration

- [x] 4.1 Create `internal/analyzer/common.go` with extractAttributesToCatalog()
- [x] 4.2 Update MetricsAnalyzer to extract label and resource attributes
- [x] 4.3 Update SpanAnalyzer to extract span and resource attributes
- [x] 4.4 Update LogAnalyzer to extract log and resource attributes
- [x] 4.5 Pass storage interface to all analyzers
- [x] 4.6 Call extractAttributesToCatalog() in each analyzer

## 5. Receiver Integration

- [x] 5.1 Update GRPC receiver to pass storage to analyzers
- [x] 5.2 Update HTTP receiver to pass storage to analyzers
- [x] 5.3 Ensure AttributeCatalog interface is available in receivers

## 6. API Endpoints

- [x] 6.1 Add GET /api/v1/attributes endpoint handler
- [x] 6.2 Parse query parameters (signal_type, scope, min_cardinality, sort_by, sort_direction, page, page_size)
- [x] 6.3 Call storage.ListAttributes with filter
- [x] 6.4 Return JSON response with attributes array and pagination metadata
- [x] 6.5 Add GET /api/v1/attributes/:key endpoint handler
- [x] 6.6 Return 404 for non-existent attributes
- [x] 6.7 Register routes in API server setup

## 7. UI Component

- [x] 7.1 Create `web/src/components/AttributesView.jsx`
- [x] 7.2 Add useState hooks for attributes, loading, error, filters, pagination
- [x] 7.3 Add useEffect to fetch data from API with query params
- [x] 7.4 Implement filter controls (signal type, scope, min cardinality, search)
- [x] 7.5 Implement statistics bar (total, high cardinality, resource attributes)
- [x] 7.6 Implement attributes table with sortable columns
- [x] 7.7 Add getCardinalityBadge() with color coding (low/medium/high)
- [x] 7.8 Add getScopeColor() for scope badge styling
- [x] 7.9 Implement pagination controls
- [x] 7.10 Add error and loading states

## 8. Application Integration

- [x] 8.1 Import AttributesView in `web/src/App.jsx`
- [x] 8.2 Add "Attributes" tab button between Logs and Noisy Neighbors
- [x] 8.3 Add conditional rendering: `{activeTab === 'attributes' && <AttributesView />}`

## 9. Testing and Validation

- [x] 9.1 Build Go binary: `go build -o bin/occ ./cmd/server`
- [x] 9.2 Test with SQLite backend (STORAGE_TYPE=sqlite)
- [x] 9.3 Send test metrics with various attributes
- [x] 9.4 Verify GET /api/v1/attributes returns correct data
- [x] 9.5 Verify cardinality estimation works (HLL)
- [x] 9.6 Build frontend: `npm run build` in web/
- [x] 9.7 Verify UI displays attributes correctly
- [x] 9.8 Test filtering, sorting, and pagination in UI

## 10. Documentation

- [x] 10.1 Update README.md with attribute catalog feature description
- [x] 10.2 Add API documentation to docs/API.md
- [x] 10.3 Add example API calls with curl
- [x] 10.4 Document query parameters and filters
- [x] 10.5 Add screenshots of UI to documentation

## Implementation Notes

### Commits
This change was implemented in 5 commits:
1. `9702b0a` - Data model and storage layer (tasks 1-3)
2. `5e9e083` - Analyzer and receiver integration (tasks 4-5)
3. `dae3f65` - API endpoints (task 6)
4. `c4d31c4` - SQLite persistence (task 3)
5. `3f711c0` - UI component (tasks 7-8)

### Testing
All functionality tested with:
- In-memory storage backend
- SQLite storage backend with migration
- Load testing with 50k observations
- Frontend build and browser verification

### Known Issues
- SQLite writes are synchronous per attribute value (performance issue under high load)
- Future optimization: Add in-memory cache with periodic batch flush to SQLite
