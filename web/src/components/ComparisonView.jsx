import { useState, useEffect } from 'react'

function ComparisonView({ onViewDetails }) {
  const [signalType, setSignalType] = useState('metrics')
  const [items, setItems] = useState([])
  const [selectedItems, setSelectedItems] = useState([])
  const [comparisonData, setComparisonData] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    setLoading(true)
    setError(null)
    setSelectedItems([])
    setComparisonData([])

    const endpoint = signalType === 'metrics' ? '/api/v1/metrics' :
                     signalType === 'spans' ? '/api/v1/spans' :
                     '/api/v1/logs'

    fetch(`${endpoint}?limit=1000`)
      .then(r => r.json())
      .then(result => {
        setItems(result.data || [])
        setLoading(false)
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [signalType])

  const handleItemSelect = (itemName) => {
    if (selectedItems.includes(itemName)) {
      setSelectedItems(selectedItems.filter(i => i !== itemName))
      setComparisonData(comparisonData.filter(d => d.name !== itemName))
    } else if (selectedItems.length < 4) {
      setSelectedItems([...selectedItems, itemName])
      
      // Fetch details for this item
      const endpoint = signalType === 'metrics' ? `/api/v1/metrics/${encodeURIComponent(itemName)}` :
                       signalType === 'spans' ? `/api/v1/spans/${encodeURIComponent(itemName)}` :
                       `/api/v1/logs/${encodeURIComponent(itemName)}`

      fetch(endpoint)
        .then(r => r.json())
        .then(data => {
          setComparisonData([...comparisonData, data])
        })
        .catch(err => console.error('Failed to fetch item details:', err))
    }
  }

  const getAllKeys = () => {
    const keysSet = new Set()
    comparisonData.forEach(item => {
      const keys = signalType === 'metrics' ? item.label_keys : item.attribute_keys
      if (keys) {
        Object.keys(keys).forEach(k => keysSet.add(k))
      }
    })
    return Array.from(keysSet).sort()
  }

  const getKeyValue = (item, key) => {
    const keys = signalType === 'metrics' ? item.label_keys : item.attribute_keys
    return keys?.[key] || null
  }

  const getCardinalityBadge = (card) => {
    if (card > 200) return 'high'
    if (card > 50) return 'medium'
    return 'low'
  }

  if (loading) return <div className="loading">Loading...</div>
  if (error) return <div className="error">Error: {error}</div>

  return (
    <div className="card">
      <h2>Compare {signalType === 'metrics' ? 'Metrics' : signalType === 'spans' ? 'Spans' : 'Logs'}</h2>
      
      <div className="filter-group">
        <select value={signalType} onChange={(e) => setSignalType(e.target.value)}>
          <option value="metrics">Metrics</option>
          <option value="spans">Spans</option>
          <option value="logs">Logs</option>
        </select>
      </div>

      <p style={{ marginTop: '10px' }} className="template-count-text">
        Select up to 4 {signalType} to compare. Selected: {selectedItems.length}/4
      </p>

      <div style={{ marginTop: '20px', marginBottom: '20px' }}>
        <h3>Available {signalType === 'metrics' ? 'Metrics' : signalType === 'spans' ? 'Spans' : 'Log Severities'}</h3>
        <div style={{ 
          maxHeight: '200px', 
          overflowY: 'auto', 
          border: '1px solid var(--border-light)', 
          borderRadius: '4px',
          padding: '10px',
          background: 'var(--bg-tertiary)'
        }}>
          {items.slice(0, 50).map((item, i) => {
            const itemName = signalType === 'logs' ? item.severity : item.name
            const isSelected = selectedItems.includes(itemName)
            
            return (
              <label 
                key={i} 
                style={{ 
                  display: 'block', 
                  padding: '5px',
                  cursor: selectedItems.length >= 4 && !isSelected ? 'not-allowed' : 'pointer',
                  opacity: selectedItems.length >= 4 && !isSelected ? 0.5 : 1
                }}
              >
                <input 
                  type="checkbox" 
                  checked={isSelected}
                  onChange={() => handleItemSelect(itemName)}
                  disabled={selectedItems.length >= 4 && !isSelected}
                  style={{ marginRight: '8px' }}
                />
                {itemName}
                <span className="template-count-text" style={{ marginLeft: '8px' }}>
                  ({item.sample_count.toLocaleString()} samples)
                </span>
              </label>
            )
          })}
        </div>
      </div>

      {comparisonData.length > 0 && (
        <>
          <h3>Comparison Table</h3>
          
          <div style={{ marginBottom: '20px' }}>
            <table>
              <thead>
                <tr>
                  <th>Property</th>
                  {comparisonData.map((item, i) => (
                    <th key={i}>
                      <span 
                        className="detail-link"
                        onClick={() => onViewDetails(signalType, signalType === 'logs' ? item.severity : item.name)}
                      >
                        {signalType === 'logs' ? item.severity : item.name}
                      </span>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td><strong>Sample Count</strong></td>
                  {comparisonData.map((item, i) => (
                    <td key={i}>{item.sample_count.toLocaleString()}</td>
                  ))}
                </tr>
                {signalType === 'metrics' && (
                  <tr>
                    <td><strong>Type</strong></td>
                    {comparisonData.map((item, i) => (
                      <td key={i}>{item.type}</td>
                    ))}
                  </tr>
                )}
                {signalType === 'spans' && (
                  <tr>
                    <td><strong>Kind</strong></td>
                    {comparisonData.map((item, i) => (
                      <td key={i}>{item.kind}</td>
                    ))}
                  </tr>
                )}
                <tr>
                  <td><strong>Service Count</strong></td>
                  {comparisonData.map((item, i) => (
                    <td key={i}>{item.services ? Object.keys(item.services).length : 0}</td>
                  ))}
                </tr>
              </tbody>
            </table>
          </div>

          <h3>{signalType === 'metrics' ? 'Label' : 'Attribute'} Comparison</h3>
          
          <table>
            <thead>
              <tr>
                <th>{signalType === 'metrics' ? 'Label' : 'Attribute'} Key</th>
                {comparisonData.map((item, i) => (
                  <th key={i}>{signalType === 'logs' ? item.severity : item.name}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {getAllKeys().map((key, i) => (
                <tr key={i}>
                  <td><code>{key}</code></td>
                  {comparisonData.map((item, j) => {
                    const value = getKeyValue(item, key)
                    if (!value) {
                      return <td key={j} className="template-label">-</td>
                    }
                    return (
                      <td key={j}>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
                          <span className={`badge ${getCardinalityBadge(value.estimated_cardinality)}`}>
                            {value.estimated_cardinality}
                          </span>
                          <span className="template-count-text-small">
                            {value.percentage.toFixed(1)}% usage
                          </span>
                        </div>
                      </td>
                    )
                  })}
                </tr>
              ))}
            </tbody>
          </table>

          {getAllKeys().length === 0 && (
            <p style={{ textAlign: 'center', padding: '20px' }} className="template-count-text">
              No {signalType === 'metrics' ? 'labels' : 'attributes'} found
            </p>
          )}
        </>
      )}

      {comparisonData.length === 0 && (
        <p style={{ textAlign: 'center', padding: '40px' }} className="template-count-text">
          Select items above to start comparing
        </p>
      )}
    </div>
  )
}

export default ComparisonView
