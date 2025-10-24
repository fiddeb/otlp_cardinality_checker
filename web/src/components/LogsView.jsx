import { useState, useEffect } from 'react'

function LogsView({ onViewDetails }) {
  const [logs, setLogs] = useState([])
  const [patterns, setPatterns] = useState([])
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
        const logsData = result.data || []
        setLogs(logsData)
        
        // Extract all patterns from all severities
        const allPatterns = []
        logsData.forEach(log => {
          if (log.body_templates) {
            log.body_templates.forEach(tmpl => {
              allPatterns.push({
                ...tmpl,
                severity: log.severity,
                totalSeveritySamples: log.sample_count
              })
            })
          }
        })
        
        setPatterns(allPatterns)
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

  const filteredPatterns = patterns.filter(pattern => {
    if (pattern.count < filter.minSamples) return false
    return true
  })

  const totalPages = Math.ceil(filteredPatterns.length / itemsPerPage)
  const startIndex = (currentPage - 1) * itemsPerPage
  const endIndex = startIndex + itemsPerPage
  const currentPatterns = filteredPatterns.slice(startIndex, endIndex)

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

  const totalSamples = logs.reduce((sum, log) => sum + log.sample_count, 0)
  const totalPatternOccurrences = patterns.reduce((sum, p) => sum + p.count, 0)

  return (
    <div className="card">
      <h2>Log Message Patterns</h2>
      
      <div className="filter-group">
        <div className="threshold-input">
          <label>Min Pattern Count:</label>
          <input 
            type="number" 
            value={filter.minSamples} 
            onChange={(e) => setFilter({...filter, minSamples: Number(e.target.value)})}
            min="0"
          />
        </div>
      </div>

      <p style={{ marginTop: '10px', color: '#666' }}>
        Showing {startIndex + 1}-{Math.min(endIndex, filteredPatterns.length)} of {filteredPatterns.length} patterns
        {totalPages > 1 && ` (Page ${currentPage} of ${totalPages})`}
      </p>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '15px', marginTop: '20px' }}>
        <div className="stat-card">
          <div className="stat-label">Unique Patterns</div>
          <div className="stat-value">{filteredPatterns.length}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Total Logs</div>
          <div className="stat-value">{totalSamples.toLocaleString()}</div>
        </div>
      </div>

      <h3 style={{ marginTop: '20px', marginBottom: '12px' }}>Message Patterns</h3>
      
      <table>
        <thead>
          <tr>
            <th style={{ width: '40%' }}>Pattern Template</th>
            <th>Severity</th>
            <th>Count</th>
            <th>% of Severity</th>
            <th>Example</th>
          </tr>
        </thead>
        <tbody>
          {currentPatterns
            .sort((a, b) => b.count - a.count)
            .map((pattern, i) => {
              return (
                <tr key={i}>
                  <td>
                    <code style={{ 
                      fontSize: '13px',
                      wordBreak: 'break-word',
                      display: 'block',
                      padding: '8px',
                      background: '#f5f5f5',
                      borderRadius: '4px'
                    }}>
                      {pattern.template}
                    </code>
                  </td>
                  <td>
                    <span 
                      className="detail-link"
                      onClick={() => onViewDetails('logs', pattern.severity)}
                      style={{ 
                        fontWeight: 'bold',
                        color: getSeverityColor(pattern.severity)
                      }}
                    >
                      {pattern.severity}
                    </span>
                  </td>
                  <td>{pattern.count.toLocaleString()}</td>
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
                          width: `${pattern.percentage}%`,
                          height: '100%',
                          background: getSeverityColor(pattern.severity),
                          transition: 'width 0.3s'
                        }}></div>
                      </div>
                      <span>{pattern.percentage.toFixed(1)}%</span>
                    </div>
                  </td>
                  <td style={{ 
                    fontSize: '12px',
                    color: '#666',
                    fontStyle: 'italic',
                    maxWidth: '300px',
                    wordBreak: 'break-word'
                  }}>
                    {pattern.example}
                  </td>
                </tr>
              )
            })}
        </tbody>
      </table>      {totalPages > 1 && (
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

      {filteredPatterns.length === 0 && (
        <p style={{ textAlign: 'center', padding: '20px', color: '#666' }}>
          No patterns match the current filters
        </p>
      )}
    </div>
  )
}

export default LogsView
