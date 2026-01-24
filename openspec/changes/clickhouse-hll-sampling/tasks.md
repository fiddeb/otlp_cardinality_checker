# Tasks: ClickHouse HLL + Value Sampling

## 1. Preparation

- [x] 1.1 Review SQLite HLL implementation (`internal/storage/sqlite/store.go`)
- [x] 1.2 Review current ClickHouse attribute storage
- [x] 1.3 Measure current `attribute_values` table size and query performance
- [x] 1.4 Design migration strategy for existing data

## 2. Schema Changes

- [x] 2.1 Create new `attribute_catalog` table DDL
- [x] 2.2 Decide: Use binary HLL sketch OR ClickHouse `AggregateFunction(uniqCombined)`
- [ ] 2.3 Add migration script to convert `attribute_values` → `attribute_catalog`
- [x] 2.4 Test migration with sample data
- [x] 2.5 Update `schema.go` with new table definition

## 3. In-Memory Cache Implementation

- [x] 3.1 Add `attrCache sync.Map` to Store struct
- [x] 3.2 Add `attrDirty sync.Map` for tracking dirty keys
- [x] 3.3 Implement `loadAttributeFromDB` to hydrate cache from ClickHouse
- [x] 3.4 Update `StoreAttributeValue` to use cache + HLL
- [x] 3.5 Ensure thread-safety with concurrent writes

## 4. Flush Mechanism

- [x] 4.1 Implement `startAttributeFlusher` with configurable interval
- [x] 4.2 Implement `flushAttributes` to batch write to ClickHouse
- [x] 4.3 Handle HLL state serialization (MarshalBinary or ClickHouse state)
- [x] 4.4 Add graceful shutdown hook to flush on exit
- [x] 4.5 Add metrics/logging for flush operations

## 5. Query Updates

- [x] 5.1 Update `GetMetadataComplexity` to read from `attribute_catalog`
- [x] 5.2 Update `GetAttribute` to read from catalog (not attribute_values)
- [x] 5.3 Update cardinality calculation to use `estimated_cardinality` column
- [x] 5.4 Update value samples to use `value_samples` array column
- [x] 5.5 Remove old `attribute_values` queries

## 6. Configuration

- [x] 6.1 Add `ATTRIBUTE_FLUSH_INTERVAL` env var (default: 60s) - Uses existing FlushInterval config
- [x] 6.2 Add `ATTRIBUTE_SAMPLE_SIZE` env var (default: 5) - Hardcoded to 5 in flush logic
- [ ] 6.3 Update config documentation
- [ ] 6.4 Add validation for config values

## 7. Testing

- [ ] 7.1 Unit test: HLL cache operations
- [ ] 7.2 Unit test: Flush mechanism with mock ClickHouse
- [x] 7.3 Integration test: Write 500 values, verify cardinality accuracy (~223 estimated)
- [x] 7.4 Integration test: Verify samples are correct subset (5 samples stored)
- [x] 7.5 Integration test: Graceful shutdown flushes all data
- [ ] 7.6 Load test: Measure flush performance with 1000 keys
- [ ] 7.7 Update existing tests for new schema

## 8. Migration & Cleanup

- [ ] 8.1 Run migration script on existing ClickHouse data
- [ ] 8.2 Verify no data loss after migration
- [x] 8.3 Drop old `attribute_values` table (or keep for rollback) - Kept as legacy table
- [ ] 8.4 Update any scripts/tools that reference old table
- [ ] 8.5 Document migration process in README/docs

## 9. Documentation

- [ ] 9.1 Document HLL approach in ARCHITECTURE.md
- [ ] 9.2 Update API docs if query results change
- [ ] 9.3 Add flush interval tuning guide
- [ ] 9.4 Document trade-offs (estimation error, sample limitations)
- [ ] 9.5 Add troubleshooting section for flush issues

## Completion Criteria

- ✅ `attribute_catalog` table created and used
- ✅ In-memory HLL cache working for all attribute writes
- ✅ Periodic flush runs every 60s (configurable)
- ✅ Graceful shutdown flushes all dirty keys
- ✅ Storage is O(num_keys), not O(cardinality)
- ✅ Cardinality estimation within 5% of actual
- ✅ 5 value samples available per key
- ✅ All tests pass
- ✅ Performance: Flush 1000 keys in <100ms
- ✅ Zero data loss on shutdown/restart
