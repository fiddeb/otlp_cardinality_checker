import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { fetchJSON } from '@/lib/fetchJSON'

function ServicesView({ onViewService }) {
  const [serviceStats, setServiceStats] = useState({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [search, setSearch] = useState('')
  const [sortField, setSortField] = useState('total')
  const [sortDir, setSortDir] = useState('desc')

  useEffect(() => {
    Promise.all([
      fetchJSON('/api/v1/metrics?limit=0'),
      fetchJSON('/api/v1/spans?limit=0'),
      fetchJSON('/api/v1/logs?limit=0'),
    ])
      .then(([allMetrics, allSpans, allLogs]) => {
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
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [])

  const handleSort = (field) => {
    if (sortField === field) {
      setSortDir(d => d === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDir('desc')
    }
  }

  const sortIndicator = (field) => {
    if (sortField !== field) return null
    return sortDir === 'asc' ? ' ↑' : ' ↓'
  }

  const rows = Object.entries(serviceStats)
    .filter(([name]) => !search || name.toLowerCase().includes(search.toLowerCase()))
    .sort((a, b) => {
      const av = sortField === 'name' ? a[0] : a[1][sortField]
      const bv = sortField === 'name' ? b[0] : b[1][sortField]
      if (typeof av === 'string') {
        return sortDir === 'asc' ? av.localeCompare(bv) : bv.localeCompare(av)
      }
      return sortDir === 'asc' ? av - bv : bv - av
    })

  if (error) {
    return (
      <Card>
        <CardContent className="pt-6">
          <p className="text-destructive">Failed to load services: {error}</p>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader>
          <CardTitle>Services</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <Input
            placeholder="Filter services..."
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="max-w-sm"
          />
          <div className="relative w-full overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead
                    className="cursor-pointer select-none"
                    onClick={() => handleSort('name')}
                  >
                    Service{sortIndicator('name')}
                  </TableHead>
                  <TableHead
                    className="cursor-pointer select-none"
                    onClick={() => handleSort('total')}
                  >
                    Total Samples{sortIndicator('total')}
                  </TableHead>
                  <TableHead
                    className="cursor-pointer select-none"
                    onClick={() => handleSort('metrics')}
                  >
                    Metrics{sortIndicator('metrics')}
                  </TableHead>
                  <TableHead
                    className="cursor-pointer select-none"
                    onClick={() => handleSort('spans')}
                  >
                    Spans{sortIndicator('spans')}
                  </TableHead>
                  <TableHead
                    className="cursor-pointer select-none"
                    onClick={() => handleSort('logs')}
                  >
                    Logs{sortIndicator('logs')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading
                  ? [...Array(10)].map((_, i) => (
                    <TableRow key={i}>
                      <TableCell><Skeleton className="h-4 w-40" /></TableCell>
                      <TableCell><Skeleton className="h-4 w-20" /></TableCell>
                      <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                      <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                      <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                    </TableRow>
                  ))
                  : rows.length === 0
                    ? (
                      <TableRow>
                        <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                          {search ? 'No services match your filter.' : 'No services found.'}
                        </TableCell>
                      </TableRow>
                    )
                    : rows.map(([name, s]) => (
                      <TableRow key={name}>
                        <TableCell className="whitespace-nowrap">
                          <Button
                            variant="link"
                            className="h-auto p-0"
                            onClick={() => onViewService(name)}
                          >
                            {name}
                          </Button>
                        </TableCell>
                        <TableCell className="font-semibold whitespace-nowrap">
                          {s.total.toLocaleString()}
                        </TableCell>
                        <TableCell className="whitespace-nowrap">
                          {s.metrics.toLocaleString()}
                        </TableCell>
                        <TableCell className="whitespace-nowrap">
                          {s.spans.toLocaleString()}
                        </TableCell>
                        <TableCell className="whitespace-nowrap">
                          {s.logs.toLocaleString()}
                        </TableCell>
                      </TableRow>
                    ))
                }
              </TableBody>
            </Table>
          </div>
          {!loading && rows.length > 0 && (
            <p className="text-xs text-muted-foreground">
              {rows.length} service{rows.length !== 1 ? 's' : ''}
              {search && ` matching "${search}"`}
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export default ServicesView
