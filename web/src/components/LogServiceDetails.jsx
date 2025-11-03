import { useState, useEffect } from 'react'

function LogServiceDetails({ serviceName, severity, onBack, onViewPattern }) {
  const [templates, setTemplates] = useState([])
  const [attributeKeys, setAttributeKeys] = useState({})
  const [resourceKeys, setResourceKeys] = useState({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [filter, setFilter] = useState({
    minCount: 0,
  })

  useEffect(() => {
    setLoading(true)
    setError(null)
    
    fetch(`/api/v1/logs/service/${encodeURIComponent(serviceName)}/severity/${encodeURIComponent(severity)}`)
      .then(r => {
        if (!r.ok) {
          throw new Error(`HTTP ${r.status}: ${r.statusText}`)
        }
        return r.json()
      })
      .then(data => {
        setTemplates(data.body_templates || [])
        setAttributeKeys(data.attribute_keys || {})
        setResourceKeys(data.resource_keys || {})
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [serviceName, severity])

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

  const filteredTemplates = templates.filter(t => t.count >= filter.minCount)
  const totalMessages = filteredTemplates.reduce((sum, t) => sum + t.count, 0)

  if (loading) return <div className="loading">Loading patterns...</div>
  if (error) return <div className="error">Error loading patterns: {error}</div>

  return (
    <div className="card">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
        <div>
          <button 
            onClick={onBack}
            style={{
              padding: '8px 16px',
              marginBottom: '10px',
              cursor: 'pointer'
            }}
          >
            ← Back to Services
          </button>
          <h2>
            Service: <code>{serviceName}</code>
          </h2>
          <h3 style={{ 
            color: getSeverityColor(severity),
            marginTop: '8px'
          }}>
            Severity: {severity}
          </h3>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '15px', marginTop: '20px' }}>
        <div className="stat-card">
          <div className="stat-label">Total Patterns</div>
          <div className="stat-value">{filteredTemplates.length.toLocaleString()}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Total Messages</div>
          <div className="stat-value">{totalMessages.toLocaleString()}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Avg per Pattern</div>
          <div className="stat-value">
            {filteredTemplates.length > 0 
              ? Math.round(totalMessages / filteredTemplates.length).toLocaleString()
              : 0
            }
          </div>
        </div>
      </div>

      <div className="filter-group" style={{ marginTop: '20px' }}>
        <div className="threshold-input">
          <label>Min Count:</label>
          <input 
            type="number" 
            value={filter.minCount} 
            onChange={(e) => setFilter({...filter, minCount: Number(e.target.value)})}
            min="0"
          />
        </div>
      </div>

      <h3 style={{ marginTop: '20px', marginBottom: '12px' }}>
        Log Patterns ({filteredTemplates.length.toLocaleString()})
      </h3>

      {filteredTemplates.length === 0 ? (
        <p className="template-count-text">No patterns match the current filters</p>
      ) : (
        <table style={{ marginTop: '15px' }}>
          <thead>
            <tr>
              <th style={{ width: '50%' }}>Pattern Template</th>
              <th>Count</th>
              <th>Percentage</th>
              <th>Example</th>
            </tr>
          </thead>
          <tbody>
            {filteredTemplates.map((tmpl, i) => (
              <tr 
                key={i}
                onClick={() => onViewPattern && onViewPattern(serviceName, severity, tmpl.template)}
                style={{ cursor: onViewPattern ? 'pointer' : 'default' }}
                className={onViewPattern ? 'clickable-row' : ''}
              >
                <td>
                  <code className="template-code" style={{ 
                    fontSize: '13px',
                    wordBreak: 'break-word',
                    display: 'block',
                    padding: '8px',
                    background: 'var(--bg-tertiary)',
                    borderRadius: '4px'
                  }}>
                    {tmpl.template}
                  </code>
                </td>
                <td>{tmpl.count.toLocaleString()}</td>
                <td>{tmpl.percentage.toFixed(1)}%</td>
                <td className="template-count-text-small" style={{ 
                  fontStyle: 'italic',
                  maxWidth: '300px',
                  wordBreak: 'break-word'
                }}>
                  {tmpl.example}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

export default LogServiceDetails
