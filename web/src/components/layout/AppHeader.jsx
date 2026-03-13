import { useState } from 'react'
import { MoonIcon, SunIcon, Trash2Icon, SearchIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { SidebarTrigger } from '@/components/ui/sidebar'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Separator } from '@/components/ui/separator'

export function AppHeader({ darkMode, onToggleDarkMode, appVersion, currentSessionName, onOpenSearch }) {
  const [isClearing, setIsClearing] = useState(false)
  const [clearError, setClearError] = useState(null)

  const handleClearData = async () => {
    setIsClearing(true)
    setClearError(null)
    try {
      const response = await fetch('/api/v1/admin/clear', { method: 'POST' })
      if (response.ok) {
        window.location.reload()
      } else {
        const data = await response.json()
        setClearError(data.error || 'Unknown error')
      }
    } catch (error) {
      setClearError(error.message)
    } finally {
      setIsClearing(false)
    }
  }

  return (
    <header className="flex h-14 shrink-0 items-center gap-2 border-b bg-background px-4">
      <SidebarTrigger className="-ml-1" />
      <Separator orientation="vertical" className="h-4" />
      <div className="flex flex-1 items-center gap-2">
        <Button
          variant="outline"
          className="relative h-8 w-full max-w-xs justify-start gap-2 text-sm text-muted-foreground font-normal shadow-none"
          onClick={onOpenSearch}
        >
          <SearchIcon className="h-4 w-4" />
          <span>Search...</span>
          <kbd className="pointer-events-none ml-auto hidden h-5 select-none items-center gap-1 rounded border bg-muted px-1.5 font-mono text-[10px] font-medium opacity-100 sm:flex">
            ⌘K
          </kbd>
        </Button>
        {currentSessionName && (
          <span className="rounded-full bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
            {currentSessionName}
          </span>
        )}
      </div>
      {clearError && (
        <p className="text-sm text-destructive">{clearError}</p>
      )}
      <div className="flex items-center gap-1">
        {appVersion && (
          <span className="hidden text-xs text-muted-foreground sm:block">v{appVersion}</span>
        )}
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button variant="ghost" size="icon" disabled={isClearing} title="Clear all data">
              <Trash2Icon data-icon="inline-start" />
              <span className="sr-only">Clear data</span>
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Clear all data?</AlertDialogTitle>
              <AlertDialogDescription>
                This will permanently delete all telemetry metadata. This action cannot be undone.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction onClick={handleClearData}>Clear data</AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
        <Button variant="ghost" size="icon" onClick={onToggleDarkMode} title={darkMode ? 'Light mode' : 'Dark mode'}>
          {darkMode ? <SunIcon /> : <MoonIcon />}
          <span className="sr-only">Toggle theme</span>
        </Button>
      </div>
    </header>
  )
}
