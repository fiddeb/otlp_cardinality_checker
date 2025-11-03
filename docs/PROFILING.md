# Performance Profiling Guide

This guide shows how to profile the OTLP Cardinality Checker to identify performance bottlenecks.

## pprof Server

The application includes a built-in pprof server that exposes profiling endpoints.

### Configuration

Set the pprof server address (default: `localhost:6060`):

```bash
export PPROF_ADDR=localhost:6060
./bin/otlp-cardinality-checker
```

When the server starts, you'll see:
```
Profiling:
  - pprof: http://localhost:6060/debug/pprof
```

## Available Profiles

### 1. CPU Profile (Most Common)

**Identifies which functions consume the most CPU time.**

```bash
# Collect 30 seconds of CPU profile during load test
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/profile?seconds=30
```

This will:
1. Collect CPU samples for 30 seconds
2. Open an interactive web UI at http://localhost:8081
3. Show flame graphs, call graphs, and top functions

**Best Practice**: Run K6 load test while collecting CPU profile:

```bash
# Terminal 1: Start profiling
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/profile?seconds=60

# Terminal 2: Run load test immediately
k6 run --duration 30s --vus 100 scripts/k6-mixed-load-test.js
```

### 2. Memory Profile (Heap)

**Shows memory allocations and helps find memory leaks.**

```bash
# Current heap allocations
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/heap

# Allocations since program start
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/allocs
```

**What to look for**:
- Large allocations per request
- Growing memory usage over time
- Unexpected object retention

### 3. Goroutine Profile

**Shows all running goroutines and their stack traces.**

```bash
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/goroutine
```

**What to look for**:
- Goroutine leaks (count keeps growing)
- Blocked goroutines
- Deadlocks

### 4. Mutex Contention

**Shows lock contention issues.**

```bash
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/mutex
```

**What to look for**:
- High contention on specific mutexes
- Bottlenecks in concurrent code

### 5. Block Profile

**Shows where goroutines block waiting.**

```bash
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/block
```

**What to look for**:
- Channel operations blocking
- I/O wait times
- Lock waits

## Web UI Features

The `-http=:8081` flag opens an interactive web interface with:

### **View → Flame Graph** (Recommended)
- Visual representation of call stacks
- Width = time/memory consumed
- Easy to spot hot paths

### **View → Top**
- List of functions by resource usage
- Shows flat% (function itself) and cum% (function + callees)

### **View → Graph**
- Call graph showing relationships
- Follow the hot path from main

### **View → Source**
- Click on function to see annotated source code
- Lines highlighted by resource usage

## Command-Line Analysis

If you prefer terminal output:

```bash
# CPU profile with top functions
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
# In pprof prompt:
(pprof) top10
(pprof) list functionName  # Show source code
(pprof) web               # Open graph in browser

# Memory profile
go tool pprof http://localhost:6060/debug/pprof/heap
(pprof) top10
(pprof) list functionName
```

## Profiling Workflow

### 1. Baseline Profile (Before Optimization)

```bash
# Start server
./bin/otlp-cardinality-checker

# Terminal 1: Collect CPU profile
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/profile?seconds=60

# Terminal 2: Run load test
k6 run --duration 30s --vus 100 scripts/k6-mixed-load-test.js

# Save results for comparison
# In browser: View → Sample → Download (save as baseline.pb.gz)
```

### 2. Identify Bottlenecks

In the web UI:
1. Go to **View → Flame Graph**
2. Look for wide bars (high CPU usage)
3. Click on functions to drill down
4. Note the hot paths

Common bottlenecks:
- JSON parsing (`encoding/json`)
- Lock contention (`sync.(*Mutex).Lock`)
- Memory allocations (`runtime.mallocgc`)
- String operations (`strings.*`)
- Map operations

### 3. Optimize Code

Focus on:
- Reducing allocations (use sync.Pool, pre-allocate slices)
- Minimizing lock contention (read-heavy? use sync.RWMutex)
- Avoiding unnecessary work (caching, memoization)
- Using faster alternatives (goccy/go-json vs encoding/json)

### 4. Compare Results

```bash
# After optimization, collect new profile
go tool pprof -http=:8082 http://localhost:6060/debug/pprof/profile?seconds=60

# Compare with baseline
go tool pprof -http=:8083 -base baseline.pb.gz http://localhost:6060/debug/pprof/profile?seconds=60
```

The `-base` flag shows the **difference** between profiles.

## Real-Time Monitoring

### Live Goroutine Count
```bash
curl http://localhost:6060/debug/pprof/goroutine?debug=1 | head -n 1
```

### Live Memory Stats
```bash
curl http://localhost:6060/debug/pprof/heap?debug=1 | head -n 20
```

### All Available Profiles
```bash
curl http://localhost:6060/debug/pprof/
```

## Continuous Profiling

For production monitoring, consider:

1. **Periodic snapshots**: Cron job collecting profiles every hour
2. **Triggered profiling**: Collect profile when latency > threshold
3. **Profile storage**: Save to S3/GCS for historical analysis

Example automated profiling:

```bash
#!/bin/bash
# save-profile.sh

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
curl http://localhost:6060/debug/pprof/profile?seconds=30 > profiles/cpu_${TIMESTAMP}.pb.gz
curl http://localhost:6060/debug/pprof/heap > profiles/heap_${TIMESTAMP}.pb.gz

echo "Profiles saved to profiles/"
```

## Profiling Best Practices

### ✅ DO
- Profile during realistic load (use K6 tests)
- Collect profiles for sufficient duration (30-60 seconds)
- Save baseline profiles before optimization
- Focus on the hot path (top 10% of CPU time)
- Profile in production-like environment

### ❌ DON'T
- Profile idle server (no useful data)
- Make changes based on one-time profiles
- Optimize micro-benchmarks (profile real workload)
- Profile with VERBOSE_LOGGING=true (skews results)
- Ignore allocations (memory pressure affects CPU)

## Example Analysis Session

```bash
# 1. Start server
./bin/otlp-cardinality-checker

# 2. Start profiling in background
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/profile?seconds=60 &

# 3. Generate load
sleep 5  # Let pprof start
k6 run --duration 30s --vus 100 scripts/k6-mixed-load-test.js

# 4. Open http://localhost:8081 in browser
# 5. View → Flame Graph
# 6. Click on wide bars to identify bottlenecks
# 7. View → Source to see hot lines
```

## Interpreting Results

### CPU Profile
- **runtime.mallocgc** high? → Too many allocations
- **sync.(*Mutex).Lock** high? → Lock contention
- **encoding/json** high? → JSON parsing bottleneck
- **Your function** high? → Algorithm needs optimization

### Memory Profile
- **inuse_space**: Current memory usage
- **inuse_objects**: Current object count
- **alloc_space**: Total allocated (including freed)
- **alloc_objects**: Total objects created

Focus on:
- Large **inuse_space** → Memory leaks
- High **alloc_space** vs **inuse_space** → High churn (GC pressure)

### Goroutine Profile
- **Normal**: ~10-50 goroutines (server handlers + background workers)
- **Warning**: 100-1000 goroutines
- **Problem**: 1000+ goroutines (likely leak)

## Further Reading

- [Go pprof documentation](https://pkg.go.dev/net/http/pprof)
- [Profiling Go Programs](https://go.dev/blog/pprof)
- [Go Performance Tuning](https://github.com/dgryski/go-perfbook)
