import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ArrowLeftIcon } from 'lucide-react'

function LogServiceDetails({ serviceName, severity, onBack, onViewPattern }) {
  const [templates, setTemplates] = useState([])
  const [attributeKeys, setAttributeKeys] = useState({})
  const [resourceKeys, setResourceKeys] = useState({})
  const [sampleCount, setSampleCount] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [filter, setFilter] = useState({
    minCount: 0,
  })

  useEffect(() => {
    setLoading(true)
    setError(null)
    
    fetch(`/api/v1/logs/service/${encodeURIComponent(serviceName)}/severity/${encodeURIComponent(severity)}`)
      .then(r => {
        if (!r.ok) {
          throw new Error(`HTTP ${r.status}: ${r.statusText}`)
        }
        return r.json()
      })
      .then(data => {
        setTemplates(data.body_templates || [])
        setAttributeKeys(data.attribute_keys || {})
        setResourceKeys(data.resource_keys || {})
        setSampleCount(data.sample_count || 0)
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [serviceName, severity])

  const getSeverityVariant = (sev) => {
    const map = {
      'ERROR': 'destructive', 'Error': 'destructive',
      'WARN': 'outline', 'Warning': 'outline',
      'INFO': 'secondary', 'Information': 'secondary',
      'DEBUG': 'outline', 'Debug': 'outline',
      'TRACE': 'outline', 'Trace': 'outline',
    }
    return map[sev] || 'outline'
  }

  const filteredTemplates = templates.filter(t => t.count >= filter.minCount)
  const totalMessages = filteredTemplates.reduce((sum, t) => sum + t.count, 0)

  if (loading) {
    return (
      <div className="flex flex-col gap-6">
        <Skeleton className="h-8 w-32" />
        <div className="grid grid-cols-3 gap-4">
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
          <Skeleton className="h-24" />
        </div>
        <Skeleton className="h-[400px] w-full" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col gap-6">
        <Button variant="ghost" size="sm" className="w-fit" onClick={onBack}>
          <ArrowLeftIcon className="mr-2 h-4 w-4" />
          Back to Services
        </Button>
        <Card className="border-destructive">
          <CardContent className="pt-6">
            <p className="text-destructive">Error loading patterns: {error}</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <Button variant="ghost" size="sm" className="w-fit" onClick={onBack}>
        <ArrowLeftIcon className="mr-2 h-4 w-4" />
        Back to Services
      </Button>

      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight font-mono">{serviceName}</h1>
          <div className="flex items-center gap-2 mt-1">
            <p className="text-muted-foreground">Severity:</p>
            <Badge variant={getSeverityVariant(severity)}>{severity}</Badge>
          </div>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium text-muted-foreground">Total Patterns</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{filteredTemplates.length.toLocaleString()}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium text-muted-foreground">Total Messages</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{totalMessages.toLocaleString()}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium text-muted-foreground">Avg per Pattern</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">
              {filteredTemplates.length > 0
                ? Math.round(totalMessages / filteredTemplates.length).toLocaleString()
                : 0}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Filter */}
      <div className="flex items-center gap-2">
        <label className="text-sm text-muted-foreground whitespace-nowrap">Min Count:</label>
        <Input
          type="number"
          value={filter.minCount}
          onChange={(e) => setFilter({ ...filter, minCount: Number(e.target.value) })}
          min="0"
          className="w-32"
        />
      </div>

      {/* Patterns table */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            Log Patterns
            <Badge variant="secondary">{filteredTemplates.length.toLocaleString()}</Badge>
          </CardTitle>
          {onViewPattern && (
            <CardDescription>Click a row to explore pattern details</CardDescription>
          )}
        </CardHeader>
        <CardContent>
          {filteredTemplates.length === 0 ? (
            <p className="text-sm text-muted-foreground py-4 text-center">No patterns match the current filters</p>
          ) : (
            <Table className="table-fixed">
              <TableHeader>
                <TableRow>
                  <TableHead className="w-1/2">Pattern Template</TableHead>
                  <TableHead className="text-right">Count</TableHead>
                  <TableHead className="text-right">Percentage</TableHead>
                  <TableHead>Example</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredTemplates.map((tmpl, i) => (
                  <TableRow
                    key={i}
                    onClick={() => onViewPattern && onViewPattern(serviceName, severity, tmpl.template)}
                    className={onViewPattern ? 'cursor-pointer hover:bg-muted/50' : ''}
                  >
                    <TableCell className="whitespace-normal">
                      <code className="block rounded bg-muted px-2 py-1 text-xs font-mono break-all">
                        {tmpl.template}
                      </code>
                    </TableCell>
                    <TableCell className="text-right">{tmpl.count.toLocaleString()}</TableCell>
                    <TableCell className="text-right">{tmpl.percentage.toFixed(1)}%</TableCell>
                    <TableCell className="text-muted-foreground text-xs italic max-w-xs truncate">
                      {tmpl.example}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export default LogServiceDetails
