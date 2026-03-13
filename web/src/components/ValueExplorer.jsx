import { useState, useEffect, useCallback } from 'react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

const MAX_WATCHED_FIELDS = 10

function ValueExplorer({ attributeKey, onClose }) {
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [sortBy, setSortBy] = useState('count')
  const [sortDir, setSortDir] = useState('desc')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const pageSize = 100

  const fetchData = useCallback(() => {
    if (!attributeKey) return
    setLoading(true)
    setError(null)

    const params = new URLSearchParams({
      sort_by: sortBy,
      sort_direction: sortDir,
      page: String(page),
      page_size: String(pageSize),
    })
    if (search) params.set('q', search)

    fetch(`/api/v1/attributes/${encodeURIComponent(attributeKey)}/watch?${params}`)
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
  }, [attributeKey, sortBy, sortDir, search, page])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  const handleSort = (field) => {
    if (sortBy === field) {
      setSortDir(d => d === 'asc' ? 'desc' : 'asc')
    } else {
      setSortBy(field)
      setSortDir('desc')
    }
    setPage(1)
  }

  const formatDateTime = (isoStr) => {
    if (!isoStr) return ''
    return new Date(isoStr).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }

  const SortIndicator = ({ field }) => sortBy === field ? (sortDir === 'asc' ? ' ↑' : ' ↓') : ''

  return (
    <Sheet open={!!attributeKey} onOpenChange={(open) => { if (!open) onClose() }}>
      <SheetContent className="flex w-[520px] flex-col gap-0 p-0 sm:max-w-[520px]">
        <SheetHeader className="border-b px-4 py-3">
          <SheetTitle className="flex items-center gap-2 text-base">
            <Badge variant="default" className="text-xs">WATCHING</Badge>
            {data?.has_invalid_utf8 && (
              <Badge variant="destructive" className="text-xs" title="One or more values contained invalid UTF-8 bytes">⚠ INVALID UTF-8</Badge>
            )}
            <code className="text-sm font-bold">{attributeKey}</code>
          </SheetTitle>
          {data && (
            <div className="flex gap-4 text-xs text-muted-foreground">
              <span>Since {formatDateTime(data.watching_since)}</span>
              <span>{(data.unique_count || 0).toLocaleString()} unique</span>
              <span>{(data.total_observations || 0).toLocaleString()} total</span>
              {!data.active && <span className="font-semibold text-orange-500">read-only (session)</span>}
            </div>
          )}
        </SheetHeader>

        {data?.overflow && (
          <div className="border-b bg-orange-50 px-4 py-2 text-xs text-orange-700 dark:bg-orange-950 dark:text-orange-300">
            ⚠ 10,000 unique values reached — new unique values are no longer collected.
          </div>
        )}

        <div className="flex items-center gap-2 border-b px-4 py-2">
          <Input
            placeholder="Prefix filter…"
            value={search}
            onChange={e => { setSearch(e.target.value); setPage(1) }}
            className="h-8 text-sm"
          />
          <Button variant="outline" size="sm" onClick={fetchData}>Refresh</Button>
        </div>

        <div className="flex-1 overflow-y-auto">
          {loading && (
            <p className="py-8 text-center text-sm text-muted-foreground">Loading…</p>
          )}
          {error && (
            <p className="p-4 text-sm text-destructive">Error: {error}</p>
          )}
          {!loading && !error && data && (
            <>
              {(data.values || []).length === 0 ? (
                <p className="py-8 text-center text-sm text-muted-foreground">No values collected yet.</p>
              ) : (
                <Table>
                  <TableHeader className="sticky top-0 bg-background">
                    <TableRow>
                      <TableHead className="cursor-pointer select-none" onClick={() => handleSort('value')}>
                        Value<SortIndicator field="value" />
                      </TableHead>
                      <TableHead className="cursor-pointer select-none text-right w-24" onClick={() => handleSort('count')}>
                        Count<SortIndicator field="count" />
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {data.values.map((entry, i) => (
                      <TableRow key={i}>
                        <TableCell className="font-mono text-xs break-all">{entry.value}</TableCell>
                        <TableCell className="text-right tabular-nums">{entry.count.toLocaleString()}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}

              {(data.total_values || 0) > pageSize && (
                <div className="flex items-center justify-center gap-2 p-3">
                  <Button variant="outline" size="sm" onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page <= 1}>
                    ← Prev
                  </Button>
                  <span className="text-sm text-muted-foreground">
                    {page} / {Math.ceil((data.total_values || 0) / pageSize)}
                  </span>
                  <Button variant="outline" size="sm" onClick={() => setPage(p => p + 1)} disabled={(data.values || []).length < pageSize}>
                    Next →
                  </Button>
                </div>
              )}
            </>
          )}
        </div>
      </SheetContent>
    </Sheet>
  )
}

export default ValueExplorer
