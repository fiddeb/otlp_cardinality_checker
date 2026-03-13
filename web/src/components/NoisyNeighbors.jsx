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

function NoisyNeighbors() {
  const [serviceVolumes, setServiceVolumes] = useState([])
  const [highCardinalityAttrs, setHighCardinalityAttrs] = useState([])
  const [noisyServices, setNoisyServices] = useState([])
  const [threshold, setThreshold] = useState(30)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    fetchNoisyNeighbors()
  }, [threshold])

  const fetchNoisyNeighbors = async () => {
    setLoading(true)
    setError(null)

    try {
      const [metricsRes, spansRes, logsRes] = await Promise.all([
        fetch('/api/v1/metrics?limit=1000').then(r => r.json()),
        fetch('/api/v1/spans?limit=1000').then(r => r.json()),
        fetch('/api/v1/logs?limit=1000').then(r => r.json()),
      ])

      // 1. Service volumes
      const serviceData = {}

      metricsRes.data?.forEach(metric => {
        metric.services && Object.entries(metric.services).forEach(([service, count]) => {
          if (!serviceData[service]) serviceData[service] = { metrics: 0, traces: 0, logs: 0, types: new Set() }
          serviceData[service].metrics += count
          serviceData[service].types.add('metrics')
        })
      })
      spansRes.data?.forEach(span => {
        span.services && Object.entries(span.services).forEach(([service, count]) => {
          if (!serviceData[service]) serviceData[service] = { metrics: 0, traces: 0, logs: 0, types: new Set() }
          serviceData[service].traces += count
          serviceData[service].types.add('traces')
        })
      })
      logsRes.data?.forEach(log => {
        log.services && Object.entries(log.services).forEach(([service, count]) => {
          if (!serviceData[service]) serviceData[service] = { metrics: 0, traces: 0, logs: 0, types: new Set() }
          serviceData[service].logs += count
          serviceData[service].types.add('logs')
        })
      })

      setServiceVolumes(
        Object.entries(serviceData)
          .map(([service, d]) => ({ service, total: d.metrics + d.traces + d.logs, metrics: d.metrics, traces: d.traces, logs: d.logs, types: Array.from(d.types) }))
          .sort((a, b) => b.total - a.total)
          .slice(0, 10)
      )

      // 2. High cardinality attributes
      const highCardAttrs = []
      metricsRes.data?.forEach(metric => {
        metric.label_keys && Object.entries(metric.label_keys).forEach(([key, stats]) => {
          if (stats.estimated_cardinality > threshold) {
            highCardAttrs.push({ type: 'metric', name: metric.name, attribute: key, cardinality: stats.estimated_cardinality, services: Object.keys(metric.services || {}).join(', ') })
          }
        })
      })
      spansRes.data?.forEach(span => {
        span.attribute_keys && Object.entries(span.attribute_keys).forEach(([key, stats]) => {
          if (stats.estimated_cardinality > threshold) {
            highCardAttrs.push({ type: 'span', name: span.name, attribute: key, cardinality: stats.estimated_cardinality, services: Object.keys(span.services || {}).join(', ') })
          }
        })
      })
      logsRes.data?.forEach(log => {
        log.attribute_keys && Object.entries(log.attribute_keys).forEach(([key, stats]) => {
          if (stats.estimated_cardinality > threshold) {
            highCardAttrs.push({ type: 'log', name: `severity_${log.severity}`, attribute: key, cardinality: stats.estimated_cardinality, services: Object.keys(log.services || {}).join(', ') })
          }
        })
      })
      highCardAttrs.sort((a, b) => b.cardinality - a.cardinality)
      setHighCardinalityAttrs(highCardAttrs.slice(0, 10))

      // 3. Services contributing to high cardinality
      const serviceContributions = {}
      const addContribution = (serviceName, samples, itemName, type) => {
        if (!serviceContributions[serviceName]) serviceContributions[serviceName] = { samples: 0, items: [] }
        serviceContributions[serviceName].samples += samples
        serviceContributions[serviceName].items.push({ name: itemName, type, samples })
      }

      metricsRes.data?.forEach(metric => {
        const hasHighCard = metric.label_keys && Object.values(metric.label_keys).some(s => s.estimated_cardinality > threshold)
        if (hasHighCard && metric.services) Object.entries(metric.services).forEach(([service, count]) => addContribution(service, count, metric.name, 'metric'))
      })
      spansRes.data?.forEach(span => {
        const hasHighCard = span.attribute_keys && Object.values(span.attribute_keys).some(s => s.estimated_cardinality > threshold)
        if (hasHighCard && span.services) Object.entries(span.services).forEach(([service, count]) => addContribution(service, count, span.name, 'span'))
      })
      logsRes.data?.forEach(log => {
        const hasHighCard = log.attribute_keys && Object.values(log.attribute_keys).some(s => s.estimated_cardinality > threshold)
        if (hasHighCard && log.services) Object.entries(log.services).forEach(([service, count]) => addContribution(service, count, `severity_${log.severity}`, 'log'))
      })

      setNoisyServices(
        Object.entries(serviceContributions)
          .map(([service, d]) => ({ service, samples: d.samples, items: d.items }))
          .sort((a, b) => b.samples - a.samples)
          .slice(0, 10)
      )
      setLoading(false)
    } catch (err) {
      setError(err.message)
      setLoading(false)
    }
  }

  if (loading) {
    return (
      <div className="flex flex-col gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Noisy Neighbors</h1>
          <p className="text-muted-foreground">Identify services causing high cardinality or high volume</p>
        </div>
        <Skeleton className="h-[400px] w-full" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Noisy Neighbors</h1>
          <p className="text-muted-foreground">Identify services causing high cardinality or high volume</p>
        </div>
        <Card className="border-destructive">
          <CardContent className="pt-6">
            <p className="text-destructive">Error: {error}</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  const typeVariant = { metric: 'default', span: 'secondary', log: 'outline' }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Noisy Neighbors</h1>
          <p className="text-muted-foreground">Identify services causing high cardinality or high volume</p>
        </div>
        <div className="flex items-center gap-2">
          <label className="text-sm font-medium whitespace-nowrap">Cardinality Threshold:</label>
          <Input
            type="number"
            min="1"
            max="10000"
            value={threshold}
            onChange={(e) => setThreshold(parseInt(e.target.value) || 30)}
            className="w-24"
          />
          <Button variant="outline" size="sm" onClick={fetchNoisyNeighbors}>Refresh</Button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Services by Total Sample Volume</CardTitle>
          <CardDescription>Top 10 services ordered by combined telemetry volume</CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Service</TableHead>
                <TableHead>Total Samples</TableHead>
                <TableHead>Metrics</TableHead>
                <TableHead>Traces</TableHead>
                <TableHead>Logs</TableHead>
                <TableHead>Signal Types</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {serviceVolumes.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="py-8 text-center text-muted-foreground">No services found</TableCell>
                </TableRow>
              ) : serviceVolumes.map(service => (
                <TableRow key={service.service}>
                  <TableCell className="font-medium">{service.service}</TableCell>
                  <TableCell>{service.total.toLocaleString()}</TableCell>
                  <TableCell>{service.metrics.toLocaleString()}</TableCell>
                  <TableCell>{service.traces.toLocaleString()}</TableCell>
                  <TableCell>{service.logs.toLocaleString()}</TableCell>
                  <TableCell>
                    <div className="flex gap-1 flex-wrap">
                      {service.types.map(t => <Badge key={t} variant={typeVariant[t] || 'outline'} className="text-xs">{t}</Badge>)}
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>High Cardinality Attributes (&gt; {threshold})</CardTitle>
          <CardDescription>Attributes with estimated cardinality above threshold</CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          {highCardinalityAttrs.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">No high cardinality attributes found</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Type</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Attribute</TableHead>
                  <TableHead>Cardinality</TableHead>
                  <TableHead>Services</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {highCardinalityAttrs.map((attr, idx) => (
                  <TableRow key={idx}>
                    <TableCell><Badge variant={typeVariant[attr.type] || 'outline'}>{attr.type}</Badge></TableCell>
                    <TableCell className="font-mono text-xs">{attr.name}</TableCell>
                    <TableCell className="font-semibold">{attr.attribute}</TableCell>
                    <TableCell>
                      <Badge variant={attr.cardinality > 100 ? 'destructive' : 'outline'}>
                        {attr.cardinality.toLocaleString()}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">{attr.services}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Services Contributing to High Cardinality</CardTitle>
          <CardDescription>Services associated with high-cardinality signals</CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          {noisyServices.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">No services contributing to high cardinality</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Service</TableHead>
                  <TableHead>Total Samples</TableHead>
                  <TableHead>High-Cardinality Items</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {noisyServices.map(service => (
                  <TableRow key={service.service}>
                    <TableCell className="font-medium">{service.service}</TableCell>
                    <TableCell>{service.samples.toLocaleString()}</TableCell>
                    <TableCell>
                      <div className="flex flex-col gap-1">
                        {service.items.slice(0, 5).map((item, idx) => (
                          <div key={idx} className="flex items-center gap-2">
                            <Badge variant={typeVariant[item.type] || 'outline'} className="text-xs">{item.type}</Badge>
                            <code className="text-xs">{item.name}</code>
                            <span className="text-xs text-muted-foreground">({item.samples.toLocaleString()} samples)</span>
                          </div>
                        ))}
                        {service.items.length > 5 && (
                          <span className="text-xs text-muted-foreground">+ {service.items.length - 5} more</span>
                        )}
                      </div>
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

export default NoisyNeighbors
