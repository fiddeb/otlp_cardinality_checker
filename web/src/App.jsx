import { useState, useEffect } from 'react'
import Dashboard from './components/Dashboard'
import HighCardinality from './components/HighCardinality'
import MetricsView from './components/MetricsView'
import TracesView from './components/TracesView'
import LogsView from './components/LogsView'
import ComparisonView from './components/ComparisonView'
import ServiceExplorer from './components/ServiceExplorer'
import Details from './components/Details'
import MemoryView from './components/MemoryView'
import NoisyNeighbors from './components/NoisyNeighbors'

function App() {
  const [activeTab, setActiveTab] = useState('dashboard')
  const [selectedItem, setSelectedItem] = useState(null)
  const [selectedService, setSelectedService] = useState(null)
  const [darkMode, setDarkMode] = useState(() => {
    // Check localStorage or system preference
    const saved = localStorage.getItem('darkMode')
    if (saved !== null) {
      return saved === 'true'
    }
    return window.matchMedia('(prefers-color-scheme: dark)').matches
  })

  useEffect(() => {
    // Apply dark mode class to body
    if (darkMode) {
      document.body.classList.add('dark-mode')
    } else {
      document.body.classList.remove('dark-mode')
    }
    // Save preference
    localStorage.setItem('darkMode', darkMode)
  }, [darkMode])

  const toggleDarkMode = () => {
    setDarkMode(!darkMode)
  }

  const handleViewDetails = (type, name) => {
    setSelectedItem({ type, name })
    setActiveTab('details')
  }

  const handleViewService = (serviceName) => {
    setSelectedService(serviceName)
    setActiveTab('service')
  }

  const handleBack = () => {
    setSelectedItem(null)
    setSelectedService(null)
    setActiveTab('dashboard')
  }

  return (
    <div className="app">
      <header>
        <div className="header-content">
          <h1>OTLP Cardinality Checker</h1>
          <p className="subtitle">Analyze metadata structure from OpenTelemetry signals</p>
        </div>
        <button 
          className="dark-mode-toggle" 
          onClick={toggleDarkMode}
          title={darkMode ? 'Switch to light mode' : 'Switch to dark mode'}
        >
          {darkMode ? '☀️' : '🌙'}
        </button>
      </header>

      {!selectedItem && !selectedService && (
        <div className="tabs">
          <button 
            className={`tab ${activeTab === 'dashboard' ? 'active' : ''}`}
            onClick={() => setActiveTab('dashboard')}
          >
            Dashboard
          </button>
          <button 
            className={`tab ${activeTab === 'high-cardinality' ? 'active' : ''}`}
            onClick={() => setActiveTab('high-cardinality')}
          >
            High Cardinality
          </button>
          <button 
            className={`tab ${activeTab === 'metrics' ? 'active' : ''}`}
            onClick={() => setActiveTab('metrics')}
          >
            Metrics
          </button>
          <button 
            className={`tab ${activeTab === 'traces' ? 'active' : ''}`}
            onClick={() => setActiveTab('traces')}
          >
            Traces
          </button>
          <button 
            className={`tab ${activeTab === 'logs' ? 'active' : ''}`}
            onClick={() => setActiveTab('logs')}
          >
            Logs
          </button>
          <button 
            className={`tab ${activeTab === 'comparison' ? 'active' : ''}`}
            onClick={() => setActiveTab('comparison')}
          >
            Compare
          </button>
          <button 
            className={`tab ${activeTab === 'noisy-neighbors' ? 'active' : ''}`}
            onClick={() => setActiveTab('noisy-neighbors')}
          >
            Noisy Neighbors
          </button>
          <button 
            className={`tab ${activeTab === 'memory' ? 'active' : ''}`}
            onClick={() => setActiveTab('memory')}
          >
            Memory
          </button>
        </div>
      )}

      {activeTab === 'dashboard' && !selectedService && (
        <Dashboard onViewService={handleViewService} />
      )}

      {activeTab === 'high-cardinality' && (
        <HighCardinality onViewDetails={handleViewDetails} />
      )}

      {activeTab === 'metrics' && (
        <MetricsView onViewDetails={handleViewDetails} />
      )}

      {activeTab === 'traces' && (
        <TracesView onViewDetails={handleViewDetails} />
      )}

      {activeTab === 'logs' && (
        <LogsView onViewDetails={handleViewDetails} />
      )}

      {activeTab === 'comparison' && (
        <ComparisonView onViewDetails={handleViewDetails} />
      )}

      {activeTab === 'noisy-neighbors' && (
        <NoisyNeighbors />
      )}

      {activeTab === 'memory' && (
        <MemoryView />
      )}

      {activeTab === 'service' && selectedService && (
        <ServiceExplorer 
          serviceName={selectedService} 
          onBack={handleBack}
          onViewDetails={handleViewDetails}
        />
      )}

      {activeTab === 'details' && selectedItem && (
        <Details 
          type={selectedItem.type} 
          name={selectedItem.name}
          onBack={handleBack}
        />
      )}
    </div>
  )
}

export default App
