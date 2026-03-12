import { useState, useEffect } from 'react'

function LogPatternDetails({ serviceName, severity, template, onBack }) {
  const [attributes, setAttributes] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    setLoading(true)
    setError(null)
    
    // Fetch pattern-specific data (not service+severity aggregated data)
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
        // Find the specific service in the pattern data
        const serviceData = data.services?.find(s => s.service_name === serviceName)
        
        if (!serviceData) {
          throw new Error('Service not found for this pattern')
        }
        
        // Convert KeyInfo array to map with metadata for table rendering
        const resourceKeysMap = {}
        if (serviceData.resource_keys) {
          serviceData.resource_keys.forEach(key => {
            resourceKeysMap[key.name] = {
              count: serviceData.sample_count, // Use service sample count
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
            percentage: 100 // Pattern-specific view, always 100% for this service
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

  const getCardinalityBadge = (cardinality) => {
    if (cardinality === 1) return 'low'
    if (cardinality <= 10) return 'medium'
    if (cardinality <= 100) return 'high'
    return 'very-high'
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
        <h3>Attributes for This Pattern (Service: {serviceName})</h3>
        
        {Object.keys(attributes.resource_keys).length > 0 && (
          <>
            <h4 style={{ marginTop: '20px', marginBottom: '12px', color: 'var(--text-secondary)' }}>
              Resource Attributes
            </h4>
            <table>
              <thead>
                <tr>
                  <th>Key</th>
                  <th>Cardinality</th>
                  <th>Usage</th>
                  <th>Sample Values</th>
                </tr>
              </thead>
              <tbody>
                {Object.entries(attributes.resource_keys).map(([key, metadata]) => {
                  const percentage = attributes.template.count > 0 
                    ? (metadata.count / attributes.template.count * 100) 
                    : 100
                  return (
                    <tr key={key}>
                      <td><code>{key}</code></td>
                      <td>
                        <span className={`badge ${getCardinalityBadge(metadata.estimated_cardinality)}`}>
                          {metadata.estimated_cardinality}
                        </span>
                      </td>
                      <td>{percentage.toFixed(1)}%</td>
                      <td className="samples">
                        {metadata.value_samples && metadata.value_samples.length > 0 
                          ? metadata.value_samples.slice(0, 5).join(', ')
                          : '—'}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </>
        )}

        {Object.keys(attributes.body_keys).length > 0 && (
          <>
            <h4 style={{ marginTop: '20px', marginBottom: '12px', color: 'var(--text-secondary)' }}>
              Body Attributes
            </h4>
            <table>
              <thead>
                <tr>
                  <th>Key</th>
                  <th>Cardinality</th>
                  <th>Usage</th>
                  <th>Sample Values</th>
                </tr>
              </thead>
              <tbody>
                {Object.entries(attributes.body_keys).map(([key, metadata]) => {
                  const percentage = attributes.template.count > 0 
                    ? (metadata.count / attributes.template.count * 100) 
                    : 100
                  return (
                    <tr key={key}>
                      <td><code>{key}</code></td>
                      <td>
                        <span className={`badge ${getCardinalityBadge(metadata.estimated_cardinality)}`}>
                          {metadata.estimated_cardinality}
                        </span>
                      </td>
                      <td>{percentage.toFixed(1)}%</td>
                      <td className="samples">
                        {metadata.value_samples && metadata.value_samples.length > 0 
                          ? metadata.value_samples.slice(0, 5).join(', ')
                          : '—'}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </>
        )}

        {Object.keys(attributes.resource_keys).length === 0 && Object.keys(attributes.body_keys).length === 0 && (
          <p className="template-count-text">No attribute information available</p>
        )}
      </div>
    </div>
  )
}

export default LogPatternDetails
