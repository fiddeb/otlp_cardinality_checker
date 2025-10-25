import { useState, useEffect } from 'react'

function NoisyNeighbors() {
  const [serviceVolumes, setServiceVolumes] = useState([])
  const [highCardinalityAttrs, setHighCardinalityAttrs] = useState([])
  const [noisyServices, setNoisyServices] = useState([])
  const [threshold, setThreshold] = useState(30)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    fetchNoisyNeighbors()
  }, [threshold])

  const fetchNoisyNeighbors = async () => {
    setLoading(true)
    setError(null)

    try {
      // Fetch all data
      const [metricsRes, spansRes, logsRes] = await Promise.all([
        fetch('/api/v1/metrics?limit=10000').then(r => r.json()),
        fetch('/api/v1/spans?limit=10000').then(r => r.json()),
        fetch('/api/v1/logs?limit=10000').then(r => r.json()),
      ])

      // 1. Calculate service volumes
      const serviceData = {}
      
      // From metrics
      metricsRes.data?.forEach(metric => {
        if (metric.services) {
          Object.entries(metric.services).forEach(([service, count]) => {
            if (!serviceData[service]) {
              serviceData[service] = { metrics: 0, traces: 0, logs: 0, types: new Set() }
            }
            serviceData[service].metrics += count
            serviceData[service].types.add('metrics')
          })
        }
      })

      // From spans
      spansRes.data?.forEach(span => {
        if (span.services) {
          Object.entries(span.services).forEach(([service, count]) => {
            if (!serviceData[service]) {
              serviceData[service] = { metrics: 0, traces: 0, logs: 0, types: new Set() }
            }
            serviceData[service].traces += count
            serviceData[service].types.add('traces')
          })
        }
      })

      // From logs
      logsRes.data?.forEach(log => {
        if (log.services) {
          Object.entries(log.services).forEach(([service, count]) => {
            if (!serviceData[service]) {
              serviceData[service] = { metrics: 0, traces: 0, logs: 0, types: new Set() }
            }
            serviceData[service].logs += count
            serviceData[service].types.add('logs')
          })
        }
      })

      const volumes = Object.entries(serviceData)
        .map(([service, data]) => ({
          service,
          total: data.metrics + data.traces + data.logs,
          metrics: data.metrics,
          traces: data.traces,
          logs: data.logs,
          types: Array.from(data.types),
        }))
        .sort((a, b) => b.total - a.total)
        .slice(0, 10)

      setServiceVolumes(volumes)

      // 2. Find high cardinality attributes
      const highCardAttrs = []

      // From metrics
      metricsRes.data?.forEach(metric => {
        if (metric.label_keys) {
          Object.entries(metric.label_keys).forEach(([key, stats]) => {
            if (stats.estimated_cardinality > threshold) {
              highCardAttrs.push({
                type: 'metric',
                name: metric.name,
                attribute: key,
                cardinality: stats.estimated_cardinality,
                services: Object.keys(metric.services || {}).join(', '),
              })
            }
          })
        }
      })

      // From spans
      spansRes.data?.forEach(span => {
        if (span.attribute_keys) {
          Object.entries(span.attribute_keys).forEach(([key, stats]) => {
            if (stats.estimated_cardinality > threshold) {
              highCardAttrs.push({
                type: 'span',
                name: span.name,
                attribute: key,
                cardinality: stats.estimated_cardinality,
                services: Object.keys(span.services || {}).join(', '),
              })
            }
          })
        }
      })

      // From logs
      logsRes.data?.forEach(log => {
        if (log.attribute_keys) {
          Object.entries(log.attribute_keys).forEach(([key, stats]) => {
            if (stats.estimated_cardinality > threshold) {
              highCardAttrs.push({
                type: 'log',
                name: `severity_${log.severity}`,
                attribute: key,
                cardinality: stats.estimated_cardinality,
                services: Object.keys(log.services || {}).join(', '),
              })
            }
          })
        }
      })

      highCardAttrs.sort((a, b) => b.cardinality - a.cardinality)
      setHighCardinalityAttrs(highCardAttrs.slice(0, 10))

      // 3. Services contributing to high cardinality
      const serviceContributions = {}

      // Helper to add service contribution
      const addContribution = (serviceName, samples, itemName, type) => {
        if (!serviceContributions[serviceName]) {
          serviceContributions[serviceName] = { samples: 0, items: [] }
        }
        serviceContributions[serviceName].samples += samples
        serviceContributions[serviceName].items.push({ name: itemName, type, samples })
      }

      // From metrics with high cardinality
      metricsRes.data?.forEach(metric => {
        const hasHighCard = metric.label_keys && Object.values(metric.label_keys).some(
          stats => stats.estimated_cardinality > threshold
        )
        if (hasHighCard && metric.services) {
          Object.entries(metric.services).forEach(([service, count]) => {
            addContribution(service, count, metric.name, 'metric')
          })
        }
      })

      // From spans with high cardinality
      spansRes.data?.forEach(span => {
        const hasHighCard = span.attribute_keys && Object.values(span.attribute_keys).some(
          stats => stats.estimated_cardinality > threshold
        )
        if (hasHighCard && span.services) {
          Object.entries(span.services).forEach(([service, count]) => {
            addContribution(service, count, span.name, 'span')
          })
        }
      })

      // From logs with high cardinality
      logsRes.data?.forEach(log => {
        const hasHighCard = log.attribute_keys && Object.values(log.attribute_keys).some(
          stats => stats.estimated_cardinality > threshold
        )
        if (hasHighCard && log.services) {
          Object.entries(log.services).forEach(([service, count]) => {
            addContribution(service, count, `severity_${log.severity}`, 'log')
          })
        }
      })

      const noisy = Object.entries(serviceContributions)
        .map(([service, data]) => ({
          service,
          samples: data.samples,
          items: data.items,
        }))
        .sort((a, b) => b.samples - a.samples)
        .slice(0, 10)

      setNoisyServices(noisy)
      setLoading(false)
    } catch (err) {
      setError(err.message)
      setLoading(false)
    }
  }

  if (loading) return <div className="loading">Loading...</div>
  if (error) return <div className="error">Error: {error}</div>

  return (
    <>
      <div className="card noisy-neighbors-header">
        <h2>🔍 Noisy Neighbor Detection</h2>
        <p className="subtitle">Identify services causing high cardinality or high volume</p>
        
        <div className="threshold-control">
          <label htmlFor="threshold">Cardinality Threshold:</label>
          <input
            id="threshold"
            type="number"
            min="1"
            max="10000"
            value={threshold}
            onChange={(e) => setThreshold(parseInt(e.target.value) || 30)}
          />
          <button onClick={fetchNoisyNeighbors} className="refresh-btn">Refresh</button>
        </div>
      </div>

      <div className="card">
        <h2>1️⃣ Services by Total Sample Volume</h2>
        <table>
          <thead>
            <tr>
              <th>Service</th>
              <th>Total Samples</th>
              <th>Metrics</th>
              <th>Traces</th>
              <th>Logs</th>
              <th>Signal Types</th>
            </tr>
          </thead>
          <tbody>
            {serviceVolumes.length === 0 ? (
              <tr>
                <td colSpan="6" style={{ textAlign: 'center', padding: '20px' }}>
                  No services found
                </td>
              </tr>
            ) : (
              serviceVolumes.map(service => (
                <tr key={service.service}>
                  <td><strong>{service.service}</strong></td>
                  <td>{service.total.toLocaleString()}</td>
                  <td>{service.metrics.toLocaleString()}</td>
                  <td>{service.traces.toLocaleString()}</td>
                  <td>{service.logs.toLocaleString()}</td>
                  <td>{service.types.join(', ')}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <div className="card">
        <h2>2️⃣ High Cardinality Attributes (&gt; {threshold})</h2>
        {highCardinalityAttrs.length === 0 ? (
          <div style={{ padding: '20px', textAlign: 'center' }}>
            ✅ No high cardinality attributes found
          </div>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Type</th>
                <th>Name</th>
                <th>Attribute</th>
                <th>Cardinality</th>
                <th>Services</th>
              </tr>
            </thead>
            <tbody>
              {highCardinalityAttrs.map((attr, idx) => (
                <tr key={idx} className={attr.cardinality > 100 ? 'high-warning' : ''}>
                  <td>
                    <span className={`signal-badge signal-${attr.type}`}>
                      {attr.type}
                    </span>
                  </td>
                  <td><code>{attr.name}</code></td>
                  <td><strong>{attr.attribute}</strong></td>
                  <td>
                    <span className="cardinality-value">
                      {attr.cardinality.toLocaleString()}
                    </span>
                  </td>
                  <td style={{ fontSize: '12px' }}>{attr.services}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div className="card">
        <h2>3️⃣ Services Contributing to High Cardinality</h2>
        {noisyServices.length === 0 ? (
          <div style={{ padding: '20px', textAlign: 'center' }}>
            ✅ No services contributing to high cardinality
          </div>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Service</th>
                <th>Total Samples</th>
                <th>High-Cardinality Items</th>
              </tr>
            </thead>
            <tbody>
              {noisyServices.map(service => (
                <tr key={service.service}>
                  <td><strong>{service.service}</strong></td>
                  <td>{service.samples.toLocaleString()}</td>
                  <td>
                    <div className="item-list">
                      {service.items.slice(0, 5).map((item, idx) => (
                        <div key={idx} className="item-entry">
                          <span className={`signal-badge signal-${item.type}`}>
                            {item.type}
                          </span>
                          <code>{item.name}</code>
                          <span className="item-samples">
                            ({item.samples.toLocaleString()} samples)
                          </span>
                        </div>
                      ))}
                      {service.items.length > 5 && (
                        <div className="item-entry" style={{ fontStyle: 'italic', color: 'var(--text-secondary)' }}>
                          + {service.items.length - 5} more items
                        </div>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </>
  )
}

export default NoisyNeighbors
