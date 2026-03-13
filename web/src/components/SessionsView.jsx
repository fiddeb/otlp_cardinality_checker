import { useState, useEffect, useRef } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

function SessionsView({ onCompare, currentSessionName, onSessionChange }) {
  const [sessions, setSessions] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [showSaveModal, setShowSaveModal] = useState(false)
  const [saving, setSaving] = useState(false)
  const [actionInProgress, setActionInProgress] = useState(null)
  const fileInputRef = useRef(null)

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
      if (!response.ok) throw new Error('Failed to fetch sessions')
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
    if (!/^[a-zA-Z0-9][a-zA-Z0-9_-]*$/.test(saveName)) {
      setSaveError('Name must start with a letter/number and contain only letters, numbers, dashes, and underscores')
      return
    }

    setSaving(true)
    setSaveError(null)
    try {
      const response = await fetch('/api/v1/sessions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: saveName.trim(), description: saveDescription.trim() || undefined }),
      })
      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.error || 'Failed to save session')
      }
      setShowSaveModal(false)
      setSaveName('')
      setSaveDescription('')
      fetchSessions()
    } catch (err) {
      setSaveError(err.message)
    } finally {
      setSaving(false)
    }
  }

  const handleLoad = async (sessionName) => {
    if (!confirm(`Load session "${sessionName}"? This will replace the current data.`)) return
    setActionInProgress(`loading-${sessionName}`)
    try {
      const response = await fetch(`/api/v1/sessions/${encodeURIComponent(sessionName)}/load`, { method: 'POST' })
      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.error || 'Failed to load session')
      }
      if (onSessionChange) onSessionChange(sessionName)
      alert(`Session "${sessionName}" loaded successfully. Refresh views to see updated data.`)
    } catch (err) {
      alert(`Error: ${err.message}`)
    } finally {
      setActionInProgress(null)
    }
  }

  const handleMerge = async (sessionName) => {
    if (!confirm(`Merge session "${sessionName}" into current data?`)) return
    setActionInProgress(`merging-${sessionName}`)
    try {
      const response = await fetch(`/api/v1/sessions/${encodeURIComponent(sessionName)}/merge`, { method: 'POST' })
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
    }
  }

  const handleDelete = async (sessionName) => {
    if (!confirm(`Delete session "${sessionName}"? This cannot be undone.`)) return
    setActionInProgress(`deleting-${sessionName}`)
    try {
      const response = await fetch(`/api/v1/sessions/${encodeURIComponent(sessionName)}`, { method: 'DELETE' })
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
    const a = document.createElement('a')
    a.href = `/api/v1/sessions/${encodeURIComponent(sessionName)}/export`
    a.download = `${sessionName}.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
  }

  const handleImportClick = () => fileInputRef.current?.click()

  const handleImportFile = async (e) => {
    const file = e.target.files?.[0]
    if (!file) return
    setActionInProgress('importing')
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
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const formatDate = (dateStr) => {
    if (!dateStr) return '-'
    return new Date(dateStr).toLocaleString()
  }

  const formatSize = (bytes) => {
    if (!bytes) return '-'
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }

  if (loading) {
    return (
      <div className="flex flex-col gap-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Sessions</h1>
          <p className="text-muted-foreground">Saved telemetry snapshots</p>
        </div>
        <Skeleton className="h-[400px] w-full" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col gap-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Sessions</h1>
          <p className="text-muted-foreground">Saved telemetry snapshots</p>
        </div>
        <Card className="border-destructive">
          <CardContent className="pt-6">
            <p className="text-destructive">Error: {error}</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Sessions</h1>
          <p className="text-muted-foreground">
            Save snapshots, compare sessions to detect changes, and merge collected data.
          </p>
        </div>
        <div className="flex items-center gap-2">
          {currentSessionName && (
            <Badge variant="secondary">{currentSessionName}</Badge>
          )}
          <Button size="sm" onClick={() => setShowSaveModal(true)}>Save Current</Button>
          <Button variant="outline" size="sm" onClick={handleImportClick} disabled={actionInProgress === 'importing'}>
            {actionInProgress === 'importing' ? 'Importing…' : 'Import'}
          </Button>
          <Button variant="ghost" size="sm" onClick={fetchSessions}>Refresh</Button>
          <input ref={fileInputRef} type="file" accept=".json" className="hidden" onChange={handleImportFile} />
        </div>
      </div>

      <Card>
        <CardContent className="p-0">
          {sessions.length === 0 ? (
            <div className="py-12 text-center">
              <p className="text-muted-foreground">No saved sessions yet.</p>
              <p className="text-sm text-muted-foreground mt-1">Use "Save Current" to create your first snapshot.</p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Size</TableHead>
                  <TableHead title="Total data points received">Metrics</TableHead>
                  <TableHead title="Total spans received">Spans</TableHead>
                  <TableHead title="Total log messages received">Logs</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sessions.map((session) => (
                  <TableRow key={session.id} className={currentSessionName === session.id ? 'bg-primary/5' : ''}>
                    <TableCell className="font-medium">
                      {session.id}
                      {currentSessionName === session.id && (
                        <Badge variant="secondary" className="ml-2 text-xs">active</Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground max-w-xs truncate">
                      {session.description || '-'}
                    </TableCell>
                    <TableCell className="text-sm">{formatDate(session.created)}</TableCell>
                    <TableCell className="text-sm">{formatSize(session.size_bytes)}</TableCell>
                    <TableCell>{session.stats?.metrics_count || 0}</TableCell>
                    <TableCell>{session.stats?.spans_count || 0}</TableCell>
                    <TableCell>{session.stats?.logs_count || 0}</TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1">
                        <Button
                          variant="outline" size="sm"
                          onClick={() => handleLoad(session.id)}
                          disabled={!!actionInProgress}
                        >
                          {actionInProgress === `loading-${session.id}` ? '…' : 'Load'}
                        </Button>
                        <Button
                          variant="outline" size="sm"
                          onClick={() => handleMerge(session.id)}
                          disabled={!!actionInProgress}
                        >
                          {actionInProgress === `merging-${session.id}` ? '…' : 'Merge'}
                        </Button>
                        <Button
                          variant="outline" size="sm"
                          onClick={() => onCompare && onCompare(session.id)}
                        >
                          Compare
                        </Button>
                        <Button
                          variant="ghost" size="sm"
                          onClick={() => handleExport(session.id)}
                        >
                          Export
                        </Button>
                        <Button
                          variant="ghost" size="sm"
                          className="text-destructive hover:text-destructive"
                          onClick={() => handleDelete(session.id)}
                          disabled={!!actionInProgress}
                        >
                          {actionInProgress === `deleting-${session.id}` ? '…' : 'Delete'}
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog open={showSaveModal} onOpenChange={(open) => { if (!open) { setShowSaveModal(false); setSaveError(null) } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Save Current State</DialogTitle>
            <DialogDescription>Create a named snapshot of the current telemetry metadata.</DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-3 py-2">
            <div className="flex flex-col gap-1.5">
              <label className="text-sm font-medium">Session Name *</label>
              <Input
                value={saveName}
                onChange={(e) => setSaveName(e.target.value)}
                placeholder="e.g., pre-deploy-v2.5"
                autoFocus
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label className="text-sm font-medium">Description (optional)</label>
              <Textarea
                value={saveDescription}
                onChange={(e) => setSaveDescription(e.target.value)}
                placeholder="Brief description of what this snapshot contains..."
                rows={3}
              />
            </div>
            {saveError && <p className="text-sm text-destructive">{saveError}</p>}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowSaveModal(false)} disabled={saving}>Cancel</Button>
            <Button onClick={handleSave} disabled={saving || !saveName.trim()}>
              {saving ? 'Saving…' : 'Save Session'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

export default SessionsView
