import { useState, useEffect } from 'react'

function PatternExplorer() {
  const [patterns, setPatterns] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [minCardinality, setMinCardinality] = useState(1)
  const [minOccurrences, setMinOccurrences] = useState(5)
  const [minServices, setMinServices] = useState(1)
  const [searchTerm, setSearchTerm] = useState('')
  const [expandedPatterns, setExpandedPatterns] = useState(new Set())

  useEffect(() => {
    fetchPatterns()
  }, [minOccurrences, minServices])

  const fetchPatterns = async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await fetch(
        `/api/v1/logs/patterns?minCount=${minOccurrences}&minServices=${minServices}`
      )
      if (!response.ok) throw new Error('Failed to fetch patterns')
      const data = await response.json()
      setPatterns(data.patterns || [])
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const togglePattern = (index) => {
    const newExpanded = new Set(expandedPatterns)
    if (newExpanded.has(index)) {
      newExpanded.delete(index)
    } else {
      newExpanded.add(index)
    }
    setExpandedPatterns(newExpanded)
  }

  const filteredPatterns = patterns.filter(pattern => {
    // Filter by search term
    if (!pattern.template.toLowerCase().includes(searchTerm.toLowerCase())) {
      return false
    }
    
    // Filter by cardinality - pattern must have at least one key meeting the threshold
    const hasHighCardinalityKey = pattern.services.some(service => {
      const resourceKeysMatch = service.resource_keys.some(key => key.cardinality >= minCardinality)
      const attributeKeysMatch = service.attribute_keys.some(key => key.cardinality >= minCardinality)
      return resourceKeysMatch || attributeKeysMatch
    })
    
    return hasHighCardinalityKey
  })

  const getSeverityColor = (severity) => {
    const colors = {
      'ERROR': '#ef4444',
      'WARN': '#f59e0b',
      'INFO': '#3b82f6',
      'DEBUG': '#8b5cf6',
      'TRACE': '#6b7280'
    }
    return colors[severity] || '#6b7280'
  }

  const formatNumber = (num) => {
    return new Intl.NumberFormat().format(num)
  }

  const truncateExample = (text, maxLength = 300) => {
    if (!text || text.length <= maxLength) return text
    return text.substring(0, maxLength) + '...'
  }

  if (loading) {
    return (
      <div className="pattern-explorer">
        <div className="loading">Loading patterns...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="pattern-explorer">
        <div className="error">Error: {error}</div>
      </div>
    )
  }

  return (
    <div className="pattern-explorer">
      <div className="controls">
        <div className="control-group">
          <label>
            Search Template:
            <input
              type="text"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              placeholder="Filter by template text..."
            />
          </label>
        </div>
        
        <div className="control-group">
          <label>
            Min Cardinality:
            <input
              type="number"
              min="1"
              value={minCardinality}
              onChange={(e) => setMinCardinality(Number(e.target.value))}
            />
          </label>
        </div>

        <div className="control-group">
          <label>
            Min Occurrences:
            <input
              type="number"
              min="1"
              value={minOccurrences}
              onChange={(e) => setMinOccurrences(Number(e.target.value))}
            />
          </label>
        </div>

        <div className="control-group">
          <label>
            Min Services:
            <input
              type="number"
              min="1"
              value={minServices}
              onChange={(e) => setMinServices(Number(e.target.value))}
            />
          </label>
        </div>

        <button onClick={fetchPatterns} className="refresh-button">
          🔄 Refresh
        </button>
      </div>

      <div className="stats-summary">
        <span>Showing {filteredPatterns.length} of {patterns.length} patterns</span>
      </div>

      <div className="patterns-list">
        {filteredPatterns.map((pattern, patternIndex) => (
          <div key={patternIndex} className="pattern-card">
            <div 
              className="pattern-header"
              onClick={() => togglePattern(patternIndex)}
            >
              <div className="pattern-info">
                <span className="expand-icon">
                  {expandedPatterns.has(patternIndex) ? '▼' : '▶'}
                </span>
                <div className="pattern-template">
                  {pattern.template}
                </div>
              </div>
              <div className="pattern-stats">
                <span className="total-count">
                  {formatNumber(pattern.total_count)} occurrences
                </span>
                <span className="service-count">
                  {pattern.services.length} services
                </span>
              </div>
            </div>

            {expandedPatterns.has(patternIndex) && (
              <div className="pattern-details">
                {pattern.example_body && (
                  <div className="example-log">
                    <h4>Example Log:</h4>
                    <pre>{truncateExample(pattern.example_body)}</pre>
                  </div>
                )}

                <div className="severity-breakdown">
                  {Object.entries(pattern.severity_breakdown).map(([severity, count]) => (
                    <span
                      key={severity}
                      className="severity-badge"
                      style={{ backgroundColor: getSeverityColor(severity) }}
                    >
                      {severity}: {formatNumber(count)}
                    </span>
                  ))}
                </div>

                <div className="services-list">
                  {pattern.services.map((service, serviceIndex) => (
                    <div key={serviceIndex} className="service-card">
                      <div className="service-header-info">
                        <span className="service-name">
                          {service.service_name}
                        </span>
                        <span className="service-count">
                          {formatNumber(service.sample_count)} samples
                        </span>
                        <div className="service-severities">
                          {service.severities.map((sev, i) => (
                            <span
                              key={i}
                              className="severity-dot"
                              style={{ backgroundColor: getSeverityColor(sev) }}
                              title={sev}
                            />
                          ))}
                        </div>
                      </div>

                      <div className="service-details">
                        <div className="keys-section">
                          <h4>Resource Keys</h4>
                          <div className="keys-grid">
                            {service.resource_keys.map((key, keyIndex) => (
                              <div key={keyIndex} className="key-item">
                                <div className="key-header">
                                  <span className="key-name">{key.name}</span>
                                  <span 
                                    className={`cardinality ${key.cardinality > 100 ? 'high' : key.cardinality > 10 ? 'medium' : 'low'}`}
                                  >
                                    ~{key.cardinality}
                                  </span>
                                </div>
                                <div className="sample-values">
                                  {key.sample_values.slice(0, 3).map((val, i) => (
                                    <span key={i} className="sample-value">
                                      {val}
                                    </span>
                                  ))}
                                  {key.sample_values.length > 3 && (
                                    <span className="more-values">
                                      +{key.sample_values.length - 3} more
                                    </span>
                                  )}
                                </div>
                              </div>
                            ))}
                          </div>
                        </div>

                        <div className="keys-section">
                          <h4>Attribute Keys</h4>
                          <div className="keys-grid">
                            {service.attribute_keys.map((key, keyIndex) => (
                              <div key={keyIndex} className="key-item">
                                <div className="key-header">
                                  <span className="key-name">{key.name}</span>
                                  <span 
                                    className={`cardinality ${key.cardinality > 100 ? 'high' : key.cardinality > 10 ? 'medium' : 'low'}`}
                                  >
                                    ~{key.cardinality}
                                  </span>
                                </div>
                                <div className="sample-values">
                                  {key.sample_values.slice(0, 3).map((val, i) => (
                                    <span key={i} className="sample-value">
                                      {val}
                                    </span>
                                  ))}
                                  {key.sample_values.length > 3 && (
                                    <span className="more-values">
                                      +{key.sample_values.length - 3} more
                                    </span>
                                  )}
                                </div>
                              </div>
                            ))}
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

export default PatternExplorer
