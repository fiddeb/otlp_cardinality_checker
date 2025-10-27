import { useState, useEffect } from 'react'

function Dashboard({ onViewService }) {
  const [stats, setStats] = useState(null)
  const [services, setServices] = useState([])
  const [serviceStats, setServiceStats] = useState({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    // First: Load just counts for quick initial render
    Promise.all([
      fetch('/api/v1/metrics?limit=1').then(r => r.json()),
      fetch('/api/v1/spans?limit=1').then(r => r.json()),
      fetch('/api/v1/logs?limit=1').then(r => r.json()),
      fetch('/api/v1/services').then(r => r.json()),
    ])
      .then(([metrics, spans, logs, services]) => {
        // Calculate total log count from all severities (use total from API)
        const totalLogCount = logs.total || 0
        
        setStats({
          metrics: metrics.total || 0,
          spans: spans.total || 0,
          logs: totalLogCount,
        })
        setServices(services.data || [])
        setLoading(false)
        
        // Second: Load service stats in background (lazy load)
        loadServiceStats()
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [])

  const loadServiceStats = async () => {
    try {
      // Load in smaller batches with pagination to avoid overwhelming the API
      const [allMetrics, allSpans, allLogs] = await Promise.all([
        fetch('/api/v1/metrics?limit=1000').then(r => r.json()),
        fetch('/api/v1/spans?limit=1000').then(r => r.json()),
        fetch('/api/v1/logs?limit=1000').then(r => r.json()),
      ])
      
      // Calculate service statistics
      const stats = {}
      
      allMetrics.data?.forEach(metric => {
        if (metric.services) {
          Object.entries(metric.services).forEach(([service, count]) => {
            if (!stats[service]) stats[service] = { metrics: 0, spans: 0, logs: 0, total: 0 }
            stats[service].metrics += count
            stats[service].total += count
          })
        }
      })
      
      allSpans.data?.forEach(span => {
        if (span.services) {
          Object.entries(span.services).forEach(([service, count]) => {
            if (!stats[service]) stats[service] = { metrics: 0, spans: 0, logs: 0, total: 0 }
            stats[service].spans += count
            stats[service].total += count
          })
        }
      })
        
      allLogs.data?.forEach(log => {
        if (log.services) {
          Object.entries(log.services).forEach(([service, count]) => {
            if (!stats[service]) stats[service] = { metrics: 0, spans: 0, logs: 0, total: 0 }
            stats[service].logs += count
            stats[service].total += count
          })
        }
      })
        
      setServiceStats(stats)
    } catch (err) {
      console.error('Failed to load service stats:', err)
    }
  }

  if (loading) return <div className="loading">Loading...</div>
  if (error) return <div className="error">Error: {error}</div>

  return (
    <>
      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-value">{stats?.metrics || 0}</div>
          <div className="stat-label">Metrics</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{stats?.spans || 0}</div>
          <div className="stat-label">Spans</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{(stats?.logs || 0).toLocaleString()}</div>
          <div className="stat-label">Total Logs</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">{services?.length || 0}</div>
          <div className="stat-label">Services</div>
        </div>
      </div>

      <div className="card">
        <h2>Top 10 Services by Sample Volume</h2>
        <table>
          <thead>
            <tr>
              <th>Service</th>
              <th>Total Samples</th>
              <th>Metrics</th>
              <th>Spans</th>
              <th>Logs</th>
            </tr>
          </thead>
          <tbody>
            {Object.entries(serviceStats)
              .sort((a, b) => b[1].total - a[1].total)
              .slice(0, 10)
              .map(([service, stats]) => (
                <tr key={service}>
                  <td>
                    <span 
                      className="detail-link"
                      onClick={() => onViewService(service)}
                    >
                      {service}
                    </span>
                  </td>
                  <td><strong>{stats.total.toLocaleString()}</strong></td>
                  <td>{stats.metrics.toLocaleString()}</td>
                  <td>{stats.spans.toLocaleString()}</td>
                  <td>{stats.logs.toLocaleString()}</td>
                </tr>
              ))}
          </tbody>
        </table>
        {Object.keys(serviceStats).length === 0 && (
          <p style={{ textAlign: 'center', padding: '20px', color: '#666' }}>
            No service data available
          </p>
        )}
      </div>
    </>
  )
}

export default Dashboard
