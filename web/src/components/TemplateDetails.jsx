import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { ArrowLeftIcon } from 'lucide-react'

function TemplateDetails({ severity, template, onBack }) {
  const [patternData, setPatternData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    const encodedSeverity = encodeURIComponent(severity)
    const encodedTemplate = encodeURIComponent(template)

    fetch(`/api/v1/logs/patterns/${encodedSeverity}/${encodedTemplate}`)
      .then(r => {
        if (!r.ok) throw new Error(`HTTP ${r.status}: ${r.statusText}`)
        return r.json()
      })
      .then(data => {
        setPatternData(data)
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [template, severity])

  const getSeverityVariant = (sev) => {
    const variants = { 'ERROR': 'destructive', 'CRITICAL': 'destructive', 'WARN': 'outline', 'INFO': 'secondary', 'DEBUG': 'outline', 'TRACE': 'outline' }
    return variants[sev] || 'outline'
  }

  const truncateExample = (text, maxLength = 300) => {
    if (!text || text.length <= maxLength) return text
    return text.substring(0, maxLength) + '...'
  }

  const getCardinalityVariant = (cardinality) => {
    if (cardinality > 100) return 'destructive'
    if (cardinality > 10) return 'outline'
    return 'secondary'
  }

  if (loading) {
    return (
      <div className="flex flex-col gap-6">
        <Button variant="ghost" size="sm" className="w-fit" onClick={onBack}>
          <ArrowLeftIcon className="h-4 w-4" /> Back
        </Button>
        <Skeleton className="h-[400px] w-full" />
      </div>
    )
  }

  if (error || !patternData) {
    return (
      <div className="flex flex-col gap-6">
        <Button variant="ghost" size="sm" className="w-fit" onClick={onBack}>
          <ArrowLeftIcon className="h-4 w-4" /> Back
        </Button>
        <Card className="border-destructive">
          <CardContent className="pt-6">
            <p className="text-destructive">{error ? `Error: ${error}` : 'Pattern not found'}</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <Button variant="ghost" size="sm" className="w-fit" onClick={onBack}>
        <ArrowLeftIcon className="h-4 w-4" /> Back to Logs
      </Button>

      <div>
        <h1 className="text-2xl font-bold tracking-tight">Pattern Details</h1>
        <p className="text-muted-foreground">
          {(patternData.total_count || 0).toLocaleString()} occurrences across {(patternData.services || []).length} service{(patternData.services || []).length !== 1 ? 's' : ''}
        </p>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Badge variant={getSeverityVariant(severity)}>{severity}</Badge>
            <CardTitle className="text-base">Pattern</CardTitle>
          </div>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <div className="rounded-md bg-muted p-3">
            <p className="text-xs font-semibold uppercase text-muted-foreground mb-2">Template Pattern</p>
            <code className="text-sm break-words leading-relaxed">{template}</code>
          </div>
          {patternData.example_body && (
            <div className="rounded-md bg-muted p-3">
              <p className="text-xs font-semibold uppercase text-muted-foreground mb-2">Example Log</p>
              <pre className="text-sm whitespace-pre-wrap break-words">{truncateExample(patternData.example_body)}</pre>
            </div>
          )}
        </CardContent>
      </Card>

      <h2 className="text-lg font-semibold">Services Using This Pattern</h2>

      {(patternData.services || []).length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center">
            <p className="text-muted-foreground">No services found for this pattern with severity {severity}</p>
          </CardContent>
        </Card>
      ) : (
        (patternData.services || []).map((service, idx) => (
          <Card key={idx}>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-base">{service.service_name || 'unknown'}</CardTitle>
                <div className="flex items-center gap-3">
                  <span className="text-sm text-muted-foreground">{(service.sample_count || 0).toLocaleString()} samples</span>
                  {service.severities && service.severities.length > 0 && (
                    <div className="flex gap-1">
                      {service.severities.map((sev, i) => (
                        <Badge key={i} variant={getSeverityVariant(sev)} className="text-xs">{sev}</Badge>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </CardHeader>
            <CardContent className="flex flex-col gap-6">
              {service.resource_keys && service.resource_keys.length > 0 && (
                <div>
                  <p className="text-xs font-semibold uppercase text-muted-foreground mb-2">Resource Keys ({service.resource_keys.length})</p>
                  <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                    {service.resource_keys.map((key, i) => (
                      <div key={i} className="rounded-md border p-2">
                        <div className="flex items-center justify-between mb-1">
                          <span className="text-xs font-medium">{key.name}</span>
                          <Badge variant={getCardinalityVariant(key.cardinality)} className="text-xs">~{key.cardinality}</Badge>
                        </div>
                        <div className="flex flex-wrap gap-1">
                          {(key.sample_values || []).slice(0, 3).map((val, j) => (
                            <span key={j} className="rounded bg-muted px-1 text-xs text-muted-foreground">{val}</span>
                          ))}
                          {(key.sample_values || []).length > 3 && (
                            <span className="text-xs text-muted-foreground">+{key.sample_values.length - 3} more</span>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
              {service.attribute_keys && service.attribute_keys.length > 0 && (
                <div>
                  <p className="text-xs font-semibold uppercase text-muted-foreground mb-2">Attribute Keys ({service.attribute_keys.length})</p>
                  <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                    {service.attribute_keys.map((key, i) => (
                      <div key={i} className="rounded-md border p-2">
                        <div className="flex items-center justify-between mb-1">
                          <span className="text-xs font-medium">{key.name}</span>
                          <Badge variant={getCardinalityVariant(key.cardinality)} className="text-xs">~{key.cardinality}</Badge>
                        </div>
                        <div className="flex flex-wrap gap-1">
                          {(key.sample_values || []).slice(0, 3).map((val, j) => (
                            <span key={j} className="rounded bg-muted px-1 text-xs text-muted-foreground">{val}</span>
                          ))}
                          {(key.sample_values || []).length > 3 && (
                            <span className="text-xs text-muted-foreground">+{key.sample_values.length - 3} more</span>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        ))
      )}
    </div>
  )
}

export default TemplateDetails
