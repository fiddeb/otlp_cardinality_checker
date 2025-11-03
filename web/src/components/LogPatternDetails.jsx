import { useState, useEffect } from 'react'

function LogPatternDetails({ serviceName, severity, template, onBack }) {
  const [attributes, setAttributes] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    setLoading(true)
    setError(null)
    
    // Fetch template data with pattern-specific attributes
    fetch(`/api/v1/logs/service/${encodeURIComponent(serviceName)}/severity/${encodeURIComponent(severity)}`)
      .then(res => {
        if (!res.ok) {
          throw new Error(`HTTP error`)
        }
        return res.json()
      })
      .then(data => {
        // Find the specific template
        const templateData = data.body_templates?.find(t => t.template === template)
        
        if (!templateData) {
          throw new Error('Template not found')
        }
        
        // Get attribute and resource keys from response (they are at top level, not per template)
        const attributeKeys = Object.keys(data.attribute_keys || {})
        const resourceKeys = Object.keys(data.resource_keys || {})
        
        setAttributes({
          template: templateData,
          resource_keys: resourceKeys,
          body_keys: attributeKeys
        })
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [serviceName, severity, template])

  const getSeverityColor = (sev) => {
    const colors = {
      'ERROR': '#d32f2f',
      'Error': '#d32f2f',
      'WARN': '#f57c00',
      'Warning': '#f57c00',
      'INFO': '#1976d2',
      'Information': '#1976d2',
      'DEBUG': '#7b1fa2',
      'DEBUG2': '#7b1fa2',
      'Debug': '#7b1fa2',
      'TRACE': '#455a64',
      'Trace': '#455a64',
      'UNSET': '#999'
    }
    return colors[sev] || '#666'
  }

  if (loading) return <div className="loading">Loading pattern details...</div>
  if (error) return <div className="error">Error loading pattern: {error}</div>
  if (!attributes) return <div className="error">Pattern not found</div>

  return (
    <div className="card">
      <button 
        onClick={onBack}
        style={{
          padding: '8px 16px',
          marginBottom: '20px',
          cursor: 'pointer'
        }}
      >
        ← Back to Patterns
      </button>

      <div style={{ marginBottom: '30px' }}>
        <h2>Pattern Details</h2>
        <div style={{ marginTop: '15px' }}>
          <p><strong>Service:</strong> <code>{serviceName}</code></p>
          <p>
            <strong>Severity:</strong>{' '}
            <span style={{ color: getSeverityColor(severity), fontWeight: 'bold' }}>
              {severity}
            </span>
          </p>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '15px', marginBottom: '30px' }}>
        <div className="stat-card">
          <div className="stat-label">Occurrences</div>
          <div className="stat-value">{attributes.template.count.toLocaleString()}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Percentage</div>
          <div className="stat-value">{attributes.template.percentage.toFixed(2)}%</div>
        </div>
      </div>

      <div style={{ marginBottom: '30px' }}>
        <h3>Pattern Template</h3>
        <pre style={{ 
          background: 'var(--bg-tertiary)',
          padding: '15px',
          borderRadius: '6px',
          overflow: 'auto',
          fontSize: '13px',
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word'
        }}>
          {template}
        </pre>
      </div>

      <div style={{ marginBottom: '30px' }}>
        <h3>Example Log Message</h3>
        <pre style={{ 
          background: 'var(--bg-tertiary)',
          padding: '15px',
          borderRadius: '6px',
          overflow: 'auto',
          fontSize: '13px',
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word'
        }}>
          {attributes.template.example}
        </pre>
      </div>

      <div style={{ marginBottom: '30px' }}>
        <h3>Attributes for Service: {serviceName}</h3>
        
        {attributes.resource_keys && attributes.resource_keys.length > 0 && (
          <div style={{ marginBottom: '20px' }}>
            <h4 style={{ marginBottom: '10px', color: 'var(--text-secondary)' }}>
              Resource Attributes ({attributes.resource_keys.length})
            </h4>
            <div style={{ 
              display: 'flex', 
              flexWrap: 'wrap', 
              gap: '8px'
            }}>
              {attributes.resource_keys.map((key, i) => (
                <span 
                  key={i}
                  style={{
                    background: 'var(--primary-color)',
                    color: 'white',
                    padding: '4px 10px',
                    borderRadius: '4px',
                    fontSize: '13px',
                    fontFamily: 'monospace'
                  }}
                >
                  {key}
                </span>
              ))}
            </div>
          </div>
        )}

        {attributes.body_keys && attributes.body_keys.length > 0 && (
          <div style={{ marginBottom: '20px' }}>
            <h4 style={{ marginBottom: '10px', color: 'var(--text-secondary)' }}>
              Body Attributes ({attributes.body_keys.length})
            </h4>
            <div style={{ 
              display: 'flex', 
              flexWrap: 'wrap', 
              gap: '8px'
            }}>
              {attributes.body_keys.map((key, i) => (
                <span 
                  key={i}
                  style={{
                    background: '#2e7d32',
                    color: 'white',
                    padding: '4px 10px',
                    borderRadius: '4px',
                    fontSize: '13px',
                    fontFamily: 'monospace'
                  }}
                >
                  {key}
                </span>
              ))}
            </div>
          </div>
        )}

        {(!attributes.resource_keys?.length && !attributes.body_keys?.length) && (
          <p className="template-count-text">No attribute information available</p>
        )}
      </div>
    </div>
  )
}

export default LogPatternDetails
