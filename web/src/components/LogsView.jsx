import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'

function LogsView({ onViewServiceDetails }) {
  const [services, setServices] = useState([])
  const [expandedService, setExpandedService] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [sort, setSort] = useState({ field: 'name', dir: 'asc' })
  const [filter, setFilter] = useState({
    minSamples: 0,
  })

  const itemsPerPage = 100

  useEffect(() => {
    fetch('/api/v1/logs/by-service?limit=0')
      .then(r => r.json())
      .then(result => {
        const servicesData = result.data || []
        setServices(servicesData)
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [])

  // Reset to page 1 when filters or sort change
  useEffect(() => {
    setCurrentPage(1)
  }, [filter, sort])

  const toggleSort = (field) => {
    setSort(prev => prev.field === field
      ? { field, dir: prev.dir === 'asc' ? 'desc' : 'asc' }
      : { field, dir: field === 'name' ? 'asc' : 'desc' }
    )
  }

  const sortIndicator = (field) => {
    if (sort.field !== field) return <span className="text-muted-foreground/40 ml-1">↕</span>
    return <span className="ml-1">{sort.dir === 'asc' ? '↑' : '↓'}</span>
  }

  const getSeverityColor = (severity) => {
    const colors = {
      'ERROR': '#d32f2f',
      'Error': '#d32f2f',
      'WARN': '#f57c00',
      'Warning': '#f57c00',
      'INFO': '#1976d2',
      'Information': '#1976d2',
      'DEBUG': '#7b1fa2',
      'DEBUG2': '#7b1fa2',
      'Debug': '#7b1fa2',
      'TRACE': '#455a64',
      'Trace': '#455a64',
      'UNSET': '#999'
    }
    return colors[severity] || '#666'
  }

  if (loading) return (
    <Card><CardHeader><Skeleton className="h-6 w-40" /></CardHeader><CardContent><div className="flex flex-col gap-3">{[...Array(5)].map((_,i) => <Skeleton key={i} className="h-10" />)}</div></CardContent></Card>
  )
  if (error) return <p className="text-sm text-destructive">Error loading logs: {error}</p>
  if (!services || services.length === 0) return <p className="text-sm text-muted-foreground">No logs found</p>

  const totalSamples = services.reduce((sum, svc) => sum + svc.sample_count, 0)

  // Apply filters
  const filteredServices = (services || []).filter(svc => {
    if (svc.sample_count < filter.minSamples) return false
    return true
  })

  // Group services by service_name
  const serviceGroups = {}
  filteredServices.forEach(svc => {
    if (!serviceGroups[svc.service_name]) {
      serviceGroups[svc.service_name] = []
    }
    serviceGroups[svc.service_name].push(svc)
  })

  const uniqueServices = Object.keys(serviceGroups).sort((a, b) => {
    if (sort.field === 'name') {
      return sort.dir === 'asc' ? a.localeCompare(b) : b.localeCompare(a)
    }
    // field === 'count'
    const countA = serviceGroups[a].reduce((s, x) => s + x.sample_count, 0)
    const countB = serviceGroups[b].reduce((s, x) => s + x.sample_count, 0)
    return sort.dir === 'asc' ? countA - countB : countB - countA
  })
  const totalPages = Math.ceil(uniqueServices.length / itemsPerPage)
  const startIndex = (currentPage - 1) * itemsPerPage
  const endIndex = startIndex + itemsPerPage
  const currentServices = uniqueServices.slice(startIndex, endIndex)

  return (
    <div className="flex flex-col gap-6">
      <h2 className="text-xl font-semibold">Log Services</h2>

      <p className="text-sm text-muted-foreground">
        Showing {startIndex + 1}–{Math.min(endIndex, uniqueServices.length)} of {uniqueServices.length} services
        {totalPages > 1 && ` (Page ${currentPage} of ${totalPages})`}
      </p>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        <Card>
          <CardContent className="pt-4">
            <p className="text-sm text-muted-foreground">Total Services</p>
            <p className="text-2xl font-bold">{uniqueServices.length}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-4">
            <p className="text-sm text-muted-foreground">Service×Severity Combos</p>
            <p className="text-2xl font-bold">{services.length}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-4">
            <p className="text-sm text-muted-foreground">Total Log Messages</p>
            <p className="text-2xl font-bold">{totalSamples.toLocaleString()}</p>
          </CardContent>
        </Card>
      </div>

      <div>
        <div className="flex flex-wrap items-center gap-2 mb-2">
          <h3 className="text-base font-medium">Services</h3>
          <div className="flex items-center gap-1 ml-auto">
            <div className="flex items-center gap-1">
              <label className="text-sm text-muted-foreground whitespace-nowrap">Min:</label>
              <Input
                type="number"
                value={filter.minSamples}
                onChange={(e) => setFilter({...filter, minSamples: Number(e.target.value)})}
                min="0"
                className="w-20"
              />
            </div>
            <Button variant={sort.field === 'name' ? 'secondary' : 'outline'} size="sm" onClick={() => toggleSort('name')}>
              Name{sortIndicator('name')}
            </Button>
            <Button variant={sort.field === 'count' ? 'secondary' : 'outline'} size="sm" onClick={() => toggleSort('count')}>
              Logs{sortIndicator('count')}
            </Button>
          </div>
        </div>
        <div className="flex flex-col gap-2">
          {currentServices.map((serviceName) => {
            const severities = serviceGroups[serviceName]
            const totalForService = severities.reduce((sum, s) => sum + s.sample_count, 0)
            const isExpanded = expandedService === serviceName

            return (
              <div key={serviceName} className="overflow-hidden rounded-lg border border-border bg-card text-sm">
                <div
                  role="button"
                  tabIndex={0}
                  onClick={() => setExpandedService(isExpanded ? null : serviceName)}
                  onKeyDown={(e) => e.key === 'Enter' && setExpandedService(isExpanded ? null : serviceName)}
                  className="flex justify-between items-center px-4 py-1.5 cursor-pointer hover:bg-muted/50 font-medium"
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <span className="shrink-0">{isExpanded ? '▼' : '▶'}</span>
                    <span className="truncate">{serviceName}</span>
                    {!isExpanded && (
                      <div className="flex flex-wrap gap-1 ml-1">
                        {severities
                          .sort((a, b) => String(a.severity).localeCompare(String(b.severity)))
                          .map((s) => (
                            <span
                              key={s.severity}
                              className="inline-block rounded px-1.5 py-0 text-[10px] font-semibold leading-5"
                              style={{ color: getSeverityColor(s.severity), border: `1px solid ${getSeverityColor(s.severity)}` }}
                            >
                              {s.severity}
                            </span>
                          ))}
                      </div>
                    )}
                  </div>
                  <span className="text-sm text-muted-foreground shrink-0 ml-2">
                    {totalForService.toLocaleString()} logs
                  </span>
                </div>

                {isExpanded && (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Severity</TableHead>
                        <TableHead>Log Count</TableHead>
                        <TableHead>Action</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {severities
                        .sort((a, b) => String(a.severity).localeCompare(String(b.severity)))
                        .map((svc, i) => (
                          <TableRow key={i}>
                            <TableCell>
                              <span style={{ fontWeight: 'bold', color: getSeverityColor(svc.severity) }}>
                                {svc.severity}
                              </span>
                            </TableCell>
                            <TableCell>{svc.sample_count.toLocaleString()}</TableCell>
                            <TableCell>
                              <Button size="sm" onClick={() => onViewServiceDetails && onViewServiceDetails(serviceName, svc.severity)}>
                                View Patterns
                              </Button>
                            </TableCell>
                          </TableRow>
                        ))}
                    </TableBody>
                  </Table>
                )}
              </div>
            )
          })}
        </div>
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

      {currentServices.length === 0 && uniqueServices.length === 0 && (
        <p className="text-sm text-muted-foreground text-center py-5">No logs match the current filters</p>
      )}
    </div>
  )
}

export default LogsView
