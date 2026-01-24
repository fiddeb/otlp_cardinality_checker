import { useState, useEffect } from 'react'

function TracePatterns({ onViewDetails }) {
  const [patterns, setPatterns] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [expandedPatterns, setExpandedPatterns] = useState({})
  const [minSpans, setMinSpans] = useState(1)

  useEffect(() => {
    fetchPatterns()
  }, [])

  const fetchPatterns = async () => {
    try {
      setLoading(true)
      const response = await fetch('/api/v1/span-patterns')
      if (!response.ok) {
        throw new Error('Failed to fetch span patterns')
      }
      const data = await response.json()
      setPatterns(data.patterns || [])
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const toggleExpand = (pattern) => {
    setExpandedPatterns(prev => ({
      ...prev,
      [pattern]: !prev[pattern]
    }))
  }

  const filteredPatterns = patterns.filter(p => p.span_count >= minSpans)
  
  // Separate multi-span patterns (interesting) from single-span patterns
  const multiSpanPatterns = filteredPatterns.filter(p => p.span_count > 1)
  const singleSpanPatterns = filteredPatterns.filter(p => p.span_count === 1)

  if (loading) {
    return <div className="loading">Loading patterns...</div>
  }

  if (error) {
    return <div className="error">Error: {error}</div>
  }

  return (
    <div className="patterns-view">
      <div className="view-header">
        <h2>Trace Patterns</h2>
        <p className="subtitle">
          Span names aggregated by extracted patterns. Multi-span patterns indicate 
          potential high-cardinality span naming (dynamic values in span names).
        </p>
      </div>

      <div className="filters">
        <label>
          Min Spans per Pattern:
          <input
            type="number"
            min="1"
            value={minSpans}
            onChange={(e) => setMinSpans(parseInt(e.target.value) || 1)}
            style={{ width: '60px', marginLeft: '8px' }}
          />
        </label>
        <span className="filter-info">
          Showing {filteredPatterns.length} patterns ({multiSpanPatterns.length} multi-span)
        </span>
      </div>

      {multiSpanPatterns.length > 0 && (
        <section className="pattern-section">
          <h3>Multi-Span Patterns (Potential High Cardinality)</h3>
          <p className="section-info">
            These patterns match multiple distinct span names - often indicating 
            dynamic values (IDs, timestamps) embedded in span names.
          </p>
          <div className="pattern-list">
            {multiSpanPatterns.map((pg, idx) => (
              <div key={idx} className="pattern-card highlight">
                <div 
                  className="pattern-header"
                  onClick={() => toggleExpand(pg.pattern)}
                >
                  <div className="pattern-info">
                    <code className="pattern-template">{pg.pattern}</code>
                    <div className="pattern-stats">
                      <span className="badge warning">{pg.span_count} spans</span>
                      <span className="stat">{pg.total_samples.toLocaleString()} samples</span>
                    </div>
                  </div>
                  <span className="expand-icon">
                    {expandedPatterns[pg.pattern] ? '▼' : '▶'}
                  </span>
                </div>
                
                {expandedPatterns[pg.pattern] && (
                  <div className="pattern-details">
                    <table>
                      <thead>
                        <tr>
                          <th>Span Name</th>
                          <th>Kind</th>
                          <th>Samples</th>
                          <th>Services</th>
                        </tr>
                      </thead>
                      <tbody>
                        {pg.matching_spans.map((span, i) => (
                          <tr key={i}>
                            <td>
                              <span 
                                className="detail-link"
                                onClick={() => onViewDetails('spans', span.span_name)}
                              >
                                {span.span_name}
                              </span>
                            </td>
                            <td>
                              <span className="key-badge">{span.kind || 'Unknown'}</span>
                            </td>
                            <td>{span.sample_count.toLocaleString()}</td>
                            <td>{span.services?.join(', ') || '-'}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            ))}
          </div>
        </section>
      )}

      {minSpans === 1 && singleSpanPatterns.length > 0 && (
        <section className="pattern-section">
          <h3>Single-Span Patterns (Unique Names)</h3>
          <p className="section-info">
            These span names did not match any pattern with other spans.
          </p>
          <table>
            <thead>
              <tr>
                <th>Pattern / Span Name</th>
                <th>Kind</th>
                <th>Samples</th>
                <th>Services</th>
              </tr>
            </thead>
            <tbody>
              {singleSpanPatterns.slice(0, 50).map((pg, idx) => {
                const span = pg.matching_spans[0]
                return (
                  <tr key={idx}>
                    <td>
                      <span 
                        className="detail-link"
                        onClick={() => onViewDetails('spans', span.span_name)}
                      >
                        {pg.pattern}
                      </span>
                    </td>
                    <td>
                      <span className="key-badge">{span?.kind || 'Unknown'}</span>
                    </td>
                    <td>{pg.total_samples.toLocaleString()}</td>
                    <td>{span?.services?.join(', ') || '-'}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
          {singleSpanPatterns.length > 50 && (
            <p className="more-info">
              Showing 50 of {singleSpanPatterns.length} single-span patterns
            </p>
          )}
        </section>
      )}

      {filteredPatterns.length === 0 && (
        <div className="empty-state">
          <p>No span patterns found. Send some trace data to see patterns.</p>
        </div>
      )}

      <style>{`
        .patterns-view {
          padding: 20px;
        }
        .view-header {
          margin-bottom: 20px;
        }
        .view-header h2 {
          margin: 0 0 8px 0;
        }
        .subtitle {
          color: var(--text-secondary);
          margin: 0;
        }
        .filters {
          display: flex;
          gap: 20px;
          align-items: center;
          margin-bottom: 20px;
          padding: 12px;
          background: var(--bg-secondary);
          border-radius: 6px;
        }
        .filter-info {
          color: var(--text-secondary);
          font-size: 0.9em;
        }
        .pattern-section {
          margin-bottom: 30px;
        }
        .pattern-section h3 {
          margin: 0 0 8px 0;
        }
        .section-info {
          color: var(--text-secondary);
          margin: 0 0 16px 0;
          font-size: 0.9em;
        }
        .pattern-list {
          display: flex;
          flex-direction: column;
          gap: 12px;
        }
        .pattern-card {
          background: var(--bg-secondary);
          border-radius: 8px;
          overflow: hidden;
        }
        .pattern-card.highlight {
          border-left: 4px solid var(--warning-color);
        }
        .pattern-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 16px;
          cursor: pointer;
        }
        .pattern-header:hover {
          background: var(--bg-hover);
        }
        .pattern-info {
          flex: 1;
        }
        .pattern-template {
          font-size: 1.1em;
          font-weight: 500;
        }
        .pattern-stats {
          display: flex;
          gap: 12px;
          margin-top: 8px;
          align-items: center;
        }
        .stat {
          color: var(--text-secondary);
        }
        .badge.warning {
          background: var(--warning-color);
          color: white;
          padding: 2px 8px;
          border-radius: 4px;
          font-size: 0.85em;
        }
        .expand-icon {
          color: var(--text-secondary);
          font-size: 0.8em;
        }
        .pattern-details {
          padding: 0 16px 16px 16px;
        }
        .pattern-details table {
          width: 100%;
        }
        .empty-state {
          text-align: center;
          padding: 40px;
          color: var(--text-secondary);
        }
        .more-info {
          color: var(--text-secondary);
          font-size: 0.9em;
          margin-top: 12px;
        }
      `}</style>
    </div>
  )
}

export default TracePatterns
