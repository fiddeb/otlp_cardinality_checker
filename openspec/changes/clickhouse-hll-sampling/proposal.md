# Proposal: ClickHouse HLL + Value Sampling

## Summary

Implement HyperLogLog (HLL) with value sampling in ClickHouse backend to prevent cardinality explosion in `attribute_values` table. Currently, the ClickHouse backend stores **every unique value** for each attribute key, causing storage growth proportional to cardinality (e.g., 1349 rows for 12 keys with 1323 unique values). This defeats the purpose of a metadata analyzer - we should track cardinality **without** becoming a full telemetry backend ourselves.

## Why

### Problem

The current ClickHouse implementation stores all observed values:

```sql
-- Current: attribute_values table stores EVERY value
INSERT INTO attribute_values (key, value, signal_type, scope, observation_count)
VALUES ('user.id', 'user-12345', 'log', 'attribute', 1)
```

**Result**: 
- 1000 unique `user.id` values → 1000 rows in ClickHouse
- 301 unique `http.status_code` values → 301 rows
- **Storage grows with cardinality** - exactly what we want to avoid!

### Root Cause

```go
// internal/storage/clickhouse/store.go:1207
func (s *Store) StoreAttributeValue(ctx context.Context, key, value, signalType, scope string) error {
    row := AttributeRow{
        Key:              key,
        Value:            value,  // ← Stores EVERY value
        SignalType:       signalType,
        Scope:            scope,
        ObservationCount: 1,
    }
    return s.buffer.AddAttribute(row)
}
```

### Why This Matters

1. **Storage explosion**: High-cardinality attributes (user IDs, trace IDs, timestamps) create millions of rows
2. **Query performance**: `uniqExact(value)` on millions of rows is expensive
3. **Defeats purpose**: We become a telemetry storage system instead of a metadata analyzer
4. **Inconsistent with SQLite**: SQLite backend uses HLL + sampling successfully

## What Changes

### Architecture: In-Memory HLL + Periodic Flush

Similar to SQLite backend, use:

1. **In-memory `sync.Map`** for attribute metadata cache
2. **HyperLogLog (HLL)** for cardinality estimation per key
3. **Sample 5 values** per key for display
4. **Periodic flush** (e.g., every 60s) to ClickHouse with aggregated data

### Schema Changes

**Option A: Store HLL Binary Sketch**
```sql
ALTER TABLE attribute_values
ADD COLUMN hll_sketch String,  -- Binary HLL state
ADD COLUMN value_samples Array(String),  -- Max 5 samples
ADD COLUMN estimated_cardinality UInt64;
```

**Option B: Use ClickHouse's uniqCombined State** (Recommended)
```sql
CREATE TABLE attribute_catalog (
    key String,
    signal_type LowCardinality(String),
    scope LowCardinality(String),
    
    -- Cardinality tracking
    cardinality_state AggregateFunction(uniqCombined, String),  -- ClickHouse's built-in HLL
    estimated_cardinality UInt64,
    
    -- Value samples
    value_samples Array(String),  -- Max 5 examples
    
    -- Metadata
    observation_count UInt64,
    first_seen DateTime64(3),
    last_seen DateTime64(3)
    
) ENGINE = AggregatingMergeTree()
ORDER BY (key, signal_type, scope)
```

### Code Changes

**1. Add in-memory cache to ClickHouse Store**
```go
type Store struct {
    conn   clickhouse.Conn
    buffer *BufferedWriter
    
    // NEW: In-memory attribute cache
    attrCache  sync.Map  // key -> *models.AttributeMetadata
    attrDirty  sync.Map  // key -> bool
    flushTimer *time.Timer
}
```

**2. Update StoreAttributeValue to use cache**
```go
func (s *Store) StoreAttributeValue(ctx context.Context, key, value, signalType, scope string) error {
    // Get or create attribute in cache
    attrInterface, loaded := s.attrCache.LoadOrStore(key, models.NewAttributeMetadata(key))
    attr := attrInterface.(*models.AttributeMetadata)
    
    // If newly created, try to load from database
    if !loaded {
        if existing := s.loadAttributeFromDB(ctx, key); existing != nil {
            s.attrCache.Store(key, existing)
            attr = existing
        }
    }
    
    // Add value to in-memory HLL (thread-safe)
    attr.AddValue(value, signalType, scope)
    
    // Mark as dirty for next flush
    s.attrDirty.Store(key, true)
    
    return nil
}
```

