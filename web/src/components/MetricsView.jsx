import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Button } from '@/components/ui/button'

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

  if (loading) return <div className="loading">Loading...</div>
  if (error) return <div className="error">Error: {error}</div>

  const totalSamples = filteredMetrics.reduce((sum, metric) => sum + metric.sample_count, 0)
  const typeBreakdown = filteredMetrics.reduce((acc, metric) => {
    acc[metric.type] = (acc[metric.type] || 0) + 1
    return acc
  }, {})

  return (
    <Card>
      <CardHeader>
        <CardTitle>Metrics Analysis</CardTitle>
        <CardDescription>
          Showing {startIndex + 1}-{Math.min(endIndex, filteredMetrics.length)} of {filteredMetrics.length} metrics
          {totalPages > 1 && ` (Page ${currentPage} of ${totalPages})`}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Filters */}
        <div className="flex flex-wrap gap-4">
          <Input 
            type="text"
            placeholder="Search metrics..."
            value={filter.search}
            onChange={(e) => setFilter({...filter, search: e.target.value})}
            className="w-64"
          />

          <Select 
            value={filter.type} 
            onValueChange={(value) => setFilter({...filter, type: value})}
          >
            <SelectTrigger className="w-48">
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

          <div className="flex items-center gap-2">
            <label className="text-sm font-medium">Min Samples:</label>
            <Input 
              type="number" 
              value={filter.minSamples} 
              onChange={(e) => setFilter({...filter, minSamples: Number(e.target.value)})}
              min="0"
              className="w-24"
            />
          </div>

          <div className="flex items-center gap-2">
            <label className="text-sm font-medium">Min Cardinality:</label>
            <Input 
              type="number" 
              value={filter.minCardinality} 
              onChange={(e) => setFilter({...filter, minCardinality: Number(e.target.value)})}
              min="0"
              className="w-24"
            />
          </div>
        </div>

        {/* Stats Cards */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">Total Metrics</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{filteredMetrics.length}</div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">Total Samples</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{totalSamples.toLocaleString()}</div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">Metric Types</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{Object.keys(typeBreakdown).length}</div>
            </CardContent>
          </Card>
        </div>

        {/* Type Distribution */}
        <div>
          <h3 className="text-lg font-semibold mb-3">Type Distribution</h3>
          <div className="flex flex-wrap gap-2">
            {Object.entries(typeBreakdown)
              .sort((a, b) => b[1] - a[1])
              .map(([type, count]) => (
                <Badge 
                  key={type}
                  variant="secondary"
                  className="px-3 py-1"
                  style={{
                    backgroundColor: getTypeColor(type),
                    color: 'white'
                  }}
                >
                  {type}: {count}
                </Badge>
              ))}
          </div>
        </div>

        {/* Metrics Table */}
        <div>
          <h3 className="text-lg font-semibold mb-3">Metrics Breakdown</h3>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead onClick={() => handleSort('name')} className="cursor-pointer select-none">
                    Metric Name {sortField === 'name' && (sortDirection === 'asc' ? '↑' : '↓')}
                  </TableHead>
                  <TableHead onClick={() => handleSort('type')} className="cursor-pointer select-none">
                    Type {sortField === 'type' && (sortDirection === 'asc' ? '↑' : '↓')}
                  </TableHead>
                  <TableHead onClick={() => handleSort('sample_count')} className="cursor-pointer select-none">
                    Samples {sortField === 'sample_count' && (sortDirection === 'asc' ? '↑' : '↓')}
                  </TableHead>
                  <TableHead onClick={() => handleSort('labels')} className="cursor-pointer select-none">
                    Labels {sortField === 'labels' && (sortDirection === 'asc' ? '↑' : '↓')}
                  </TableHead>
                  <TableHead onClick={() => handleSort('resources')} className="cursor-pointer select-none">
                    Resources {sortField === 'resources' && (sortDirection === 'asc' ? '↑' : '↓')}
                  </TableHead>
                  <TableHead onClick={() => handleSort('cardinality')} className="cursor-pointer select-none">
                    Max Cardinality {sortField === 'cardinality' && (sortDirection === 'asc' ? '↑' : '↓')}
                  </TableHead>
                  <TableHead onClick={() => handleSort('complexity')} className="cursor-pointer select-none">
                    Complexity {sortField === 'complexity' && (sortDirection === 'asc' ? '↑' : '↓')}
                  </TableHead>
                  <TableHead onClick={() => handleSort('services')} className="cursor-pointer select-none">
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
                  
                  // Calculate complexity: total_keys × max_cardinality
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
                        <button 
                          className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 hover:underline"
                          onClick={() => onViewDetails('metrics', metric.name)}
                        >
                          {metric.name}
                        </button>
                      </TableCell>
                      <TableCell>
                        <Badge 
                          variant="secondary"
                          style={{ 
                            backgroundColor: getTypeColor(metric.type),
                            color: 'white'
                          }}
                        >
                          {metric.type}
                        </Badge>
                      </TableCell>
                      <TableCell>{metric.sample_count.toLocaleString()}</TableCell>
                      <TableCell>{labelCount}</TableCell>
                      <TableCell>{resourceCount}</TableCell>
                      <TableCell>
                        {maxCard > 0 ? (
                          <Badge variant={maxCard > 200 ? 'destructive' : maxCard > 50 ? 'default' : 'secondary'}>
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
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex justify-center items-center gap-4">
            <Button 
              onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
              disabled={currentPage === 1}
              variant="outline"
            >
              Previous
            </Button>
            
            <span className="text-sm text-muted-foreground">
              Page {currentPage} of {totalPages}
            </span>
            
            <Button 
              onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
              disabled={currentPage === totalPages}
              variant="outline"
            >
              Next
            </Button>
          </div>
        )}

        {/* Empty State */}
        {filteredMetrics.length === 0 && (
          <div className="text-center py-8 text-muted-foreground">
            No metrics match the current filters
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export default MetricsView
