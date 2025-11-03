# Database Performance Analysis

## Problem Summary

Current logs API performance is unacceptable:
- **List 10 logs**: ~60-68 seconds
- **List 3 logs**: ~18 seconds  
- **Single log (limit=1)**: <1 second

Root cause: **Inefficient database schema and N+1 query pattern**

## Current Schema Issues

### 1. Body Templates Explosion
```sql
CREATE TABLE log_body_templates (
    severity TEXT NOT NULL,
    service_name TEXT NOT NULL,
    template TEXT NOT NULL,
    example TEXT,
    count INTEGER DEFAULT 0,
    percentage REAL DEFAULT 0.0,
    PRIMARY KEY (severity, service_name, template)
);
```

**Problems:**
- Single severity (e.g., "UNSET") can have **thousands** of templates
- Example: 10 logs contained 4,796 total body templates
- Each template stores full text + example = high storage overhead
- Loading templates requires separate query per severity (N+1 pattern)

### 2. N+1 Query Pattern in ListLogs

Current implementation makes **3 queries per severity**:
```
1. SELECT severities (1 query)
2. SELECT services for each severity (N queries)
3. SELECT templates for each severity (N queries)
Total: 1 + (N * 2) queries
```

For 10 logs: **1 + 20 = 21 queries**, each potentially scanning thousands of rows

### 3. Window Function Performance

Attempted optimization using `ROW_NUMBER() OVER (PARTITION BY...)` to limit templates:
```sql
SELECT severity, template, example, count, percentage,
       ROW_NUMBER() OVER (PARTITION BY severity ORDER BY count DESC) as rn
FROM log_body_templates 
WHERE severity IN (?, ?, ...)
WHERE rn <= 100
```

**Result**: Still ~60+ seconds for 10 severities

Likely causes:
- Window function materializes entire result set before filtering
- No index on (severity, count DESC) for efficient top-N selection
- Template table size causing full table scans

## Proposed Solutions

### Option 1: Denormalized Summary Table (Recommended)

Create pre-aggregated summary for list views:

```sql
-- New table for fast list queries
CREATE TABLE log_severity_summary (
    severity TEXT PRIMARY KEY,
    total_sample_count INTEGER DEFAULT 0,
    service_count INTEGER DEFAULT 0,
    template_count INTEGER DEFAULT 0,
    top_services TEXT,      -- JSON array of top 5 services
    top_templates TEXT,     -- JSON array of top 10 templates (ids only)
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) STRICT;

-- Keep detailed tables for drill-down
-- log_services (unchanged)
-- log_body_templates (add template_id for referencing)
```

**Benefits:**
- List queries become single SELECT from summary table
- Sub-second response time
- Detailed data loaded only on demand (GetLog)

**Trade-offs:**
- Need to update summary on writes (slight write overhead)
- Small storage increase for summary data

### Option 2: Template Count Index

Add specialized index for top-N template queries:

```sql
-- Add template_id for efficient referencing
ALTER TABLE log_body_templates ADD COLUMN template_id INTEGER;

-- Index for top-N queries per severity
CREATE INDEX idx_log_templates_top_n 
    ON log_body_templates(severity, count DESC, template_id);

-- Limit templates in list view to just IDs and counts
SELECT template_id, count 
FROM log_body_templates 
WHERE severity = ? 
ORDER BY count DESC 
LIMIT 10;
```

**Benefits:**
- Leverages covering index for fast top-N
- Smaller result set (IDs vs full templates)
- No schema redesign needed

**Trade-offs:**
- Still N queries for N severities
- Requires bulk ID lookup for template details

### Option 3: Hybrid Batch Loading

Optimize batch queries with better SQL:

```sql
-- Load all services in one query (already implemented)
SELECT severity, service_name, sample_count 
FROM log_services 
WHERE severity IN (?, ?, ...)

-- Load top templates using correlated subquery
SELECT t1.severity, t1.template, t1.count, t1.percentage
FROM log_body_templates t1
WHERE t1.severity IN (?, ?, ...)
  AND t1.count >= (
    SELECT COALESCE(MIN(count), 0)
    FROM (
      SELECT count 
      FROM log_body_templates t2 
      WHERE t2.severity = t1.severity 
      ORDER BY count DESC 
      LIMIT 100
    )
  )
ORDER BY t1.severity, t1.count DESC;
```

**Benefits:**
- 2 total queries instead of 1 + 2N
- No schema changes

**Trade-offs:**
- Complex query, may still be slow
- Correlated subquery can be expensive

## Recommended Implementation Plan

### Phase 1: Quick Win - Remove Templates from List View
**Timeline**: Immediate  
**Effort**: 1 hour

```go
// ListLogs returns minimal data
func (s *Store) ListLogs(...) ([]*models.LogMetadata, int, error) {
    // Return: severity, sample_count, service_count, template_count
    // NO services map, NO templates array
}

// GetLog returns full details
func (s *Store) GetLog(severity string) (*models.LogMetadata, error) {
    // Load everything: services, keys, templates
}
```

Update UI to call GetLog only when user clicks to expand details.

**Expected result**: <1s for list, ~5s for single detail view

### Phase 2: Add Summary Table
**Timeline**: 1-2 days  
**Effort**: Medium

1. Create migration for `log_severity_summary` table
2. Add trigger/update logic to maintain summary on writes
3. Update ListLogs to query summary table
4. Add background job to rebuild summaries

**Expected result**: <100ms for list view

### Phase 3: Optimize Template Storage (if needed)
**Timeline**: 2-3 days  
**Effort**: High

- Implement template deduplication (hash-based)
- Store templates in separate table with ID references
- Add compression for template text
- Implement LRU cache for hot templates

## Performance Targets

- **List view** (all severities): <1 second
- **Detail view** (single severity with all templates): <5 seconds  
- **Dashboard initial load**: <2 seconds
- **Memory overhead**: <10% increase

## Metrics to Track

1. Query execution time (per query type)
2. Total database size
3. Template table size growth rate
4. Cache hit rate (if caching added)
5. Write throughput impact

## Next Steps

1. ✅ Create performance branch
2. ⏳ Implement Phase 1 (remove templates from list)
3. ⏳ Benchmark and validate <1s target
4. ⏳ Design Phase 2 schema migration
5. ⏳ Implement and test summary table
6. ⏳ Deploy and monitor production metrics
