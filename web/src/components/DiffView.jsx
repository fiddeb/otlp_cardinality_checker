import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ArrowLeftIcon, ChevronDownIcon, ChevronRightIcon, ArrowRightIcon } from 'lucide-react'

function DiffView({ initialFrom, onBack }) {
  const [sessions, setSessions] = useState([])
  const [fromSession, setFromSession] = useState(initialFrom || '')
  const [toSession, setToSession] = useState('')
  const [loading, setLoading] = useState(false)
  const [loadingSessions, setLoadingSessions] = useState(true)
  const [error, setError] = useState(null)
  const [diff, setDiff] = useState(null)
  const [expandedChanges, setExpandedChanges] = useState({})
  const [signalFilter, setSignalFilter] = useState('all')
  const [severityFilter, setSeverityFilter] = useState('all')
  const [serviceFilter, setServiceFilter] = useState('all')

  useEffect(() => {
    fetchSessions()
  }, [])

  const fetchSessions = async () => {
    try {
      setLoadingSessions(true)
      const response = await fetch('/api/v1/sessions')
      if (!response.ok) throw new Error('Failed to fetch sessions')
      const data = await response.json()
      setSessions(data.sessions || [])
    } catch (err) {
      setError(err.message)
    } finally {
      setLoadingSessions(false)
    }
  }

  const fetchDiff = async () => {
    if (!fromSession || !toSession) {
      setError('Please select both sessions to compare')
      return
    }

    if (fromSession === toSession) {
      setError('Please select different sessions to compare')
      return
    }

    setLoading(true)
    setError(null)
    setDiff(null)

    try {
      let url = `/api/v1/sessions/diff?from=${encodeURIComponent(fromSession)}&to=${encodeURIComponent(toSession)}`
      if (severityFilter !== 'all') {
        url += `&min_severity=${severityFilter}`
      }

      const response = await fetch(url)
      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.error || 'Failed to compute diff')
      }

      const data = await response.json()
      setDiff(data)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const toggleExpand = (changeId) => {
    setExpandedChanges((prev) => ({
      ...prev,
      [changeId]: !prev[changeId],
    }))
  }

  const getChangeTypeBadgeVariant = (type) => {
    switch (type) {
      case 'added': return 'default'
      case 'removed': return 'destructive'
      case 'changed': return 'outline'
      default: return 'secondary'
    }
  }

  const getSeverityBadgeVariant = (severity) => {
    switch (severity) {
      case 'critical': return 'destructive'
      case 'warning': return 'outline'
      case 'info': return 'secondary'
      default: return 'secondary'
    }
  }

  const getSignalBadgeVariant = (signalType) => {
    switch (signalType) {
      case 'metric': return 'default'
      case 'span': return 'secondary'
      case 'log': return 'outline'
      default: return 'outline'
    }
  }

  const formatChange = (change) => {
    const id = `${change.signal_type}-${change.name}-${change.type}`
    const isExpanded = expandedChanges[id]

    return (
      <div key={id} className="border rounded-md mb-2 overflow-hidden">
        <div
          className="flex items-center gap-2 px-4 py-3 cursor-pointer hover:bg-muted/50 transition-colors"
          onClick={() => toggleExpand(id)}
        >
          {isExpanded
            ? <ChevronDownIcon className="h-4 w-4 text-muted-foreground shrink-0" />
            : <ChevronRightIcon className="h-4 w-4 text-muted-foreground shrink-0" />}
          <Badge variant={getSignalBadgeVariant(change.signal_type)} className="shrink-0">{change.signal_type}</Badge>
          <span className="font-mono text-sm flex-1 truncate">{change.name}</span>
          <Badge variant={getChangeTypeBadgeVariant(change.type)} className="shrink-0">{change.type === 'added' ? 'NEW' : change.type}</Badge>
          <Badge variant={getSeverityBadgeVariant(change.severity)} className="shrink-0">{change.severity}</Badge>
        </div>
        {isExpanded && (
          <div className="border-t bg-muted/20 px-4 py-3 flex flex-col gap-3">
            {change.metadata && Object.keys(change.metadata).length > 0 && (
              <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
                {Object.entries(change.metadata).map(([key, value]) => (
                  <div key={key} className="flex gap-2">
                    <span className="text-muted-foreground font-medium">{key}:</span>
                    <span className="font-mono truncate">{JSON.stringify(value)}</span>
                  </div>
                ))}
              </div>
            )}
            {change.details && change.details.length > 0 && (
              <div>
                <p className="text-xs font-semibold text-muted-foreground mb-2 uppercase tracking-wide">Field Changes</p>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Field</TableHead>
                      <TableHead>From</TableHead>
                      <TableHead>To</TableHead>
                      <TableHead>Change %</TableHead>
                      <TableHead>Severity</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {change.details.map((detail, idx) => (
                      <TableRow key={idx}>
                        <TableCell><code className="font-mono text-xs">{detail.field}</code></TableCell>
                        <TableCell className="font-mono text-xs">{detail.from !== null && detail.from !== undefined ? JSON.stringify(detail.from) : '–'}</TableCell>
                        <TableCell className="font-mono text-xs">{detail.to !== null && detail.to !== undefined ? JSON.stringify(detail.to) : '–'}</TableCell>
                        <TableCell className="text-right">
                          {detail.change_pct !== undefined && detail.change_pct !== 0
                            ? <span className={detail.change_pct > 0 ? 'text-destructive' : 'text-green-600'}>
                                {detail.change_pct > 0 ? '+' : ''}{detail.change_pct.toFixed(1)}%
                              </span>
                            : '–'}
                        </TableCell>
                        <TableCell>
                          <Badge variant={getSeverityBadgeVariant(detail.severity)}>{detail.severity}</Badge>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
            {change.message && (
              <p className="text-sm text-muted-foreground italic">{change.message}</p>
            )}
          </div>
        )}
      </div>
    )
  }

  const getAllChanges = () => {
    if (!diff?.changes) return []

    const all = []

    const signalTypes = ['metrics', 'spans', 'logs']
    for (const signalType of signalTypes) {
      const signalChanges = diff.changes[signalType]
      if (signalChanges) {
        if (signalChanges.added) all.push(...signalChanges.added)
        if (signalChanges.removed) all.push(...signalChanges.removed)
        if (signalChanges.changed) all.push(...signalChanges.changed)
      }
    }

    let filtered = all
    if (signalFilter !== 'all') {
      filtered = filtered.filter((c) => c.signal_type === signalFilter)
    }

    if (serviceFilter !== 'all') {
      filtered = filtered.filter((c) => {
        if (c.metadata?.services && serviceFilter in c.metadata.services) {
          return true
        }
        return c.name.startsWith(serviceFilter + '.')
      })
    }

    const severityOrder = { critical: 0, warning: 1, info: 2 }
    filtered.sort((a, b) => {
      const severityDiff = (severityOrder[a.severity] || 3) - (severityOrder[b.severity] || 3)
      if (severityDiff !== 0) return severityDiff
      return a.name.localeCompare(b.name)
    })

    return filtered
  }

  const getSummary = () => {
    if (!diff?.summary) return null

    const { summary } = diff
    return {
      total: summary.total_changes || 0,
      added: summary.added || 0,
      removed: summary.removed || 0,
      changed: summary.changed || 0,
      critical: summary.critical || 0,
      warning: summary.warning || 0,
    }
  }

  const getAvailableServices = () => {
    if (!diff?.changes) return []

    const services = new Set()
    const signalTypes = ['metrics', 'spans', 'logs']
    
    for (const signalType of signalTypes) {
      const signalChanges = diff.changes[signalType]
      if (signalChanges) {
        const allChanges = [
          ...(signalChanges.added || []),
          ...(signalChanges.removed || []),
          ...(signalChanges.changed || []),
        ]
        for (const change of allChanges) {
          if (change.metadata?.services) {
            Object.keys(change.metadata.services).forEach((s) => services.add(s))
          }
          const nameParts = change.name.split('.')
          if (nameParts.length > 1) {
            services.add(nameParts[0])
          }
        }
      }
    }

    return Array.from(services).sort()
  }

  if (loadingSessions) {
    return (
      <div className="flex flex-col gap-6">
        <Skeleton className="h-8 w-32" />
        <Skeleton className="h-32 w-full" />
      </div>
    )
  }

  const summary = getSummary()
  const availableServices = getAvailableServices()
  const allChanges = getAllChanges()

  return (
    <div className="flex flex-col gap-6">
      <Button variant="ghost" size="sm" className="w-fit" onClick={onBack}>
        <ArrowLeftIcon className="mr-2 h-4 w-4" />
        Back to Sessions
      </Button>

      <div>
        <h1 className="text-2xl font-bold tracking-tight">Compare Sessions</h1>
        <p className="text-muted-foreground">Select two sessions to compare and detect changes in metrics, spans, and logs.</p>
      </div>

      {/* Session selector */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex flex-col sm:flex-row items-start sm:items-end gap-4">
            <div className="flex flex-col gap-2 flex-1">
              <label className="text-sm font-medium">From (baseline)</label>
              <Select value={fromSession} onValueChange={setFromSession}>
                <SelectTrigger>
                  <SelectValue placeholder="Select session..." />
                </SelectTrigger>
                <SelectContent>
                  {sessions.map((s) => (
                    <SelectItem key={s.id} value={s.id}>{s.id}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="hidden sm:flex items-center pb-2">
              <ArrowRightIcon className="h-5 w-5 text-muted-foreground" />
            </div>

            <div className="flex flex-col gap-2 flex-1">
              <label className="text-sm font-medium">To (comparison)</label>
              <Select value={toSession} onValueChange={setToSession}>
                <SelectTrigger>
                  <SelectValue placeholder="Select session..." />
                </SelectTrigger>
                <SelectContent>
                  {sessions.map((s) => (
                    <SelectItem key={s.id} value={s.id}>{s.id}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <Button
              onClick={fetchDiff}
              disabled={!fromSession || !toSession || loading}
              className="sm:mb-0"
            >
              {loading ? 'Comparing…' : 'Compare'}
            </Button>
          </div>
        </CardContent>
      </Card>

      {error && (
        <Card className="border-destructive">
          <CardContent className="pt-6">
            <p className="text-destructive text-sm">{error}</p>
          </CardContent>
        </Card>
      )}

      {diff && (
        <>
          {/* Summary */}
          {summary && (
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium text-muted-foreground">Total</CardTitle>
                </CardHeader>
                <CardContent><p className="text-2xl font-bold">{summary.total}</p></CardContent>
              </Card>
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium text-muted-foreground">Added</CardTitle>
                </CardHeader>
                <CardContent><p className="text-2xl font-bold text-primary">{summary.added}</p></CardContent>
              </Card>
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium text-muted-foreground">Removed</CardTitle>
                </CardHeader>
                <CardContent><p className="text-2xl font-bold text-destructive">{summary.removed}</p></CardContent>
              </Card>
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium text-muted-foreground">Changed</CardTitle>
                </CardHeader>
                <CardContent><p className="text-2xl font-bold">{summary.changed}</p></CardContent>
              </Card>
              {summary.critical > 0 && (
                <Card className="border-destructive">
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium text-destructive">Critical</CardTitle>
                  </CardHeader>
                  <CardContent><p className="text-2xl font-bold text-destructive">{summary.critical}</p></CardContent>
                </Card>
              )}
              {summary.warning > 0 && (
                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium text-muted-foreground">Warnings</CardTitle>
                  </CardHeader>
                  <CardContent><p className="text-2xl font-bold">{summary.warning}</p></CardContent>
                </Card>
              )}
            </div>
          )}

          {/* Filters */}
          <div className="flex flex-wrap items-center gap-2">
            <div className="flex gap-1">
              {['all', 'metric', 'span', 'log'].map(f => (
                <Button
                  key={f}
                  variant={signalFilter === f ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => setSignalFilter(f)}
                >
                  {f === 'all' ? 'All' : f.charAt(0).toUpperCase() + f.slice(1) + 's'}
                </Button>
              ))}
            </div>

            {availableServices.length > 0 && (
              <div className="flex items-center gap-2 ml-2">
                <span className="text-sm text-muted-foreground">Service:</span>
                <Select value={serviceFilter} onValueChange={setServiceFilter}>
                  <SelectTrigger className="w-48">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Services</SelectItem>
                    {availableServices.map((service) => (
                      <SelectItem key={service} value={service}>{service}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}
          </div>

          {/* Changes list */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-base flex items-center gap-2">
                Changes
                <Badge variant="secondary">{allChanges.length}</Badge>
              </CardTitle>
              {allChanges.length > 0 && (
                <CardDescription>Click a row to expand field-level changes</CardDescription>
              )}
            </CardHeader>
            <CardContent>
              {allChanges.length === 0 ? (
                <p className="text-sm text-muted-foreground text-center py-8">
                  No changes detected between these sessions.
                </p>
              ) : (
                <div>{allChanges.map(formatChange)}</div>
              )}
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}

export default DiffView
