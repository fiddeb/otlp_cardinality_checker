# Automatic Log Template Extraction (Research Summary)

## Task
Evaluate tokenizer-based/auto-template log parsing methods that can run at 20–30k events/sec in our Go pipeline, then recommend an approach and outline a concrete integration and benchmark plan.

## Constraints and goals
- Throughput target: 20–30k events/sec (≈33–50µs/log on a single node, excluding I/O).
- Hot path must be streaming/online and memory-bounded (LRU for templates).
- Keep our core principle: store only templates/keys, never values.
- Prefer Go-native implementation; avoid heavy dependencies; use our existing regex masking (config/patterns.yaml).

## Algorithms and references
- Drain (ICWS’17) — fixed-depth prefix tree, online clustering by token similarity.
  - Repo: https://github.com/logpai/logparser/tree/main/logparser/Drain
  - Production variant: Drain3 (Python) adds snapshotting, masking, LRU, inference mode.
    - Repo: https://github.com/logpai/Drain3
    - Notable features: streaming, masks (IP/NUM/etc.), Kafka/Redis/file persistence, max_clusters (LRU), inference-only.
- Spell (ICDM’16) — streaming LCS-based clustering.
  - Repo: https://github.com/logpai/logparser/tree/main/logparser/Spell
- IPLoM (KDD’09/TKDE’12) — iterative partitioning using log characteristics (token count, position frequency, etc.).
  - Repo: https://github.com/logpai/logparser/tree/main/logparser/IPLoM
- LogMine (CIKM’16) — fast pattern recognition, reported as fast and memory-efficient.
  - Repo: https://github.com/logpai/logparser/tree/main/logparser/LogMine
- LenMa (CNSM’15) — length-of-words signatures; simple and fast.
  - Repo: https://github.com/logpai/logparser/tree/main/logparser/LenMa
- SHISO (SCC’13) — incremental tree-based online mining.
  - Repo: https://github.com/logpai/logparser/tree/main/logparser/SHISO
- LKE (ICDM’09) — rules + weighted edit distance hierarchical clustering.
  - Repo: https://github.com/logpai/logparser/tree/main/logparser/LKE
- Brain (TSC’23) — bidirectional parallel tree; paper claims ~1M lines in ~46s (≈21.7k lines/s) in their environment.
  - Repo: https://github.com/logpai/logparser/tree/main/logparser/Brain

Datasets for evaluation:
- Loghub (ISSRE’23/ISSTA’24) — large public log datasets (HDFS, BGL, OpenSSH, Thunderbird, etc.).
  - Repo: https://github.com/logpai/loghub

Notes
- Logparser repo is research/benchmark oriented; Drain3 documents practical production features (masking, persistence, max_clusters, inference mode).
- Reported benchmark tables in the repo primarily cover parsing accuracy; performance depends heavily on implementation and language.

## Shortlist for high-throughput streaming in Go
- Drain-style token tree (with masking) — excellent streaming characteristics, tunable (max depth, similarity threshold, max children), stable behavior. Conceptually simple to port to Go with careful memory management.
- LenMa/LogMine — simple signatures that can accelerate candidate search or serve as first-stage grouping; can complement Drain.
- Spell — LCS on hot path risks higher per-line cost; keep as reference, not first choice for 30k EPS.

## Recommended approach (hybrid, Go)
1) Pre-masking (we already have this):
   - Use our YAML-configured compiled regex patterns to replace known variable substrings (UUID, IP, email, numbers, URLs, durations, sizes, etc.) with placeholders before tokenization.
   - Keep patterns ordered and cheap; prefer anchored/specific regex to control cost.

2) Tokenization:
   - Split by whitespace + configured extra delimiters (e.g., [":=/[](),"]). Normalize multi-space to single space.

3) Sharded fixed-depth token tree (Drain-like):
   - Shard keys by (tokenCount, firstToken hash) modulo N to reduce lock contention.
   - Each shard maintains a small tree limited by maxDepth (default 4) and maxChildren per node.
   - Leaf holds a small list of clusters with: templateTokens []string, size int64, lastUsed, and an optional cached regex for parameter extraction (disabled by default).
   - Similarity score s = matchedConstantTokens / max(lenA, lenB); match if s ≥ simThreshold (default 0.4–0.6).
   - On match, optionally generalize tokens where they differ into a placeholder (e.g., <*> or <NUM>) respecting masks.

4) Bounded memory:
   - Global maxClusters per shard; evict by LRU when limit is reached (similar to Drain3’s policy).
   - Avoid storing raw messages; store only template tokens and counters.

5) Inference vs training mode:
   - Training: allow new clusters and template generalization.
   - Inference: match-only with exact templates; if no match, return None. Useful under high load or when pre-trained from snapshots.

