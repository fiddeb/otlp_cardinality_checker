import { useState, useEffect } from 'react'

function MetricsView({ onViewDetails }) {
  const [metrics, setMetrics] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [filter, setFilter] = useState({
    type: 'all',
    minSamples: 0,
    minCardinality: 0,
    search: ''
  })
  const [sortField, setSortField] = useState('sample_count')
  const [sortDirection, setSortDirection] = useState('desc')

  const itemsPerPage = 100

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

  const getMaxCardinality = (metric) => {
    if (!metric.label_keys) return 0
    return Math.max(...Object.values(metric.label_keys).map(k => k.estimated_cardinality || 0))
  }

  const filteredMetrics = (metrics || []).filter(metric => {
    if (filter.type !== 'all' && metric.type !== filter.type) return false
    if (metric.sample_count < filter.minSamples) return false
    if (getMaxCardinality(metric) < filter.minCardinality) return false
    if (filter.search && !metric.name.toLowerCase().includes(filter.search.toLowerCase())) return false
    return true
  })

  const totalPages = Math.ceil(filteredMetrics.length / itemsPerPage)
  const startIndex = (currentPage - 1) * itemsPerPage
  const endIndex = startIndex + itemsPerPage
  const currentMetrics = filteredMetrics.slice(startIndex, endIndex)

  // Reset to page 1 when filters change
  useEffect(() => {
    setCurrentPage(1)
  }, [filter])

  const metricTypes = ['all', ...new Set((metrics || []).map(m => m.type).filter(Boolean))]

  const getCardinalityBadge = (card) => {
    if (card > 200) return 'high'
    if (card > 50) return 'medium'
    return 'low'
  }

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

  const handleSort = (field) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDirection('desc')
    }
  }

  const getSortedMetrics = (metrics) => {
    return [...metrics].sort((a, b) => {
      let aVal, bVal
      
      switch(sortField) {
        case 'name':
          aVal = a.name
          bVal = b.name
          break
        case 'type':
          aVal = a.type
          bVal = b.type
          break
        case 'sample_count':
          aVal = a.sample_count
          bVal = b.sample_count
          break
        case 'labels':
          aVal = a.label_keys ? Object.keys(a.label_keys).length : 0
          bVal = b.label_keys ? Object.keys(b.label_keys).length : 0
          break
        case 'resources':
          aVal = a.resource_keys ? Object.keys(a.resource_keys).length : 0
          bVal = b.resource_keys ? Object.keys(b.resource_keys).length : 0
          break
        case 'cardinality':
          aVal = getMaxCardinality(a)
          bVal = getMaxCardinality(b)
          break
        case 'complexity':
          // Calculate complexity inline
          const aLabels = a.label_keys ? Object.keys(a.label_keys).length : 0
          const aResources = a.resource_keys ? Object.keys(a.resource_keys).length : 0
          let aBuckets = 0
          if (a.type === 'Histogram' && a.data && a.data.explicit_bounds) {
            aBuckets = a.data.explicit_bounds.length + 1
          } else if (a.type === 'ExponentialHistogram' && a.data && a.data.scales) {
            aBuckets = a.data.scales.length * 10
          }
          aVal = (aLabels + aResources + aBuckets) * getMaxCardinality(a)
          
          const bLabels = b.label_keys ? Object.keys(b.label_keys).length : 0
          const bResources = b.resource_keys ? Object.keys(b.resource_keys).length : 0
          let bBuckets = 0
          if (b.type === 'Histogram' && b.data && b.data.explicit_bounds) {
            bBuckets = b.data.explicit_bounds.length + 1
          } else if (b.type === 'ExponentialHistogram' && b.data && b.data.scales) {
            bBuckets = b.data.scales.length * 10
          }
          bVal = (bLabels + bResources + bBuckets) * getMaxCardinality(b)
          break
        case 'services':
          aVal = a.services ? Object.keys(a.services).length : 0
          bVal = b.services ? Object.keys(b.services).length : 0
          break
        default:
          aVal = a.sample_count
          bVal = b.sample_count
      }
      
      if (typeof aVal === 'string') {
        return sortDirection === 'asc' 
          ? aVal.localeCompare(bVal)
          : bVal.localeCompare(aVal)
      } else {
        return sortDirection === 'asc' ? aVal - bVal : bVal - aVal
      }
    })
  }

  if (loading) return <div className="loading">Loading...</div>
  if (error) return <div className="error">Error: {error}</div>

  const totalSamples = filteredMetrics.reduce((sum, metric) => sum + metric.sample_count, 0)
  const typeBreakdown = filteredMetrics.reduce((acc, metric) => {
    acc[metric.type] = (acc[metric.type] || 0) + 1
    return acc
  }, {})

  return (
    <div className="card">
      <h2>Metrics Analysis</h2>
      
      <div className="filter-group">
        <input 
          type="text"
          placeholder="Search metrics..."
          value={filter.search}
          onChange={(e) => setFilter({...filter, search: e.target.value})}
          style={{ width: '200px' }}
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

        <div className="threshold-input">
          <label>Min Samples:</label>
          <input 
            type="number" 
            value={filter.minSamples} 
            onChange={(e) => setFilter({...filter, minSamples: Number(e.target.value)})}
            min="0"
          />
        </div>

        <div className="threshold-input">
          <label>Min Cardinality:</label>
          <input 
            type="number" 
            value={filter.minCardinality} 
            onChange={(e) => setFilter({...filter, minCardinality: Number(e.target.value)})}
            min="0"
          />
        </div>
      </div>

      <p className="template-count-text" style={{ marginTop: '10px' }}>
        Showing {startIndex + 1}-{Math.min(endIndex, filteredMetrics.length)} of {filteredMetrics.length} metrics
        {totalPages > 1 && ` (Page ${currentPage} of ${totalPages})`}
      </p>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '15px', marginTop: '20px' }}>
        <div className="stat-card">
          <div className="stat-label">Total Metrics</div>
          <div className="stat-value">{filteredMetrics.length}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Total Samples</div>
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

      <h3 style={{ marginTop: '20px', marginBottom: '12px' }}>Metrics Breakdown</h3>
      
      <table>
        <thead>
          <tr>
            <th onClick={() => handleSort('name')} style={{ cursor: 'pointer' }}>
              Metric Name {sortField === 'name' && (sortDirection === 'asc' ? '↑' : '↓')}
            </th>
            <th onClick={() => handleSort('type')} style={{ cursor: 'pointer' }}>
              Type {sortField === 'type' && (sortDirection === 'asc' ? '↑' : '↓')}
            </th>
            <th onClick={() => handleSort('sample_count')} style={{ cursor: 'pointer' }}>
              Samples {sortField === 'sample_count' && (sortDirection === 'asc' ? '↑' : '↓')}
            </th>
            <th onClick={() => handleSort('labels')} style={{ cursor: 'pointer' }}>
              Labels {sortField === 'labels' && (sortDirection === 'asc' ? '↑' : '↓')}
            </th>
            <th onClick={() => handleSort('resources')} style={{ cursor: 'pointer' }}>
              Resources {sortField === 'resources' && (sortDirection === 'asc' ? '↑' : '↓')}
            </th>
            <th onClick={() => handleSort('cardinality')} style={{ cursor: 'pointer' }}>
              Max Cardinality {sortField === 'cardinality' && (sortDirection === 'asc' ? '↑' : '↓')}
            </th>
            <th onClick={() => handleSort('complexity')} style={{ cursor: 'pointer' }}>
              Complexity {sortField === 'complexity' && (sortDirection === 'asc' ? '↑' : '↓')}
            </th>
            <th onClick={() => handleSort('services')} style={{ cursor: 'pointer' }}>
              Services {sortField === 'services' && (sortDirection === 'asc' ? '↑' : '↓')}
            </th>
          </tr>
        </thead>
        <tbody>
          {getSortedMetrics(currentMetrics)
            .map((metric, i) => {
              const maxCard = getMaxCardinality(metric)
              const labelCount = metric.label_keys ? Object.keys(metric.label_keys).length : 0
              const resourceCount = metric.resource_keys ? Object.keys(metric.resource_keys).length : 0
              const serviceCount = metric.services ? Object.keys(metric.services).length : 0
              
              // Calculate complexity: total_keys × max_cardinality
              let bucketCount = 0
              if (metric.type === 'Histogram' && metric.data && metric.data.explicit_bounds) {
                bucketCount = metric.data.explicit_bounds.length + 1
              } else if (metric.type === 'ExponentialHistogram' && metric.data && metric.data.scales) {
                bucketCount = metric.data.scales.length * 10
              }
              
              const totalKeys = labelCount + resourceCount + bucketCount
              const complexity = totalKeys * maxCard
              
              return (
                <tr key={i}>
                  <td>
                    <span 
                      className="detail-link"
                      onClick={() => onViewDetails('metrics', metric.name)}
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
                  <td>{metric.sample_count.toLocaleString()}</td>
                  <td>{labelCount}</td>
                  <td>{resourceCount}</td>
                  <td>
                    {maxCard > 0 ? (
                      <span className={`badge ${getCardinalityBadge(maxCard)}`}>
                        {maxCard}
                      </span>
                    ) : '-'}
                  </td>
                  <td>{complexity > 0 ? complexity.toLocaleString() : '-'}</td>
                  <td>{serviceCount}</td>
                </tr>
              )
            })}
        </tbody>
      </table>

      {totalPages > 1 && (
        <div style={{ 
          display: 'flex', 
          justifyContent: 'center', 
          alignItems: 'center',
          gap: '10px',
          marginTop: '20px',
          padding: '20px'
        }}>
          <button 
            onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
            disabled={currentPage === 1}
            style={{
              padding: '8px 16px',
              cursor: currentPage === 1 ? 'not-allowed' : 'pointer',
              opacity: currentPage === 1 ? 0.5 : 1
            }}
          >
            Previous
          </button>
          
          <span className="template-count-text">
            Page {currentPage} of {totalPages}
          </span>
          
          <button 
            onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
            disabled={currentPage === totalPages}
            style={{
              padding: '8px 16px',
              cursor: currentPage === totalPages ? 'not-allowed' : 'pointer',
              opacity: currentPage === totalPages ? 0.5 : 1
            }}
          >
            Next
          </button>
        </div>
      )}

      {filteredMetrics.length === 0 && (
        <p className="template-count-text" style={{ textAlign: 'center', padding: '20px' }}>
          No metrics match the current filters
        </p>
      )}
    </div>
  )
}

export default MetricsView
