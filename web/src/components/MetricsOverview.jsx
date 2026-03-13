import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

function MetricsOverview({ onViewMetric }) {
  const [metrics, setMetrics] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [filter, setFilter] = useState({ type: 'all', search: '' })
  const [sortField, setSortField] = useState('sample_count')
  const [sortDirection, setSortDirection] = useState('desc')

  useEffect(() => {
    fetch('/api/v1/metrics?limit=1000')
      .then(r => r.json())
      .then(result => {
        setMetrics(result.data || [])
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [])

  const filteredMetrics = (metrics || []).filter(metric => {
    if (filter.type !== 'all' && metric.type !== filter.type) return false
    if (filter.search && !metric.name.toLowerCase().includes(filter.search.toLowerCase())) return false
    return true
  })

  const metricTypes = ['all', ...new Set((metrics || []).map(m => m.type).filter(Boolean))]

  const getTypeVariant = (type) => {
    const variants = { 'Sum': 'default', 'Gauge': 'secondary', 'Histogram': 'outline', 'Summary': 'secondary', 'ExponentialHistogram': 'destructive' }
    return variants[type] || 'outline'
  }

  const getComplexity = (metric) => {
    const labelCount = metric.label_keys ? Object.keys(metric.label_keys).length : 0
    const resourceCount = metric.resource_keys ? Object.keys(metric.resource_keys).length : 0
    let bucketCount = 0
    if (metric.type === 'Histogram' && metric.data && metric.data.explicit_bounds) {
      bucketCount = metric.data.explicit_bounds.length + 1
    } else if (metric.type === 'ExponentialHistogram' && metric.data && metric.data.scales) {
      bucketCount = metric.data.scales.length * 10
    }
    const totalKeys = labelCount + resourceCount + bucketCount
    let maxCardinality = 0
    if (metric.label_keys) {
      const vals = Object.values(metric.label_keys).map(v => v.estimated_cardinality || 0)
      maxCardinality = Math.max(maxCardinality, ...vals)
    }
    if (metric.resource_keys) {
      const vals = Object.values(metric.resource_keys).map(v => v.estimated_cardinality || 0)
      maxCardinality = Math.max(maxCardinality, ...vals)
    }
    return totalKeys * maxCardinality
  }

  const handleSort = (field) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDirection('desc')
    }
  }

  const getSortedMetrics = (list) => {
    return [...list].sort((a, b) => {
      let aVal, bVal
      switch (sortField) {
        case 'name': aVal = a.name; bVal = b.name; break
        case 'type': aVal = a.type; bVal = b.type; break
        case 'unit': aVal = a.unit || ''; bVal = b.unit || ''; break
        case 'complexity': aVal = getComplexity(a); bVal = getComplexity(b); break
        default: aVal = a.sample_count; bVal = b.sample_count
      }
      if (typeof aVal === 'string') {
        return sortDirection === 'asc' ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal)
      }
      return sortDirection === 'asc' ? aVal - bVal : bVal - aVal
    })
  }

  if (loading) {
    return (
      <div className="flex flex-col gap-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Metrics Overview</h1>
          <p className="text-muted-foreground">All observed metrics and their cardinality</p>
        </div>
        <Skeleton className="h-[400px] w-full" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col gap-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Metrics Overview</h1>
          <p className="text-muted-foreground">All observed metrics and their cardinality</p>
        </div>
        <Card className="border-destructive">
          <CardContent className="pt-6">
            <p className="text-destructive">Error: {error}</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  const totalSamples = filteredMetrics.reduce((sum, m) => sum + m.sample_count, 0)
  const typeBreakdown = filteredMetrics.reduce((acc, m) => {
    acc[m.type] = (acc[m.type] || 0) + 1
    return acc
  }, {})

  const SortIndicator = ({ field }) => sortField === field ? (sortDirection === 'asc' ? ' ▲' : ' ▼') : ''

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Metrics Overview</h1>
        <p className="text-muted-foreground">All observed metrics and their cardinality</p>
      </div>

      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium text-muted-foreground">Total Metrics</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{filteredMetrics.length}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium text-muted-foreground">Total Observations</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalSamples.toLocaleString()}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium text-muted-foreground">Metric Types</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{Object.keys(typeBreakdown).length}</div>
          </CardContent>
        </Card>
      </div>

      {Object.keys(typeBreakdown).length > 0 && (
        <div className="flex flex-wrap gap-2">
          {Object.entries(typeBreakdown)
            .sort((a, b) => b[1] - a[1])
            .map(([type, count]) => (
              <Badge key={type} variant={getTypeVariant(type)} className="text-xs">
                {type}: {count}
              </Badge>
            ))}
        </div>
      )}

      <div className="flex items-center gap-2">
        <Input
          placeholder="Search metrics..."
          value={filter.search}
          onChange={(e) => setFilter({ ...filter, search: e.target.value })}
          className="max-w-xs"
        />
        <Select value={filter.type} onValueChange={(v) => setFilter({ ...filter, type: v })}>
          <SelectTrigger className="w-44">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {metricTypes.map(type => (
              <SelectItem key={type} value={type}>
                {type === 'all' ? 'All Types' : `Type: ${type}`}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort('name')}>
                  Metric Name<SortIndicator field="name" />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort('type')}>
                  Type<SortIndicator field="type" />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort('unit')}>
                  Unit<SortIndicator field="unit" />
                </TableHead>
                <TableHead>Description</TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort('sample_count')}>
                  Observations<SortIndicator field="sample_count" />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort('complexity')}>
                  Complexity<SortIndicator field="complexity" />
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {getSortedMetrics(filteredMetrics).map((metric, i) => (
                <TableRow
                  key={i}
                  className={onViewMetric ? 'cursor-pointer' : ''}
                  onClick={() => onViewMetric && onViewMetric(metric.name)}
                >
                  <TableCell className="font-mono text-xs">{metric.name}</TableCell>
                  <TableCell>
                    <Badge variant={getTypeVariant(metric.type)}>{metric.type}</Badge>
                  </TableCell>
                  <TableCell>{metric.unit || '-'}</TableCell>
                  <TableCell className="max-w-xs truncate text-muted-foreground text-xs">
                    {metric.description || '-'}
                  </TableCell>
                  <TableCell>{metric.sample_count.toLocaleString()}</TableCell>
                  <TableCell>{getComplexity(metric).toLocaleString()}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {filteredMetrics.length === 0 && (
            <p className="py-8 text-center text-sm text-muted-foreground">
              No metrics match the current filters
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export default MetricsOverview
