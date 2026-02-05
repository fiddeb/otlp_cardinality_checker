import { useState, useEffect } from 'react'

function ActiveSeries() {
  const [data, setData] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [showAll, setShowAll] = useState(false)
  const [sortBy, setSortBy] = useState('series-prom') // 'series-otlp', 'series-prom', 'name', 'samples'

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

  const getOtlpSeries = (metric) => metric.active_series_otlp ?? metric.active_series ?? 0
  const getPromSeries = (metric) => metric.active_series_prometheus ?? getOtlpSeries(metric)

  // Sort metrics
  const sorted = [...data].sort((a, b) => {
    switch (sortBy) {
      case 'series-otlp':
        return getOtlpSeries(b) - getOtlpSeries(a)
      case 'series-prom':
        return getPromSeries(b) - getPromSeries(a)
      case 'name':
        return a.name.localeCompare(b.name)
      case 'samples':
        return b.sample_count - a.sample_count
      default:
        return 0
    }
  })

  // Calculate statistics
  const totalOtlpSeries = sorted.reduce((sum, m) => sum + getOtlpSeries(m), 0)
  const totalPromSeries = sorted.reduce((sum, m) => sum + getPromSeries(m), 0)
  const avgOtlpSeries = totalOtlpSeries / (sorted.length || 1)
  const maxPromSeries = Math.max(...sorted.map(m => getPromSeries(m)))

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
              Total Active Series (OTLP)
            </div>
            <div style={{ fontSize: '1.5em', fontWeight: 'bold' }}>
              {totalOtlpSeries.toLocaleString()}
            </div>
          </div>
          
          <div style={{ padding: '12px', backgroundColor: 'var(--bg-secondary)', borderRadius: '6px' }}>
            <div style={{ fontSize: '0.85em', color: 'var(--text-secondary)', marginBottom: '4px' }}>
              Total Active Series (Prometheus)
            </div>
            <div style={{ fontSize: '1.5em', fontWeight: 'bold' }}>
              {totalPromSeries.toLocaleString()}
            </div>
          </div>
          
          <div style={{ padding: '12px', backgroundColor: 'var(--bg-secondary)', borderRadius: '6px' }}>
            <div style={{ fontSize: '0.85em', color: 'var(--text-secondary)', marginBottom: '4px' }}>
              Average per Metric (OTLP)
            </div>
            <div style={{ fontSize: '1.5em', fontWeight: 'bold' }}>
              {Math.round(avgOtlpSeries).toLocaleString()}
            </div>
          </div>

          <div style={{ padding: '12px', backgroundColor: 'var(--bg-secondary)', borderRadius: '6px' }}>
            <div style={{ fontSize: '0.85em', color: 'var(--text-secondary)', marginBottom: '4px' }}>
              Highest Cardinality (Prometheus)
            </div>
            <div style={{ fontSize: '1.5em', fontWeight: 'bold', color: maxPromSeries > 1000 ? 'var(--danger)' : 'var(--success)' }}>
              {maxPromSeries.toLocaleString()}
            </div>
          </div>
        </div>

        <div style={{ marginBottom: '16px', display: 'flex', gap: '8px', alignItems: 'center' }}>
          <span style={{ fontWeight: '500' }}>Sort by:</span>
          <button 
            onClick={() => setSortBy('series-otlp')}
            className={sortBy === 'series-otlp' ? 'sort-button active' : 'sort-button'}
          >
            OTLP Series
          </button>
          <button 
            onClick={() => setSortBy('series-prom')}
            className={sortBy === 'series-prom' ? 'sort-button active' : 'sort-button'}
          >
            Prometheus Series
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
              <th>Active Series (OTLP)</th>
              <th>Active Series (Prometheus)</th>
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
                  <span className={`badge ${getSeriesBadge(getOtlpSeries(metric))}`}>
                    {getOtlpSeries(metric).toLocaleString()}
                  </span>
                </td>
                <td>
                  <span className={`badge ${getSeriesBadge(getPromSeries(metric))}`}>
                    {getPromSeries(metric).toLocaleString()}
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
