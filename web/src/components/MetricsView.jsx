import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'

function MetricsView({ onViewDetails }) {
  const [metrics, setMetrics] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [filter, setFilter] = useState({
    type: 'all',
    minSamples: 0,
    minCardinality: 0,
    search: ''
  })
  const [sortField, setSortField] = useState('sample_count')
  const [sortDirection, setSortDirection] = useState('desc')

  const itemsPerPage = 100

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

  const getMaxCardinality = (metric) => {
    if (!metric.label_keys) return 0
    return Math.max(...Object.values(metric.label_keys).map(k => k.estimated_cardinality || 0))
  }

  const filteredMetrics = (metrics || []).filter(metric => {
    if (filter.type !== 'all' && metric.type !== filter.type) return false
    if (metric.sample_count < filter.minSamples) return false
    if (getMaxCardinality(metric) < filter.minCardinality) return false
    if (filter.search && !metric.name.toLowerCase().includes(filter.search.toLowerCase())) return false
    return true
  })

  const totalPages = Math.ceil(filteredMetrics.length / itemsPerPage)
  const startIndex = (currentPage - 1) * itemsPerPage
  const endIndex = startIndex + itemsPerPage
  const currentMetrics = filteredMetrics.slice(startIndex, endIndex)

  // Reset to page 1 when filters change
  useEffect(() => {
    setCurrentPage(1)
  }, [filter])

  const metricTypes = ['all', ...new Set((metrics || []).map(m => m.type).filter(Boolean))]

  const getCardinalityBadge = (card) => {
    if (card > 200) return 'high'
    if (card > 50) return 'medium'
    return 'low'
  }

  const getTypeColor = (type) => {
    const colors = {
      'Sum': '#1976d2',
      'Gauge': '#388e3c',
      'Histogram': '#f57c00',
      'Summary': '#7b1fa2',
      'ExponentialHistogram': '#d32f2f'
    }
    return colors[type] || 'var(--text-secondary)'
  }

  const handleSort = (field) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDirection('desc')
    }
  }

  const getSortedMetrics = (metrics) => {
    return [...metrics].sort((a, b) => {
      let aVal, bVal
      
      switch(sortField) {
        case 'name':
          aVal = a.name
          bVal = b.name
          break
        case 'type':
          aVal = a.type
          bVal = b.type
          break
        case 'sample_count':
          aVal = a.sample_count
          bVal = b.sample_count
          break
        case 'labels':
          aVal = a.label_keys ? Object.keys(a.label_keys).length : 0
          bVal = b.label_keys ? Object.keys(b.label_keys).length : 0
          break
        case 'resources':
          aVal = a.resource_keys ? Object.keys(a.resource_keys).length : 0
          bVal = b.resource_keys ? Object.keys(b.resource_keys).length : 0
          break
        case 'cardinality':
          aVal = getMaxCardinality(a)
          bVal = getMaxCardinality(b)
          break
        case 'complexity':
          // Calculate complexity inline
          const aLabels = a.label_keys ? Object.keys(a.label_keys).length : 0
          const aResources = a.resource_keys ? Object.keys(a.resource_keys).length : 0
          let aBuckets = 0
          if (a.type === 'Histogram' && a.data && a.data.explicit_bounds) {
            aBuckets = a.data.explicit_bounds.length + 1
          } else if (a.type === 'ExponentialHistogram' && a.data && a.data.scales) {
            aBuckets = a.data.scales.length * 10
          }
          aVal = (aLabels + aResources + aBuckets) * getMaxCardinality(a)
          
          const bLabels = b.label_keys ? Object.keys(b.label_keys).length : 0
          const bResources = b.resource_keys ? Object.keys(b.resource_keys).length : 0
          let bBuckets = 0
          if (b.type === 'Histogram' && b.data && b.data.explicit_bounds) {
            bBuckets = b.data.explicit_bounds.length + 1
          } else if (b.type === 'ExponentialHistogram' && b.data && b.data.scales) {
            bBuckets = b.data.scales.length * 10
          }
          bVal = (bLabels + bResources + bBuckets) * getMaxCardinality(b)
          break
        case 'services':
          aVal = a.services ? Object.keys(a.services).length : 0
          bVal = b.services ? Object.keys(b.services).length : 0
          break
        default:
          aVal = a.sample_count
          bVal = b.sample_count
      }
      
      if (typeof aVal === 'string') {
        return sortDirection === 'asc' 
          ? aVal.localeCompare(bVal)
          : bVal.localeCompare(aVal)
      } else {
        return sortDirection === 'asc' ? aVal - bVal : bVal - aVal
      }
    })
  }

  if (loading) return (
    <Card><CardHeader><Skeleton className="h-6 w-40" /></CardHeader><CardContent><div className="flex flex-col gap-3">{[...Array(5)].map((_,i) => <Skeleton key={i} className="h-10" />)}</div></CardContent></Card>
  )
  if (error) return <p className="text-sm text-destructive">Error: {error}</p>

  const totalSamples = filteredMetrics.reduce((sum, metric) => sum + metric.sample_count, 0)
  const typeBreakdown = filteredMetrics.reduce((acc, metric) => {
    acc[metric.type] = (acc[metric.type] || 0) + 1
    return acc
  }, {})

  return (
    <div className="flex flex-col gap-4">
      <h2 className="text-xl font-semibold">Metrics Analysis</h2>

      <div className="flex flex-wrap items-end gap-2">
        <Input
          placeholder="Search metrics..."
          value={filter.search}
          onChange={(e) => setFilter({...filter, search: e.target.value})}
          className="w-48"
        />

        <Select value={filter.type} onValueChange={(v) => setFilter({...filter, type: v})}>
          <SelectTrigger className="w-44">
            <SelectValue placeholder="All Types" />
          </SelectTrigger>
          <SelectContent>
            {metricTypes.map(type => (
              <SelectItem key={type} value={type}>
                {type === 'all' ? 'All Types' : `Type: ${type}`}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <div className="flex items-center gap-1">
          <label className="text-sm text-muted-foreground whitespace-nowrap">Min Samples:</label>
          <Input
            type="number"
            value={filter.minSamples}
            onChange={(e) => setFilter({...filter, minSamples: Number(e.target.value)})}
            min="0"
            className="w-24"
          />
        </div>

        <div className="flex items-center gap-1">
          <label className="text-sm text-muted-foreground whitespace-nowrap">Min Cardinality:</label>
          <Input
            type="number"
            value={filter.minCardinality}
            onChange={(e) => setFilter({...filter, minCardinality: Number(e.target.value)})}
            min="0"
            className="w-24"
          />
        </div>
      </div>

      <p className="text-sm text-muted-foreground">
        Showing {startIndex + 1}–{Math.min(endIndex, filteredMetrics.length)} of {filteredMetrics.length} metrics
        {totalPages > 1 && ` (Page ${currentPage} of ${totalPages})`}
      </p>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        <Card>
          <CardContent className="pt-4">
            <p className="text-sm text-muted-foreground">Total Metrics</p>
            <p className="text-2xl font-bold">{filteredMetrics.length}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-4">
            <p className="text-sm text-muted-foreground">Total Samples</p>
            <p className="text-2xl font-bold">{totalSamples.toLocaleString()}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-4">
            <p className="text-sm text-muted-foreground">Metric Types</p>
            <p className="text-2xl font-bold">{Object.keys(typeBreakdown).length}</p>
          </CardContent>
        </Card>
      </div>

      <div>
        <h3 className="text-base font-medium mb-2">Type Distribution</h3>
        <div className="flex flex-wrap gap-2">
          {Object.entries(typeBreakdown)
            .sort((a, b) => b[1] - a[1])
            .map(([type, count]) => (
              <span
                key={type}
                style={{ background: getTypeColor(type) }}
                className="px-3 py-1 rounded text-white text-sm font-medium"
              >
                {type}: {count}
              </span>
            ))}
        </div>
      </div>

      <div>
        <h3 className="text-base font-medium mb-2">Metrics Breakdown</h3>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="cursor-pointer" onClick={() => handleSort('name')}>
                Metric Name {sortField === 'name' && (sortDirection === 'asc' ? '↑' : '↓')}
              </TableHead>
              <TableHead className="cursor-pointer" onClick={() => handleSort('type')}>
                Type {sortField === 'type' && (sortDirection === 'asc' ? '↑' : '↓')}
              </TableHead>
              <TableHead className="cursor-pointer" onClick={() => handleSort('sample_count')}>
                Samples {sortField === 'sample_count' && (sortDirection === 'asc' ? '↑' : '↓')}
              </TableHead>
              <TableHead className="cursor-pointer" onClick={() => handleSort('labels')}>
                Labels {sortField === 'labels' && (sortDirection === 'asc' ? '↑' : '↓')}
              </TableHead>
              <TableHead className="cursor-pointer" onClick={() => handleSort('resources')}>
                Resources {sortField === 'resources' && (sortDirection === 'asc' ? '↑' : '↓')}
              </TableHead>
              <TableHead className="cursor-pointer" onClick={() => handleSort('cardinality')}>
                Max Cardinality {sortField === 'cardinality' && (sortDirection === 'asc' ? '↑' : '↓')}
              </TableHead>
              <TableHead className="cursor-pointer" onClick={() => handleSort('complexity')}>
                Complexity {sortField === 'complexity' && (sortDirection === 'asc' ? '↑' : '↓')}
              </TableHead>
              <TableHead className="cursor-pointer" onClick={() => handleSort('services')}>
                Services {sortField === 'services' && (sortDirection === 'asc' ? '↑' : '↓')}
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {getSortedMetrics(currentMetrics).map((metric, i) => {
              const maxCard = getMaxCardinality(metric)
              const labelCount = metric.label_keys ? Object.keys(metric.label_keys).length : 0
              const resourceCount = metric.resource_keys ? Object.keys(metric.resource_keys).length : 0
              const serviceCount = metric.services ? Object.keys(metric.services).length : 0

              let bucketCount = 0
              if (metric.type === 'Histogram' && metric.data && metric.data.explicit_bounds) {
                bucketCount = metric.data.explicit_bounds.length + 1
              } else if (metric.type === 'ExponentialHistogram' && metric.data && metric.data.scales) {
                bucketCount = metric.data.scales.length * 10
              }

              const totalKeys = labelCount + resourceCount + bucketCount
              const complexity = totalKeys * maxCard

              return (
                <TableRow key={i}>
                  <TableCell>
                    <Button variant="link" className="h-auto p-0 font-normal" onClick={() => onViewDetails('metrics', metric.name)}>
                      {metric.name}
                    </Button>
                  </TableCell>
                  <TableCell>
                    <Badge style={{ background: getTypeColor(metric.type) }} className="text-white border-0">
                      {metric.type}
                    </Badge>
                  </TableCell>
                  <TableCell>{metric.sample_count.toLocaleString()}</TableCell>
                  <TableCell>{labelCount}</TableCell>
                  <TableCell>{resourceCount}</TableCell>
                  <TableCell>
                    {maxCard > 0 ? (
                      <Badge variant={getCardinalityBadge(maxCard) === 'high' ? 'destructive' : getCardinalityBadge(maxCard) === 'medium' ? 'secondary' : 'outline'}>
                        {maxCard}
                      </Badge>
                    ) : '-'}
                  </TableCell>
                  <TableCell>{complexity > 0 ? complexity.toLocaleString() : '-'}</TableCell>
                  <TableCell>{serviceCount}</TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      </div>

      {totalPages > 1 && (
        <div className="flex justify-center items-center gap-2 py-2">
          <Button variant="outline" size="sm" onClick={() => setCurrentPage(p => Math.max(1, p - 1))} disabled={currentPage === 1}>
            Previous
          </Button>
          <span className="text-sm text-muted-foreground">Page {currentPage} of {totalPages}</span>
          <Button variant="outline" size="sm" onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))} disabled={currentPage === totalPages}>
            Next
          </Button>
        </div>
      )}

      {filteredMetrics.length === 0 && (
        <p className="text-sm text-muted-foreground text-center py-5">No metrics match the current filters</p>
      )}
    </div>
  )
}

export default MetricsView
