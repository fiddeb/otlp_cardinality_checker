import { useState, useEffect } from 'react'

function ActiveSeries() {
  const [data, setData] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [showAll, setShowAll] = useState(false)
  const [sortBy, setSortBy] = useState('series') // 'series', 'name', 'samples'

  useEffect(() => {
    fetch('/api/v1/metrics')
      .then(r => r.json())
      .then(response => {
        // Extract metrics from paginated response
        const metrics = response.data || response
        setData(metrics)
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [])

  if (loading) return <div className="loading">Loading...</div>
  if (error) return <div className="error">Error: {error}</div>

  // Sort metrics
  const sorted = [...data].sort((a, b) => {
    switch (sortBy) {
      case 'series':
        return (b.active_series || 0) - (a.active_series || 0)
      case 'name':
        return a.name.localeCompare(b.name)
      case 'samples':
        return b.sample_count - a.sample_count
      default:
        return 0
    }
  })

  // Calculate statistics
  const totalSeries = sorted.reduce((sum, m) => sum + (m.active_series || 0), 0)
  const avgSeries = totalSeries / (sorted.length || 1)
  const maxSeries = Math.max(...sorted.map(m => m.active_series || 0))

  // Display limit
  const displayLimit = showAll ? sorted.length : 20
  const displayed = sorted.slice(0, displayLimit)

  const getSeriesBadge = (count) => {
    if (count > 1000) return 'high'
    if (count > 100) return 'medium'
    return 'low'
  }

  return (
    <div className="view-container">
      <h1>📊 Active Series</h1>
      
      <div className="card">
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '16px', marginBottom: '20px' }}>
          <div style={{ padding: '12px', backgroundColor: 'var(--bg-secondary)', borderRadius: '6px' }}>
            <div style={{ fontSize: '0.85em', color: 'var(--text-secondary)', marginBottom: '4px' }}>
              Total Metrics
            </div>
            <div style={{ fontSize: '1.5em', fontWeight: 'bold' }}>
              {sorted.length.toLocaleString()}
            </div>
          </div>
          
          <div style={{ padding: '12px', backgroundColor: 'var(--bg-secondary)', borderRadius: '6px' }}>
            <div style={{ fontSize: '0.85em', color: 'var(--text-secondary)', marginBottom: '4px' }}>
              Total Active Series
            </div>
            <div style={{ fontSize: '1.5em', fontWeight: 'bold' }}>
              {totalSeries.toLocaleString()}
            </div>
          </div>
          
          <div style={{ padding: '12px', backgroundColor: 'var(--bg-secondary)', borderRadius: '6px' }}>
            <div style={{ fontSize: '0.85em', color: 'var(--text-secondary)', marginBottom: '4px' }}>
              Average per Metric
            </div>
            <div style={{ fontSize: '1.5em', fontWeight: 'bold' }}>
              {Math.round(avgSeries).toLocaleString()}
            </div>
          </div>
          
          <div style={{ padding: '12px', backgroundColor: 'var(--bg-secondary)', borderRadius: '6px' }}>
            <div style={{ fontSize: '0.85em', color: 'var(--text-secondary)', marginBottom: '4px' }}>
              Highest Cardinality
            </div>
            <div style={{ fontSize: '1.5em', fontWeight: 'bold', color: maxSeries > 1000 ? 'var(--danger)' : 'var(--success)' }}>
              {maxSeries.toLocaleString()}
            </div>
          </div>
        </div>

        <div style={{ marginBottom: '16px', display: 'flex', gap: '8px', alignItems: 'center' }}>
          <span style={{ fontWeight: '500' }}>Sort by:</span>
          <button 
            onClick={() => setSortBy('series')}
            className={sortBy === 'series' ? 'sort-button active' : 'sort-button'}
          >
            Active Series
          </button>
          <button 
            onClick={() => setSortBy('name')}
            className={sortBy === 'name' ? 'sort-button active' : 'sort-button'}
          >
            Name
          </button>
          <button 
            onClick={() => setSortBy('samples')}
            className={sortBy === 'samples' ? 'sort-button active' : 'sort-button'}
          >
            Sample Count
          </button>
        </div>

        <table>
          <thead>
            <tr>
              <th>Rank</th>
              <th>Metric Name</th>
              <th>Type</th>
              <th>Active Series</th>
              <th>Label Keys</th>
              <th>Samples</th>
            </tr>
          </thead>
          <tbody>
            {displayed.map((metric, idx) => (
              <tr key={metric.name}>
                <td style={{ fontWeight: '500' }}>#{idx + 1}</td>
                <td>
                  <code style={{ fontSize: '0.9em' }}>{metric.name}</code>
                </td>
                <td>
                  <span className="type-badge">{metric.type || 'Unknown'}</span>
                </td>
                <td>
                  <span className={`badge ${getSeriesBadge(metric.active_series || 0)}`}>
                    {(metric.active_series || 0).toLocaleString()}
                  </span>
                </td>
                <td>
                  {Object.keys(metric.label_keys || {}).length} keys
                </td>
                <td>
                  {metric.sample_count.toLocaleString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {sorted.length > displayLimit && (
          <div style={{ marginTop: '16px', textAlign: 'center' }}>
            <button 
              onClick={() => setShowAll(!showAll)}
              style={{
                padding: '8px 16px',
                backgroundColor: 'var(--link-color)',
                color: 'white',
                border: 'none',
                borderRadius: '4px',
                cursor: 'pointer'
              }}
            >
              {showAll ? 'Show Top 20' : `Show All ${sorted.length} Metrics`}
            </button>
          </div>
        )}
      </div>

      <style jsx>{`
        .sort-button {
          padding: 6px 12px;
          border: 1px solid var(--border-color);
          background-color: var(--bg-secondary);
          border-radius: 4px;
          cursor: pointer;
          transition: all 0.2s;
        }
        
        .sort-button:hover {
          background-color: var(--bg-hover);
        }
        
        .sort-button.active {
          background-color: var(--link-color);
          color: white;
          border-color: var(--link-color);
        }
        
        .type-badge {
          padding: 2px 8px;
          background-color: var(--bg-secondary);
          border-radius: 3px;
          font-size: 0.85em;
        }
      `}</style>
    </div>
  )
}

export default ActiveSeries
