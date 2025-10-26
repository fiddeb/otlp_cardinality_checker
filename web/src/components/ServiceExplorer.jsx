import { useState, useEffect } from 'react'

function ServiceExplorer({ serviceName, onBack, onViewDetails }) {
  const [overview, setOverview] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  // Pagination state
  const [metricsPage, setMetricsPage] = useState(1)
  const [spansPage, setSpansPage] = useState(1)
  const [logsPage, setLogsPage] = useState(1)
  const itemsPerPage = 100

  useEffect(() => {
    fetch(`/api/v1/services/${encodeURIComponent(serviceName)}/overview`)
      .then(r => r.json())
      .then(data => {
        setOverview(data)
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [serviceName])

  if (loading) return <div className="loading">Loading...</div>
  if (error) return <div className="error">Error: {error}</div>

  // Pagination helpers
  const paginate = (items, page) => {
    const start = (page - 1) * itemsPerPage
    const end = start + itemsPerPage
    return items.slice(start, end)
  }

  const totalPages = (items) => Math.ceil(items.length / itemsPerPage)

  const paginatedMetrics = paginate(overview?.metrics || [], metricsPage)
  const paginatedSpans = paginate(overview?.spans || [], spansPage)
  const paginatedLogs = paginate(overview?.logs || [], logsPage)

  return (
    <>
      <button className="back-button" onClick={onBack}>← Back</button>

      <div className="card">
        <h2>Service: {serviceName}</h2>
        
        <h3 style={{ marginTop: '20px', marginBottom: '12px' }}>Metrics ({overview?.metrics?.length || 0})</h3>
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Type</th>
              <th>Samples</th>
            </tr>
          </thead>
          <tbody>
            {paginatedMetrics.map(m => (
              <tr key={m.name}>
                <td>
                  <span 
                    className="detail-link"
                    onClick={() => onViewDetails('metrics', m.name)}
                  >
                    {m.name}
                  </span>
                </td>
                <td>{m.type}</td>
                <td>{m.sample_count}</td>
              </tr>
            ))}
          </tbody>
        </table>
        {(overview?.metrics?.length || 0) > itemsPerPage && (
          <div className="pagination">
            <button 
              onClick={() => setMetricsPage(p => Math.max(1, p - 1))}
              disabled={metricsPage === 1}
            >
              Previous
            </button>
            <span className="template-count-text">
              Page {metricsPage} of {totalPages(overview?.metrics || [])} 
              (Showing {(metricsPage - 1) * itemsPerPage + 1}-{Math.min(metricsPage * itemsPerPage, overview?.metrics?.length || 0)} of {overview?.metrics?.length || 0})
            </span>
            <button 
              onClick={() => setMetricsPage(p => Math.min(totalPages(overview?.metrics || []), p + 1))}
              disabled={metricsPage === totalPages(overview?.metrics || [])}
            >
              Next
            </button>
          </div>
        )}

        <h3 style={{ marginTop: '20px', marginBottom: '12px' }}>Spans ({overview?.spans?.length || 0})</h3>
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Kind</th>
              <th>Samples</th>
            </tr>
          </thead>
          <tbody>
            {paginatedSpans.map(s => (
              <tr key={s.name}>
                <td>
                  <span 
                    className="detail-link"
                    onClick={() => onViewDetails('spans', s.name)}
                  >
                    {s.name}
                  </span>
                </td>
                <td>{s.kind}</td>
                <td>{s.sample_count}</td>
              </tr>
            ))}
          </tbody>
        </table>
        {(overview?.spans?.length || 0) > itemsPerPage && (
          <div className="pagination">
            <button 
              onClick={() => setSpansPage(p => Math.max(1, p - 1))}
              disabled={spansPage === 1}
            >
              Previous
            </button>
            <span className="template-count-text">
              Page {spansPage} of {totalPages(overview?.spans || [])} 
              (Showing {(spansPage - 1) * itemsPerPage + 1}-{Math.min(spansPage * itemsPerPage, overview?.spans?.length || 0)} of {overview?.spans?.length || 0})
            </span>
            <button 
              onClick={() => setSpansPage(p => Math.min(totalPages(overview?.spans || []), p + 1))}
              disabled={spansPage === totalPages(overview?.spans || [])}
            >
              Next
            </button>
          </div>
        )}

        <h3 style={{ marginTop: '20px', marginBottom: '12px' }}>Logs ({overview?.logs?.length || 0})</h3>
        <table>
          <thead>
            <tr>
              <th>Severity</th>
              <th>Samples</th>
            </tr>
          </thead>
          <tbody>
            {paginatedLogs.map(l => (
              <tr key={l.severity}>
                <td>
                  <span 
                    className="detail-link"
                    onClick={() => onViewDetails('logs', l.severity)}
                  >
                    {l.severity}
                  </span>
                </td>
                <td>{l.sample_count}</td>
              </tr>
            ))}
          </tbody>
        </table>
        {(overview?.logs?.length || 0) > itemsPerPage && (
          <div className="pagination">
            <button 
              onClick={() => setLogsPage(p => Math.max(1, p - 1))}
              disabled={logsPage === 1}
            >
              Previous
            </button>
            <span className="template-count-text">
              Page {logsPage} of {totalPages(overview?.logs || [])} 
              (Showing {(logsPage - 1) * itemsPerPage + 1}-{Math.min(logsPage * itemsPerPage, overview?.logs?.length || 0)} of {overview?.logs?.length || 0})
            </span>
            <button 
              onClick={() => setLogsPage(p => Math.min(totalPages(overview?.logs || []), p + 1))}
              disabled={logsPage === totalPages(overview?.logs || [])}
            >
              Next
            </button>
          </div>
        )}
      </div>
    </>
  )
}

export default ServiceExplorer
