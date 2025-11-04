import { useState, useEffect } from 'react'

function AttributesView() {
  const [attributes, setAttributes] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [filter, setFilter] = useState({
    signalType: 'all',
    scope: 'all',
    minCardinality: 0,
    search: ''
  })
  const [sortField, setSortField] = useState('cardinality')
  const [sortDirection, setSortDirection] = useState('desc')

  const itemsPerPage = 100

  useEffect(() => {
    let url = '/api/v1/attributes?limit=1000'
    
    // Add filters
    if (filter.signalType !== 'all') {
      url += `&signal_type=${filter.signalType}`
    }
    if (filter.scope !== 'all') {
      url += `&scope=${filter.scope}`
    }
    if (filter.minCardinality > 0) {
      url += `&min_cardinality=${filter.minCardinality}`
    }
    
    // Add sorting
    url += `&sort_by=${sortField}&sort_order=${sortDirection}`
    
    fetch(url)
      .then(r => r.json())
      .then(result => {
        setAttributes(result.data || [])
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [filter, sortField, sortDirection])

  const filteredAttributes = (attributes || []).filter(attr => {
    if (filter.search && !attr.key.toLowerCase().includes(filter.search.toLowerCase())) return false
    return true
  })

  const totalPages = Math.ceil(filteredAttributes.length / itemsPerPage)
  const startIndex = (currentPage - 1) * itemsPerPage
  const endIndex = startIndex + itemsPerPage
  const currentAttributes = filteredAttributes.slice(startIndex, endIndex)

  // Reset to page 1 when filters change
  useEffect(() => {
    setCurrentPage(1)
  }, [filter])

  const getCardinalityBadge = (card) => {
    if (card > 1000) return 'high'
    if (card > 100) return 'medium'
    return 'low'
  }

  const getScopeColor = (scope) => {
    const colors = {
      'resource': '#1976d2',
      'attribute': '#388e3c',
      'both': '#f57c00'
    }
    return colors[scope] || 'var(--text-secondary)'
  }

  const handleSort = (field) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDirection('desc')
    }
  }

  if (loading) {
    return <div className="loading">Loading attributes...</div>
  }

  if (error) {
    return <div className="error">Error: {error}</div>
  }

  return (
    <div className="view-container">
      <div className="view-header">
        <h2>Attribute Catalog</h2>
        <p className="view-description">
          Global attribute cardinality tracking across all signals (metrics, spans, logs)
        </p>
      </div>

      <div className="filters">
        <div className="filter-group">
          <label>Signal Type:</label>
          <select 
            value={filter.signalType} 
            onChange={e => setFilter({...filter, signalType: e.target.value})}
          >
            <option value="all">All Signals</option>
            <option value="metric">Metrics</option>
            <option value="span">Spans</option>
            <option value="log">Logs</option>
          </select>
        </div>

        <div className="filter-group">
          <label>Scope:</label>
          <select 
            value={filter.scope} 
            onChange={e => setFilter({...filter, scope: e.target.value})}
          >
            <option value="all">All Scopes</option>
            <option value="resource">Resource Attributes</option>
            <option value="attribute">Data Attributes</option>
            <option value="both">Both</option>
          </select>
        </div>

        <div className="filter-group">
          <label>Min Cardinality:</label>
          <input
            type="number"
            value={filter.minCardinality}
            onChange={e => setFilter({...filter, minCardinality: parseInt(e.target.value) || 0})}
            placeholder="0"
            min="0"
          />
        </div>

        <div className="filter-group">
          <label>Search:</label>
          <input
            type="text"
            value={filter.search}
            onChange={e => setFilter({...filter, search: e.target.value})}
            placeholder="Filter by key..."
          />
        </div>
      </div>

      <div className="stats-bar">
        <div className="stat">
          <span className="stat-label">Total Attributes:</span>
          <span className="stat-value">{filteredAttributes.length}</span>
        </div>
        <div className="stat">
          <span className="stat-label">High Cardinality (&gt;1000):</span>
          <span className="stat-value">
            {filteredAttributes.filter(a => a.estimated_cardinality > 1000).length}
          </span>
        </div>
        <div className="stat">
          <span className="stat-label">Resource Attributes:</span>
          <span className="stat-value">
            {filteredAttributes.filter(a => a.scope === 'resource' || a.scope === 'both').length}
          </span>
        </div>
      </div>

      <div className="table-container">
        <table className="data-table">
          <thead>
            <tr>
              <th onClick={() => handleSort('key')} style={{cursor: 'pointer'}}>
                Attribute Key {sortField === 'key' && (sortDirection === 'asc' ? '↑' : '↓')}
              </th>
              <th onClick={() => handleSort('cardinality')} style={{cursor: 'pointer'}}>
                Cardinality {sortField === 'cardinality' && (sortDirection === 'asc' ? '↑' : '↓')}
              </th>
              <th onClick={() => handleSort('count')} style={{cursor: 'pointer'}}>
                Count {sortField === 'count' && (sortDirection === 'asc' ? '↑' : '↓')}
              </th>
              <th>Sample Values</th>
              <th>Signal Types</th>
              <th>Scope</th>
            </tr>
          </thead>
          <tbody>
            {currentAttributes.map((attr, idx) => (
              <tr key={idx}>
                <td>
                  <code className="attribute-key">{attr.key}</code>
                </td>
                <td>
                  <span className={`cardinality-badge ${getCardinalityBadge(attr.estimated_cardinality)}`}>
                    {attr.estimated_cardinality?.toLocaleString() || 0}
                  </span>
                </td>
                <td>{attr.count?.toLocaleString() || 0}</td>
                <td>
                  <div className="value-samples">
                    {(attr.value_samples || []).slice(0, 5).map((val, i) => (
                      <code key={i} className="sample-value">{val}</code>
                    ))}
                    {(attr.value_samples?.length || 0) > 5 && (
                      <span className="more-indicator">+{attr.value_samples.length - 5} more</span>
                    )}
                  </div>
                </td>
                <td>
                  <div className="signal-types">
                    {(attr.signal_types || []).map((type, i) => (
                      <span key={i} className="signal-type-badge">{type}</span>
                    ))}
                  </div>
                </td>
                <td>
                  <span 
                    className="scope-badge" 
                    style={{backgroundColor: getScopeColor(attr.scope)}}
                  >
                    {attr.scope}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div className="pagination">
          <button 
            onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
            disabled={currentPage === 1}
          >
            Previous
          </button>
          <span>Page {currentPage} of {totalPages}</span>
          <button 
            onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
            disabled={currentPage === totalPages}
          >
            Next
          </button>
        </div>
      )}
    </div>
  )
}

export default AttributesView