6) Concurrency and performance:
   - RWMutex per shard; reads mostly under RLock; short WLock on cluster update/create.
   - Use sync.Pool for token slices and builders to reduce GC pressure.
   - Pre-allocate small slices (cap ~16–32) for tokens/clusters.

7) Persistence and ops:
   - Periodic snapshots (JSON) of shards: templates, sizes, and tree shape; load on startup.
   - Configurable via our existing config system: maxDepth, simThreshold, maxChildren, maxClusters, extraDelimiters, shards, training mode flag.

Why this fits 20–30k EPS
- Operations are linear in number of tokens with bounded tree navigation (depth ≤ 4) and tiny candidate sets per leaf.
- Masking is regex-based but on short substrings; keep the list tight and compiled; avoid catastrophic patterns.
- Go implementation with pooling + sharding should fit ≈33–50µs/log on a modern CPU, matching or exceeding Drain3 Python and approaching Brain’s Python numbers in a lower-overhead language.

## Implementation sketch (interfaces)
- Package: internal/analyzer/autotemplate
- Core types
  - type ShardedMiner struct { shards []*MinerShard }
  - type MinerShard struct { tree *node; lru *LRU; mu sync.RWMutex; cfg Config }
  - type node struct { children map[string]*node; wildcard *node; clusters []*cluster; depth int }
  - type cluster struct { tokens []string; size int64; lastUsed uint64 }
  - type Config struct { Shards, MaxDepth, MaxChildren, MaxClusters int; SimThreshold float64; ExtraDelims []rune; Training bool }
- Key methods
  - func (m *ShardedMiner) Add(line string) (template string, matched bool)
  - func (m *ShardedMiner) Match(line string) (template string, ok bool)
  - func (m *ShardedMiner) Snapshot()/Load()

## Benchmark plan
Datasets
- Use Loghub subsets: OpenSSH, HDFS, BGL, Thunderbird (diverse sizes and structures).
- Use our k6 synthetic logs (varied patterns) to stress high-cardinality masking.

Methodology
- Single-node benchmarks; pin GOMAXPROCS to CPU count; test shards ∈ {1,2,4,8}.
- Config sweep: simThreshold {0.4, 0.5, 0.6}, maxDepth {3,4,5}, maxChildren {100,200}.
- Modes: training vs inference-only.

Metrics
- Throughput (events/sec) and per-line latency (mean, p95, p99) via Go benchmark harness.
- Memory (AllocMB, SysMB), GC (#, pause), cluster count, eviction rate under maxClusters.
- Parsing quality proxy: cluster count vs known template count; manual spot checks on small sets.

Success criteria
- ≥20k EPS sustained in training mode on at least one dataset; ≥30k EPS in inference-only mode.
- Memory footprint bounded with stable GC behavior under steady load.

## Integration with existing codebase
- Reuse config/patterns.yaml for pre-masking.
- Introduce an optional autotemplate miner alongside current regex-based logs analyzer; gated by a feature flag.
- Expose minimal stats via /api/v1/health or a new endpoint: clusters, evictions, match rate, training mode.
- Provide a CLI to load/save snapshots.

## Next steps
1) Prototype MinerShard with fixed-depth navigation and similarity scoring (no persistence yet).
2) Add simple benchmark in Go (testing.B) that reads a log file and measures EPS.
3) Wire feature flag to use miner for log body templating in our analyzer path.
4) Add snapshot/restore + config options; iterate on perf (pooling, sharding).

## Sources
- Drain (ICWS’17): https://github.com/logpai/logparser/tree/main/logparser/Drain
- Drain3 (production features): https://github.com/logpai/Drain3
- Spell (ICDM’16): https://github.com/logpai/logparser/tree/main/logparser/Spell
- IPLoM (KDD’09/TKDE’12): https://github.com/logpai/logparser/tree/main/logparser/IPLoM
- LogMine (CIKM’16): https://github.com/logpai/logparser/tree/main/logparser/LogMine
- LenMa (CNSM’15): https://github.com/logpai/logparser/tree/main/logparser/LenMa
- SHISO (SCC’13): https://github.com/logpai/logparser/tree/main/logparser/SHISO
- LKE (ICDM’09): https://github.com/logpai/logparser/tree/main/logparser/LKE
- Brain (TSC’23): https://github.com/logpai/logparser/tree/main/logparser/Brain
- Loghub datasets: https://github.com/logpai/loghub

Notes on licensing: Refer to each repository’s LICENSE; the logparser repo aggregates multiple implementations for research/benchmarking and is not positioned as production-ready. Drain3 documents production-oriented features to emulate.
