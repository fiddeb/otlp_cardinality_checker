import { useState, useEffect } from 'react'

function CrossSignalCardinality() {
  const [threshold, setThreshold] = useState(100)
  const [limit, setLimit] = useState(50)
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    setLoading(true)
    setError(null)

    fetch(`/api/v1/cardinality/high?threshold=${threshold}&limit=${limit}`)
      .then(r => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        return r.json()
      })
      .then(result => {
        setData(result)
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [threshold, limit])

  const getSignalTypeColor = (type) => {
    switch(type) {
      case 'metric': return '#4CAF50'
      case 'span': return '#2196F3'
      case 'log': return '#FF9800'
      default: return '#757575'
    }
  }

  const getCardinalityColor = (cardinality) => {
    if (cardinality < 100) return '#4CAF50'
    if (cardinality < 1000) return '#FF9800'
    return '#f44336'
  }

  if (loading) return <div className="loading">Loading cross-signal cardinality data...</div>
  if (error) return <div className="error">Error: {error}</div>
  if (!data || !data.high_cardinality_keys) return <div className="no-data">No high-cardinality keys found</div>

  return (
    <div className="cross-signal-cardinality">
      <div className="controls">
        <div className="control-group">
          <label>
            Min Cardinality:
            <input
              type="number"
              value={threshold}
              onChange={(e) => setThreshold(Number(e.target.value))}
              min="1"
              step="10"
            />
          </label>
        </div>
        <div className="control-group">
          <label>
            Max Results:
            <input
              type="number"
              value={limit}
              onChange={(e) => setLimit(Number(e.target.value))}
              min="10"
              max="1000"
              step="10"
            />
          </label>
        </div>
      </div>

      <div className="stats">
        <div className="stat-card">
          <div className="stat-value">{data.total}</div>
          <div className="stat-label">High-Cardinality Keys</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{threshold}</div>
          <div className="stat-label">Threshold</div>
        </div>
      </div>

      <div className="keys-table">
        <table>
          <thead>
            <tr>
              <th>Signal Type</th>
              <th>Signal Name</th>
              <th>Key Scope</th>
              <th>Key Name</th>
              <th>Cardinality</th>
              <th>Count</th>
              <th>Sample Values</th>
            </tr>
          </thead>
          <tbody>
            {data.high_cardinality_keys.map((key, idx) => (
              <tr key={idx}>
                <td>
                  <span 
                    className="signal-type-badge"
                    style={{ backgroundColor: getSignalTypeColor(key.signal_type) }}
                  >
                    {key.signal_type}
                  </span>
                </td>
                <td className="signal-name">{key.signal_name}</td>
                <td className="key-scope">{key.key_scope}</td>
                <td className="key-name">
                  <code>{key.key_name}</code>
                  {key.event_name && <span className="event-name"> ({key.event_name})</span>}
                </td>
                <td>
                  <span 
                    className="cardinality-badge"
                    style={{ 
                      backgroundColor: getCardinalityColor(key.estimated_cardinality),
                      color: 'white',
                      padding: '2px 8px',
                      borderRadius: '4px',
                      fontWeight: 'bold'
                    }}
                  >
                    {key.estimated_cardinality.toLocaleString()}
                  </span>
                </td>
                <td>{key.key_count.toLocaleString()}</td>
                <td className="sample-values">
                  {key.value_samples && key.value_samples.length > 0 ? (
                    key.value_samples.slice(0, 3).map((sample, i) => (
                      <span key={i} className="sample-value">{sample}</span>
                    ))
                  ) : (
                    <span className="no-samples">No samples</span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {data.high_cardinality_keys.length === 0 && (
        <div className="no-results">
          No keys found with cardinality ≥ {threshold}. Try lowering the threshold.
        </div>
      )}
    </div>
  )
}

export default CrossSignalCardinality
