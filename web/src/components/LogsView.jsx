import { useState, useEffect } from 'react'

function LogsView({ onViewDetails }) {
  const [logs, setLogs] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [filter, setFilter] = useState({
    minSamples: 0,
    minCardinality: 0
  })

  const itemsPerPage = 100

  useEffect(() => {
    fetch('/api/v1/logs?limit=1000')
      .then(r => r.json())
      .then(result => {
        setLogs(result.data || [])
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [])

  const getMaxCardinality = (log) => {
    if (!log.attribute_keys) return 0
    return Math.max(...Object.values(log.attribute_keys).map(k => k.estimated_cardinality || 0))
  }

  const filteredLogs = logs.filter(log => {
    if (log.sample_count < filter.minSamples) return false
    if (getMaxCardinality(log) < filter.minCardinality) return false
    return true
  })

  const totalPages = Math.ceil(filteredLogs.length / itemsPerPage)
  const startIndex = (currentPage - 1) * itemsPerPage
  const endIndex = startIndex + itemsPerPage
  const currentLogs = filteredLogs.slice(startIndex, endIndex)

  // Reset to page 1 when filters change
  useEffect(() => {
    setCurrentPage(1)
  }, [filter])

  const getCardinalityBadge = (card) => {
    if (card > 200) return 'high'
    if (card > 50) return 'medium'
    return 'low'
  }

  const getSeverityColor = (severity) => {
    const colors = {
      'ERROR': '#d32f2f',
      'WARN': '#f57c00',
      'INFO': '#1976d2',
      'DEBUG': '#7b1fa2',
      'TRACE': '#455a64'
    }
    return colors[severity] || '#666'
  }

  if (loading) return <div className="loading">Loading...</div>
  if (error) return <div className="error">Error: {error}</div>

  const totalSamples = filteredLogs.reduce((sum, log) => sum + log.sample_count, 0)

  return (
    <div className="card">
      <h2>Logs Analysis</h2>
      
      <div className="filter-group">
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

      <p style={{ marginTop: '10px', color: '#666' }}>
        Showing {startIndex + 1}-{Math.min(endIndex, filteredLogs.length)} of {filteredLogs.length} log severities
        {totalPages > 1 && ` (Page ${currentPage} of ${totalPages})`}
      </p>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '15px', marginTop: '20px' }}>
        <div className="stat-card">
          <div className="stat-label">Total Severities</div>
          <div className="stat-value">{filteredLogs.length}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Total Log Records</div>
          <div className="stat-value">{totalSamples.toLocaleString()}</div>
        </div>
      </div>

      <h3 style={{ marginTop: '20px', marginBottom: '12px' }}>Log Severity Breakdown</h3>
      
      <table>
        <thead>
          <tr>
            <th>Severity</th>
            <th>Sample Count</th>
            <th>Percentage</th>
            <th>Attributes</th>
            <th>Max Cardinality</th>
            <th>Services</th>
          </tr>
        </thead>
        <tbody>
          {currentLogs
            .sort((a, b) => b.sample_count - a.sample_count)
            .map((log, i) => {
              const maxCard = getMaxCardinality(log)
              const attrCount = log.attribute_keys ? Object.keys(log.attribute_keys).length : 0
              const serviceCount = log.services ? Object.keys(log.services).length : 0
              const percentage = totalSamples > 0 ? (log.sample_count / totalSamples * 100) : 0
              
              return (
                <tr key={i}>
                  <td>
                    <span 
                      className="detail-link"
                      onClick={() => onViewDetails('logs', log.severity)}
                      style={{ 
                        fontWeight: 'bold',
                        color: getSeverityColor(log.severity)
                      }}
                    >
                      {log.severity}
                    </span>
                  </td>
                  <td>{log.sample_count.toLocaleString()}</td>
                  <td>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                      <div style={{ 
                        flex: '0 0 60px',
                        height: '8px',
                        background: '#e0e0e0',
                        borderRadius: '4px',
                        overflow: 'hidden'
                      }}>
                        <div style={{
                          width: `${percentage}%`,
                          height: '100%',
                          background: getSeverityColor(log.severity),
                          transition: 'width 0.3s'
                        }}></div>
                      </div>
                      <span>{percentage.toFixed(1)}%</span>
                    </div>
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
          
          <span style={{ color: '#666' }}>
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

      {filteredLogs.length === 0 && (
        <p style={{ textAlign: 'center', padding: '20px', color: '#666' }}>
          No logs match the current filters
        </p>
      )}
    </div>
  )
}

export default LogsView
