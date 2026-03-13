import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { ArrowLeftIcon, ChevronDownIcon, ChevronRightIcon } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

function Details({ type, name, onBack }) {
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [showTemplates, setShowTemplates] = useState(true)
  const [showSeriesExplanation, setShowSeriesExplanation] = useState(false)
  const [watchedKeys, setWatchedKeys] = useState({})
  const [watchLoading, setWatchLoading] = useState({})

  useEffect(() => {
    const endpoint = type === 'metrics' || type === 'metric' ? `/api/v1/metrics/${encodeURIComponent(name)}` :
                     type === 'spans' || type === 'span' ? `/api/v1/spans/${encodeURIComponent(name)}` :
                     `/api/v1/logs/${encodeURIComponent(name)}`
    fetch(endpoint)
      .then(r => r.json())
      .then(data => { setData(data); setLoading(false) })
      .catch(err => { setError(err.message); setLoading(false) })
  }, [type, name])

  const handleWatch = async (key) => {
    setWatchLoading(prev => ({ ...prev, [key]: true }))
    try {
      await fetch(`/api/v1/attributes/${encodeURIComponent(key)}/watch`, { method: 'POST' })
      setWatchedKeys(prev => ({ ...prev, [key]: true }))
    } catch (e) { /* ignore */ } finally {
      setWatchLoading(prev => ({ ...prev, [key]: false }))
    }
  }

  const getCardinalityVariant = (card) => {
    if (card > 200) return 'destructive'
    if (card > 50) return 'outline'
    return 'secondary'
  }

  if (loading) {
    return (
      <div className="flex flex-col gap-4">
        <Button variant="ghost" size="sm" className="w-fit" onClick={onBack}>
          <ArrowLeftIcon className="h-4 w-4" /> Back
        </Button>
        <Skeleton className="h-[400px] w-full" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col gap-4">
        <Button variant="ghost" size="sm" className="w-fit" onClick={onBack}>
          <ArrowLeftIcon className="h-4 w-4" /> Back
        </Button>
        <Card className="border-destructive">
          <CardContent className="pt-6">
            <p className="text-destructive">Error: {error}</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  const keys = type === 'metrics' ? data.label_keys : data.attribute_keys
  const otlpSeries = data.active_series_otlp ?? data.active_series ?? 0
  const promSeries = data.active_series_prometheus ?? otlpSeries
  const histogramBucketCount = data.type === 'Histogram' && data.data?.explicit_bounds
    ? data.data.explicit_bounds.length + 1
    : data.type === 'ExponentialHistogram' && data.data?.scales
      ? data.data.scales.length * 10
      : null

  return (
    <div className="flex flex-col gap-4">
      <Button variant="ghost" size="sm" className="w-fit" onClick={onBack}>
        <ArrowLeftIcon className="h-4 w-4" /> Back
      </Button>

      <div>
        <h1 className="text-2xl font-bold tracking-tight">{name}</h1>
        <div className="flex items-center gap-2 mt-1">
          <Badge variant="secondary">{type}</Badge>
          {type === 'metrics' && <Badge variant="outline">{data.type}</Badge>}
          {type === 'spans' && data.kind && <Badge variant="outline">{data.kind}</Badge>}
          <span className="text-sm text-muted-foreground">{data.sample_count?.toLocaleString()} samples</span>
        </div>
      </div>

      {/* Active Series (metrics only) */}
      {type === 'metrics' && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Active Series</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-xs text-muted-foreground mb-1">OTLP Active Series</p>
                <p className={`text-2xl font-bold ${otlpSeries > 1000 ? 'text-destructive' : ''}`}>
                  {otlpSeries.toLocaleString()}
                </p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground mb-1">Prometheus Active Series</p>
                <p className={`text-2xl font-bold ${promSeries > 1000 ? 'text-destructive' : ''}`}>
                  {promSeries.toLocaleString()}
                </p>
              </div>
            </div>
            {promSeries > 1000 && (
              <p className="mt-3 text-sm text-destructive">
                High cardinality detected. This metric generates many unique series which may impact storage and query performance.
              </p>
            )}
            <Button variant="ghost" size="sm" className="mt-2 text-xs" onClick={() => setShowSeriesExplanation(!showSeriesExplanation)}>
              {showSeriesExplanation ? 'Hide' : 'How is this calculated?'}
            </Button>
            {showSeriesExplanation && (
              <div className="mt-2 rounded-md bg-muted p-3 text-xs leading-relaxed text-muted-foreground">
                <p className="font-semibold text-foreground mb-1">What is an active series?</p>
                <p>An active series is a unique combination of all label values for this metric. The system tracks actual observed combinations.</p>
                {(data.type === 'Histogram' || data.type === 'ExponentialHistogram') && histogramBucketCount !== null && (
                  <p className="mt-2">Histogram: Prometheus Active Series = OTLP × (bucket_count + 2). bucket_count = {histogramBucketCount}.</p>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Histogram buckets */}
      {type === 'metrics' && data.type === 'Histogram' && data.data?.explicit_bounds && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Histogram Buckets</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground mb-2">
              {data.data.explicit_bounds.length + 1} buckets · Aggregation: {data.data.aggregation_temporality === 1 ? 'Delta' : data.data.aggregation_temporality === 2 ? 'Cumulative' : 'Unknown'}
            </p>
            <div className="flex flex-wrap gap-1">
              <Badge variant="outline" className="font-mono text-xs">(-∞, {data.data.explicit_bounds[0]}]</Badge>
              {data.data.explicit_bounds.map((bound, idx) => (
                <Badge key={idx} variant="outline" className="font-mono text-xs">
                  ({bound}, {data.data.explicit_bounds[idx + 1] || '∞'}]
                </Badge>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Body templates (logs only) */}
      {type === 'logs' && data.body_templates && data.body_templates.length > 0 && (
        <Card>
          <CardHeader>
            <button
              className="flex w-full items-center gap-2 text-left"
              onClick={() => setShowTemplates(!showTemplates)}
            >
              {showTemplates ? <ChevronDownIcon className="h-4 w-4" /> : <ChevronRightIcon className="h-4 w-4" />}
              <CardTitle className="text-base">Message Templates</CardTitle>
              <span className="text-sm text-muted-foreground">({data.body_templates.length} patterns from {data.sample_count} messages)</span>
            </button>
          </CardHeader>
          {showTemplates && (
            <CardContent className="flex flex-col gap-2">
              {data.body_templates.slice(0, 10).map((tmpl, idx) => (
                <div key={idx} className="rounded-md border p-3">
                  <div className="flex items-start justify-between gap-2 mb-2">
                    <code className="text-xs break-all flex-1">{tmpl.template}</code>
                    <div className="text-right shrink-0">
                      <p className="font-semibold">{tmpl.count.toLocaleString()}</p>
                      <p className="text-xs text-muted-foreground">{tmpl.percentage?.toFixed(1)}%</p>
                    </div>
                  </div>
                  <div className="h-1 overflow-hidden rounded-full bg-muted">
                    <div className="h-full bg-primary transition-all" style={{ width: `${tmpl.percentage}%` }} />
                  </div>
                  {tmpl.example && (
                    <p className="mt-2 text-xs text-muted-foreground truncate">
                      Example: &quot;{tmpl.example.length > 80 ? tmpl.example.substring(0, 80) + '…' : tmpl.example}&quot;
                    </p>
                  )}
                </div>
              ))}
              {data.body_templates.length > 10 && (
                <p className="text-center text-xs text-muted-foreground">Showing top 10 of {data.body_templates.length} templates</p>
              )}
            </CardContent>
          )}
        </Card>
      )}

      {/* Label / Attribute keys */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{type === 'metrics' ? 'Labels' : 'Attributes'}</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Key</TableHead>
                <TableHead>Cardinality</TableHead>
                <TableHead>Usage</TableHead>
                <TableHead>Sample Values</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {Object.entries(keys || {}).map(([key, metadata]) => (
                <TableRow key={key} className={metadata.has_invalid_utf8 ? 'bg-destructive/5' : ''}>
                  <TableCell>
                    <code className={metadata.has_invalid_utf8 ? 'text-destructive text-xs' : 'text-xs'}>{key}</code>
                    {metadata.has_invalid_utf8 && (
                      <span className="ml-1 text-destructive" title="Invalid UTF-8 bytes observed">⚠</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge variant={getCardinalityVariant(metadata.estimated_cardinality)}>
                      {metadata.estimated_cardinality}
                    </Badge>
                  </TableCell>
                  <TableCell>{metadata.percentage?.toFixed(1)}%</TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {metadata.value_samples?.slice(0, 5).join(', ')}
                    {metadata.has_invalid_utf8 && !watchedKeys[key] && (
                      <Button
                        variant="outline"
                        size="sm"
                        className="ml-2 h-5 text-xs text-destructive border-destructive"
                        onClick={() => handleWatch(key)}
                        disabled={watchLoading[key]}
                      >
                        {watchLoading[key] ? '…' : 'Watch'}
                      </Button>
                    )}
                    {watchedKeys[key] && (
                      <span className="ml-2 text-xs text-green-600">Watching</span>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Resource Attributes */}
      {data.resource_keys && Object.keys(data.resource_keys).length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Resource Attributes</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Key</TableHead>
                  <TableHead>Cardinality</TableHead>
                  <TableHead>Sample Values</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {Object.entries(data.resource_keys).map(([key, metadata]) => (
                  <TableRow key={key}>
                    <TableCell><code className="text-xs">{key}</code></TableCell>
                    <TableCell>
                      <Badge variant={getCardinalityVariant(metadata.estimated_cardinality)}>
                        {metadata.estimated_cardinality}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {metadata.value_samples?.slice(0, 5).join(', ')}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {/* Services */}
      {data.services && Object.keys(data.services).length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Services</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {Object.entries(data.services).map(([service, count]) => (
                <Badge key={service} variant="outline">
                  {service}: {count.toLocaleString()} samples
                </Badge>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

export default Details
