import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

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

  if (loading) return (
    <div className="flex flex-col gap-4">
      <Card><CardHeader><Skeleton className="h-6 w-32" /></CardHeader><CardContent><div className="grid grid-cols-2 gap-4 md:grid-cols-5">{[...Array(5)].map((_,i) => <Skeleton key={i} className="h-16" />)}</div></CardContent></Card>
    </div>
  )
  if (error) return <p className="text-sm text-destructive">Error: {error}</p>

  const memItems = [
    { label: 'Allocated', desc: 'Current usage', value: `${health.memory.alloc_mb} MB` },
    { label: 'Total Allocated', desc: 'Cumulative', value: `${health.memory.total_alloc_mb} MB` },
    { label: 'System', desc: 'From OS', value: `${health.memory.sys_mb} MB` },
    { label: 'GC Runs', desc: 'Collections', value: health.memory.num_gc },
    { label: 'Uptime', desc: 'Duration', value: health.uptime },
  ]

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader>
          <CardTitle>Memory Usage</CardTitle>
          <CardDescription>Updates every 5 seconds</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-4 md:grid-cols-5">
            {memItems.map(({ label, desc, value }) => (
              <div key={label} className="flex flex-col gap-1">
                <p className="text-sm font-medium">{label}</p>
                <p className="text-xs text-muted-foreground">{desc}</p>
                <p className="text-lg font-semibold">{value}</p>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Memory Metrics Explained</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3 text-sm">
          <p><strong>Allocated:</strong> The most important metric — memory currently in use by the application.</p>
          <p><strong>Total Allocated:</strong> Cumulative total of all allocations since start, including freed memory.</p>
          <p><strong>System:</strong> Total memory obtained from the OS. Go may hold freed memory for future allocations.</p>
          <p><strong>GC Runs:</strong> Number of garbage collection cycles completed. Frequent GC can impact performance.</p>
          <p><strong>Uptime:</strong> How long the server has been running since last restart.</p>
        </CardContent>
      </Card>
    </div>
  )
}

export default MemoryView
