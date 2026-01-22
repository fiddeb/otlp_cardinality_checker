import { useState, useEffect } from 'react'
import Dashboard from './components/Dashboard'
import MetadataComplexity from './components/MetadataComplexity'
import MetricsView from './components/MetricsView'
import MetricsOverview from './components/MetricsOverview'
import TracesView from './components/TracesView'
import LogsView from './components/LogsView'
import ServiceExplorer from './components/ServiceExplorer'
import Details from './components/Details'
import MemoryView from './components/MemoryView'
import NoisyNeighbors from './components/NoisyNeighbors'
import TemplateDetails from './components/TemplateDetails'
import LogServiceDetails from './components/LogServiceDetails'
import LogPatternDetails from './components/LogPatternDetails'
import AttributesView from './components/AttributesView'
import ActiveSeries from './components/ActiveSeries'

function App() {
  const [activeTab, setActiveTab] = useState('dashboard')
  const [selectedItem, setSelectedItem] = useState(null)
  const [selectedService, setSelectedService] = useState(null)
  const [selectedTemplate, setSelectedTemplate] = useState(null)
  const [selectedLogService, setSelectedLogService] = useState(null)
  const [selectedLogPattern, setSelectedLogPattern] = useState(null)
  const [navigationHistory, setNavigationHistory] = useState([])
  const [isClearing, setIsClearing] = useState(false)
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

  useEffect(() => {
    // Handle browser back/forward button
    const handlePopState = (event) => {
      event.preventDefault()
      // Trigger our internal back handler
      handleBack()
    }

    // Push initial state when app loads
    window.history.pushState({ page: 'app' }, '', window.location.href)
    
    // Listen for popstate (browser back/forward)
    window.addEventListener('popstate', handlePopState)

    return () => {
      window.removeEventListener('popstate', handlePopState)
    }
  }, [navigationHistory]) // Re-attach when history changes

  const toggleDarkMode = () => {
    setDarkMode(!darkMode)
  }

  const handleClearData = async () => {
    if (!confirm('Are you sure you want to clear ALL data? This cannot be undone!')) {
      return
    }

    setIsClearing(true)
    try {
      const response = await fetch('/api/v1/admin/clear', {
        method: 'POST',
      })

      if (response.ok) {
        alert('All data cleared successfully!')
        // Refresh the current view
        window.location.reload()
      } else {
        const data = await response.json()
        alert(`Failed to clear data: ${data.error || 'Unknown error'}`)
      }
    } catch (error) {
      alert(`Failed to clear data: ${error.message}`)
    } finally {
      setIsClearing(false)
    }
  }

  const pushNavigation = (tab, state = {}) => {
    // Save current state to history before navigating
    const currentState = {
      tab: activeTab,
      selectedItem,
      selectedService,
      selectedTemplate,
      selectedLogService,
      selectedLogPattern
    }
    setNavigationHistory(prev => [...prev, currentState])
    
    // Push to browser history to prevent leaving the page
    window.history.pushState({ page: 'app' }, '', window.location.href)
    
    // Navigate to new state
    setActiveTab(tab)
    setSelectedItem(state.selectedItem || null)
    setSelectedService(state.selectedService || null)
    setSelectedTemplate(state.selectedTemplate || null)
    setSelectedLogService(state.selectedLogService || null)
    setSelectedLogPattern(state.selectedLogPattern || null)
  }

  const handleViewDetails = (type, name) => {
    pushNavigation('details', { selectedItem: { type, name } })
  }

  const handleViewService = (serviceName) => {
    pushNavigation('service', { selectedService: serviceName })
  }

  const handleViewTemplate = (severity, template) => {
    pushNavigation('template-details', { selectedTemplate: { severity, template } })
  }

  const handleViewLogService = (serviceName, severity) => {
    pushNavigation('log-service-details', { selectedLogService: { serviceName, severity } })
  }

  const handleViewLogPattern = (serviceName, severity, template) => {
    pushNavigation('log-pattern-details', { selectedLogPattern: { serviceName, severity, template } })
  }

  const handleBack = () => {
    if (navigationHistory.length === 0) {
      // No history, go to dashboard
      setSelectedItem(null)
      setSelectedService(null)
      setSelectedTemplate(null)
      setSelectedLogService(null)
      setSelectedLogPattern(null)
      setActiveTab('dashboard')
      return
    }

    // Pop last state from history
    const previousState = navigationHistory[navigationHistory.length - 1]
    setNavigationHistory(prev => prev.slice(0, -1))
    
    // Restore previous state
    setActiveTab(previousState.tab)
    setSelectedItem(previousState.selectedItem)
    setSelectedService(previousState.selectedService)
    setSelectedTemplate(previousState.selectedTemplate)
    setSelectedLogService(previousState.selectedLogService)
    setSelectedLogPattern(previousState.selectedLogPattern)
  }

  const handleBackToServiceDetails = () => {
    // This is a special case for log pattern -> log service navigation
    // Instead of using history, we know we want to go back to log-service-details
    setSelectedLogPattern(null)
    setActiveTab('log-service-details')
  }

  return (
    <div className="app">
      <header>
        <div className="header-content">
          <h1>OTLP Cardinality Checker</h1>
          <p className="subtitle">Analyze metadata structure from OpenTelemetry signals</p>
        </div>
        <div className="header-actions">
          <button 
            className="clear-button" 
            onClick={handleClearData}
            disabled={isClearing}
            title="Clear all data from database"
          >
            {isClearing ? '🔄' : '🗑️'} Clear Data
          </button>
          <button 
            className="dark-mode-toggle" 
            onClick={toggleDarkMode}
            title={darkMode ? 'Switch to light mode' : 'Switch to dark mode'}
          >
            {darkMode ? '☀️' : '🌙'}
          </button>
        </div>
      </header>

      {!selectedItem && !selectedService && !selectedTemplate && (
        <div className="tabs">
          <button 
            className={`tab ${activeTab === 'dashboard' ? 'active' : ''}`}
            onClick={() => {
              setActiveTab('dashboard')
              setNavigationHistory([]) // Clear history when clicking tabs
            }}
          >
            Dashboard
          </button>
          <button 
            className={`tab ${activeTab === 'metadata-complexity' ? 'active' : ''}`}
            onClick={() => {
              setActiveTab('metadata-complexity')
              setNavigationHistory([])
            }}
          >
            Metadata Complexity
          </button>
          <button 
            className={`tab ${activeTab === 'metrics-overview' ? 'active' : ''}`}
            onClick={() => {
              setActiveTab('metrics-overview')
              setNavigationHistory([])
            }}
          >
            Metrics Overview
          </button>
          <button 
            className={`tab ${activeTab === 'active-series' ? 'active' : ''}`}
            onClick={() => {
              setActiveTab('active-series')
              setNavigationHistory([])
            }}
          >
            Active Series
          </button>
          <button 
            className={`tab ${activeTab === 'metrics' ? 'active' : ''}`}
            onClick={() => {
              setActiveTab('metrics')
              setNavigationHistory([])
            }}
          >
            Metrics Details
          </button>
          <button 
            className={`tab ${activeTab === 'traces' ? 'active' : ''}`}
            onClick={() => {
              setActiveTab('traces')
              setNavigationHistory([])
            }}
          >
            Traces
          </button>
          <button 
            className={`tab ${activeTab === 'logs' ? 'active' : ''}`}
            onClick={() => {
              setActiveTab('logs')
              setNavigationHistory([])
            }}
          >
            Logs
          </button>
          <button 
            className={`tab ${activeTab === 'attributes' ? 'active' : ''}`}
            onClick={() => {
              setActiveTab('attributes')
              setNavigationHistory([])
            }}
          >
            Attributes
          </button>
          <button 
            className={`tab ${activeTab === 'noisy-neighbors' ? 'active' : ''}`}
            onClick={() => {
              setActiveTab('noisy-neighbors')
              setNavigationHistory([])
            }}
          >
            Noisy Neighbors
          </button>
          <button 
            className={`tab ${activeTab === 'memory' ? 'active' : ''}`}
            onClick={() => {
              setActiveTab('memory')
              setNavigationHistory([])
            }}
          >
            Memory
          </button>
        </div>
      )}

      {activeTab === 'dashboard' && !selectedService && (
        <Dashboard onViewService={handleViewService} />
      )}

      {activeTab === 'metadata-complexity' && (
        <MetadataComplexity onViewDetails={handleViewDetails} />
      )}

      {activeTab === 'metrics-overview' && (
        <MetricsOverview onViewMetric={(name) => handleViewDetails('metrics', name)} />
      )}

      {activeTab === 'active-series' && (
        <ActiveSeries />
      )}

      {activeTab === 'metrics' && !selectedItem && (
        <MetricsView onViewDetails={handleViewDetails} />
      )}

      {activeTab === 'traces' && (
        <TracesView onViewDetails={handleViewDetails} />
      )}

      {activeTab === 'logs' && (
        <LogsView onViewServiceDetails={handleViewLogService} />
      )}

      {activeTab === 'attributes' && (
        <AttributesView />
      )}

      {activeTab === 'noisy-neighbors' && (
        <NoisyNeighbors />
      )}

      {activeTab === 'memory' && (
        <MemoryView />
      )}

      {activeTab === 'template-details' && selectedTemplate && (
        <TemplateDetails 
          severity={selectedTemplate.severity}
          template={selectedTemplate.template}
          onBack={handleBack}
        />
      )}

      {activeTab === 'log-service-details' && selectedLogService && (
        <LogServiceDetails 
          serviceName={selectedLogService.serviceName}
          severity={selectedLogService.severity}
          onBack={handleBack}
          onViewPattern={handleViewLogPattern}
        />
      )}

      {activeTab === 'log-pattern-details' && selectedLogPattern && (
        <LogPatternDetails 
          serviceName={selectedLogPattern.serviceName}
          severity={selectedLogPattern.severity}
          template={selectedLogPattern.template}
          onBack={handleBackToServiceDetails}
        />
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
