import { useState, useEffect } from 'react'

function MetadataComplexity({ onViewDetails }) {
  const [threshold, setThreshold] = useState(10)
  const [limit, setLimit] = useState(50)
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [sortField, setSortField] = useState('complexity_score')
  const [sortDirection, setSortDirection] = useState('desc')

  useEffect(() => {
    setLoading(true)
    setError(null)

    fetch(`/api/v1/cardinality/complexity?threshold=${threshold}&limit=${limit}`)
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

  const getComplexityColor = (score) => {
    if (score < 1000) return '#4CAF50'
    if (score < 10000) return '#FF9800'
    return '#f44336'
  }

  const handleSort = (field) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDirection('desc')
    }
  }

  const sortData = (signals) => {
    if (!signals) return []
    
    return [...signals].sort((a, b) => {
      let aVal = a[sortField]
      let bVal = b[sortField]
      
      if (sortDirection === 'asc') {
        return aVal > bVal ? 1 : -1
      } else {
        return aVal < bVal ? 1 : -1
      }
    })
  }

  if (loading) return <div className="loading">Loading metadata complexity data...</div>
  if (error) return <div className="error">Error: {error}</div>
  if (!data || !data.signals) return <div className="no-data">No signals found</div>

  const sortedSignals = sortData(data.signals)

  return (
    <div className="metadata-complexity">
      <div className="header">
        <h2>Metadata Complexity Analysis</h2>
        <p>Identify signals with excessive instrumentation that may cause cardinality issues</p>
      </div>

      <div className="controls">
        <div className="control-group">
          <label>
            Min Total Keys:
            <input
              type="number"
              value={threshold}
              onChange={(e) => setThreshold(Number(e.target.value))}
              min="1"
              step="5"
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
          <div className="stat-label">Complex Signals</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{threshold}+</div>
          <div className="stat-label">Key Threshold</div>
        </div>
      </div>

      <div className="complexity-table">
        <table>
          <thead>
            <tr>
              <th onClick={() => handleSort('signal_type')} className="sortable">
                Signal Type {sortField === 'signal_type' && (sortDirection === 'asc' ? '▲' : '▼')}
              </th>
              <th onClick={() => handleSort('signal_name')} className="sortable">
                Signal Name {sortField === 'signal_name' && (sortDirection === 'asc' ? '▲' : '▼')}
              </th>
              <th onClick={() => handleSort('total_keys')} className="sortable">
                Total Keys {sortField === 'total_keys' && (sortDirection === 'asc' ? '▲' : '▼')}
              </th>
              <th onClick={() => handleSort('attribute_key_count')} className="sortable">
                Attributes {sortField === 'attribute_key_count' && (sortDirection === 'asc' ? '▲' : '▼')}
              </th>
              <th onClick={() => handleSort('resource_key_count')} className="sortable">
                Resources {sortField === 'resource_key_count' && (sortDirection === 'asc' ? '▲' : '▼')}
              </th>
              <th onClick={() => handleSort('event_key_count')} className="sortable">
                Events {sortField === 'event_key_count' && (sortDirection === 'asc' ? '▲' : '▼')}
              </th>
              <th onClick={() => handleSort('link_key_count')} className="sortable">
                Links {sortField === 'link_key_count' && (sortDirection === 'asc' ? '▲' : '▼')}
              </th>
              <th onClick={() => handleSort('max_cardinality')} className="sortable">
                Max Card {sortField === 'max_cardinality' && (sortDirection === 'asc' ? '▲' : '▼')}
              </th>
              <th onClick={() => handleSort('high_cardinality_count')} className="sortable">
                High Card Keys {sortField === 'high_cardinality_count' && (sortDirection === 'asc' ? '▲' : '▼')}
              </th>
              <th onClick={() => handleSort('complexity_score')} className="sortable">
                Complexity {sortField === 'complexity_score' && (sortDirection === 'asc' ? '▲' : '▼')}
              </th>
            </tr>
          </thead>
          <tbody>
            {sortedSignals.map((signal, idx) => (
              <tr 
                key={idx}
                onClick={() => onViewDetails(signal.signal_type, signal.signal_name)}
                style={{ cursor: 'pointer' }}
                className="clickable-row"
              >
                <td>
                  <span 
                    className="signal-type-badge"
                    style={{ backgroundColor: getSignalTypeColor(signal.signal_type) }}
                  >
                    {signal.signal_type}
                  </span>
                </td>
                <td className="signal-name">
                  <code>{signal.signal_name}</code>
                </td>
                <td>
                  <strong>{signal.total_keys}</strong>
                </td>
                <td>{signal.attribute_key_count}</td>
                <td>{signal.resource_key_count}</td>
                <td>{signal.event_key_count || 0}</td>
                <td>{signal.link_key_count || 0}</td>
                <td>{signal.max_cardinality.toLocaleString()}</td>
                <td>
                  {signal.high_cardinality_count > 0 ? (
                    <span className="warning-badge">
                      {signal.high_cardinality_count}
                    </span>
                  ) : (
                    <span className="ok-badge">0</span>
                  )}
                </td>
                <td>
                  <span 
                    className="complexity-badge"
                    style={{ 
                      backgroundColor: getComplexityColor(signal.complexity_score),
                      color: 'white',
                      padding: '4px 10px',
                      borderRadius: '4px',
                      fontWeight: 'bold'
                    }}
                  >
                    {signal.complexity_score.toLocaleString()}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {sortedSignals.length === 0 && (
        <div className="no-results">
          No signals found with {threshold}+ total keys. Try lowering the threshold.
        </div>
      )}

      <div className="legend">
        <h3>Understanding Complexity Score</h3>
        <p>
          <strong>Complexity Score</strong> = Total Keys × Max Cardinality
        </p>
        <ul>
          <li><span style={{color: '#4CAF50'}}>Green (&lt;1000)</span>: Low complexity, efficient</li>
          <li><span style={{color: '#FF9800'}}>Orange (1000-10000)</span>: Medium complexity, monitor</li>
          <li><span style={{color: '#f44336'}}>Red (&gt;10000)</span>: High complexity, optimize</li>
        </ul>
        <p>
          <strong>High Card Keys</strong>: Number of keys with estimated cardinality &gt; 100
        </p>
      </div>
    </div>
  )
}

export default MetadataComplexity
