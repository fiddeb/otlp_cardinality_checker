import { useState, useEffect } from 'react'

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

  const getSeverityColor = (sev) => {
    const colors = {
      'ERROR': '#ef4444',
      'WARN': '#f59e0b',
      'INFO': '#3b82f6',
      'DEBUG': '#8b5cf6',
      'CRITICAL': '#dc2626',
      'TRACE': '#6b7280'
    }
    return colors[sev] || '#6b7280'
  }

  const formatNumber = (num) => {
    return new Intl.NumberFormat().format(num)
  }

  const truncateExample = (text, maxLength = 300) => {
    if (!text || text.length <= maxLength) return text
    return text.substring(0, maxLength) + '...'
  }

  if (loading) return <div className="loading">Loading pattern details...</div>
  if (error) return <div className="error">Error: {error}</div>
  if (!patternData) return <div className="error">Pattern not found</div>

  return (
    <>
      <button className="back-button" onClick={onBack}>← Back to Logs</button>

      <div className="card">
        <h2>Pattern Details</h2>
        
        <div style={{ marginBottom: '20px' }}>
          <div style={{ 
            display: 'flex', 
            gap: '10px', 
            alignItems: 'center',
            marginBottom: '12px'
          }}>
            <span style={{
              backgroundColor: getSeverityColor(severity),
              color: 'white',
              padding: '4px 12px',
              borderRadius: '12px',
              fontSize: '12px',
              fontWeight: '600',
              textTransform: 'uppercase'
            }}>
              {severity}
            </span>
            <span className="template-count-text">
              {formatNumber(patternData.total_count)} occurrences across {(patternData.services || []).length} service{(patternData.services || []).length !== 1 ? 's' : ''}
            </span>
          </div>

          <div style={{
            background: 'var(--bg-tertiary)',
            padding: '16px',
            borderRadius: '8px',
            marginTop: '12px'
          }}>
            <div style={{ 
              fontSize: '11px',
              color: 'var(--text-secondary)',
              marginBottom: '6px',
              textTransform: 'uppercase',
              fontWeight: '600'
            }}>
              Template Pattern
            </div>
            <code style={{
              display: 'block',
              fontSize: '14px',
              wordBreak: 'break-word',
              lineHeight: '1.5'
            }}>
              {template}
            </code>
          </div>

          {/* Example log */}
          {patternData.example_body && (
            <div style={{
              background: 'var(--bg-tertiary)',
              padding: '16px',
              borderRadius: '8px',
              marginTop: '12px'
            }}>
              <div style={{ 
                fontSize: '11px',
                color: 'var(--text-secondary)',
                marginBottom: '6px',
                textTransform: 'uppercase',
                fontWeight: '600'
              }}>
                Example Log
              </div>
              <pre style={{
                fontSize: '13px',
                wordBreak: 'break-word',
                whiteSpace: 'pre-wrap',
                margin: 0
              }}>
                {truncateExample(patternData.example_body)}
              </pre>
            </div>
          )}
        </div>

        {/* Per-service breakdown */}
        <h3 style={{ marginTop: '24px', marginBottom: '16px' }}>
          Services Using This Pattern
        </h3>
        
        {patternData.services.map((service, idx) => {
          return (
            <div key={idx} style={{
              background: 'var(--bg-tertiary)',
              padding: '20px',
              borderRadius: '8px',
              marginBottom: '16px'
            }}>
              <div style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                marginBottom: '16px'
              }}>
                <h4 style={{ margin: 0, display: 'flex', alignItems: 'center', gap: '8px' }}>
                  🔧 {service.service_name || 'unknown'}
                </h4>
                <div style={{
                  display: 'flex',
                  gap: '12px',
                  alignItems: 'center'
                }}>
                  <div style={{
                    fontSize: '14px',
                    color: 'var(--text-secondary)'
                  }}>
                    {formatNumber(service.sample_count)} samples
                  </div>
                  {service.severities && service.severities.length > 0 && (
                    <div style={{ display: 'flex', gap: '4px' }}>
                      {service.severities.map((sev, i) => (
                        <span
                          key={i}
                          style={{
                            width: '12px',
                            height: '12px',
                            borderRadius: '50%',
                            backgroundColor: getSeverityColor(sev),
                            display: 'inline-block'
                          }}
                          title={sev}
                        />
                      ))}
                    </div>
                  )}
                </div>
              </div>

              {/* Resource keys */}
              {service.resource_keys && service.resource_keys.length > 0 && (
                <div style={{ marginBottom: '16px' }}>
                  <div style={{
                    fontSize: '12px',
                    fontWeight: '600',
                    color: 'var(--text-secondary)',
                    marginBottom: '12px',
                    textTransform: 'uppercase'
                  }}>
                    Resource Keys ({service.resource_keys.length})
                  </div>
                  <div className="keys-grid">
                    {service.resource_keys.map((key, keyIndex) => (
                      <div key={keyIndex} className="key-item">
                        <div className="key-header">
                          <span className="key-name">{key.name}</span>
                          <span 
                            className={`cardinality ${key.cardinality > 100 ? 'high' : key.cardinality > 10 ? 'medium' : 'low'}`}
                          >
                            ~{key.cardinality}
                          </span>
                        </div>
                        <div className="sample-values">
                          {key.sample_values && key.sample_values.slice(0, 3).map((val, i) => (
                            <span key={i} className="sample-value">
                              {val}
                            </span>
                          ))}
                          {key.sample_values && key.sample_values.length > 3 && (
                            <span className="more-values">
                              +{key.sample_values.length - 3} more
                            </span>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Attribute keys */}
              {service.attribute_keys && service.attribute_keys.length > 0 && (
                <div>
                  <div style={{
                    fontSize: '12px',
                    fontWeight: '600',
                    color: 'var(--text-secondary)',
                    marginBottom: '12px',
                    textTransform: 'uppercase'
                  }}>
                    Attribute Keys ({service.attribute_keys.length})
                  </div>
                  <div className="keys-grid">
                    {service.attribute_keys.map((key, keyIndex) => (
                      <div key={keyIndex} className="key-item">
                        <div className="key-header">
                          <span className="key-name">{key.name}</span>
                          <span 
                            className={`cardinality ${key.cardinality > 100 ? 'high' : key.cardinality > 10 ? 'medium' : 'low'}`}
                          >
                            ~{key.cardinality}
                          </span>
                        </div>
                        <div className="sample-values">
                          {key.sample_values && key.sample_values.slice(0, 3).map((val, i) => (
                            <span key={i} className="sample-value">
                              {val}
                            </span>
                          ))}
                          {key.sample_values && key.sample_values.length > 3 && (
                            <span className="more-values">
                              +{key.sample_values.length - 3} more
                            </span>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )
        })}

        {patternData.services.length === 0 && (
          <div style={{
            padding: '40px',
            textAlign: 'center',
            color: 'var(--text-secondary)',
            background: 'var(--bg-tertiary)',
            borderRadius: '8px'
          }}>
            No services found for this pattern with severity {severity}
          </div>
        )}
      </div>
    </>
  )
}

export default TemplateDetails
