import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { ChevronDownIcon, ChevronRightIcon } from 'lucide-react'

function TracePatterns({ onViewDetails }) {
  const [patterns, setPatterns] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [expandedPatterns, setExpandedPatterns] = useState({})
  const [minSpans, setMinSpans] = useState(1)
  const [showOnlyNormalized, setShowOnlyNormalized] = useState(false)

  useEffect(() => {
    fetchPatterns()
  }, [])

  const fetchPatterns = async () => {
    try {
      setLoading(true)
      const response = await fetch('/api/v1/span-patterns')
      if (!response.ok) throw new Error('Failed to fetch span patterns')
      const data = await response.json()
      setPatterns(data.patterns || [])
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const toggleExpand = (pattern) => {
    setExpandedPatterns(prev => ({ ...prev, [pattern]: !prev[pattern] }))
  }

  const isNormalized = (pattern) => /<[A-Z_]+>/i.test(pattern)

  const getExample = (pg) => {
    if (pg.matching_spans && pg.matching_spans.length > 0) {
      const firstSpan = pg.matching_spans[0].span_name
      if (firstSpan === pg.pattern) return null
      return firstSpan
    }
    return null
  }

  const filteredPatterns = patterns.filter(p => {
    if (p.span_count < minSpans) return false
    if (showOnlyNormalized && !isNormalized(p.pattern)) return false
    return true
  })

  const multiSpanPatterns = filteredPatterns.filter(p => p.span_count > 1)
  const singleSpanPatterns = filteredPatterns.filter(p => p.span_count === 1)
  const normalizedCount = filteredPatterns.filter(p => isNormalized(p.pattern)).length

  if (loading) {
    return (
      <div className="flex flex-col gap-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Trace Patterns</h1>
          <p className="text-muted-foreground">Span names aggregated by extracted patterns</p>
        </div>
        <Skeleton className="h-[400px] w-full" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col gap-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Trace Patterns</h1>
          <p className="text-muted-foreground">Span names aggregated by extracted patterns</p>
        </div>
        <Card className="border-destructive">
          <CardContent className="pt-6">
            <p className="text-destructive">Error: {error}</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Trace Patterns</h1>
        <p className="text-muted-foreground">
          Span names aggregated by extracted patterns. Multi-variant patterns indicate high-cardinality span naming.
        </p>
      </div>

      <div className="flex items-center gap-4 flex-wrap">
        <div className="flex items-center gap-2">
          <label className="text-sm font-medium whitespace-nowrap">Min Variants:</label>
          <Input
            type="number"
            min="1"
            value={minSpans}
            onChange={(e) => setMinSpans(parseInt(e.target.value) || 1)}
            className="w-20"
          />
        </div>
        <label className="flex items-center gap-2 text-sm font-medium cursor-pointer">
          <input
            type="checkbox"
            checked={showOnlyNormalized}
            onChange={(e) => setShowOnlyNormalized(e.target.checked)}
            className="rounded"
          />
          Normalized only
        </label>
        <span className="text-sm text-muted-foreground ml-auto">
          {filteredPatterns.length} patterns · {normalizedCount} normalized · {multiSpanPatterns.length} multi-variant
        </span>
      </div>

      <div className="flex gap-2 flex-wrap">
        <Badge variant="default" className="gap-1">Normalized — dynamic values replaced with placeholders</Badge>
        <Badge variant="outline" className="gap-1">Original — no dynamic values detected</Badge>
      </div>

      {multiSpanPatterns.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>High Cardinality Patterns ({multiSpanPatterns.length})</CardTitle>
            <CardDescription>
              Patterns with 2+ variants indicate dynamic values (IDs, paths) embedded in span names.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-2 p-3">
            {multiSpanPatterns.map((pg, idx) => {
              const normalized = isNormalized(pg.pattern)
              const example = getExample(pg)
              const expanded = expandedPatterns[pg.pattern]
              return (
                <div key={idx} className="rounded-md border">
                  <button
                    className="flex w-full items-start gap-2 p-3 text-left hover:bg-muted/50 transition-colors"
                    onClick={() => toggleExpand(pg.pattern)}
                  >
                    {expanded ? (
                      <ChevronDownIcon className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
                    ) : (
                      <ChevronRightIcon className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
                    )}
                    <div className="flex flex-1 flex-col gap-1">
                      <div className="flex items-center gap-2 flex-wrap">
                        <Badge variant={normalized ? 'default' : 'outline'}>
                          {normalized ? 'Normalized' : 'Original'}
                        </Badge>
                        <code className="text-sm">{pg.pattern}</code>
                      </div>
                      {example && (
                        <span className="text-xs text-muted-foreground">
                          Example: <code>{example}</code>
                        </span>
                      )}
                      <div className="flex items-center gap-2">
                        <Badge variant="destructive" className="text-xs">{pg.span_count} variants</Badge>
                        <span className="text-xs text-muted-foreground">{pg.total_samples.toLocaleString()} samples</span>
                      </div>
                    </div>
                  </button>
                  {expanded && (
                    <div className="border-t">
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>Span Name (Variant)</TableHead>
                            <TableHead>Kind</TableHead>
                            <TableHead>Samples</TableHead>
                            <TableHead>Services</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {pg.matching_spans.map((span, i) => (
                            <TableRow key={i}>
                              <TableCell
                                className="font-mono text-xs cursor-pointer text-primary hover:underline"
                                onClick={() => onViewDetails('spans', span.span_name)}
                              >
                                {span.span_name}
                              </TableCell>
                              <TableCell>
                                <Badge variant="outline">{span.kind || 'Unknown'}</Badge>
                              </TableCell>
                              <TableCell>{span.sample_count.toLocaleString()}</TableCell>
                              <TableCell className="text-xs text-muted-foreground">
                                {span.services?.join(', ') || '-'}
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </div>
                  )}
                </div>
              )
            })}
          </CardContent>
        </Card>
      )}

      {minSpans === 1 && singleSpanPatterns.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Single-Variant Patterns ({singleSpanPatterns.length})</CardTitle>
            <CardDescription>
              Each pattern matches only one span name.
            </CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Type</TableHead>
                  <TableHead>Pattern</TableHead>
                  <TableHead>Example</TableHead>
                  <TableHead>Kind</TableHead>
                  <TableHead>Samples</TableHead>
                  <TableHead>Services</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {singleSpanPatterns.slice(0, 50).map((pg, idx) => {
                  const span = pg.matching_spans[0]
                  const normalized = isNormalized(pg.pattern)
                  const example = getExample(pg)
                  return (
                    <TableRow key={idx}>
                      <TableCell>
                        <Badge variant={normalized ? 'default' : 'outline'} className="text-xs">
                          {normalized ? 'Normalized' : 'Original'}
                        </Badge>
                      </TableCell>
                      <TableCell className="font-mono text-xs">{pg.pattern}</TableCell>
                      <TableCell>
                        {example ? (
                          <span
                            className="font-mono text-xs cursor-pointer text-primary hover:underline"
                            onClick={() => onViewDetails('spans', span.span_name)}
                          >
                            {example}
                          </span>
                        ) : (
                          <span className="text-muted-foreground">-</span>
                        )}
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline">{span?.kind || 'Unknown'}</Badge>
                      </TableCell>
                      <TableCell>{pg.total_samples.toLocaleString()}</TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {span?.services?.slice(0, 3).join(', ')}{span?.services?.length > 3 ? '...' : ''}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
            {singleSpanPatterns.length > 50 && (
              <p className="py-3 text-center text-xs text-muted-foreground">
                Showing 50 of {singleSpanPatterns.length} patterns
              </p>
            )}
          </CardContent>
        </Card>
      )}

      {filteredPatterns.length === 0 && (
        <Card>
          <CardContent className="py-8 text-center">
            <p className="text-muted-foreground">No span patterns found. Send some trace data to see patterns.</p>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

export default TracePatterns
