import { useState, useEffect } from 'react'
import ValueExplorer from './ValueExplorer'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'

const MAX_WATCHED_FIELDS = 10

function AttributesView() {
  const [attributes, setAttributes] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [filter, setFilter] = useState({
    signalType: 'all',
    scope: 'all',
    minCardinality: 0,
    search: ''
  })
  const [sortField, setSortField] = useState('cardinality')
  const [sortDirection, setSortDirection] = useState('desc')
  const [explorerKey, setExplorerKey] = useState(null) // key for ValueExplorer panel
  const [watchToggling, setWatchToggling] = useState({}) // key -> bool (loading)
  const [expandedSamples, setExpandedSamples] = useState({}) // key -> bool

  const itemsPerPage = 100

  const fetchAttributes = () => {
    let url = '/api/v1/attributes?limit=0'

    if (filter.signalType !== 'all') url += `&signal_type=${filter.signalType}`
    if (filter.scope !== 'all') url += `&scope=${filter.scope}`
    if (filter.minCardinality > 0) url += `&min_cardinality=${filter.minCardinality}`
    url += `&sort_by=${sortField}&sort_order=${sortDirection}`

    fetch(url)
      .then(r => r.json())
      .then(result => {
        setAttributes(result.data || [])
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }

  useEffect(() => {
    fetchAttributes()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filter, sortField, sortDirection])

  const filteredAttributes = (attributes || []).filter(attr => {
    if (filter.search && !attr.key.toLowerCase().includes(filter.search.toLowerCase())) return false
    return true
  })

  const watchedCount = filteredAttributes.filter(a => a.watched).length
  const limitReached = watchedCount >= MAX_WATCHED_FIELDS

  const totalPages = Math.ceil(filteredAttributes.length / itemsPerPage)
  const startIndex = (currentPage - 1) * itemsPerPage
  const endIndex = startIndex + itemsPerPage
  const currentAttributes = filteredAttributes.slice(startIndex, endIndex)

  useEffect(() => {
    setCurrentPage(1)
  }, [filter])

  const getCardinalityBadge = (card) => {
    if (card > 1000) return 'high'
    if (card > 100) return 'medium'
    return 'low'
  }

  const getScopeColor = (scope) => {
    const colors = {
      'resource': '#1976d2',
      'attribute': '#388e3c',
      'both': '#f57c00'
    }
    return colors[scope] || 'var(--text-secondary)'
  }

  const handleSort = (field) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDirection('desc')
    }
  }

  const handleWatchToggle = async (attr) => {
    const key = attr.key
    setWatchToggling(prev => ({ ...prev, [key]: true }))
    try {
      if (attr.watched) {
        await fetch(`/api/v1/attributes/${encodeURIComponent(key)}/watch`, { method: 'DELETE' })
        if (explorerKey === key) setExplorerKey(null)
      } else {
        const res = await fetch(`/api/v1/attributes/${encodeURIComponent(key)}/watch`, { method: 'POST' })
        if (res.ok) setExplorerKey(key)
      }
      fetchAttributes()
    } catch (err) {
      console.error('[watch] toggle failed:', err)
    } finally {
      setWatchToggling(prev => ({ ...prev, [key]: false }))
    }
  }

  if (loading) return (
    <Card><CardHeader><Skeleton className="h-6 w-40" /></CardHeader><CardContent><div className="flex flex-col gap-3">{[...Array(5)].map((_,i) => <Skeleton key={i} className="h-10" />)}</div></CardContent></Card>
  )
  if (error) return <p className="text-sm text-destructive">Error: {error}</p>

  return (
    <div className="flex flex-col gap-6" style={{ marginRight: explorerKey ? '540px' : 0 }}>
      <div>
        <h2 className="text-xl font-semibold">Attribute Catalog</h2>
        <p className="text-sm text-muted-foreground">
          Global attribute cardinality tracking across all signals (metrics, spans, logs)
        </p>
      </div>

      <div className="flex flex-wrap items-end gap-2">
        <div className="flex items-center gap-1">
          <label className="text-sm text-muted-foreground whitespace-nowrap">Signal Type:</label>
          <Select value={filter.signalType} onValueChange={v => setFilter({...filter, signalType: v})}>
            <SelectTrigger className="w-36">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Signals</SelectItem>
              <SelectItem value="metric">Metrics</SelectItem>
              <SelectItem value="span">Spans</SelectItem>
              <SelectItem value="log">Logs</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="flex items-center gap-1">
          <label className="text-sm text-muted-foreground whitespace-nowrap">Scope:</label>
          <Select value={filter.scope} onValueChange={v => setFilter({...filter, scope: v})}>
            <SelectTrigger className="w-44">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Scopes</SelectItem>
              <SelectItem value="resource">Resource Attributes</SelectItem>
              <SelectItem value="attribute">Data Attributes</SelectItem>
              <SelectItem value="both">Both</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="flex items-center gap-1">
          <label className="text-sm text-muted-foreground whitespace-nowrap">Min Cardinality:</label>
          <Input
            type="number"
            value={filter.minCardinality}
            onChange={e => setFilter({...filter, minCardinality: parseInt(e.target.value) || 0})}
            placeholder="0"
            min="0"
            className="w-24"
          />
        </div>

        <Input
          placeholder="Filter by key..."
          value={filter.search}
          onChange={e => setFilter({...filter, search: e.target.value})}
          className="w-48"
        />
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Card>
          <CardContent className="pt-4">
            <p className="text-sm text-muted-foreground">Total Attributes</p>
            <p className="text-2xl font-bold">{filteredAttributes.length}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-4">
            <p className="text-sm text-muted-foreground">High Cardinality (&gt;1000)</p>
            <p className="text-2xl font-bold">{filteredAttributes.filter(a => a.estimated_cardinality > 1000).length}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-4">
            <p className="text-sm text-muted-foreground">Resource Attributes</p>
            <p className="text-2xl font-bold">{filteredAttributes.filter(a => a.scope === 'resource' || a.scope === 'both').length}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-4">
            <p className="text-sm text-muted-foreground">Watching</p>
            <p className="text-2xl font-bold">{watchedCount} / {MAX_WATCHED_FIELDS}</p>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardContent className="p-0">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="cursor-pointer" onClick={() => handleSort('key')}>
              Attribute Key {sortField === 'key' && (sortDirection === 'asc' ? '↑' : '↓')}
            </TableHead>
            <TableHead className="cursor-pointer" onClick={() => handleSort('cardinality')}>
              Cardinality (est.) {sortField === 'cardinality' && (sortDirection === 'asc' ? '↑' : '↓')}
            </TableHead>
            <TableHead className="cursor-pointer" onClick={() => handleSort('count')}>
              Count {sortField === 'count' && (sortDirection === 'asc' ? '↑' : '↓')}
            </TableHead>
            <TableHead className="max-w-[340px] w-[340px]">Sample Values</TableHead>
            <TableHead>Signal Types</TableHead>
            <TableHead>Scope</TableHead>
            <TableHead>Watch</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {currentAttributes.map((attr, idx) => (
            <TableRow key={idx} style={attr.has_invalid_utf8 ? { backgroundColor: 'rgba(220, 38, 38, 0.07)' } : undefined}>
              <TableCell>
                {attr.watched ? (
                  <Button
                    variant="link"
                    className="h-auto p-0 font-mono font-normal"
                    style={attr.has_invalid_utf8 ? { color: 'var(--destructive)' } : undefined}
                    onClick={() => setExplorerKey(explorerKey === attr.key ? null : attr.key)}
                    title="Open Value Explorer"
                  >
                    {attr.key}
                    {attr.has_invalid_utf8 && <span className="ml-1" title="One or more values contained invalid UTF-8 bytes (replaced with \uFFFD)">⚠</span>}
                  </Button>
                ) : (
                  <code className="attribute-key" style={attr.has_invalid_utf8 ? { color: 'var(--destructive)' } : undefined}>
                    {attr.key}
                    {attr.has_invalid_utf8 && <span className="ml-1" title="One or more values contained invalid UTF-8 bytes (replaced with \uFFFD)">⚠</span>}
                  </code>
                )}
              </TableCell>
              <TableCell>
                <Badge variant={getCardinalityBadge(attr.estimated_cardinality) === 'high' ? 'destructive' : getCardinalityBadge(attr.estimated_cardinality) === 'medium' ? 'secondary' : 'outline'}>
                  {attr.estimated_cardinality?.toLocaleString() || 0}
                </Badge>
              </TableCell>
              <TableCell>{attr.count?.toLocaleString() || 0}</TableCell>
              <TableCell className="max-w-[340px]">
                <div className="sample-values">
                  {(attr.value_samples || [])
                    .slice(0, expandedSamples[attr.key] ? undefined : 5)
                    .map((val, i) => (
                      <code key={i} className="sample-value">{val}</code>
                    ))
                  }
                  {!expandedSamples[attr.key] && (attr.value_samples?.length || 0) > 5 && (
                    <button
                      className="more-indicator"
                      onClick={() => setExpandedSamples(prev => ({ ...prev, [attr.key]: true }))}
                      title="Show all sample values"
                    >
                      +{attr.value_samples.length - 5} more
                    </button>
                  )}
                  {expandedSamples[attr.key] && (
                    <button
                      className="more-indicator"
                      onClick={() => setExpandedSamples(prev => ({ ...prev, [attr.key]: false }))}
                      title="Show fewer"
                    >
                      show less
                    </button>
                  )}
                </div>
              </TableCell>
              <TableCell>
                <div className="flex flex-wrap gap-1">
                  {(attr.signal_types || []).map((type, i) => (
                    <Badge key={i} variant="secondary" className="text-xs">{type}</Badge>
                  ))}
                </div>
              </TableCell>
              <TableCell>
                <Badge
                  style={{ background: getScopeColor(attr.scope) }}
                  className="text-white border-0"
                >
                  {attr.scope}
                </Badge>
              </TableCell>
              <TableCell>
                <button
                  onClick={() => handleWatchToggle(attr)}
                  disabled={watchToggling[attr.key] || (!attr.watched && limitReached)}
                  title={
                    !attr.watched && limitReached
                      ? `Limit of ${MAX_WATCHED_FIELDS} watched fields reached`
                      : attr.watched
                        ? 'Stop watching'
                        : 'Start deep watch'
                  }
                  style={{
                    padding: '3px 10px',
                    borderRadius: 4,
                    border: '1px solid',
                    cursor: (watchToggling[attr.key] || (!attr.watched && limitReached)) ? 'not-allowed' : 'pointer',
                    opacity: (!attr.watched && limitReached) ? 0.4 : 1,
                    background: attr.watched ? (attr.has_invalid_utf8 ? 'rgba(220,38,38,0.10)' : '#e3f2fd') : 'transparent',
                    borderColor: attr.watched ? (attr.has_invalid_utf8 ? 'var(--danger, #dc2626)' : '#1976d2') : '#ccc',
                    color: attr.watched ? (attr.has_invalid_utf8 ? 'var(--danger, #dc2626)' : '#1976d2') : '#666',
                    fontSize: 12,
                    fontWeight: attr.watched ? 600 : 400,
                    whiteSpace: 'nowrap',
                  }}
                >
                  {watchToggling[attr.key] ? '…' : attr.watched ? (attr.has_invalid_utf8 ? '⚠ Watching' : 'Watching') : 'Watch'}
                </button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
        </CardContent>
      </Card>

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

      {/* Value Explorer side panel */}
      {explorerKey && (
        <ValueExplorer
          attributeKey={explorerKey}
          onClose={() => setExplorerKey(null)}
        />
      )}
    </div>
  )
}

export default AttributesView

