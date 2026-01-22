import { Button } from '@/components/ui/button'
import { useTheme } from '@/context/theme-provider'
import { Sun, Moon, Trash2 } from 'lucide-react'
import { useState } from 'react'

export function AppHeader() {
  const { theme, setTheme } = useTheme()
  const [isClearing, setIsClearing] = useState(false)

  const handleClearData = async () => {
    if (!confirm('Are you sure you want to clear ALL data? This cannot be undone!')) {
      return
    }

    setIsClearing(true)
    try {
      const response = await fetch('/api/v1/admin/clear', { method: 'POST' })
      if (response.ok) {
        alert('All data cleared successfully!')
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

  return (
    <header className="sticky top-0 z-40 border-b bg-background px-6 py-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">OTLP Cardinality Checker</h1>
          <p className="text-sm text-muted-foreground">
            Analyze metadata structure from OpenTelemetry signals
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Button
            variant="destructive"
            size="sm"
            onClick={handleClearData}
            disabled={isClearing}
          >
            <Trash2 className="h-4 w-4 mr-2" />
            {isClearing ? 'Clearing...' : 'Clear Data'}
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
            aria-label="Toggle theme"
          >
            {theme === 'dark' ? <Sun className="h-5 w-5" /> : <Moon className="h-5 w-5" />}
          </Button>
        </div>
      </div>
    </header>
  )
}