**3. Add periodic flush mechanism**
```go
func (s *Store) startAttributeFlusher(ctx context.Context, interval time.Duration) {
    ticker := time.NewTicker(interval)
    go func() {
        for {
            select {
            case <-ticker.C:
                s.flushAttributes(ctx)
            case <-ctx.Done():
                ticker.Stop()
                s.flushAttributes(ctx)  // Final flush
                return
            }
        }
    }()
}

func (s *Store) flushAttributes(ctx context.Context) error {
    batch, err := s.conn.PrepareBatch(ctx, `
        INSERT INTO attribute_catalog (
            key, signal_type, scope, cardinality_state, 
            estimated_cardinality, value_samples, observation_count, 
            first_seen, last_seen
        )
    `)
    
    s.attrDirty.Range(func(keyInterface, _ interface{}) bool {
        key := keyInterface.(string)
        if attrInterface, ok := s.attrCache.Load(key); ok {
            attr := attrInterface.(*models.AttributeMetadata)
            
            // Serialize HLL state or use -State combinator
            hllState := attr.HLL.MarshalBinary()
            samples := attr.GetSamples(5)
            
            batch.Append(
                key, attr.SignalType, attr.Scope,
                hllState,
                attr.EstimatedCardinality,
                samples,
                attr.Count,
                attr.FirstSeen, attr.LastSeen,
            )
        }
        s.attrDirty.Delete(keyInterface)
        return true
    })
    
    return batch.Send()
}
```

**4. Update GetMetadataComplexity to use attribute_catalog**
```go
cardQuery := `
    SELECT 
        signal_type,
        key,
        estimated_cardinality
    FROM attribute_catalog
    WHERE signal_type = ?
`
```

### Benefits

1. **Storage efficiency**: 12 keys → 12 rows (not 1349)
2. **Query performance**: No `uniqExact()` on millions of rows
3. **Memory efficiency**: HLL uses ~12KB per key regardless of cardinality
4. **Consistent architecture**: Matches SQLite backend design
5. **Scales to production**: Can handle millions of unique values per key

### Trade-offs

1. **Cardinality is estimated**: HLL has ~2% error rate (acceptable for metadata analysis)
2. **Limited samples**: Only 5 example values per key (sufficient for debugging)
3. **Flush delay**: Cardinality updates visible after flush interval (60s)
4. **Memory usage**: All keys kept in memory (manageable - ~100KB for 1000 keys)

## Scope

### In Scope

- Implement in-memory HLL cache in ClickHouse Store
- Add `attribute_catalog` table with HLL state
- Implement periodic flush mechanism
- Update `GetMetadataComplexity` to use catalog
- Update `GetAttribute` to read from catalog
- Migrate existing `attribute_values` data to catalog
- Update tests for new behavior

### Out of Scope

- Changes to OTLP receiver logic
- Changes to frontend UI
- Changes to SQLite backend (already has HLL)
- Changes to memory backend
- Performance optimization of flush interval (start with 60s)
- Distributed HLL merging (single instance only)

### Success Criteria

✅ `attribute_values` replaced with `attribute_catalog`  
✅ Storage grows linearly with **number of keys**, not cardinality  
✅ Cardinality estimation within 5% of actual (HLL standard)  
✅ 5 value samples available per key  
✅ Flush completes within 100ms for 1000 keys  
✅ No data loss on graceful shutdown  
✅ Existing tests pass with minor adjustments  

## Dependencies

- ClickHouse native Go driver (already used)
- HyperLogLog library: `github.com/axiomhq/hyperloglog` (already in go.mod from SQLite)
- No new external dependencies

## Timeline Estimate

- Schema migration: 2 hours
- In-memory cache implementation: 4 hours
- Flush mechanism: 3 hours
- Query updates: 2 hours
- Testing: 3 hours
- **Total: ~14 hours** (2 work days)

## References

- SQLite implementation: `internal/storage/sqlite/store.go:304` (StoreAttributeValue)
- HLL library docs: https://github.com/axiomhq/hyperloglog
- ClickHouse AggregateFunction: https://clickhouse.com/docs/en/sql-reference/data-types/aggregatefunction
- Original cardinality explosion issue: PR #13 (ClickHouse migration)
