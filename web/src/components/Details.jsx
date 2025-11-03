import { useState, useEffect } from 'react'

function Details({ type, name, onBack }) {
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [showTemplates, setShowTemplates] = useState(true)

  useEffect(() => {
    const endpoint = type === 'metrics' ? `/api/v1/metrics/${encodeURIComponent(name)}` :
                     type === 'spans' ? `/api/v1/spans/${encodeURIComponent(name)}` :
                     `/api/v1/logs/${encodeURIComponent(name)}`

    fetch(endpoint)
      .then(r => r.json())
      .then(data => {
        setData(data)
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [type, name])

  if (loading) return <div className="loading">Loading...</div>
  if (error) return <div className="error">Error: {error}</div>

  const keys = type === 'metrics' ? data.label_keys : data.attribute_keys

  const getCardinalityBadge = (card) => {
    if (card > 200) return 'high'
    if (card > 50) return 'medium'
    return 'low'
  }

  return (
    <>
      <button className="back-button" onClick={onBack}>← Back</button>

      <div className="card">
        <h2>{type === 'logs' ? 'Severity' : type.slice(0, -1)}: {name}</h2>
        
        {type === 'metrics' && <p>Type: {data.type}</p>}
        {type === 'spans' && <p>Kind: {data.kind}</p>}
        <p>Samples: {data.sample_count}</p>

        {/* Histogram Bucket Distribution (only for histogram metrics) */}
        {type === 'metrics' && data.type === 'Histogram' && data.data && data.data.explicit_bounds && (
          <>
            <h3 style={{ marginTop: '20px', marginBottom: '12px' }}>📊 Histogram Buckets</h3>
            <div className="histogram-info">
              <p>
                <strong>Total Buckets:</strong> {data.data.explicit_bounds.length + 1} 
                <span style={{ color: 'var(--text-secondary)', marginLeft: '8px' }}>
                  ({data.data.explicit_bounds.length} explicit boundaries + ∞)
                </span>
              </p>
              <p><strong>Aggregation:</strong> {data.data.aggregation_temporality === 1 ? 'Delta' : data.data.aggregation_temporality === 2 ? 'Cumulative' : 'Unknown'}</p>
            </div>
            <div style={{ marginTop: '12px' }}>
              <strong>Bucket Boundaries:</strong>
              <div style={{ 
                display: 'flex', 
                flexWrap: 'wrap', 
                gap: '8px', 
                marginTop: '8px',
                fontFamily: 'monospace',
                fontSize: '0.9em'
              }}>
                <span className="key-badge">(-∞, {data.data.explicit_bounds[0]}]</span>
                {data.data.explicit_bounds.map((bound, idx) => (
                  <span key={idx} className="key-badge">
                    ({bound}, {data.data.explicit_bounds[idx + 1] || '∞'}]
                  </span>
                ))}
              </div>
            </div>
          </>
        )}

        {/* Body Templates Section (only for logs) */}
        {type === 'logs' && data.body_templates && data.body_templates.length > 0 && (
          <>
            <div style={{ 
              marginTop: '20px',
              marginBottom: '12px',
              display: 'flex',
              alignItems: 'center',
              gap: '10px',
              cursor: 'pointer',
              userSelect: 'none'
            }} onClick={() => setShowTemplates(!showTemplates)}>
              <h3 style={{ margin: 0 }}>📋 Message Templates</h3>
              <span className="template-count-text">
                ({data.body_templates.length} patterns from {data.sample_count} messages)
              </span>
              <span style={{ fontSize: '20px', marginLeft: 'auto' }}>
                {showTemplates ? '▼' : '▶'}
              </span>
            </div>

            {showTemplates && (
              <div className="template-container">
                {data.body_templates.slice(0, 10).map((tmpl, idx) => (
                  <div key={idx} className="template-card">
                    <div style={{ 
                      display: 'flex', 
                      justifyContent: 'space-between',
                      alignItems: 'start',
                      marginBottom: '8px'
                    }}>
                      <div style={{ flex: 1 }}>
                        <div className="template-label">
                          Template #{idx + 1}
                        </div>
                        <code className="template-code">
                          {tmpl.template}
                        </code>
                      </div>
                      <div style={{ 
                        marginLeft: '12px',
                        textAlign: 'right',
                        minWidth: '100px'
                      }}>
                        <div className="template-count">
                          {tmpl.count.toLocaleString()}
                        </div>
                        <div className="template-percentage">
                          {tmpl.percentage.toFixed(1)}%
                        </div>
                      </div>
                    </div>
                    
                    {/* Progress bar */}
                    <div className="template-progress-bg">
                      <div 
                        className="template-progress-bar"
                        style={{ width: `${tmpl.percentage}%` }}
                      ></div>
                    </div>

                    {/* Example */}
                    {tmpl.example && (
                      <div className="template-example">
                        💬 Example: "{tmpl.example.length > 80 ? tmpl.example.substring(0, 80) + '...' : tmpl.example}"
                      </div>
                    )}
                  </div>
                ))}
                
                {data.body_templates.length > 10 && (
                  <div className="template-show-more">
                    Showing top 10 of {data.body_templates.length} templates
                  </div>
                )}
              </div>
            )}
          </>
        )}

        <h3 style={{ marginTop: '20px', marginBottom: '12px' }}>
          {type === 'metrics' ? 'Labels' : 'Attributes'}
        </h3>
        
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
            {Object.entries(keys).map(([key, metadata]) => (
              <tr key={key}>
                <td><code>{key}</code></td>
                <td>
                  <span className={`badge ${getCardinalityBadge(metadata.estimated_cardinality)}`}>
                    {metadata.estimated_cardinality}
                  </span>
                </td>
                <td>{metadata.percentage.toFixed(1)}%</td>
                <td className="samples">{metadata.value_samples.slice(0, 5).join(', ')}</td>
              </tr>
            ))}
          </tbody>
        </table>

        <h3 style={{ marginTop: '20px', marginBottom: '12px' }}>Resource Attributes</h3>
        
        <table>
          <thead>
            <tr>
              <th>Key</th>
              <th>Cardinality</th>
              <th>Sample Values</th>
            </tr>
          </thead>
          <tbody>
            {Object.entries(data.resource_keys).map(([key, metadata]) => (
              <tr key={key}>
                <td><code>{key}</code></td>
                <td>
                  <span className={`badge ${getCardinalityBadge(metadata.estimated_cardinality)}`}>
                    {metadata.estimated_cardinality}
                  </span>
                </td>
                <td className="samples">{metadata.value_samples.slice(0, 5).join(', ')}</td>
              </tr>
            ))}
          </tbody>
        </table>

        {data.services && (
          <>
            <h3 style={{ marginTop: '20px', marginBottom: '12px' }}>Services</h3>
            <div className="key-list">
              {Object.entries(data.services).map(([service, count]) => (
                <span key={service} className="key-badge">
                  {service}: {count} samples
                </span>
              ))}
            </div>
          </>
        )}
      </div>
    </>
  )
}

export default Details
