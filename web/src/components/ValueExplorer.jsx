import { useState, useEffect, useCallback } from 'react'

const MAX_WATCHED_FIELDS = 10

function ValueExplorer({ attributeKey, onClose }) {
  const [data, setData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [sortBy, setSortBy] = useState('count')
  const [sortDir, setSortDir] = useState('desc')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const pageSize = 100

  const fetchData = useCallback(() => {
    if (!attributeKey) return
    setLoading(true)
    setError(null)

    const params = new URLSearchParams({
      sort_by: sortBy,
      sort_direction: sortDir,
      page: String(page),
      page_size: String(pageSize),
    })
    if (search) {
      params.set('q', search)
    }

    fetch(`/api/v1/attributes/${encodeURIComponent(attributeKey)}/watch?${params}`)
      .then(r => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
        return r.json()
      })
      .then(d => {
        setData(d)
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [attributeKey, sortBy, sortDir, search, page])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  const handleSort = (field) => {
    if (sortBy === field) {
      setSortDir(d => d === 'asc' ? 'desc' : 'asc')
    } else {
      setSortBy(field)
      setSortDir('desc')
    }
    setPage(1)
  }

  const formatDateTime = (isoStr) => {
    if (!isoStr) return ''
    const d = new Date(isoStr)
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }

  return (
    <div style={{
      position: 'fixed', top: 0, right: 0, bottom: 0,
      width: '520px', background: 'var(--bg-primary, #fff)',
      boxShadow: '-4px 0 24px rgba(0,0,0,0.15)',
      zIndex: 1000, display: 'flex', flexDirection: 'column',
      overflow: 'hidden',
    }}>
      {/* Header */}
      <div style={{
        padding: '16px 20px', borderBottom: '1px solid var(--border-color, #e0e0e0)',
        display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between',
        flexShrink: 0,
      }}>
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
            <span style={{
              background: '#1976d2', color: '#fff', fontSize: 11,
              padding: '2px 8px', borderRadius: 10, fontWeight: 600,
            }}>WATCHING</span>
            <code style={{ fontSize: 14, fontWeight: 700 }}>{attributeKey}</code>
          </div>
          {data && (
            <div style={{ fontSize: 12, color: 'var(--text-secondary, #666)', display: 'flex', gap: 16 }}>
              <span>Since {formatDateTime(data.watching_since)}</span>
              <span>{(data.unique_count || 0).toLocaleString()} unique</span>
              <span>{(data.total_observations || 0).toLocaleString()} total</span>
              {!data.active && (
                <span style={{ color: '#f57c00', fontWeight: 600 }}>read-only (session)</span>
              )}
            </div>
          )}
        </div>
        <button
          onClick={onClose}
          style={{
            background: 'none', border: 'none', cursor: 'pointer',
            fontSize: 20, color: 'var(--text-secondary, #666)',
            padding: '0 4px', lineHeight: 1,
          }}
          aria-label="Close Value Explorer"
        >✕</button>
      </div>

      {/* Overflow warning */}
      {data?.overflow && (
        <div style={{
          background: '#fff3e0', border: '1px solid #ffe0b2',
          padding: '8px 16px', fontSize: 13, color: '#e65100',
          flexShrink: 0, display: 'flex', alignItems: 'center', gap: 8,
        }}>
          <span>⚠</span>
          <span>10,000 unique values reached — new unique values are no longer collected.</span>
        </div>
      )}

      {/* Controls */}
      <div style={{
        padding: '10px 16px', borderBottom: '1px solid var(--border-color, #e0e0e0)',
        display: 'flex', gap: 8, alignItems: 'center', flexShrink: 0,
      }}>
        <input
          type="text"
          placeholder="Prefix filter…"
          value={search}
          onChange={e => { setSearch(e.target.value); setPage(1) }}
          style={{
            flex: 1, padding: '6px 10px', border: '1px solid #ccc',
            borderRadius: 4, fontSize: 13,
          }}
        />
        <button
          onClick={fetchData}
          style={{
            padding: '6px 14px', background: '#1976d2', color: '#fff',
            border: 'none', borderRadius: 4, cursor: 'pointer', fontSize: 13,
          }}
        >
          Refresh
        </button>
      </div>

      {/* Body */}
      <div style={{ flex: 1, overflowY: 'auto' }}>
        {loading && (
          <div style={{ padding: 24, textAlign: 'center', color: '#999' }}>Loading…</div>
        )}
        {error && (
          <div style={{ padding: 24, color: '#c62828' }}>Error: {error}</div>
        )}
        {!loading && !error && data && (
          <>
            {(data.values || []).length === 0 ? (
              <div style={{ padding: 32, textAlign: 'center', color: '#999' }}>
                No values collected yet.
              </div>
            ) : (
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
                <thead>
                  <tr style={{ background: 'var(--bg-secondary, #f5f5f5)', position: 'sticky', top: 0 }}>
                    <th
                      onClick={() => handleSort('value')}
                      style={{ padding: '8px 16px', textAlign: 'left', cursor: 'pointer', userSelect: 'none' }}
                    >
                      Value {sortBy === 'value' && (sortDir === 'asc' ? '↑' : '↓')}
                    </th>
                    <th
                      onClick={() => handleSort('count')}
                      style={{ padding: '8px 16px', textAlign: 'right', cursor: 'pointer', userSelect: 'none', width: 100 }}
                    >
                      Count {sortBy === 'count' && (sortDir === 'asc' ? '↑' : '↓')}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {data.values.map((entry, i) => (
                    <tr
                      key={i}
                      style={{ borderBottom: '1px solid var(--border-color, #f0f0f0)' }}
                    >
                      <td style={{ padding: '6px 16px', fontFamily: 'monospace', wordBreak: 'break-all' }}>
                        {entry.value}
                      </td>
                      <td style={{ padding: '6px 16px', textAlign: 'right', fontVariantNumeric: 'tabular-nums' }}>
                        {entry.count.toLocaleString()}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}

            {/* Pagination */}
            {(data.total_values || 0) > pageSize && (
              <div style={{ padding: '10px 16px', display: 'flex', justifyContent: 'center', gap: 8 }}>
                <button
                  onClick={() => setPage(p => Math.max(1, p - 1))}
                  disabled={page <= 1}
                  style={{ padding: '4px 12px', cursor: 'pointer' }}
                >← Prev</button>
                <span style={{ padding: '4px 8px', fontSize: 13 }}>
                  {page} / {Math.ceil((data.total_values || 0) / pageSize)}
                </span>
                <button
                  onClick={() => setPage(p => p + 1)}
                  disabled={(data.values || []).length < pageSize}
                  style={{ padding: '4px 12px', cursor: 'pointer' }}
                >Next →</button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}

export default ValueExplorer
export { MAX_WATCHED_FIELDS }
