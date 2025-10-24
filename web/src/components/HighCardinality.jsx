import { useState, useEffect } from 'react'

function HighCardinality({ onViewDetails }) {
  const [signalType, setSignalType] = useState('metrics')
  const [threshold, setThreshold] = useState(50)
  const [data, setData] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [currentPage, setCurrentPage] = useState(1)

  const itemsPerPage = 100

  useEffect(() => {
    setLoading(true)
    setError(null)

    const endpoint = signalType === 'metrics' ? '/api/v1/metrics' :
                     signalType === 'spans' ? '/api/v1/spans' :
                     '/api/v1/logs'

    fetch(`${endpoint}?limit=1000`)
      .then(r => r.json())
      .then(result => {
        const items = []
        
        result.data.forEach(item => {
          const keys = signalType === 'metrics' ? item.label_keys :
                       signalType === 'spans' ? item.attribute_keys :
                       item.attribute_keys
          
          if (keys) {
            Object.entries(keys).forEach(([key, metadata]) => {
              if (metadata.estimated_cardinality > threshold) {
                items.push({
                  name: signalType === 'logs' ? item.severity : item.name,
                  key,
                  cardinality: metadata.estimated_cardinality,
                  samples: metadata.value_samples.slice(0, 3),
                  percentage: metadata.percentage,
                })
              }
            })
          }
        })

        items.sort((a, b) => b.cardinality - a.cardinality)
        setData(items)
        setLoading(false)
        setCurrentPage(1) // Reset to page 1 when data changes
      })
      .catch(err => {
        setError(err.message)
        setLoading(false)
      })
  }, [signalType, threshold])

  const totalPages = Math.ceil(data.length / itemsPerPage)
  const startIndex = (currentPage - 1) * itemsPerPage
  const endIndex = startIndex + itemsPerPage
  const currentData = data.slice(startIndex, endIndex)

  const getCardinalityBadge = (card) => {
    if (card > 200) return 'high'
    if (card > 50) return 'medium'
    return 'low'
  }

  return (
    <div className="card">
      <h2>High Cardinality Detection</h2>
      
      <div className="filter-group">
        <select value={signalType} onChange={(e) => setSignalType(e.target.value)}>
          <option value="metrics">Metrics</option>
          <option value="spans">Spans</option>
          <option value="logs">Logs</option>
        </select>

        <div className="threshold-input">
          <label>Threshold:</label>
          <input 
            type="number" 
            value={threshold} 
            onChange={(e) => setThreshold(Number(e.target.value))}
            min="1"
          />
        </div>
      </div>

      {loading && <div className="loading">Loading...</div>}
      {error && <div className="error">Error: {error}</div>}

      {!loading && !error && (
        <>
          <p style={{ marginTop: '10px', marginBottom: '10px', color: '#666' }}>
            Found {data.length} items above threshold. 
            {totalPages > 1 && ` Showing ${startIndex + 1}-${Math.min(endIndex, data.length)} (Page ${currentPage} of ${totalPages})`}
          </p>
          {data.length === 0 ? (
            <p>No high cardinality items found above threshold {threshold}</p>
          ) : (
            <>
              <table>
                <thead>
                  <tr>
                    <th>{signalType === 'logs' ? 'Severity' : 'Name'}</th>
                    <th>Key</th>
                    <th>Cardinality</th>
                    <th>Usage</th>
                    <th>Samples</th>
                  </tr>
              </thead>
              <tbody>
                {currentData.map((item, i) => (
                  <tr key={i}>
                    <td>
                      <span 
                        className="detail-link"
                        onClick={() => onViewDetails(signalType, item.name)}
                      >
                        {item.name}
                      </span>
                    </td>
                    <td><code>{item.key}</code></td>
                    <td>
                      <span className={`badge ${getCardinalityBadge(item.cardinality)}`}>
                        {item.cardinality}
                      </span>
                    </td>
                    <td>{item.percentage.toFixed(1)}%</td>
                    <td className="samples">{item.samples.join(', ')}</td>
                  </tr>
                ))}
              </tbody>
            </table>

            {totalPages > 1 && (
              <div style={{ 
                display: 'flex', 
                justifyContent: 'center', 
                alignItems: 'center',
                gap: '10px',
                marginTop: '20px',
                padding: '20px'
              }}>
                <button 
                  onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                  disabled={currentPage === 1}
                  style={{
                    padding: '8px 16px',
                    cursor: currentPage === 1 ? 'not-allowed' : 'pointer',
                    opacity: currentPage === 1 ? 0.5 : 1
                  }}
                >
                  Previous
                </button>
                
                <span style={{ color: '#666' }}>
                  Page {currentPage} of {totalPages}
                </span>
                
                <button 
                  onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                  disabled={currentPage === totalPages}
                  style={{
                    padding: '8px 16px',
                    cursor: currentPage === totalPages ? 'not-allowed' : 'pointer',
                    opacity: currentPage === totalPages ? 0.5 : 1
                  }}
                >
                  Next
                </button>
              </div>
            )}
          </>
        )}
        </>
      )}
    </div>
  )
}

export default HighCardinality
