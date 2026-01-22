import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Button } from '@/components/ui/button'

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

  if (loading) return <div className="loading">Loading...</div>
  if (error) return <div className="error">Error: {error}</div>

  return (
    <Card>
      <CardHeader>
        <CardTitle>Traces Analysis</CardTitle>
        <CardDescription>
          Showing {startIndex + 1}-{Math.min(endIndex, filteredSpans.length)} of {filteredSpans.length} span operations
          {totalPages > 1 && ` (Page ${currentPage} of ${totalPages})`}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Filters */}
        <div className="flex flex-wrap gap-4">
          <Input 
            type="text"
            placeholder="Search spans..."
            value={filter.search}
            onChange={(e) => setFilter({...filter, search: e.target.value})}
            className="w-64"
          />

          <Select 
            value={filter.kind} 
            onValueChange={(value) => setFilter({...filter, kind: value})}
          >
            <SelectTrigger className="w-48">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {spanKinds.map(kind => (
                <SelectItem key={kind} value={kind}>
                  {kind === 'all' ? 'All Kinds' : `Kind: ${kind}`}
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

        {/* Spans Table */}
        <div className="rounded-md border">
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
                      <button 
                        className="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 hover:underline"
                        onClick={() => onViewDetails('spans', span.name)}
                      >
                        {span.name}
                      </button>
                    </TableCell>
                    <TableCell>
                      <Badge variant="secondary">{span.kind || 'Unknown'}</Badge>
                    </TableCell>
                    <TableCell>{span.sample_count.toLocaleString()}</TableCell>
                    <TableCell>{attrCount}</TableCell>
                    <TableCell>
                      {maxCard > 0 ? (
                        <Badge variant={maxCard > 200 ? 'destructive' : maxCard > 50 ? 'default' : 'secondary'}>
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
        {filteredSpans.length === 0 && (
          <div className="text-center py-8 text-muted-foreground">
            No spans match the current filters
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export default TracesView
