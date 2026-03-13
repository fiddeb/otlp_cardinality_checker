import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'

function Dashboard({ onViewService }) {
  const [stats, setStats] = useState(null)
  const [services, setServices] = useState([])
  const [serviceStats, setServiceStats] = useState({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    // First: Load just counts for quick initial render
    Promise.all([
      fetch('/api/v1/metrics?limit=1').then(r => r.json()),
      fetch('/api/v1/spans?limit=1').then(r => r.json()),
      fetch('/api/v1/logs?limit=1').then(r => r.json()),
      fetch('/api/v1/services').then(r => r.json()),
    ])
      .then(([metrics, spans, logs, services]) => {
        setStats({
          metrics: metrics.total || 0,
          spans: spans.total || 0,
          logs: logs.total_sample_count || 0, // Total log messages, not severity count
        })
        setServices(services.data || [])
        setLoading(false)
        
        // Second: Load service stats in background (lazy load)
        loadServiceStats()
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [])

  const loadServiceStats = async () => {
    try {
      // Load in smaller batches with pagination to avoid overwhelming the API
      const [allMetrics, allSpans, allLogs] = await Promise.all([
        fetch('/api/v1/metrics?limit=1000').then(r => r.json()),
        fetch('/api/v1/spans?limit=1000').then(r => r.json()),
        fetch('/api/v1/logs?limit=1000').then(r => r.json()),
      ])
      
      // Calculate service statistics
      const stats = {}
      
      allMetrics.data?.forEach(metric => {
        if (metric.services) {
          Object.entries(metric.services).forEach(([service, count]) => {
            if (!stats[service]) stats[service] = { metrics: 0, spans: 0, logs: 0, total: 0 }
            stats[service].metrics += count
            stats[service].total += count
          })
        }
      })
      
      allSpans.data?.forEach(span => {
        if (span.services) {
          Object.entries(span.services).forEach(([service, count]) => {
            if (!stats[service]) stats[service] = { metrics: 0, spans: 0, logs: 0, total: 0 }
            stats[service].spans += count
            stats[service].total += count
          })
        }
      })
        
      allLogs.data?.forEach(log => {
        if (log.services) {
          Object.entries(log.services).forEach(([service, count]) => {
            if (!stats[service]) stats[service] = { metrics: 0, spans: 0, logs: 0, total: 0 }
            stats[service].logs += count
            stats[service].total += count
          })
        }
      })
        
      setServiceStats(stats)
    } catch (err) {
      console.error('Failed to load service stats:', err)
    }
  }

  if (loading) return (
    <div className="flex flex-col gap-4">
      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        {[...Array(4)].map((_, i) => (
          <Card key={i}><CardContent className="pt-6"><Skeleton className="h-8 w-16 mb-2" /><Skeleton className="h-4 w-24" /></CardContent></Card>
        ))}
      </div>
      <Card><CardHeader><Skeleton className="h-6 w-48" /></CardHeader><CardContent><Skeleton className="h-40 w-full" /></CardContent></Card>
    </div>
  )
  if (error) return <p className="text-sm text-destructive">Error: {error}</p>

  return (
    <div className="flex flex-col gap-4">
      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        {[
          { label: 'Metrics', value: stats?.metrics || 0 },
          { label: 'Spans', value: stats?.spans || 0 },
          { label: 'Total Logs', value: (stats?.logs || 0).toLocaleString() },
          { label: 'Services', value: services?.length || 0 },
        ].map(({ label, value }) => (
          <Card key={label}>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">{label}</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-2xl font-bold">{value}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Top 10 Services by Sample Volume</CardTitle>
        </CardHeader>
        <CardContent>
          {Object.keys(serviceStats).length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">No service data available</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Service</TableHead>
                  <TableHead>Total Samples</TableHead>
                  <TableHead>Metrics</TableHead>
                  <TableHead>Spans</TableHead>
                  <TableHead>Logs</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {Object.entries(serviceStats)
                  .sort((a, b) => b[1].total - a[1].total)
                  .slice(0, 10)
                  .map(([service, stats]) => (
                    <TableRow key={service}>
                      <TableCell>
                        <Button variant="link" className="h-auto p-0" onClick={() => onViewService(service)}>
                          {service}
                        </Button>
                      </TableCell>
                      <TableCell className="font-semibold">{stats.total.toLocaleString()}</TableCell>
                      <TableCell>{stats.metrics.toLocaleString()}</TableCell>
                      <TableCell>{stats.spans.toLocaleString()}</TableCell>
                      <TableCell>{stats.logs.toLocaleString()}</TableCell>
                    </TableRow>
                  ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export default Dashboard
