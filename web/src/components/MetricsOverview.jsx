import { useState, useEffect } from 'react'

function MetricsOverview({ onViewMetric }) {
  const [metrics, setMetrics] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [filter, setFilter] = useState({
    type: 'all',
    search: ''
  })

  useEffect(() => {
    fetch('/api/v1/metrics?limit=1000')
      .then(r => r.json())
      .then(result => {
        setMetrics(result.data || [])
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [])

  const filteredMetrics = (metrics || []).filter(metric => {
    if (filter.type !== 'all' && metric.type !== filter.type) return false
    if (filter.search && !metric.name.toLowerCase().includes(filter.search.toLowerCase())) return false
    return true
  })

  const metricTypes = ['all', ...new Set((metrics || []).map(m => m.type).filter(Boolean))]

  const getTypeColor = (type) => {
    const colors = {
      'Sum': '#1976d2',
      'Gauge': '#388e3c',
      'Histogram': '#f57c00',
      'Summary': '#7b1fa2',
      'ExponentialHistogram': '#d32f2f'
    }
    return colors[type] || 'var(--text-secondary)'
  }

  const getBucketCount = (metric) => {
    if (metric.type === 'Histogram' && metric.data && metric.data.explicit_bounds) {
      // Number of buckets = number of bounds + 1 (including infinity bucket)
      return metric.data.explicit_bounds.length + 1
    }
    if (metric.type === 'ExponentialHistogram' && metric.data && metric.data.scales) {
      // Exponential histograms have dynamic buckets based on scale
      return `Scale: ${metric.data.scales.join(', ')}`
    }
    return '-'
  }

  if (loading) return <div className="loading">Loading metrics...</div>
  if (error) return <div className="error">Error: {error}</div>

  const totalSamples = filteredMetrics.reduce((sum, metric) => sum + metric.sample_count, 0)
  const typeBreakdown = filteredMetrics.reduce((acc, metric) => {
    acc[metric.type] = (acc[metric.type] || 0) + 1
    return acc
  }, {})

  return (
    <div className="card">
      <h2>Metrics Overview</h2>
      
      <div className="stats-row">
        <div className="stat-card">
          <div className="stat-label">Total Metrics</div>
          <div className="stat-value">{filteredMetrics.length}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Total Observations</div>
          <div className="stat-value">{totalSamples.toLocaleString()}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Metric Types</div>
          <div className="stat-value">{Object.keys(typeBreakdown).length}</div>
        </div>
      </div>

      <h3 style={{ marginTop: '20px', marginBottom: '12px' }}>Type Distribution</h3>
      
      <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap', marginBottom: '20px' }}>
        {Object.entries(typeBreakdown)
          .sort((a, b) => b[1] - a[1])
          .map(([type, count]) => (
            <div 
              key={type}
              style={{
                padding: '8px 16px',
                background: getTypeColor(type),
                color: 'white',
                borderRadius: '4px',
                fontSize: '0.9em',
                fontWeight: '500'
              }}
            >
              {type}: {count}
            </div>
          ))}
      </div>

      <div className="filter-group">
        <input 
          type="text"
          placeholder="Search metrics..."
          value={filter.search}
          onChange={(e) => setFilter({...filter, search: e.target.value})}
          style={{ width: '250px' }}
        />

        <select 
          value={filter.type} 
          onChange={(e) => setFilter({...filter, type: e.target.value})}
        >
          {metricTypes.map(type => (
            <option key={type} value={type}>
              {type === 'all' ? 'All Types' : `Type: ${type}`}
            </option>
          ))}
        </select>
      </div>

      <h3 style={{ marginTop: '20px', marginBottom: '12px' }}>
        Metrics ({filteredMetrics.length})
      </h3>

      <table>
        <thead>
          <tr>
            <th>Metric Name</th>
            <th>Type</th>
            <th>Unit</th>
            <th>Description</th>
            <th>Observations</th>
            <th>Buckets</th>
          </tr>
        </thead>
        <tbody>
          {filteredMetrics
            .sort((a, b) => b.sample_count - a.sample_count)
            .map((metric, i) => (
              <tr key={i}>
                <td>
                  <span 
                    className="detail-link"
                    onClick={() => onViewMetric && onViewMetric(metric.name)}
                    style={{ cursor: onViewMetric ? 'pointer' : 'default' }}
                  >
                    {metric.name}
                  </span>
                </td>
                <td>
                  <span 
                    className="key-badge"
                    style={{ 
                      background: getTypeColor(metric.type),
                      color: 'white'
                    }}
                  >
                    {metric.type}
                  </span>
                </td>
                <td>{metric.unit || '-'}</td>
                <td style={{ maxWidth: '300px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {metric.description || '-'}
                </td>
                <td>{metric.sample_count.toLocaleString()}</td>
                <td>{getBucketCount(metric)}</td>
              </tr>
            ))}
        </tbody>
      </table>

      {filteredMetrics.length === 0 && (
        <p className="template-count-text" style={{ textAlign: 'center', padding: '20px' }}>
          No metrics match the current filters
        </p>
      )}
    </div>
  )
}

export default MetricsOverview
