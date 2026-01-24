import { useState, useEffect } from 'react'

function TracesView({ onViewDetails }) {
  const [spans, setSpans] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [filter, setFilter] = useState({
    kind: 'all',
    minSamples: 0,
    minCardinality: 0,
    search: ''
  })

  const itemsPerPage = 100

  useEffect(() => {
    fetch('/api/v1/spans?limit=1000')
      .then(r => r.json())
      .then(result => {
        setSpans(result.data || [])
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [])

  const getMaxCardinality = (span) => {
    if (!span.attribute_keys) return 0
    return Math.max(...Object.values(span.attribute_keys).map(k => k.estimated_cardinality || 0))
  }

  const filteredSpans = spans.filter(span => {
    if (filter.kind !== 'all' && span.kind !== filter.kind) return false
    if (span.sample_count < filter.minSamples) return false
    if (getMaxCardinality(span) < filter.minCardinality) return false
    if (filter.search && !span.name.toLowerCase().includes(filter.search.toLowerCase())) return false
    return true
  })

  const totalPages = Math.ceil(filteredSpans.length / itemsPerPage)
  const startIndex = (currentPage - 1) * itemsPerPage
  const endIndex = startIndex + itemsPerPage
  const currentSpans = filteredSpans.slice(startIndex, endIndex)

  // Reset to page 1 when filters change
  useEffect(() => {
    setCurrentPage(1)
  }, [filter])

  const spanKinds = ['all', ...new Set(spans.map(s => s.kind).filter(Boolean))]

  const getCardinalityBadge = (card) => {
    if (card > 200) return 'high'
    if (card > 50) return 'medium'
    return 'low'
  }

  if (loading) return <div className="loading">Loading...</div>
  if (error) return <div className="error">Error: {error}</div>

  return (
    <div className="card">
      <h2>Traces Analysis</h2>
      
      <div className="filter-group">
        <input 
          type="text"
          placeholder="Search spans..."
          value={filter.search}
          onChange={(e) => setFilter({...filter, search: e.target.value})}
          style={{ width: '200px' }}
        />

        <select 
          value={filter.kind} 
          onChange={(e) => setFilter({...filter, kind: e.target.value})}
        >
          {spanKinds.map(kind => (
            <option key={kind} value={kind}>
              {kind === 'all' ? 'All Kinds' : `Kind: ${kind}`}
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
        Showing {startIndex + 1}-{Math.min(endIndex, filteredSpans.length)} of {filteredSpans.length} span operations
        {totalPages > 1 && ` (Page ${currentPage} of ${totalPages})`}
      </p>

      <table>
        <thead>
          <tr>
            <th>Span Name</th>
            <th>Kind</th>
            <th>Samples</th>
            <th>Patterns</th>
            <th>Attributes</th>
            <th>Max Cardinality</th>
            <th>Services</th>
          </tr>
        </thead>
        <tbody>
          {currentSpans.map((span, i) => {
            const maxCard = getMaxCardinality(span)
            const attrCount = span.attribute_keys ? Object.keys(span.attribute_keys).length : 0
            const serviceCount = span.services ? Object.keys(span.services).length : 0
            const patternCount = span.name_patterns ? span.name_patterns.length : 0
            const topPattern = span.name_patterns && span.name_patterns[0] ? span.name_patterns[0].template : null
            
            return (
              <tr key={i}>
                <td>
                  <span 
                    className="detail-link"
                    onClick={() => onViewDetails('spans', span.name)}
                  >
                    {span.name}
                  </span>
                </td>
                <td>
                  <span className="key-badge">{span.kind || 'Unknown'}</span>
                </td>
                <td>{span.sample_count.toLocaleString()}</td>
                <td>
                  {patternCount > 0 ? (
                    <span 
                      className="key-badge" 
                      title={topPattern || 'No pattern'}
                      style={{ cursor: 'help' }}
                    >
                      {patternCount}
                    </span>
                  ) : '-'}
                </td>
                <td>{attrCount}</td>
                <td>
                  {maxCard > 0 ? (
                    <span className={`badge ${getCardinalityBadge(maxCard)}`}>
                      {maxCard}
                    </span>
                  ) : '-'}
                </td>
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

      {filteredSpans.length === 0 && (
        <p className="template-count-text" style={{ textAlign: 'center', padding: '20px' }}>
          No spans match the current filters
        </p>
      )}
    </div>
  )
}

export default TracesView
