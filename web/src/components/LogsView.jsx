import { useState, useEffect } from 'react'

function LogsView({ onViewServiceDetails }) {
  const [services, setServices] = useState([])
  const [expandedService, setExpandedService] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [filter, setFilter] = useState({
    minSamples: 0,
  })

  const itemsPerPage = 100

  useEffect(() => {
    fetch('/api/v1/logs/by-service?limit=1000')
      .then(r => r.json())
      .then(result => {
        const servicesData = result.data || []
        setServices(servicesData)
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [])

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

  if (loading) return <div className="loading">Loading logs...</div>
  if (error) return <div className="error">Error loading logs: {error}</div>
  if (!services || services.length === 0) return <div className="error">No logs found</div>

  const totalSamples = services.reduce((sum, svc) => sum + svc.sample_count, 0)

  // Apply filters
  const filteredServices = (services || []).filter(svc => {
    if (svc.sample_count < filter.minSamples) return false
    return true
  })

  // Group services by service_name
  const serviceGroups = {}
  filteredServices.forEach(svc => {
    if (!serviceGroups[svc.service_name]) {
      serviceGroups[svc.service_name] = []
    }
    serviceGroups[svc.service_name].push(svc)
  })

  const uniqueServices = Object.keys(serviceGroups).sort()
  const totalPages = Math.ceil(uniqueServices.length / itemsPerPage)
  const startIndex = (currentPage - 1) * itemsPerPage
  const endIndex = startIndex + itemsPerPage
  const currentServices = uniqueServices.slice(startIndex, endIndex)

  return (
    <div className="card">
      <h2>Log Services</h2>
      
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
        Showing {startIndex + 1}-{Math.min(endIndex, uniqueServices.length)} of {uniqueServices.length} services
        {totalPages > 1 && ` (Page ${currentPage} of ${totalPages})`}
      </p>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '15px', marginTop: '20px' }}>
        <div className="stat-card">
          <div className="stat-label">Total Services</div>
          <div className="stat-value">{uniqueServices.length}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Service×Severity Combos</div>
          <div className="stat-value">{services.length}</div>
        </div>
        <div className="stat-card">
          <div className="stat-label">Total Log Messages</div>
          <div className="stat-value">{totalSamples.toLocaleString()}</div>
        </div>
      </div>

      <h3 style={{ marginTop: '20px', marginBottom: '12px' }}>Services</h3>
      
      <div style={{ marginTop: '15px' }}>
        {currentServices.map((serviceName) => {
          const severities = serviceGroups[serviceName]
          const totalForService = severities.reduce((sum, s) => sum + s.sample_count, 0)
          const isExpanded = expandedService === serviceName

          return (
            <div key={serviceName} style={{ 
              marginBottom: '10px',
              border: '1px solid var(--border-color)',
              borderRadius: '6px',
              overflow: 'hidden'
            }}>
              <div
                onClick={() => setExpandedService(isExpanded ? null : serviceName)}
                style={{
                  padding: '12px 16px',
                  background: isExpanded ? 'var(--bg-secondary)' : 'var(--bg-primary)',
                  cursor: 'pointer',
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  fontWeight: '500'
                }}
              >
                <span>
                  {isExpanded ? '▼' : '▶'} {serviceName}
                </span>
                <span className="template-count-text">
                  {severities.length} severities · {totalForService.toLocaleString()} logs
                </span>
              </div>

              {isExpanded && (
                <div style={{ padding: '0' }}>
                  <table style={{ width: '100%', margin: 0 }}>
                    <thead>
                      <tr>
                        <th>Severity</th>
                        <th>Log Count</th>
                        <th>Action</th>
                      </tr>
                    </thead>
                    <tbody>
                      {severities
                        .sort((a, b) => b.sample_count - a.sample_count)
                        .map((svc, i) => (
                          <tr key={i}>
                            <td>
                              <span style={{ 
                                fontWeight: 'bold',
                                color: getSeverityColor(svc.severity)
                              }}>
                                {svc.severity}
                              </span>
                            </td>
                            <td>{svc.sample_count.toLocaleString()}</td>
                            <td>
                              <button 
                                onClick={() => onViewServiceDetails && onViewServiceDetails(serviceName, svc.severity)}
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
                                View Patterns
                              </button>
                            </td>
                          </tr>
                        ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )
        })}
      </div>

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

      {currentServices.length === 0 && uniqueServices.length === 0 && (
        <p style={{ textAlign: 'center', padding: '20px' }} className="template-count-text">
          No logs match the current filters
        </p>
      )}
    </div>
  )
}

export default LogsView
