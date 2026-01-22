import { useState, useEffect } from 'react'

function Details({ type, name, onBack }) {
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [showTemplates, setShowTemplates] = useState(true)
  const [showSeriesExplanation, setShowSeriesExplanation] = useState(false)

  useEffect(() => {
    console.log('Details useEffect - type:', type, 'name:', name)
    const endpoint = type === 'metrics' || type === 'metric' ? `/api/v1/metrics/${encodeURIComponent(name)}` :
                     type === 'spans' || type === 'span' ? `/api/v1/spans/${encodeURIComponent(name)}` :
                     type === 'logs' || type === 'log' ? `/api/v1/logs/${encodeURIComponent(name)}` :
                     `/api/v1/logs/${encodeURIComponent(name)}`  // fallback

    console.log('Fetching from endpoint:', endpoint)
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

        {/* Active Series (only for metrics) */}
        {type === 'metrics' && data.active_series !== undefined && (
          <>
            <div style={{ 
              marginTop: '16px',
              padding: '12px',
              backgroundColor: 'var(--bg-secondary)',
              borderRadius: '6px',
              border: '1px solid var(--border-color)'
            }}>
              <div style={{ 
                display: 'flex', 
                alignItems: 'center',
                gap: '8px',
                marginBottom: '4px'
              }}>
                <strong>📈 Active Series:</strong>
                <span style={{ 
                  fontSize: '1.2em',
                  fontWeight: 'bold',
                  color: data.active_series > 1000 ? 'var(--danger)' : 
                         data.active_series > 100 ? 'var(--warning)' : 
                         'var(--success)'
                }}>
                  {data.active_series.toLocaleString()}
                </span>
              </div>
              <div style={{ 
                fontSize: '0.9em',
                color: 'var(--text-secondary)',
                marginTop: '4px'
              }}>
                <button 
                  onClick={() => setShowSeriesExplanation(!showSeriesExplanation)}
                  style={{
                    marginLeft: '8px',
                    padding: '2px 8px',
                    fontSize: '0.85em',
                    border: '1px solid var(--border-color)',
                    borderRadius: '4px',
                    backgroundColor: 'var(--bg-primary)',
                    cursor: 'pointer',
                    color: 'var(--link-color)'
                  }}
                >
                  {showSeriesExplanation ? 'Hide' : 'How is this calculated?'}
                </button>
              </div>

              {showSeriesExplanation && (
                <div style={{
                  marginTop: '12px',
                  padding: '12px',
                  backgroundColor: 'var(--bg-primary)',
                  borderRadius: '4px',
                  fontSize: '0.85em',
                  lineHeight: '1.6'
                }}>
                  <div style={{ marginBottom: '8px' }}>
                    <strong>Vad är en aktiv serie?</strong>
                  </div>
                  <div style={{ marginBottom: '8px' }}>
                    En aktiv serie är en unik kombination av alla label-värden för denna metric.
                    Systemet spårar faktiska kombinationer som observerats, inte teoretiska möjligheter.
                  </div>
                  <div style={{ 
                    fontFamily: 'monospace',
                    padding: '8px',
                    backgroundColor: 'rgba(0, 0, 0, 0.2)',
                    borderRadius: '4px',
                    marginBottom: '8px'
                  }}>
                    {Object.keys(data.label_keys).length > 0 ? (
                      <>
                        <div><strong>Observerade labels:</strong></div>
                        {Object.entries(data.label_keys)
                          .slice(0, 5)
                          .map(([key, meta]) => (
                            <div key={key}>
                              • {key}: {meta.estimated_cardinality} unika värden
                            </div>
                          ))}
                        {Object.keys(data.label_keys).length > 5 && (
                          <div>... och {Object.keys(data.label_keys).length - 5} fler labels</div>
                        )}
                        <div style={{ marginTop: '8px', borderTop: '1px solid var(--border-color)', paddingTop: '8px' }}>
                          <strong>{data.active_series.toLocaleString()}</strong> unika kombinationer observerade
                        </div>
                      </>
                    ) : (
                      <div>Inga labels → 1 konstant serie</div>
                    )}
                  </div>
                  <div>
                    <strong>Exempel:</strong> Om metric har labels method=GET,status=200 och method=POST,status=404 
                    = 2 unika kombinationer = 2 aktiva serier.
                  </div>
                </div>
              )}

              {data.active_series > 1000 && (
                <div style={{
                  marginTop: '8px',
                  padding: '8px',
                  backgroundColor: 'rgba(220, 38, 38, 0.1)',
                  borderRadius: '4px',
                  fontSize: '0.85em',
                  color: 'var(--danger)'
                }}>
                  ⚠️ High cardinality detected! This metric generates many unique series which may impact storage and query performance.
                </div>
              )}
            </div>
          </>
        )}

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
