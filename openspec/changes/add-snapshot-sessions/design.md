# Design: Snapshot Sessions with Diff Mode

## Context

OCC is an in-memory diagnostic tool. Users collect telemetry, analyze it, then the data is gone on restart. This design addresses the need to persist, merge, and compare analysis results.

### Stakeholders
- Platform engineers doing pre/post deploy validation
- SREs investigating telemetry changes
- Developers analyzing service instrumentation

### Constraints
- Must not change ephemeral-first philosophy (sessions are opt-in)
- Must handle HyperLogLog state serialization
- File-based storage (no external database dependency)
- Sessions must be portable (JSON export/import)

## Goals / Non-Goals

### Goals
- Save current state as named session
- Load session to restore previous state
- Merge multiple sessions (combine signals collected separately)
- Compare two sessions and detect changes
- Filter by service when saving/loading
- UI for session management and diff visualization

### Non-Goals
- Real-time sync between OCC instances
- Session versioning/history within a session
- Automatic session creation (always explicit)
- Cloud storage backends (file-only for MVP)

## Decisions

### Decision 1: File-Based Storage with JSON Format

**Choice**: Store sessions as gzip-compressed JSON files in `data/sessions/`

**Rationale**:
- No external dependencies
- Human-readable (when uncompressed)
- Easy backup/transfer
- Matches existing config pattern

**Alternatives considered**:
- SQLite: Adds complexity, overkill for occasional saves
- BoltDB: Another dependency, not human-readable
- Memory-only: Doesn't solve the problem

### Decision 2: HyperLogLog Serialization

**Choice**: Serialize HLL registers as base64-encoded bytes

**Rationale**:
- HLL state is just a byte array (16KB per sketch)
- Base64 is JSON-safe
- Preserves exact cardinality estimation

**Implementation**:
```go
type SerializedHLL struct {
    Precision uint8  `json:"precision"`
    Registers string `json:"registers"` // base64
}
```

### Decision 3: Merge Semantics

**Choice**: Additive merge with HLL union for cardinality

**Rationale**:
- HLL supports union operation (no double-counting)
- Sample counts are additive
- Value samples use first-seen priority (cap at 10)

**Merge rules**:
| Field | Strategy |
|-------|----------|
| SampleCount | Sum |
| HLL | Union (mathematical merge) |
| ValueSamples | Union, keep first 10 |
| FirstSeen | Min(a.FirstSeen, b.FirstSeen) |
| LastSeen | Max(a.LastSeen, b.LastSeen) |
| Services | Merge maps, sum counts |

### Decision 4: Diff Algorithm

**Choice**: Three-way classification with severity scoring

**Categories**:
- `added`: Present in "to" but not "from"
- `removed`: Present in "from" but not "to"
- `changed`: Present in both, with differences

**Severity scoring**:
| Condition | Severity |
|-----------|----------|
| Cardinality increase > 10x | critical |
| Cardinality increase > 2x | warning |
| Sample rate increase > 5x | warning |
| New high-cardinality attribute (>1000) | warning |
| Any other change | info |

### Decision 5: Session Storage Limits

**Choice**: Configurable limits with sensible defaults

```bash
OCC_SESSION_DIR=/var/lib/occ/sessions  # default: ./data/sessions
OCC_MAX_SESSION_SIZE=100MB             # per session
OCC_MAX_SESSIONS=50                    # total sessions
```

**Cleanup policy**: Manual deletion only (no auto-eviction)

## Risks / Trade-offs

### Risk: Large Sessions
**Issue**: Heavy telemetry could create multi-GB sessions
**Mitigation**: 
- Gzip compression (typically 5-10x reduction)
- Size limit per session (default 100MB)
- Warning in UI when approaching limit

### Risk: HLL Precision Loss on Merge
**Issue**: Merging HLLs from different collection periods may have minor precision differences
**Mitigation**: 
- Use same HLL precision (14) everywhere
- Document that merged cardinality is estimated
- Error margin is still ~0.81%

### Risk: Service Filter Complexity
**Issue**: Filtering by service requires understanding service attribution across all signals
**Mitigation**: 
- Use existing `Services` map on each metadata type
- Filter at save time, not load time
- Clear documentation on behavior

## Migration Plan

No migration needed - new capability, backward compatible.

## Open Questions

1. **Should sessions auto-expire?** 
   - Current answer: No, manual cleanup only
   - Revisit if users complain about disk usage

2. **Should we support partial loads?**
   - e.g., Load only metrics from a session
   - Current answer: Yes, via `signals` filter parameter

3. **WebSocket for diff progress?**
   - Large diffs might take time
   - Current answer: No, keep it simple (HTTP response)
