import { useState, useEffect } from 'react'

function TracePatterns({ onViewDetails }) {
  const [patterns, setPatterns] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [expandedPatterns, setExpandedPatterns] = useState({})
  const [minSpans, setMinSpans] = useState(1)
  const [showOnlyNormalized, setShowOnlyNormalized] = useState(false)

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

  // Check if a pattern was normalized (contains placeholders like <NUM>, <URL>, <HEX>, <UUID>)
  const isNormalized = (pattern) => {
    return /<(NUM|URL|HEX|UUID|ID|DATE|TIME|TIMESTAMP)>/i.test(pattern)
  }

  // Get an example span name that differs from the pattern
  const getExample = (pg) => {
    if (pg.matching_spans && pg.matching_spans.length > 0) {
      const firstSpan = pg.matching_spans[0].span_name
      // If pattern equals span name, it wasn't normalized
      if (firstSpan === pg.pattern) {
        return null
      }
      return firstSpan
    }
    return null
  }

  const filteredPatterns = patterns.filter(p => {
    if (p.span_count < minSpans) return false
    if (showOnlyNormalized && !isNormalized(p.pattern)) return false
    return true
  })
  
  // Separate multi-span patterns (interesting) from single-span patterns
  const multiSpanPatterns = filteredPatterns.filter(p => p.span_count > 1)
  const singleSpanPatterns = filteredPatterns.filter(p => p.span_count === 1)
  
  // Count normalized patterns
  const normalizedCount = filteredPatterns.filter(p => isNormalized(p.pattern)).length

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
          Span names aggregated by extracted patterns. Multi-variant patterns indicate 
          high-cardinality span naming (dynamic values in span names).
        </p>
      </div>

      <div className="filters">
        <label>
          Min Variants:
          <input
            type="number"
            min="1"
            value={minSpans}
            onChange={(e) => setMinSpans(parseInt(e.target.value) || 1)}
            style={{ width: '60px', marginLeft: '8px' }}
          />
        </label>
        <label style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <input
            type="checkbox"
            checked={showOnlyNormalized}
            onChange={(e) => setShowOnlyNormalized(e.target.checked)}
          />
          Show only normalized patterns
        </label>
        <span className="filter-info">
          {filteredPatterns.length} patterns ({normalizedCount} normalized, {multiSpanPatterns.length} multi-variant)
        </span>
      </div>

      <div className="legend">
        <span className="legend-item">
          <span className="legend-badge normalized">Normalized</span>
          Pattern extracted - dynamic values replaced with placeholders
        </span>
        <span className="legend-item">
          <span className="legend-badge original">Original</span>
          Already well-named - no dynamic values detected
        </span>
      </div>

      {multiSpanPatterns.length > 0 && (
        <section className="pattern-section">
          <h3>High Cardinality Patterns ({multiSpanPatterns.length})</h3>
          <p className="section-info">
            These patterns match multiple distinct span names - dynamic values (IDs, timestamps) in span names.
          </p>
          <div className="pattern-list">
            {multiSpanPatterns.map((pg, idx) => {
              const normalized = isNormalized(pg.pattern)
              const example = getExample(pg)
              
              return (
                <div key={idx} className={`pattern-card ${normalized ? 'normalized' : 'original'}`}>
                  <div 
                    className="pattern-header"
                    onClick={() => toggleExpand(pg.pattern)}
                  >
                    <div className="pattern-info">
                      <div className="pattern-line">
                        <span className={`type-badge ${normalized ? 'normalized' : 'original'}`}>
                          {normalized ? 'Normalized' : 'Original'}
                        </span>
                        <code className="pattern-template">{pg.pattern}</code>
                      </div>
                      {example && (
                        <div className="example-line">
                          <span className="example-label">Example:</span>
                          <code className="example-name">{example}</code>
                        </div>
                      )}
                      <div className="pattern-stats">
                        <span className="badge warning">{pg.span_count} variants</span>
                        <span className="stat">{pg.total_samples.toLocaleString()} total samples</span>
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
                            <th>Span Name (Variant)</th>
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
              )
            })}
          </div>
        </section>
      )}

      {minSpans === 1 && singleSpanPatterns.length > 0 && (
        <section className="pattern-section">
          <h3>Single-Variant Patterns ({singleSpanPatterns.length})</h3>
          <p className="section-info">
            Each pattern matches only one span name. Normalized = we detected dynamic values but only saw one variant.
          </p>
          <table>
            <thead>
              <tr>
                <th>Type</th>
                <th>Pattern</th>
                <th>Example</th>
                <th>Kind</th>
                <th>Samples</th>
                <th>Services</th>
              </tr>
            </thead>
            <tbody>
              {singleSpanPatterns.slice(0, 50).map((pg, idx) => {
                const span = pg.matching_spans[0]
                const normalized = isNormalized(pg.pattern)
                const example = getExample(pg)
                
                return (
                  <tr key={idx} className={normalized ? 'row-normalized' : ''}>
                    <td>
                      <span className={`type-badge small ${normalized ? 'normalized' : 'original'}`}>
                        {normalized ? 'Normalized' : 'Original'}
                      </span>
                    </td>
                    <td>
                      <code className="pattern-template">{pg.pattern}</code>
                    </td>
                    <td>
                      {example ? (
                        <span 
                          className="detail-link"
                          onClick={() => onViewDetails('spans', span.span_name)}
                        >
                          {example}
                        </span>
                      ) : (
                        <span className="same-as-pattern">= pattern</span>
                      )}
                    </td>
                    <td>
                      <span className="key-badge">{span?.kind || 'Unknown'}</span>
                    </td>
                    <td>{pg.total_samples.toLocaleString()}</td>
                    <td>{span?.services?.slice(0, 3).join(', ')}{span?.services?.length > 3 ? '...' : ''}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
          {singleSpanPatterns.length > 50 && (
            <p className="more-info">
              Showing 50 of {singleSpanPatterns.length} single-variant patterns
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
          margin-bottom: 16px;
          padding: 12px;
          background: var(--bg-secondary);
          border-radius: 6px;
        }
        .filter-info {
          color: var(--text-secondary);
          font-size: 0.9em;
          margin-left: auto;
        }
        .legend {
          display: flex;
          gap: 24px;
          margin-bottom: 20px;
          padding: 10px 12px;
          background: var(--bg-tertiary);
          border-radius: 6px;
          font-size: 0.85em;
        }
        .legend-item {
          display: flex;
          align-items: center;
          gap: 8px;
          color: var(--text-secondary);
        }
        .legend-badge {
          padding: 2px 6px;
          border-radius: 3px;
          font-size: 0.8em;
          font-weight: 500;
        }
        .legend-badge.normalized {
          background: #7c3aed;
          color: white;
        }
        .legend-badge.original {
          background: #059669;
          color: white;
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
        .pattern-card.normalized {
          border-left: 4px solid #7c3aed;
        }
        .pattern-card.original {
          border-left: 4px solid #059669;
        }
        .pattern-header {
          display: flex;
          justify-content: space-between;
          align-items: flex-start;
          padding: 16px;
          cursor: pointer;
        }
        .pattern-header:hover {
          background: var(--bg-hover);
        }
        .pattern-info {
          flex: 1;
        }
        .pattern-line {
          display: flex;
          align-items: center;
          gap: 10px;
          margin-bottom: 6px;
        }
        .type-badge {
          padding: 2px 8px;
          border-radius: 4px;
          font-size: 0.75em;
          font-weight: 600;
          text-transform: uppercase;
        }
        .type-badge.small {
          padding: 1px 6px;
          font-size: 0.7em;
        }
        .type-badge.normalized {
          background: #7c3aed;
          color: white;
        }
        .type-badge.original {
          background: #059669;
          color: white;
        }
        .pattern-template {
          font-size: 1.05em;
          font-weight: 500;
        }
        .example-line {
          display: flex;
          align-items: center;
          gap: 8px;
          margin-bottom: 8px;
          font-size: 0.9em;
        }
        .example-label {
          color: var(--text-secondary);
        }
        .example-name {
          color: var(--text-secondary);
          font-style: italic;
        }
        .pattern-stats {
          display: flex;
          gap: 12px;
          align-items: center;
        }
        .stat {
          color: var(--text-secondary);
          font-size: 0.9em;
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
          padding-top: 4px;
        }
        .pattern-details {
          padding: 0 16px 16px 16px;
        }
        .pattern-details table {
          width: 100%;
        }
        .row-normalized {
          background: rgba(124, 58, 237, 0.1);
        }
        .same-as-pattern {
          color: var(--text-tertiary);
          font-style: italic;
          font-size: 0.9em;
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
