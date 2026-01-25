import { useState, useEffect, useRef } from 'react'

function SessionsView({ onCompare, currentSessionName, onSessionChange }) {
  const [sessions, setSessions] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [showSaveModal, setShowSaveModal] = useState(false)
  const [saving, setSaving] = useState(false)
  const [actionInProgress, setActionInProgress] = useState(null)
  const [progressInfo, setProgressInfo] = useState(null) // { title, description }
  const fileInputRef = useRef(null)

  // Save modal state
  const [saveName, setSaveName] = useState('')
  const [saveDescription, setSaveDescription] = useState('')
  const [saveError, setSaveError] = useState(null)

  useEffect(() => {
    fetchSessions()
  }, [])

  const fetchSessions = async () => {
    try {
      setLoading(true)
      const response = await fetch('/api/v1/sessions')
      if (!response.ok) {
        throw new Error('Failed to fetch sessions')
      }
      const data = await response.json()
      setSessions(data.sessions || [])
      setError(null)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleSave = async () => {
    if (!saveName.trim()) {
      setSaveError('Session name is required')
      return
    }

    // Validate name format (kebab-case or alphanumeric with dashes/underscores)
    if (!/^[a-zA-Z0-9][a-zA-Z0-9_-]*$/.test(saveName)) {
      setSaveError('Name must start with a letter/number and contain only letters, numbers, dashes, and underscores')
      return
    }

    setSaving(true)
    setSaveError(null)
    setProgressInfo({ title: 'Saving Session', description: `Creating snapshot "${saveName.trim()}"...` })

    try {
      const response = await fetch('/api/v1/sessions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: saveName.trim(),
          description: saveDescription.trim() || undefined,
        }),
      })

      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.error || 'Failed to save session')
      }

      // Success - close modal and refresh
      setShowSaveModal(false)
      setSaveName('')
      setSaveDescription('')
      fetchSessions()
    } catch (err) {
      setSaveError(err.message)
    } finally {
      setSaving(false)
      setProgressInfo(null)
    }
  }

  const handleLoad = async (sessionName) => {
    if (!confirm(`Load session "${sessionName}"? This will replace the current data.`)) {
      return
    }

    setActionInProgress(`loading-${sessionName}`)
    setProgressInfo({ title: 'Loading Session', description: `Restoring data from "${sessionName}"...` })
    try {
      const response = await fetch(`/api/v1/sessions/${encodeURIComponent(sessionName)}/load`, {
        method: 'POST',
      })

      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.error || 'Failed to load session')
      }

      // Notify parent of session change
      if (onSessionChange) {
        onSessionChange(sessionName)
      }
      
      alert(`Session "${sessionName}" loaded successfully. Refresh views to see updated data.`)
    } catch (err) {
      alert(`Error: ${err.message}`)
    } finally {
      setActionInProgress(null)
      setProgressInfo(null)
    }
  }

  const handleMerge = async (sessionName) => {
    if (!confirm(`Merge session "${sessionName}" into current data? This will combine the data additively.`)) {
      return
    }

    setActionInProgress(`merging-${sessionName}`)
    setProgressInfo({ title: 'Merging Session', description: `Combining data from "${sessionName}"...` })
    try {
      const response = await fetch(`/api/v1/sessions/${encodeURIComponent(sessionName)}/merge`, {
        method: 'POST',
      })

      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.error || 'Failed to merge session')
      }

      const result = await response.json()
      const merged = result.merged || {}
      alert(`Session "${sessionName}" merged successfully.\n\nMerged: ${merged.metrics || 0} metrics, ${merged.spans || 0} spans, ${merged.logs || 0} logs, ${merged.attributes || 0} attributes`)
    } catch (err) {
      alert(`Error: ${err.message}`)
    } finally {
      setActionInProgress(null)
      setProgressInfo(null)
    }
  }

  const handleDelete = async (sessionName) => {
    if (!confirm(`Delete session "${sessionName}"? This cannot be undone.`)) {
      return
    }

    setActionInProgress(`deleting-${sessionName}`)
    try {
      const response = await fetch(`/api/v1/sessions/${encodeURIComponent(sessionName)}`, {
        method: 'DELETE',
      })

      if (!response.ok && response.status !== 204) {
        const data = await response.json()
        throw new Error(data.error || 'Failed to delete session')
      }

      fetchSessions()
    } catch (err) {
      alert(`Error: ${err.message}`)
    } finally {
      setActionInProgress(null)
    }
  }

  const handleExport = (sessionName) => {
    // Use direct link download for large files (avoids fetch timeout/memory issues)
    const url = `/api/v1/sessions/${encodeURIComponent(sessionName)}/export`
    const a = document.createElement('a')
    a.href = url
    a.download = `${sessionName}.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
  }

  const handleImportClick = () => {
    fileInputRef.current?.click()
  }

  const handleImportFile = async (e) => {
    const file = e.target.files?.[0]
    if (!file) return

    setActionInProgress('importing')
    setProgressInfo({ title: 'Importing Session', description: `Processing "${file.name}"...` })
    try {
      const text = await file.text()
      const session = JSON.parse(text)

      const response = await fetch('/api/v1/sessions/import', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: text,
      })

      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.error || 'Failed to import session')
      }

      alert(`Session "${session.id || 'unknown'}" imported successfully`)
      fetchSessions()
    } catch (err) {
      alert(`Error: ${err.message}`)
    } finally {
      setActionInProgress(null)
      setProgressInfo(null)
      // Reset file input
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    }
  }

  const formatDate = (dateStr) => {
    if (!dateStr) return '-'
    const date = new Date(dateStr)
    return date.toLocaleString()
  }

  const formatSize = (bytes) => {
    if (!bytes) return '-'
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }

  if (loading) return <div className="loading">Loading sessions...</div>
  if (error) return <div className="error">Error: {error}</div>

  return (
    <>
      {/* Current session indicator */}
      {currentSessionName && (
        <div className="session-indicator">
          <span className="session-indicator-label">Active Session:</span>
          <span className="session-indicator-name">{currentSessionName}</span>
        </div>
      )}

      {/* Actions bar */}
      <div className="card sessions-actions">
        <div className="sessions-actions-row">
          <h2>Sessions</h2>
          <div className="sessions-buttons">
            <button
              className="action-button primary"
              onClick={() => setShowSaveModal(true)}
            >
              Save Current
            </button>
            <button
              className="action-button"
              onClick={handleImportClick}
              disabled={actionInProgress === 'importing'}
            >
              {actionInProgress === 'importing' ? 'Importing...' : 'Import'}
            </button>
            <input
              ref={fileInputRef}
              type="file"
              accept=".json"
              style={{ display: 'none' }}
              onChange={handleImportFile}
            />
            <button
              className="action-button"
              onClick={fetchSessions}
            >
              Refresh
            </button>
          </div>
        </div>
        <p className="sessions-subtitle">
          Save snapshots of the current telemetry state, compare sessions to detect changes, 
          and merge data collected at different times.
        </p>
      </div>

      {/* Sessions list */}
      <div className="card">
        {sessions.length === 0 ? (
          <div className="empty-state">
            <p>No saved sessions yet.</p>
            <p className="empty-hint">Use "Save Current" to create your first snapshot.</p>
          </div>
        ) : (
          <table className="sessions-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Description</th>
                <th>Created</th>
                <th>Size</th>
                <th title="Total data points received">Data Points</th>
                <th title="Total spans received">Spans</th>
                <th title="Total log messages received">Log Messages</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {sessions.map((session) => (
                <tr key={session.id} className={currentSessionName === session.id ? 'active-session-row' : ''}>
                  <td>
                    <span className="session-name">{session.id}</span>
                  </td>
                  <td className="session-description">
                    {session.description || <span className="text-muted">-</span>}
                  </td>
                  <td className="session-date">{formatDate(session.created)}</td>
                  <td>{formatSize(session.size_bytes)}</td>
                  <td>{session.stats?.metrics_count || 0}</td>
                  <td>{session.stats?.spans_count || 0}</td>
                  <td>{session.stats?.logs_count || 0}</td>
                  <td className="session-actions">
                    <button
                      className="action-btn load-btn"
                      onClick={() => handleLoad(session.id)}
                      disabled={actionInProgress?.startsWith('loading')}
                      title="Load session (replaces current data)"
                    >
                      {actionInProgress === `loading-${session.id}` ? '...' : 'Load'}
                    </button>
                    <button
                      className="action-btn merge-btn"
                      onClick={() => handleMerge(session.id)}
                      disabled={actionInProgress?.startsWith('merging')}
                      title="Merge session into current data"
                    >
                      {actionInProgress === `merging-${session.id}` ? '...' : 'Merge'}
                    </button>
                    <button
                      className="action-btn compare-btn"
                      onClick={() => onCompare && onCompare(session.id)}
                      title="Compare with another session"
                    >
                      Compare
                    </button>
                    <button
                      className="action-btn export-btn"
                      onClick={() => handleExport(session.id)}
                      title="Export session as JSON"
                    >
                      Export
                    </button>
                    <button
                      className="action-btn delete-btn"
                      onClick={() => handleDelete(session.id)}
                      disabled={actionInProgress?.startsWith('deleting')}
                      title="Delete session"
                    >
                      {actionInProgress === `deleting-${session.id}` ? '...' : 'Delete'}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Save Modal */}
      {showSaveModal && (
        <div className="modal-overlay" onClick={() => setShowSaveModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h3>Save Current State as Session</h3>
            <div className="modal-content">
              <div className="form-group">
                <label htmlFor="session-name">Session Name *</label>
                <input
                  id="session-name"
                  type="text"
                  value={saveName}
                  onChange={(e) => setSaveName(e.target.value)}
                  placeholder="e.g., pre-deploy-v2.5"
                  autoFocus
                />
              </div>
              <div className="form-group">
                <label htmlFor="session-description">Description (optional)</label>
                <textarea
                  id="session-description"
                  value={saveDescription}
                  onChange={(e) => setSaveDescription(e.target.value)}
                  placeholder="Brief description of what this snapshot contains..."
                  rows={3}
                />
              </div>
              {saveError && <div className="modal-error">{saveError}</div>}
            </div>
            <div className="modal-actions">
              <button
                className="action-button"
                onClick={() => setShowSaveModal(false)}
                disabled={saving}
              >
                Cancel
              </button>
              <button
                className="action-button primary"
                onClick={handleSave}
                disabled={saving || !saveName.trim()}
              >
                {saving ? 'Saving...' : 'Save Session'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Progress Overlay */}
      {progressInfo && (
        <div className="progress-overlay">
          <div className="progress-modal">
            <div className="progress-spinner"></div>
            <div className="progress-title">{progressInfo.title}</div>
            <div className="progress-description">{progressInfo.description}</div>
          </div>
        </div>
      )}
    </>
  )
}

export default SessionsView
