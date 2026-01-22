import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

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
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Memory Usage</CardTitle>
          <CardDescription>Real-time memory statistics (updates every 5 seconds)</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
            <div className="space-y-2" title="Current memory actively in use by the application">
              <div className="text-sm font-medium text-muted-foreground">
                Allocated
                <div className="text-xs text-muted-foreground/70">Current usage</div>
              </div>
              <div className="text-2xl font-bold">{health.memory.alloc_mb} MB</div>
            </div>
            <div className="space-y-2" title="Cumulative total allocated since start (including freed memory)">
              <div className="text-sm font-medium text-muted-foreground">
                Total Allocated
                <div className="text-xs text-muted-foreground/70">Cumulative</div>
              </div>
              <div className="text-2xl font-bold">{health.memory.total_alloc_mb} MB</div>
            </div>
            <div className="space-y-2" title="Total memory obtained from the OS">
              <div className="text-sm font-medium text-muted-foreground">
                System
                <div className="text-xs text-muted-foreground/70">From OS</div>
              </div>
              <div className="text-2xl font-bold">{health.memory.sys_mb} MB</div>
            </div>
            <div className="space-y-2" title="Number of completed garbage collection cycles">
              <div className="text-sm font-medium text-muted-foreground">
                GC Runs
                <div className="text-xs text-muted-foreground/70">Collections</div>
              </div>
              <div className="text-2xl font-bold">{health.memory.num_gc}</div>
            </div>
            <div className="space-y-2" title="How long the server has been running">
              <div className="text-sm font-medium text-muted-foreground">
                Uptime
                <div className="text-xs text-muted-foreground/70">Duration</div>
              </div>
              <div className="text-2xl font-bold">{health.uptime}</div>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Memory Metrics Explained</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm leading-relaxed">
          <p>
            <strong className="font-semibold">Allocated:</strong> The most important metric - this is the memory currently in use by the application. 
            Monitor this to understand actual memory consumption.
          </p>
          <p>
            <strong className="font-semibold">Total Allocated:</strong> Cumulative total of all memory allocations since the application started, 
            including memory that has been freed. This number will always increase and shows allocation activity.
          </p>
          <p>
            <strong className="font-semibold">System:</strong> Total memory obtained from the operating system. Go manages this memory pool 
            and may hold onto freed memory for future allocations rather than returning it to the OS immediately.
          </p>
          <p>
            <strong className="font-semibold">GC Runs:</strong> Number of garbage collection cycles completed. Higher numbers indicate more 
            allocation/deallocation activity. Frequent GC can impact performance.
          </p>
          <p>
            <strong className="font-semibold">Uptime:</strong> How long the server has been running since last restart.
          </p>
        </CardContent>
      </Card>
    </div>
  )
}

export default MemoryView
