import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ArrowLeftIcon } from 'lucide-react'

function ServiceExplorer({ serviceName, onBack, onViewDetails }) {
  const [overview, setOverview] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  // Pagination state
  const [metricsPage, setMetricsPage] = useState(1)
  const [spansPage, setSpansPage] = useState(1)
  const [logsPage, setLogsPage] = useState(1)
  const itemsPerPage = 100

  useEffect(() => {
    fetch(`/api/v1/services/${encodeURIComponent(serviceName)}/overview`)
      .then(r => r.json())
      .then(data => {
        setOverview(data)
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [serviceName])

  // Pagination helpers
  const paginate = (items, page) => {
    const start = (page - 1) * itemsPerPage
    const end = start + itemsPerPage
    return items.slice(start, end)
  }

  const totalPages = (items) => Math.ceil(items.length / itemsPerPage)

  if (loading) {
    return (
      <div className="flex flex-col gap-6">
        <Skeleton className="h-8 w-32" />
        <Skeleton className="h-[400px] w-full" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col gap-6">
        <Button variant="ghost" size="sm" className="w-fit" onClick={onBack}>
          <ArrowLeftIcon className="mr-2 h-4 w-4" />
          Back
        </Button>
        <Card className="border-destructive">
          <CardContent className="pt-6">
            <p className="text-destructive">Error: {error}</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  const paginatedMetrics = paginate(overview?.metrics || [], metricsPage)
  const paginatedSpans = paginate(overview?.spans || [], spansPage)
  const paginatedLogs = paginate(overview?.logs || [], logsPage)

  const PaginationRow = ({ items, page, setPage }) => {
    const total = totalPages(items)
    if (items.length <= itemsPerPage) return null
    return (
      <div className="flex items-center justify-between mt-2">
        <Button variant="outline" size="sm" onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1}>
          Previous
        </Button>
        <span className="text-sm text-muted-foreground">
          Page {page} of {total} ({(page - 1) * itemsPerPage + 1}–{Math.min(page * itemsPerPage, items.length)} of {items.length})
        </span>
        <Button variant="outline" size="sm" onClick={() => setPage(p => Math.min(total, p + 1))} disabled={page === total}>
          Next
        </Button>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <Button variant="ghost" size="sm" className="w-fit" onClick={onBack}>
        <ArrowLeftIcon className="mr-2 h-4 w-4" />
        Back
      </Button>

      <div>
        <h1 className="text-2xl font-bold tracking-tight">{serviceName}</h1>
        <p className="text-muted-foreground">Service telemetry overview</p>
      </div>

      {/* Metrics */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            Metrics
            <Badge variant="secondary">{overview?.metrics?.length || 0}</Badge>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Type</TableHead>
                <TableHead className="text-right">Samples</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {paginatedMetrics.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={3} className="text-center text-muted-foreground">No metrics</TableCell>
                </TableRow>
              ) : (
                paginatedMetrics.map(m => (
                  <TableRow key={m.name} className="cursor-pointer hover:bg-muted/50" onClick={() => onViewDetails('metrics', m.name)}>
                    <TableCell className="font-mono text-sm text-primary">{m.name}</TableCell>
                    <TableCell><Badge variant="outline">{m.type}</Badge></TableCell>
                    <TableCell className="text-right">{m.sample_count?.toLocaleString()}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
          <PaginationRow items={overview?.metrics || []} page={metricsPage} setPage={setMetricsPage} />
        </CardContent>
      </Card>

      {/* Spans */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            Spans
            <Badge variant="secondary">{overview?.spans?.length || 0}</Badge>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Kind</TableHead>
                <TableHead className="text-right">Samples</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {paginatedSpans.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={3} className="text-center text-muted-foreground">No spans</TableCell>
                </TableRow>
              ) : (
                paginatedSpans.map(s => (
                  <TableRow key={s.name} className="cursor-pointer hover:bg-muted/50" onClick={() => onViewDetails('spans', s.name)}>
                    <TableCell className="font-mono text-sm text-primary">{s.name}</TableCell>
                    <TableCell><Badge variant="outline">{s.kind}</Badge></TableCell>
                    <TableCell className="text-right">{s.sample_count?.toLocaleString()}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
          <PaginationRow items={overview?.spans || []} page={spansPage} setPage={setSpansPage} />
        </CardContent>
      </Card>

      {/* Logs */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            Logs
            <Badge variant="secondary">{overview?.logs?.length || 0}</Badge>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Severity</TableHead>
                <TableHead className="text-right">Samples</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {paginatedLogs.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={2} className="text-center text-muted-foreground">No logs</TableCell>
                </TableRow>
              ) : (
                paginatedLogs.map(l => (
                  <TableRow key={l.severity} className="cursor-pointer hover:bg-muted/50" onClick={() => onViewDetails('logs', l.severity)}>
                    <TableCell className="font-medium text-primary">{l.severity}</TableCell>
                    <TableCell className="text-right">{l.sample_count?.toLocaleString()}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
          <PaginationRow items={overview?.logs || []} page={logsPage} setPage={setLogsPage} />
        </CardContent>
      </Card>
    </div>
  )
}

export default ServiceExplorer
