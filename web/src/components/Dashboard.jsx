import { useState, useEffect } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

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

  if (loading) return <div className="loading">Loading...</div>
  if (error) return <div className="error">Error: {error}</div>

  return (
    <>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Metrics
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.metrics || 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Spans
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats?.spans || 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Total Logs
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{(stats?.logs || 0).toLocaleString()}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Services
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{services?.length || 0}</div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Top 10 Services by Sample Volume</CardTitle>
        </CardHeader>
        <CardContent>
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
                  <TableRow key={service} className="cursor-pointer">
                    <TableCell>
                      <span 
                        className="text-primary hover:underline cursor-pointer"
                        onClick={() => onViewService(service)}
                      >
                        {service}
                      </span>
                    </TableCell>
                    <TableCell className="font-medium">{stats.total.toLocaleString()}</TableCell>
                    <TableCell>{stats.metrics.toLocaleString()}</TableCell>
                    <TableCell>{stats.spans.toLocaleString()}</TableCell>
                    <TableCell>{stats.logs.toLocaleString()}</TableCell>
                  </TableRow>
                ))}
            </TableBody>
          </Table>
          {Object.keys(serviceStats).length === 0 && (
            <p className="text-center py-5 text-muted-foreground">
              No service data available
            </p>
          )}
        </CardContent>
      </Card>
    </>
  )
}

export default Dashboard
