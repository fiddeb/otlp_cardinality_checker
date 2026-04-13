import { useState, useEffect } from 'react'
import Dashboard from './components/Dashboard'
import MetadataComplexity from './components/MetadataComplexity'
import MetricsView from './components/MetricsView'
import MetricsOverview from './components/MetricsOverview'
import TracesView from './components/TracesView'
import TracePatterns from './components/TracePatterns'
import LogsView from './components/LogsView'
import ServiceExplorer from './components/ServiceExplorer'
import Details from './components/Details'
import MemoryView from './components/MemoryView'
import NoisyNeighbors from './components/NoisyNeighbors'
import TemplateDetails from './components/TemplateDetails'
import LogServiceDetails from './components/LogServiceDetails'
import LogPatternDetails from './components/LogPatternDetails'
import AttributesView from './components/AttributesView'
import AttributeExplorer from './components/AttributeExplorer'
import ServicesView from './components/ServicesView'
import ActiveSeries from './components/ActiveSeries'
import SessionsView from './components/SessionsView'
import DiffView from './components/DiffView'
import { AppSidebar } from './components/layout/AppSidebar'
import { AppHeader } from './components/layout/AppHeader'
import { SidebarProvider, SidebarInset } from './components/ui/sidebar'
import { TooltipProvider } from './components/ui/tooltip'


function App() {
  const [activeTab, setActiveTab] = useState('dashboard')
  const [selectedItem, setSelectedItem] = useState(null)
  const [selectedService, setSelectedService] = useState(null)
  const [selectedTemplate, setSelectedTemplate] = useState(null)
  const [selectedLogService, setSelectedLogService] = useState(null)
  const [selectedLogPattern, setSelectedLogPattern] = useState(null)
  const [selectedAttribute, setSelectedAttribute] = useState(null)
  const [navigationHistory, setNavigationHistory] = useState([])
  const [currentSessionName, setCurrentSessionName] = useState(null)
  const [diffFromSession, setDiffFromSession] = useState(null)
  const [appVersion, setAppVersion] = useState(null)
  const [darkMode, setDarkMode] = useState(() => {
    // Check localStorage or system preference
    const saved = localStorage.getItem('darkMode')
    if (saved !== null) {
      return saved === 'true'
    }
    return window.matchMedia('(prefers-color-scheme: dark)').matches
  })

  useEffect(() => {
    fetch('/api/v1/version')
      .then(r => r.ok ? r.json() : null)
      .then(data => { if (data) setAppVersion(data.version) })
      .catch(err => console.warn('[version] fetch failed:', err))
  }, [])

  useEffect(() => {
    // Apply dark mode class to html element (shadcn uses .dark on ancestor)
    if (darkMode) {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
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

  const pushNavigation = (tab, state = {}) => {
    // Save current state to history before navigating
    const currentState = {
      tab: activeTab,
      selectedItem,
      selectedService,
      selectedTemplate,
      selectedLogService,
      selectedLogPattern,
      selectedAttribute
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
    setSelectedAttribute(state.selectedAttribute || null)
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

  const handleViewAttribute = (attributeKey) => {
    pushNavigation('attribute-explorer', { selectedAttribute: attributeKey })
  }

  const handleBack = () => {
    if (navigationHistory.length === 0) {
      // No history, go to dashboard
      setSelectedItem(null)
      setSelectedService(null)
      setSelectedTemplate(null)
      setSelectedLogService(null)
      setSelectedLogPattern(null)
      setSelectedAttribute(null)
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
    setSelectedAttribute(previousState.selectedAttribute)
  }

  const handleBackToServiceDetails = () => {
    // This is a special case for log pattern -> log service navigation
    // Instead of using history, we know we want to go back to log-service-details
    setSelectedLogPattern(null)
    setActiveTab('log-service-details')
  }

  return (
    <TooltipProvider>
      <SidebarProvider>
        <AppSidebar activeTab={activeTab} onNavigate={(tab) => {
          setActiveTab(tab)
          setNavigationHistory([])
        }} />
        <SidebarInset className="min-w-0 overflow-x-hidden">
          <AppHeader
            darkMode={darkMode}
            onToggleDarkMode={toggleDarkMode}
            appVersion={appVersion}
            currentSessionName={currentSessionName}
          />
          <div className="flex flex-1 flex-col gap-4 px-4 py-6 overflow-x-hidden">
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

            {activeTab === 'trace-patterns' && (
              <TracePatterns onViewDetails={handleViewDetails} />
            )}

            {activeTab === 'logs' && (
              <LogsView onViewServiceDetails={handleViewLogService} />
            )}

            {activeTab === 'attributes' && (
              <AttributesView onViewAttribute={handleViewAttribute} />
            )}

            {activeTab === 'attribute-explorer' && selectedAttribute && (
              <AttributeExplorer
                attributeKey={selectedAttribute}
                onBack={handleBack}
                onViewService={handleViewService}
              />
            )}

            {activeTab === 'services' && (
              <ServicesView onViewService={handleViewService} />
            )}

            {activeTab === 'noisy-neighbors' && (
              <NoisyNeighbors />
            )}

            {activeTab === 'memory' && (
              <MemoryView />
            )}

            {activeTab === 'sessions' && (
              <SessionsView
                currentSessionName={currentSessionName}
                onSessionChange={setCurrentSessionName}
                onCompare={(sessionName) => {
                  setDiffFromSession(sessionName)
                  setActiveTab('diff')
                }}
              />
            )}

            {activeTab === 'diff' && (
              <DiffView
                initialFrom={diffFromSession}
                onBack={() => {
                  setDiffFromSession(null)
                  setActiveTab('sessions')
                }}
              />
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
                onViewLogDetails={(severity) => handleViewLogService(selectedService, severity)}
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
        </SidebarInset>
      </SidebarProvider>

    </TooltipProvider>
  )
}

export default App
