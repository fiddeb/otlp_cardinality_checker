import { useState, useEffect } from 'react'

function MemoryView() {
  const [health, setHealth] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    const fetchHealth = () => {
      fetch('/api/v1/health')
        .then(r => r.json())
        .then(data => {
          setHealth(data)
          setLoading(false)
        })
        .catch(err => {
          setError(err.message)
          setLoading(false)
        })
    }

    fetchHealth()
    const interval = setInterval(fetchHealth, 5000) // Update every 5 seconds

    return () => clearInterval(interval)
  }, [])

  if (loading) return <div className="loading">Loading...</div>
  if (error) return <div className="error">Error: {error}</div>

  return (
    <>
      <div className="card memory-stats">
        <h2>Memory Usage</h2>
        <div className="memory-grid">
          <div className="memory-item" title="Current memory actively in use by the application">
            <div className="memory-label">
              Allocated
              <span className="memory-info">Current usage</span>
            </div>
            <div className="memory-value">{health.memory.alloc_mb} MB</div>
          </div>
          <div className="memory-item" title="Cumulative total allocated since start (including freed memory)">
            <div className="memory-label">
              Total Allocated
              <span className="memory-info">Cumulative</span>
            </div>
            <div className="memory-value">{health.memory.total_alloc_mb} MB</div>
          </div>
          <div className="memory-item" title="Total memory obtained from the OS">
            <div className="memory-label">
              System
              <span className="memory-info">From OS</span>
            </div>
            <div className="memory-value">{health.memory.sys_mb} MB</div>
          </div>
          <div className="memory-item" title="Number of completed garbage collection cycles">
            <div className="memory-label">
              GC Runs
              <span className="memory-info">Collections</span>
            </div>
            <div className="memory-value">{health.memory.num_gc}</div>
          </div>
          <div className="memory-item" title="How long the server has been running">
            <div className="memory-label">
              Uptime
              <span className="memory-info">Duration</span>
            </div>
            <div className="memory-value">{health.uptime}</div>
          </div>
        </div>
      </div>

      <div className="card">
        <h2>Memory Metrics Explained</h2>
        <div style={{ lineHeight: '1.6' }}>
          <p style={{ marginBottom: '12px' }}>
            <strong>Allocated:</strong> The most important metric - this is the memory currently in use by the application. 
            Monitor this to understand actual memory consumption.
          </p>
          <p style={{ marginBottom: '12px' }}>
            <strong>Total Allocated:</strong> Cumulative total of all memory allocations since the application started, 
            including memory that has been freed. This number will always increase and shows allocation activity.
          </p>
          <p style={{ marginBottom: '12px' }}>
            <strong>System:</strong> Total memory obtained from the operating system. Go manages this memory pool 
            and may hold onto freed memory for future allocations rather than returning it to the OS immediately.
          </p>
          <p style={{ marginBottom: '12px' }}>
            <strong>GC Runs:</strong> Number of garbage collection cycles completed. Higher numbers indicate more 
            allocation/deallocation activity. Frequent GC can impact performance.
          </p>
          <p>
            <strong>Uptime:</strong> How long the server has been running since last restart.
          </p>
        </div>
      </div>
    </>
  )
}

export default MemoryView
