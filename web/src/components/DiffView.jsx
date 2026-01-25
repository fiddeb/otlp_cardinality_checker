import { useState, useEffect } from 'react'

function DiffView({ initialFrom, onBack }) {
  const [sessions, setSessions] = useState([])
  const [fromSession, setFromSession] = useState(initialFrom || '')
  const [toSession, setToSession] = useState('')
  const [loading, setLoading] = useState(false)
  const [loadingSessions, setLoadingSessions] = useState(true)
  const [error, setError] = useState(null)
  const [diff, setDiff] = useState(null)
  const [expandedChanges, setExpandedChanges] = useState({})
  const [signalFilter, setSignalFilter] = useState('all')
  const [severityFilter, setSeverityFilter] = useState('all')
  const [serviceFilter, setServiceFilter] = useState('all')

  useEffect(() => {
    fetchSessions()
  }, [])

  const fetchSessions = async () => {
    try {
      setLoadingSessions(true)
      const response = await fetch('/api/v1/sessions')
      if (!response.ok) throw new Error('Failed to fetch sessions')
      const data = await response.json()
      setSessions(data.sessions || [])
    } catch (err) {
      setError(err.message)
    } finally {
      setLoadingSessions(false)
    }
  }

  const fetchDiff = async () => {
    if (!fromSession || !toSession) {
      setError('Please select both sessions to compare')
      return
    }

    if (fromSession === toSession) {
      setError('Please select different sessions to compare')
      return
    }

    setLoading(true)
    setError(null)
    setDiff(null)

    try {
      let url = `/api/v1/sessions/diff?from=${encodeURIComponent(fromSession)}&to=${encodeURIComponent(toSession)}`
      if (severityFilter !== 'all') {
        url += `&min_severity=${severityFilter}`
      }

      const response = await fetch(url)
      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.error || 'Failed to compute diff')
      }

      const data = await response.json()
      setDiff(data)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const toggleExpand = (changeId) => {
    setExpandedChanges((prev) => ({
      ...prev,
      [changeId]: !prev[changeId],
    }))
  }

  const getChangeTypeClass = (type) => {
    switch (type) {
      case 'added': return 'change-added'
      case 'removed': return 'change-removed'
      case 'changed': return 'change-modified'
      default: return ''
    }
  }

  const getSeverityClass = (severity) => {
    switch (severity) {
      case 'critical': return 'severity-critical'
      case 'warning': return 'severity-warning'
      case 'info': return 'severity-info'
      default: return ''
    }
  }

  const getSignalBadgeClass = (signalType) => {
    switch (signalType) {
      case 'metric': return 'signal-metric'
      case 'span': return 'signal-span'
      case 'log': return 'signal-log'
      default: return ''
    }
  }

  const formatChange = (change) => {
    const id = `${change.signal_type}-${change.name}-${change.type}`
    const isExpanded = expandedChanges[id]

    return (
      <div key={id} className={`change-item ${getChangeTypeClass(change.type)}`}>
        <div className="change-header" onClick={() => toggleExpand(id)}>
          <span className="expand-icon">{isExpanded ? '▼' : '▶'}</span>
          <span className={`signal-badge ${getSignalBadgeClass(change.signal_type)}`}>
            {change.signal_type}
          </span>
          <span className="change-name">{change.name}</span>
          <span className={`change-type-badge ${change.type}`}>{change.type}</span>
          <span className={`severity-badge ${getSeverityClass(change.severity)}`}>
            {change.severity}
          </span>
        </div>
        {isExpanded && (
          <div className="change-details">
            {change.metadata && (
              <div className="change-metadata">
                {Object.entries(change.metadata).map(([key, value]) => (
                  <div key={key} className="metadata-item">
                    <span className="metadata-key">{key}:</span>
                    <span className="metadata-value">{JSON.stringify(value)}</span>
                  </div>
                ))}
              </div>
            )}
            {change.details && change.details.length > 0 && (
              <div className="field-changes">
                <h5>Field Changes</h5>
                <table className="field-changes-table">
                  <thead>
                    <tr>
                      <th>Field</th>
                      <th>From</th>
                      <th>To</th>
                      <th>Change %</th>
                      <th>Severity</th>
                    </tr>
                  </thead>
                  <tbody>
                    {change.details.map((detail, idx) => (
                      <tr key={idx}>
                        <td><code>{detail.field}</code></td>
                        <td>{detail.from !== null ? JSON.stringify(detail.from) : '-'}</td>
                        <td>{detail.to !== null ? JSON.stringify(detail.to) : '-'}</td>
                        <td>
                          {detail.change_pct !== undefined && detail.change_pct !== 0
                            ? `${detail.change_pct > 0 ? '+' : ''}${detail.change_pct.toFixed(1)}%`
                            : '-'}
                        </td>
                        <td>
                          <span className={`severity-badge ${getSeverityClass(detail.severity)}`}>
                            {detail.severity}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            {change.message && (
              <div className="change-message">{change.message}</div>
            )}
          </div>
        )}
      </div>
    )
  }

  const getAllChanges = () => {
    if (!diff?.changes) return []

    const all = []

    // Collect all changes from all signal types
    const signalTypes = ['metrics', 'spans', 'logs']
    for (const signalType of signalTypes) {
      const signalChanges = diff.changes[signalType]
      if (signalChanges) {
        if (signalChanges.added) all.push(...signalChanges.added)
        if (signalChanges.removed) all.push(...signalChanges.removed)
        if (signalChanges.changed) all.push(...signalChanges.changed)
      }
    }

    // Apply signal filter
    let filtered = all
    if (signalFilter !== 'all') {
      filtered = filtered.filter((c) => c.signal_type === signalFilter)
    }

    // Apply service filter
    if (serviceFilter !== 'all') {
      filtered = filtered.filter((c) => {
        // Check metadata.services
        if (c.metadata?.services && serviceFilter in c.metadata.services) {
          return true
        }
        // Check name pattern
        return c.name.startsWith(serviceFilter + '.')
      })
    }

    // Sort by severity (critical first)
    const severityOrder = { critical: 0, warning: 1, info: 2 }
    filtered.sort((a, b) => {
      const severityDiff = (severityOrder[a.severity] || 3) - (severityOrder[b.severity] || 3)
      if (severityDiff !== 0) return severityDiff
      return a.name.localeCompare(b.name)
    })

    return filtered
  }

  const getSummary = () => {
    if (!diff?.summary) return null

    const { summary } = diff
    return {
      total: summary.total_changes || 0,
      added: summary.added || 0,
      removed: summary.removed || 0,
      changed: summary.changed || 0,
      critical: summary.critical || 0,
      warning: summary.warning || 0,
    }
  }

  // Extract unique services from all changes for the service filter
  const getAvailableServices = () => {
    if (!diff?.changes) return []

    const services = new Set()
    const signalTypes = ['metrics', 'spans', 'logs']
    
    for (const signalType of signalTypes) {
      const signalChanges = diff.changes[signalType]
      if (signalChanges) {
        const allChanges = [
          ...(signalChanges.added || []),
          ...(signalChanges.removed || []),
          ...(signalChanges.changed || []),
        ]
        for (const change of allChanges) {
          // Check metadata.services if available
          if (change.metadata?.services) {
            Object.keys(change.metadata.services).forEach((s) => services.add(s))
          }
          // Also check the name pattern (service.name format)
          const nameParts = change.name.split('.')
          if (nameParts.length > 1) {
            services.add(nameParts[0])
          }
        }
      }
    }

    return Array.from(services).sort()
  }

  if (loadingSessions) return <div className="loading">Loading sessions...</div>

  return (
    <div className="diff-view">
      <button className="back-button" onClick={onBack}>
        ← Back to Sessions
      </button>

      <div className="card">
        <h2>Compare Sessions</h2>
        <p className="diff-subtitle">
          Select two sessions to compare and detect changes in metrics, spans, and logs.
        </p>

        <div className="diff-controls">
          <div className="session-selector">
            <label>From (baseline)</label>
            <select
              value={fromSession}
              onChange={(e) => setFromSession(e.target.value)}
            >
              <option value="">Select session...</option>
              {sessions.map((s) => (
                <option key={s.id} value={s.id}>{s.id}</option>
              ))}
            </select>
          </div>

          <div className="diff-arrow">→</div>

          <div className="session-selector">
            <label>To (comparison)</label>
            <select
              value={toSession}
              onChange={(e) => setToSession(e.target.value)}
            >
              <option value="">Select session...</option>
              {sessions.map((s) => (
                <option key={s.id} value={s.id}>{s.id}</option>
              ))}
            </select>
          </div>

          <button
            className="action-button primary"
            onClick={fetchDiff}
            disabled={!fromSession || !toSession || loading}
          >
            {loading ? 'Comparing...' : 'Compare'}
          </button>
        </div>
      </div>

      {error && <div className="error">{error}</div>}

      {diff && (
        <>
          {/* Summary */}
          <div className="diff-summary">
            <div className="summary-stat">
              <span className="summary-value">{getSummary()?.total || 0}</span>
              <span className="summary-label">Total Changes</span>
            </div>
            <div className="summary-stat added">
              <span className="summary-value">{getSummary()?.added || 0}</span>
              <span className="summary-label">Added</span>
            </div>
            <div className="summary-stat removed">
              <span className="summary-value">{getSummary()?.removed || 0}</span>
              <span className="summary-label">Removed</span>
            </div>
            <div className="summary-stat changed">
              <span className="summary-value">{getSummary()?.changed || 0}</span>
              <span className="summary-label">Changed</span>
            </div>
            {getSummary()?.critical > 0 && (
              <div className="summary-stat critical">
                <span className="summary-value">{getSummary()?.critical}</span>
                <span className="summary-label">Critical</span>
              </div>
            )}
            {getSummary()?.warning > 0 && (
              <div className="summary-stat warning">
                <span className="summary-value">{getSummary()?.warning}</span>
                <span className="summary-label">Warnings</span>
              </div>
            )}
          </div>

          {/* Filters */}
          <div className="diff-filters">
            <div className="filter-tabs signal-tabs">
              <button
                className={`filter-tab ${signalFilter === 'all' ? 'active' : ''}`}
                onClick={() => setSignalFilter('all')}
              >
                All
              </button>
              <button
                className={`filter-tab ${signalFilter === 'metric' ? 'active' : ''}`}
                onClick={() => setSignalFilter('metric')}
              >
                Metrics
              </button>
              <button
                className={`filter-tab ${signalFilter === 'span' ? 'active' : ''}`}
                onClick={() => setSignalFilter('span')}
              >
                Spans
              </button>
              <button
                className={`filter-tab ${signalFilter === 'log' ? 'active' : ''}`}
                onClick={() => setSignalFilter('log')}
              >
                Logs
              </button>
            </div>

            {getAvailableServices().length > 0 && (
              <div className="service-filter">
                <label>Service:</label>
                <select
                  value={serviceFilter}
                  onChange={(e) => setServiceFilter(e.target.value)}
                >
                  <option value="all">All Services</option>
                  {getAvailableServices().map((service) => (
                    <option key={service} value={service}>{service}</option>
                  ))}
                </select>
              </div>
            )}
          </div>

          {/* Changes list */}
          <div className="card changes-list">
            <h3>Changes</h3>
            {getAllChanges().length === 0 ? (
              <div className="no-changes">
                No changes detected between these sessions.
              </div>
            ) : (
              <div className="changes-container">
                {getAllChanges().map(formatChange)}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  )
}

export default DiffView
