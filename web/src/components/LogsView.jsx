import { useState, useEffect } from 'react'

function LogsView({ onViewTemplate }) {
  const [logs, setLogs] = useState([])
  const [selectedLog, setSelectedLog] = useState(null)
  const [templates, setTemplates] = useState([])
  const [loadingTemplates, setLoadingTemplates] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [filter, setFilter] = useState({
    minSamples: 0,
  })

  const itemsPerPage = 100

  useEffect(() => {
    fetch('/api/v1/logs?limit=100')
      .then(r => r.json())
      .then(result => {
        const logsData = result.data || []
        setLogs(logsData)
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [])

  const loadTemplates = async (severity) => {
    setLoadingTemplates(true)
    setSelectedLog(severity)
    try {
      const response = await fetch(`/api/v1/logs/${encodeURIComponent(severity)}`)
      const data = await response.json()
      setTemplates(data.body_templates || [])
    } catch (err) {
      console.error('Failed to load templates:', err)
      setTemplates([])
    } finally {
      setLoadingTemplates(false)
    }
  }

  const filteredLogs = (logs || []).filter(log => {
    if (log.sample_count < filter.minSamples) return false
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

  const getSeverityColor = (severity) => {
    const colors = {
      'ERROR': '#d32f2f',
      'Error': '#d32f2f',
      'WARN': '#f57c00',
      'Warning': '#f57c00',
      'INFO': '#1976d2',
      'Information': '#1976d2',
      'DEBUG': '#7b1fa2',
      'DEBUG2': '#7b1fa2',
      'Debug': '#7b1fa2',
      'TRACE': '#455a64',
      'Trace': '#455a64',
      'UNSET': '#999'
    }
    return colors[severity] || '#666'
  }

  if (loading) return <div className="loading">Loading...</div>
  if (error) return <div className="error">Error: {error}</div>

  const totalSamples = logs.reduce((sum, log) => sum + log.sample_count, 0)
  const totalTemplates = logs.reduce((sum, log) => sum + (log.template_count || 0), 0)

  return (
    <div className="card">
      <h2>Log Severities</h2>
      
      <div className="filter-group">
        <div className="threshold-input">
          <label>Min Sample Count:</label>
          <input 
            type="number" 
            value={filter.minSamples} 
            onChange={(e) => setFilter({...filter, minSamples: Number(e.target.value)})}
            min="0"
          />
        </div>
      </div>

      <p style={{ marginTop: '10px' }} className="template-count-text">
        Showing {startIndex + 1}-{Math.min(endIndex, filteredLogs.length)} of {filteredLogs.length} severities
        {totalPages > 1 && ` (Page ${currentPage} of ${totalPages})`}
      </p>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '15px', marginTop: '20px' }}>
        <div className="stat-card">
          <div className="stat-label">Total Severities</div>
          <div className="stat-value">{filteredLogs.length}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Total Logs</div>
          <div className="stat-value">{totalSamples.toLocaleString()}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Total Patterns</div>
          <div className="stat-value">{totalTemplates.toLocaleString()}</div>
        </div>
      </div>

      <h3 style={{ marginTop: '20px', marginBottom: '12px' }}>Log Severities</h3>
      
      <table>
        <thead>
          <tr>
            <th>Severity</th>
            <th>Log Samples</th>
            <th>Services</th>
            <th>Unique Patterns</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          {currentLogs
            .sort((a, b) => b.sample_count - a.sample_count)
            .map((log, i) => {
              const serviceCount = Object.keys(log.services || {}).length
              return (
                <tr key={i}>
                  <td>
                    <span 
                      style={{ 
                        fontWeight: 'bold',
                        color: getSeverityColor(log.severity)
                      }}
                    >
                      {log.severity}
                    </span>
                  </td>
                  <td>{log.sample_count.toLocaleString()}</td>
                  <td>{serviceCount}</td>
                  <td>{(log.template_count || 0).toLocaleString()}</td>
                  <td>
                    <button 
                      onClick={() => loadTemplates(log.severity)}
                      disabled={loadingTemplates && selectedLog === log.severity}
                      style={{
                        padding: '6px 12px',
                        fontSize: '13px',
                        cursor: 'pointer',
                        background: 'var(--primary-color)',
                        color: 'white',
                        border: 'none',
                        borderRadius: '4px'
                      }}
                    >
                      {loadingTemplates && selectedLog === log.severity ? 'Loading...' : 'View Patterns'}
                    </button>
                  </td>
                </tr>
              )
            })}
        </tbody>
      </table>

      {selectedLog && templates.length > 0 && (
        <div style={{ marginTop: '30px', borderTop: '2px solid var(--border-color)', paddingTop: '20px' }}>
          <h3>Patterns for {selectedLog} ({templates.length.toLocaleString()} total)</h3>
          <p className="template-count-text">
            Showing top {Math.min(100, templates.length)} patterns
          </p>
          <table style={{ marginTop: '15px' }}>
            <thead>
              <tr>
                <th style={{ width: '50%' }}>Pattern Template</th>
                <th>Count</th>
                <th>Percentage</th>
                <th>Example</th>
              </tr>
            </thead>
            <tbody>
              {templates.slice(0, 100).map((tmpl, i) => (
                <tr 
                  key={i}
                  onClick={() => onViewTemplate && onViewTemplate(selectedLog, tmpl.template)}
                  style={{ cursor: onViewTemplate ? 'pointer' : 'default' }}
                  className={onViewTemplate ? 'clickable-row' : ''}
                >
                  <td>
                    <code className="template-code" style={{ 
                      fontSize: '13px',
                      wordBreak: 'break-word',
                      display: 'block',
                      padding: '8px',
                      background: 'var(--bg-tertiary)',
                      borderRadius: '4px'
                    }}>
                      {tmpl.template}
                    </code>
                  </td>
                  <td>{tmpl.count.toLocaleString()}</td>
                  <td>{tmpl.percentage.toFixed(1)}%</td>
                  <td className="template-count-text-small" style={{ 
                    fontStyle: 'italic',
                    maxWidth: '300px',
                    wordBreak: 'break-word'
                  }}>
                    {tmpl.example}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}      {totalPages > 1 && (
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

      {filteredPatterns.length === 0 && (
        <p style={{ textAlign: 'center', padding: '20px' }} className="template-count-text">
          No patterns match the current filters
        </p>
      )}
    </div>
  )
}

export default LogsView
