import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ArrowLeftIcon } from 'lucide-react'

function AttributeExplorer({ attributeKey, onBack, onViewService }) {

  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  // Pagination state
  const [metricsPage, setMetricsPage] = useState(1)
  const [spansPage, setSpansPage] = useState(1)
  const [logsPage, setLogsPage] = useState(1)
  const itemsPerPage = 100

  useEffect(() => {
    setLoading(true)
    setError(null)
    fetch(`/api/v1/attributes/${encodeURIComponent(attributeKey)}/telemetry`)
      .then(r => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        return r.json()
      })
      .then(d => {
        setData(d)
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [attributeKey])

  // Pagination helpers
  const paginate = (items, page) => {
    const start = (page - 1) * itemsPerPage
    const end = start + itemsPerPage
    return (items || []).slice(start, end)
  }

  const totalPages = (items) => Math.ceil((items || []).length / itemsPerPage)

  const getCardinalityBadge = (card) => {
    if (card > 1000) return 'destructive'
    if (card > 100) return 'secondary'
    return 'outline'
  }

  const getScopeColor = (scope) => {
    const colors = { 'resource': '#1976d2', 'attribute': '#388e3c', 'both': '#f57c00' }
    return colors[scope] || 'var(--text-secondary)'
  }

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

  const attr = data?.attribute
  const metrics = data?.metrics || []
  const spans = data?.spans || []
  const logs = data?.logs || []

  const paginatedMetrics = paginate(metrics, metricsPage)
  const paginatedSpans = paginate(spans, spansPage)
  const paginatedLogs = paginate(logs, logsPage)

  // Collect unique services across all signals
  const allServices = new Set()
  metrics.forEach(m => Object.keys(m.services || {}).forEach(s => allServices.add(s)))
  spans.forEach(s => Object.keys(s.services || {}).forEach(svc => allServices.add(svc)))
  logs.forEach(l => Object.keys(l.services || {}).forEach(s => allServices.add(s)))

  const PaginationRow = ({ items, page, setPage }) => {
    const total = totalPages(items)
    if ((items || []).length <= itemsPerPage) return null
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
        <h1 className="text-2xl font-bold tracking-tight font-mono">{attributeKey}</h1>
        <p className="text-muted-foreground">Attribute telemetry overview</p>
      </div>

      {/* Attribute summary cards */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-5">
        <Card>
          <CardContent className="pt-4">
            <p className="text-sm text-muted-foreground">Cardinality (est.)</p>
            <p className="text-2xl font-bold">
              <Badge variant={getCardinalityBadge(attr?.estimated_cardinality)} className="text-base px-2 py-0.5">
                {(attr?.estimated_cardinality || 0).toLocaleString()}
              </Badge>
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-4">
            <p className="text-sm text-muted-foreground">Observations</p>
            <p className="text-2xl font-bold">{(attr?.count || 0).toLocaleString()}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-4">
            <p className="text-sm text-muted-foreground">Scope</p>
            <Badge style={{ background: getScopeColor(attr?.scope) }} className="text-white border-0 mt-1">
              {attr?.scope || 'unknown'}
            </Badge>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-4">
            <p className="text-sm text-muted-foreground">Signal Types</p>
            <div className="flex flex-wrap gap-1 mt-1">
              {(attr?.signal_types || []).map(t => (
                <Badge key={t} variant="secondary" className="text-xs">{t}</Badge>
              ))}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-4">
            <p className="text-sm text-muted-foreground">Services</p>
            <p className="text-2xl font-bold">{allServices.size}</p>
          </CardContent>
        </Card>
      </div>

      {/* Sample values */}
      {attr?.value_samples?.length > 0 && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-base">Sample Values</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-1">
              {attr.value_samples.map((v, i) => (
                <code key={i} className="rounded bg-muted px-1.5 py-0.5 text-xs">{v}</code>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Services list */}
      {allServices.size > 0 && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-base flex items-center gap-2">
              Services
              <Badge variant="secondary">{allServices.size}</Badge>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-1.5">
              {[...allServices].sort().map(svc => (
                <button
                  key={svc}
                  onClick={() => onViewService?.(svc)}
                  className="inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-mono cursor-pointer hover:bg-muted/50 transition-colors"
                >
                  {svc}
                </button>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Metrics */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            Metrics
            <Badge variant="secondary">{metrics.length}</Badge>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Services</TableHead>
                <TableHead className="text-right">Observations</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {paginatedMetrics.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={4} className="text-center text-muted-foreground">No metrics use this attribute</TableCell>
                </TableRow>
              ) : (
                paginatedMetrics.map(m => (
                  <TableRow key={m.name}>
                    <TableCell className="font-mono text-sm">{m.name}</TableCell>
                    <TableCell><Badge variant="outline">{m.type}</Badge></TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {Object.keys(m.services || {}).sort().map(svc => (
                          <Badge key={svc} variant="outline" className="text-xs font-mono font-normal">{svc}</Badge>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell className="text-right">{m.attribute_count?.toLocaleString()}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
          <PaginationRow items={metrics} page={metricsPage} setPage={setMetricsPage} />
        </CardContent>
      </Card>

      {/* Spans */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            Spans
            <Badge variant="secondary">{spans.length}</Badge>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Kind</TableHead>
                <TableHead>Services</TableHead>
                <TableHead className="text-right">Observations</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {paginatedSpans.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={4} className="text-center text-muted-foreground">No spans use this attribute</TableCell>
                </TableRow>
              ) : (
                paginatedSpans.map(sp => (
                  <TableRow key={sp.name}>
                    <TableCell className="font-mono text-sm">{sp.name}</TableCell>
                    <TableCell><Badge variant="outline">{sp.kind}</Badge></TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {Object.keys(sp.services || {}).sort().map(svc => (
                          <Badge key={svc} variant="outline" className="text-xs font-mono font-normal">{svc}</Badge>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell className="text-right">{sp.attribute_count?.toLocaleString()}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
          <PaginationRow items={spans} page={spansPage} setPage={setSpansPage} />
        </CardContent>
      </Card>

      {/* Logs */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            Logs
            <Badge variant="secondary">{logs.length}</Badge>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Severity</TableHead>
                <TableHead>Services</TableHead>
                <TableHead className="text-right">Observations</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {paginatedLogs.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={3} className="text-center text-muted-foreground">No logs use this attribute</TableCell>
                </TableRow>
              ) : (
                paginatedLogs.map(l => (
                  <TableRow key={l.severity}>
                    <TableCell className="font-medium">{l.severity}</TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {Object.keys(l.services || {}).sort().map(svc => (
                          <Badge key={svc} variant="outline" className="text-xs font-mono font-normal">{svc}</Badge>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell className="text-right">{l.attribute_count?.toLocaleString()}</TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
          <PaginationRow items={logs} page={logsPage} setPage={setLogsPage} />
        </CardContent>
      </Card>
    </div>
  )
}

export default AttributeExplorer
