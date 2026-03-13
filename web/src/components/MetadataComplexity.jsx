import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

function MetadataComplexity({ onViewDetails }) {
  const [threshold, setThreshold] = useState(10)
  const [limit, setLimit] = useState(50)
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [sortField, setSortField] = useState('complexity_score')
  const [sortDirection, setSortDirection] = useState('desc')

  useEffect(() => {
    setLoading(true)
    setError(null)

    fetch(`/api/v1/cardinality/complexity?threshold=${threshold}&limit=${limit}`)
      .then(r => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        return r.json()
      })
      .then(result => {
        setData(result)
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [threshold, limit])

  const getSignalTypeVariant = (type) => {
    switch(type) {
      case 'metric': return 'default'
      case 'span': return 'secondary'
      case 'log': return 'outline'
      default: return 'outline'
    }
  }

  const getComplexityVariant = (score) => {
    if (score < 50) return 'secondary'
    if (score < 200) return 'outline'
    return 'destructive'
  }

  const handleSort = (field) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDirection('desc')
    }
  }

  const sortData = (signals) => {
    if (!signals) return []
    return [...signals].sort((a, b) => {
      const aVal = a[sortField]
      const bVal = b[sortField]
      if (sortDirection === 'asc') return aVal > bVal ? 1 : -1
      return aVal < bVal ? 1 : -1
    })
  }

  if (loading) {
    return (
      <div className="flex flex-col gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Metadata Complexity</h1>
          <p className="text-muted-foreground">Identify signals with excessive instrumentation</p>
        </div>
        <Skeleton className="h-[400px] w-full" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Metadata Complexity</h1>
          <p className="text-muted-foreground">Identify signals with excessive instrumentation</p>
        </div>
        <Card className="border-destructive">
          <CardContent className="pt-6">
            <p className="text-destructive">Error: {error}</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!data || !data.signals) {
    return (
      <div className="flex flex-col gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Metadata Complexity</h1>
          <p className="text-muted-foreground">Identify signals with excessive instrumentation</p>
        </div>
        <p className="text-muted-foreground">No signals found</p>
      </div>
    )
  }

  const sortedSignals = sortData(data.signals)

  const SortIndicator = ({ field }) => sortField === field ? (sortDirection === 'asc' ? ' ▲' : ' ▼') : ''

  return (
    <div className="flex flex-col gap-4">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Metadata Complexity</h1>
        <p className="text-muted-foreground">Identify signals with excessive instrumentation that may cause cardinality issues</p>
      </div>

      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Complex Signals</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data.total}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Key Threshold</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{threshold}+</div>
          </CardContent>
        </Card>
      </div>

      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2">
          <label className="text-sm font-medium whitespace-nowrap">Min Total Keys:</label>
          <Input
            type="number"
            value={threshold}
            onChange={(e) => setThreshold(Number(e.target.value))}
            min="1"
            step="5"
            className="w-24"
          />
        </div>
        <div className="flex items-center gap-2">
          <label className="text-sm font-medium whitespace-nowrap">Max Results:</label>
          <Input
            type="number"
            value={limit}
            onChange={(e) => setLimit(Number(e.target.value))}
            min="10"
            max="1000"
            step="10"
            className="w-24"
          />
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Signals ({sortedSignals.length})</CardTitle>
          <CardDescription>Click a row to view signal details</CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort('signal_type')}>
                  Type<SortIndicator field="signal_type" />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort('signal_name')}>
                  Signal Name<SortIndicator field="signal_name" />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort('total_keys')}>
                  Total Keys<SortIndicator field="total_keys" />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort('attribute_key_count')}>
                  Attributes<SortIndicator field="attribute_key_count" />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort('resource_key_count')}>
                  Resources<SortIndicator field="resource_key_count" />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort('max_cardinality')}>
                  Max Card<SortIndicator field="max_cardinality" />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort('high_cardinality_count')}>
                  High Card Keys<SortIndicator field="high_cardinality_count" />
                </TableHead>
                <TableHead className="cursor-pointer select-none" onClick={() => handleSort('complexity_score')}>
                  Complexity<SortIndicator field="complexity_score" />
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {sortedSignals.map((signal, idx) => {
                const typeMap = { 'metric': 'metrics', 'span': 'spans', 'log': 'logs' }
                const pluralType = typeMap[signal.signal_type] || signal.signal_type
                return (
                  <TableRow
                    key={idx}
                    className="cursor-pointer"
                    onClick={() => onViewDetails(pluralType, signal.signal_name)}
                  >
                    <TableCell>
                      <Badge variant={getSignalTypeVariant(signal.signal_type)}>
                        {signal.signal_type}
                      </Badge>
                    </TableCell>
                    <TableCell className="font-mono text-xs">{signal.signal_name}</TableCell>
                    <TableCell className="font-semibold">{signal.total_keys}</TableCell>
                    <TableCell>{signal.attribute_key_count}</TableCell>
                    <TableCell>{signal.resource_key_count}</TableCell>
                    <TableCell>{signal.max_cardinality.toLocaleString()}</TableCell>
                    <TableCell>
                      {signal.high_cardinality_count > 0 ? (
                        <Badge variant="destructive">{signal.high_cardinality_count}</Badge>
                      ) : (
                        <span className="text-muted-foreground">0</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <Badge variant={getComplexityVariant(signal.complexity_score)}>
                        {signal.complexity_score.toLocaleString()}
                      </Badge>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
          {sortedSignals.length === 0 && (
            <p className="py-8 text-center text-sm text-muted-foreground">
              No signals found with {threshold}+ total keys. Try lowering the threshold.
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export default MetadataComplexity
