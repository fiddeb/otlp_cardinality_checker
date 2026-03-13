import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'

function TracesView({ onViewDetails }) {
  const [spans, setSpans] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [filter, setFilter] = useState({
    kind: 'all',
    minSamples: 0,
    minCardinality: 0,
    search: ''
  })

  const itemsPerPage = 100

  useEffect(() => {
    fetch('/api/v1/spans?limit=1000')
      .then(r => r.json())
      .then(result => {
        setSpans(result.data || [])
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [])

  const getMaxCardinality = (span) => {
    if (!span.attribute_keys) return 0
    return Math.max(...Object.values(span.attribute_keys).map(k => k.estimated_cardinality || 0))
  }

  const filteredSpans = spans.filter(span => {
    if (filter.kind !== 'all' && span.kind !== filter.kind) return false
    if (span.sample_count < filter.minSamples) return false
    if (getMaxCardinality(span) < filter.minCardinality) return false
    if (filter.search && !span.name.toLowerCase().includes(filter.search.toLowerCase())) return false
    return true
  })

  const totalPages = Math.ceil(filteredSpans.length / itemsPerPage)
  const startIndex = (currentPage - 1) * itemsPerPage
  const endIndex = startIndex + itemsPerPage
  const currentSpans = filteredSpans.slice(startIndex, endIndex)

  // Reset to page 1 when filters change
  useEffect(() => {
    setCurrentPage(1)
  }, [filter])

  const spanKinds = ['all', ...new Set(spans.map(s => s.kind).filter(Boolean))]

  const getCardinalityBadge = (card) => {
    if (card > 200) return 'high'
    if (card > 50) return 'medium'
    return 'low'
  }

  if (loading) return (
    <Card><CardHeader><Skeleton className="h-6 w-40" /></CardHeader><CardContent><div className="flex flex-col gap-3">{[...Array(5)].map((_,i) => <Skeleton key={i} className="h-10" />)}</div></CardContent></Card>
  )
  if (error) return <p className="text-sm text-destructive">Error: {error}</p>

  return (
    <div className="flex flex-col gap-4">
      <h2 className="text-xl font-semibold">Traces Analysis</h2>

      <div className="flex flex-wrap items-end gap-2">
        <Input
          placeholder="Search spans..."
          value={filter.search}
          onChange={(e) => setFilter({...filter, search: e.target.value})}
          className="w-48"
        />

        <Select value={filter.kind} onValueChange={(v) => setFilter({...filter, kind: v})}>
          <SelectTrigger className="w-44">
            <SelectValue placeholder="All Kinds" />
          </SelectTrigger>
          <SelectContent>
            {spanKinds.map(kind => (
              <SelectItem key={kind} value={kind}>
                {kind === 'all' ? 'All Kinds' : `Kind: ${kind}`}
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
        Showing {startIndex + 1}–{Math.min(endIndex, filteredSpans.length)} of {filteredSpans.length} span operations
        {totalPages > 1 && ` (Page ${currentPage} of ${totalPages})`}
      </p>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Span Name</TableHead>
            <TableHead>Kind</TableHead>
            <TableHead>Samples</TableHead>
            <TableHead>Attributes</TableHead>
            <TableHead>Max Cardinality</TableHead>
            <TableHead>Services</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {currentSpans.map((span, i) => {
            const maxCard = getMaxCardinality(span)
            const attrCount = span.attribute_keys ? Object.keys(span.attribute_keys).length : 0
            const serviceCount = span.services ? Object.keys(span.services).length : 0

            return (
              <TableRow key={i}>
                <TableCell>
                  <Button variant="link" className="h-auto p-0 font-normal" onClick={() => onViewDetails('spans', span.name)}>
                    {span.name}
                  </Button>
                </TableCell>
                <TableCell>
                  <Badge variant="secondary">{span.kind || 'Unknown'}</Badge>
                </TableCell>
                <TableCell>{span.sample_count.toLocaleString()}</TableCell>
                <TableCell>{attrCount}</TableCell>
                <TableCell>
                  {maxCard > 0 ? (
                    <Badge variant={getCardinalityBadge(maxCard) === 'high' ? 'destructive' : getCardinalityBadge(maxCard) === 'medium' ? 'secondary' : 'outline'}>
                      {maxCard}
                    </Badge>
                  ) : '-'}
                </TableCell>
                <TableCell>{serviceCount}</TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>

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

      {filteredSpans.length === 0 && (
        <p className="text-sm text-muted-foreground text-center py-5">No spans match the current filters</p>
      )}
    </div>
  )
}

export default TracesView
