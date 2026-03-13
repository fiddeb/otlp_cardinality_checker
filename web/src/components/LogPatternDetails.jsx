import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ArrowLeftIcon } from 'lucide-react'

function LogPatternDetails({ serviceName, severity, template, onBack }) {
  const [attributes, setAttributes] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    setLoading(true)
    setError(null)
    
    const encodedSeverity = encodeURIComponent(severity)
    const encodedTemplate = encodeURIComponent(template)
    
    fetch(`/api/v1/logs/patterns/${encodedSeverity}/${encodedTemplate}`)
      .then(res => {
        if (!res.ok) {
          throw new Error(`HTTP error`)
        }
        return res.json()
      })
      .then(data => {
        const serviceData = data.services?.find(s => s.service_name === serviceName)
        
        if (!serviceData) {
          throw new Error('Service not found for this pattern')
        }
        
        const resourceKeysMap = {}
        if (serviceData.resource_keys) {
          serviceData.resource_keys.forEach(key => {
            resourceKeysMap[key.name] = {
              count: serviceData.sample_count,
              estimated_cardinality: key.cardinality,
              value_samples: key.sample_values || []
            }
          })
        }
        
        const attributeKeysMap = {}
        if (serviceData.attribute_keys) {
          serviceData.attribute_keys.forEach(key => {
            attributeKeysMap[key.name] = {
              count: serviceData.sample_count,
              estimated_cardinality: key.cardinality,
              value_samples: key.sample_values || []
            }
          })
        }
        
        setAttributes({
          template: {
            template: data.template,
            example: data.example_body,
            count: serviceData.sample_count,
            percentage: 100
          },
          resource_keys: resourceKeysMap,
          body_keys: attributeKeysMap
        })
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [serviceName, severity, template])

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

  const getCardinalityVariant = (cardinality) => {
    if (cardinality === 1) return 'secondary'
    if (cardinality <= 10) return 'outline'
    if (cardinality <= 100) return 'outline'
    return 'destructive'
  }

  const getCardinalityLabel = (cardinality) => {
    if (cardinality === 1) return 'low'
    if (cardinality <= 10) return 'medium'
    if (cardinality <= 100) return 'high'
    return 'very-high'
  }

  if (loading) {
    return (
      <div className="flex flex-col gap-6">
        <Skeleton className="h-8 w-32" />
        <Skeleton className="h-[500px] w-full" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col gap-6">
        <Button variant="ghost" size="sm" className="w-fit" onClick={onBack}>
          <ArrowLeftIcon className="mr-2 h-4 w-4" />
          Back to Patterns
        </Button>
        <Card className="border-destructive">
          <CardContent className="pt-6">
            <p className="text-destructive">Error loading pattern: {error}</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!attributes) {
    return (
      <div className="flex flex-col gap-6">
        <Button variant="ghost" size="sm" className="w-fit" onClick={onBack}>
          <ArrowLeftIcon className="mr-2 h-4 w-4" />
          Back to Patterns
        </Button>
        <Card className="border-destructive">
          <CardContent className="pt-6">
            <p className="text-destructive">Pattern not found</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  const AttributesTable = ({ keysMap, title }) => {
    if (Object.keys(keysMap).length === 0) return null
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium text-muted-foreground">{title}</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Key</TableHead>
                <TableHead>Cardinality</TableHead>
                <TableHead className="text-right">Usage</TableHead>
                <TableHead>Sample Values</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {Object.entries(keysMap).map(([key, metadata]) => {
                const percentage = attributes.template.count > 0
                  ? (metadata.count / attributes.template.count * 100)
                  : 100
                return (
                  <TableRow key={key}>
                    <TableCell><code className="font-mono text-sm">{key}</code></TableCell>
                    <TableCell>
                      <Badge variant={getCardinalityVariant(metadata.estimated_cardinality)}>
                        {getCardinalityLabel(metadata.estimated_cardinality)} ({metadata.estimated_cardinality})
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">{percentage.toFixed(1)}%</TableCell>
                    <TableCell className="text-muted-foreground text-xs max-w-xs truncate">
                      {metadata.value_samples && metadata.value_samples.length > 0
                        ? metadata.value_samples.slice(0, 5).join(', ')
                        : '—'}
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <Button variant="ghost" size="sm" className="w-fit" onClick={onBack}>
        <ArrowLeftIcon className="mr-2 h-4 w-4" />
        Back to Patterns
      </Button>

      <div>
        <h1 className="text-2xl font-bold tracking-tight">Pattern Details</h1>
        <div className="flex items-center gap-2 mt-1 text-sm text-muted-foreground">
          <span>Service:</span>
          <code className="font-mono text-foreground">{serviceName}</code>
          <span>·</span>
          <span>Severity:</span>
          <Badge variant={getSeverityVariant(severity)}>{severity}</Badge>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium text-muted-foreground">Occurrences</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{attributes.template.count.toLocaleString()}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium text-muted-foreground">Percentage</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{attributes.template.percentage.toFixed(2)}%</p>
          </CardContent>
        </Card>
      </div>

      {/* Pattern template */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium text-muted-foreground">Pattern Template</CardTitle>
        </CardHeader>
        <CardContent>
          <pre className="rounded bg-muted p-4 text-sm font-mono whitespace-pre-wrap break-words overflow-auto">
            {template}
          </pre>
        </CardContent>
      </Card>

      {/* Example log */}
      {attributes.template.example && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium text-muted-foreground">Example Log Message</CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="rounded bg-muted p-4 text-sm font-mono whitespace-pre-wrap break-words overflow-auto">
              {attributes.template.example}
            </pre>
          </CardContent>
        </Card>
      )}

      {/* Attribute tables */}
      <AttributesTable keysMap={attributes.resource_keys} title="Resource Attributes" />
      <AttributesTable keysMap={attributes.body_keys} title="Body Attributes" />

      {Object.keys(attributes.resource_keys).length === 0 && Object.keys(attributes.body_keys).length === 0 && (
        <p className="text-sm text-muted-foreground text-center py-4">No attribute information available</p>
      )}
    </div>
  )
}

export default LogPatternDetails
