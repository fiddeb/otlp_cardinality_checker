import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

function ActiveSeries() {
  const [data, setData] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [showAll, setShowAll] = useState(false)
  const [sortBy, setSortBy] = useState('series-prom')

  useEffect(() => {
    fetch('/api/v1/metrics')
      .then(r => r.json())
      .then(response => {
        setData(response.data || response)
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [])

  if (loading) {
    return (
      <div className="flex flex-col gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Active Series</h1>
          <p className="text-muted-foreground">Cardinality breakdown per metric</p>
        </div>
        <Skeleton className="h-[400px] w-full" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Active Series</h1>
          <p className="text-muted-foreground">Cardinality breakdown per metric</p>
        </div>
        <Card className="border-destructive">
          <CardContent className="pt-6">
            <p className="text-destructive">Error: {error}</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  const getOtlpSeries = (metric) => metric.active_series_otlp ?? metric.active_series ?? 0
  const getPromSeries = (metric) => metric.active_series_prometheus ?? getOtlpSeries(metric)

  const sorted = [...data].sort((a, b) => {
    switch (sortBy) {
      case 'series-otlp': return getOtlpSeries(b) - getOtlpSeries(a)
      case 'series-prom': return getPromSeries(b) - getPromSeries(a)
      case 'name': return a.name.localeCompare(b.name)
      case 'samples': return b.sample_count - a.sample_count
      default: return 0
    }
  })

  const totalOtlpSeries = sorted.reduce((sum, m) => sum + getOtlpSeries(m), 0)
  const totalPromSeries = sorted.reduce((sum, m) => sum + getPromSeries(m), 0)
  const avgOtlpSeries = totalOtlpSeries / (sorted.length || 1)
  const maxPromSeries = sorted.length > 0 ? Math.max(...sorted.map(m => getPromSeries(m))) : 0

  const displayLimit = showAll ? sorted.length : 20
  const displayed = sorted.slice(0, displayLimit)

  const getSeriesVariant = (count) => {
    if (count > 1000) return 'destructive'
    if (count > 100) return 'outline'
    return 'secondary'
  }

  const sortButtons = [
    { id: 'series-prom', label: 'Prometheus Series' },
    { id: 'series-otlp', label: 'OTLP Series' },
    { id: 'name', label: 'Name' },
    { id: 'samples', label: 'Sample Count' },
  ]

  return (
    <div className="flex flex-col gap-4">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Active Series</h1>
        <p className="text-muted-foreground">Cardinality breakdown per metric</p>
      </div>

      <div className="grid grid-cols-2 gap-4 sm:grid-cols-5">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Total Metrics</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{sorted.length.toLocaleString()}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Total Series (OTLP)</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalOtlpSeries.toLocaleString()}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Total Series (Prometheus)</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalPromSeries.toLocaleString()}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Average per Metric</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{Math.round(avgOtlpSeries).toLocaleString()}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Highest (Prometheus)</CardTitle>
          </CardHeader>
          <CardContent>
            <div className={`text-2xl font-bold ${maxPromSeries > 1000 ? 'text-destructive' : ''}`}>
              {maxPromSeries.toLocaleString()}
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="flex items-center gap-2">
        <span className="text-sm font-medium">Sort by:</span>
        {sortButtons.map(({ id, label }) => (
          <Button
            key={id}
            variant={sortBy === id ? 'default' : 'outline'}
            size="sm"
            onClick={() => setSortBy(id)}
          >
            {label}
          </Button>
        ))}
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-16">Rank</TableHead>
                <TableHead>Metric Name</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Active Series (OTLP)</TableHead>
                <TableHead>Active Series (Prometheus)</TableHead>
                <TableHead>Label Keys</TableHead>
                <TableHead>Samples</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {displayed.map((metric, idx) => (
                <TableRow key={metric.name}>
                  <TableCell className="font-medium">#{idx + 1}</TableCell>
                  <TableCell className="font-mono text-xs">{metric.name}</TableCell>
                  <TableCell>
                    <Badge variant="outline">{metric.type || 'Unknown'}</Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={getSeriesVariant(getOtlpSeries(metric))}>
                      {getOtlpSeries(metric).toLocaleString()}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={getSeriesVariant(getPromSeries(metric))}>
                      {getPromSeries(metric).toLocaleString()}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    {Object.keys(metric.label_keys || {}).length} keys
                  </TableCell>
                  <TableCell>{metric.sample_count.toLocaleString()}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {sorted.length > displayLimit && (
            <div className="flex justify-center p-4">
              <Button variant="outline" onClick={() => setShowAll(!showAll)}>
                {showAll ? 'Show Top 20' : `Show All ${sorted.length} Metrics`}
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export default ActiveSeries
