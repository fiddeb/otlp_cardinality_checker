# Tasks: Gradual ShadcnUI Migration

**Change ID:** `migrate-shadcn-ui`

**Status:** On Hold - Tailwind CSS compilation issue blocking progress

## Progress Summary

✅ **Phase 1: Foundation Setup** - COMPLETE (100%)
✅ **Phase 2: Layout Migration** - COMPLETE (100%)  
🚧 **Phase 3: Component Migration** - PARTIAL (60%) - **BLOCKED**
⏸️ **Phase 4: Interactive Components** - NOT STARTED
⏸️ **Phase 5: Polish & Accessibility** - NOT STARTED

## Current Blocker

**Issue:** Tailwind CSS classes not applying to shadcn components despite correct configuration.

**Symptoms:**
- Card components render with no styling (no borders, padding, background)
- Tailwind utility classes like `bg-card`, `text-muted-foreground`, `border` not generating CSS
- Build succeeds without errors (22.4KB CSS generated)
- Dev server runs without errors
- All imports resolve correctly

**Configuration verified:**
- ✅ tailwind.config.js has correct color mappings
- ✅ theme.css has CSS variables defined
- ✅ index.css imports theme.css and Tailwind directives
- ✅ vite.config.js has @tailwindcss/vite plugin
- ✅ Card components use correct Tailwind classes

**Next steps to investigate:**
1. Check if Tailwind v4 has breaking changes with CSS variable syntax
2. Verify content path in tailwind.config.js matches all JSX files
3. Try downgrading to Tailwind v3
4. Check browser DevTools computed styles for Card elements
5. Verify CSS file actually loads in browser (Network tab)

## Overview

This task list implements the gradual migration strategy outlined in `proposal.md`. Each task is small, verifiable, and delivers incremental progress. Tasks are grouped by phase with clear dependencies.

---

## Phase 1: Foundation Setup ✅ COMPLETE

### 1.1 Install Core Dependencies ✅

**Status:** COMPLETE  
**Commit:** `chore(web): add shadcn-ui dependencies`

### 1.2 Configure Tailwind CSS v4 ✅

**Status:** COMPLETE  
**Commit:** `feat(web): configure Tailwind CSS v4`

### 1.3 Configure Path Aliases ✅

**Status:** COMPLETE  
**Commit:** `feat(web): add path aliases and cn() utility`

### 1.4 Initialize shadcn-ui Configuration ✅

**Status:** COMPLETE  
**Commit:** Included in task 1.7

### 1.5 Define CSS Theme Variables ✅

**Status:** COMPLETE  
**Commit:** `feat(web): define CSS theme variables`

### 1.6 Create ThemeProvider Context ✅

**Status:** COMPLETE  
**Commit:** `feat(web): add ThemeProvider context`

### 1.7 Add Base UI Components ✅

**Status:** COMPLETE  
**Commit:** `feat(web): add shadcn base components`

---

## Phase 2: Layout Migration ✅ COMPLETE

### 2.1 Install Sidebar Component ✅

**Status:** COMPLETE  
**Commit:** `feat(web): add sidebar component and lucide-react icons`

### 2.2 Create AppSidebar Component ✅

**Status:** COMPLETE  
**Commit:** `feat(web): integrate sidebar layout into App`

### 2.3 Create AppHeader Component ✅

**Status:** COMPLETE  
**Commit:** `feat(web): integrate sidebar layout into App`

### 2.4 Integrate Sidebar and Header into App ✅

**Status:** COMPLETE  
**Commit:** `feat(web): integrate sidebar layout into App`

**Changes:**
- Replaced tab navigation with sidebar
- Integrated AppHeader with theme toggle and clear data button
- Removed old header CSS
- App now uses SidebarProvider layout

### 2.5 Remove Legacy Tab Navigation ✅

**Status:** COMPLETE (CSS cleanup pending)  
**Note:** Old tab CSS still exists but unused. Will be removed in Phase 5 cleanup.

---

## Phase 3: Component Migration 🚧 PARTIAL

### 3.1 Migrate Dashboard Stats to Cards ✅

**Status:** COMPLETE  
**Commit:** `feat(web): migrate Dashboard to shadcn Card and Table components`

**Changes:**
- Converted `.stat-card` divs to shadcn `Card` components
- Migrated service table to shadcn `Table` component
- Updated styling to use Tailwind classes

### 3.2 Migrate Tables to shadcn Table Component ✅

**Status:** COMPLETE
- ✅ Dashboard table migrated
- ✅ MetricsView table migrated
- ✅ TracesView table migrated
- ⏸️ LogsView, AttributesView tables pending

**Commits:**
- `feat(web): migrate MetricsView and TracesView to shadcn components`

### 3.3 Migrate MemoryView ✅

**Status:** COMPLETE
**Commit:** `feat(web): migrate MemoryView to shadcn Card components`

**Changes:**
- Replaced `.card` and `.memory-grid` with shadcn Card components
- Used Tailwind grid utilities for responsive layout
- Added CardDescription for real-time update info

### 3.4 Migrate Badges ⏸️

**Status:** PARTIAL
- ✅ MetricsView badges migrated (using variant system)
- ✅ TracesView badges migrated  
- ⏸️ Other views pending

### 3.5 Migrate Search and Filter Inputs ✅

**Status:** COMPLETE (for migrated views)
- ✅ MetricsView filters using shadcn Input and Select
- ✅ TracesView filters using shadcn Input and Select
- ⏸️ LogsView, AttributesView filters pending

---

## Phase 4: Interactive Components ⏸️ NOT STARTED

All tasks in this phase are pending.

---

## Phase 5: Polish & Accessibility ⏸️ NOT STARTED

All tasks in this phase are pending.

---

## Next Steps

1. **Complete Phase 3** - Migrate remaining tables, badges, and inputs
2. **Phase 4** - Migrate modals/dialogs and loading states
3. **Phase 5** - Remove legacy CSS, accessibility audit, bundle size check
4. **Testing** - Full integration testing across all views
5. **PR** - Create pull request with all changes

---

## Commits Made

1. `chore(web): add shadcn-ui dependencies`
2. `feat(web): configure Tailwind CSS v4`
3. `feat(web): add path aliases and cn() utility`
4. `feat(web): define CSS theme variables`
5. `feat(web): add ThemeProvider context`
6. `feat(web): add shadcn base components`
7. `feat(web): add sidebar component and lucide-react icons`
8. `feat(web): integrate sidebar layout into App`
9. `feat(web): migrate Dashboard to shadcn Card and Table components`
10. `refactor(web): clean up CSS conflicts for shadcn migration`
11. `feat(web): migrate MetricsView and TracesView to shadcn components`
12. `feat(web): migrate MemoryView to shadcn Card components`
13. `fix(web): remove @layer base from theme.css for Tailwind v4 compatibility`

## Known Issues

### Tailwind CSS Not Applying (Critical)
**Status:** Unresolved - blocking all further migration  
**Affected:** All migrated components (Dashboard, MetricsView, TracesView, MemoryView)  
**Impact:** Components render without styling - no borders, backgrounds, or spacing

**What Works:**
- Build process completes successfully
- No JavaScript errors in console
- Component structure is correct
- All imports resolve

**What Doesn't Work:**
- Tailwind utility classes don't generate CSS
- CSS variables from theme.css may not be loading
- shadcn component styles (bg-card, text-muted-foreground) have no effect

**Attempted Solutions:**
1. ✅ Removed `@layer base` wrapper from theme.css
2. ✅ Verified tailwind.config.js color mappings
3. ✅ Checked vite.config.js Tailwind plugin
4. ❌ Hard refresh browser
5. ❌ Restart dev server
6. ❌ Clear Vite cache

---
2. Update `web/vite.config.js` to include Tailwind plugin:
```js
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  // ... rest unchanged
})
```
3. Create `web/src/styles/theme.css` with basic structure (empty for now)
4. Update `web/src/index.css` to import Tailwind directives:
```css
@tailwind base;
@tailwind components;
@tailwind utilities;

/* Existing custom CSS below */
```
5. Test build: `npm run dev`
6. Commit: `feat(web): configure Tailwind CSS v4`

**Validation:**
```bash
npm run build  # Should succeed
# Check dist/assets/*.css includes Tailwind classes
```

---

### 1.3 Configure Path Aliases

**Goal:** Enable `@/` imports for components  
**Dependencies:** None  
**Validation:** Sample import resolves correctly

**Steps:**
1. Update `web/vite.config.js`:
```js
import path from 'path'

export default defineConfig({
  // ...
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
})
```
2. Create `web/src/lib/utils.js`:
```js
import { clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs) {
  return twMerge(clsx(inputs))
}
```
3. Test import in `App.jsx`: `import { cn } from '@/lib/utils'`
4. Verify build succeeds
5. Commit: `feat(web): add path aliases and cn() utility`

**Validation:**
```bash
# Should compile without errors
npm run dev
```

---

### 1.4 Initialize shadcn-ui Configuration

**Goal:** Create components.json for shadcn CLI  
**Dependencies:** 1.2, 1.3  
**Validation:** `npx shadcn@latest add button` works

**Steps:**
1. Create `web/components.json`:
```json
{
  "$schema": "https://ui.shadcn.com/schema.json",
  "style": "default",
  "rsc": false,
  "tsx": false,
  "tailwind": {
    "config": "tailwind.config.js",
    "css": "src/styles/index.css",
    "baseColor": "slate",
    "cssVariables": true
  },
  "aliases": {
    "components": "@/components",
    "utils": "@/lib",
    "ui": "@/components/ui"
  }
}
```
2. Create `web/src/components/ui/.gitkeep` (empty directory)
3. Test CLI: `npx shadcn@latest add button --yes`
4. Verify `src/components/ui/button.jsx` is created
5. Remove button.jsx (just testing)
6. Commit: `chore(web): configure shadcn-ui CLI`

**Validation:**
```bash
cd web
npx shadcn@latest add button --yes
ls src/components/ui/button.jsx  # Should exist
git restore src/components/ui/button.jsx  # Clean up
```

---

### 1.5 Define CSS Theme Variables

**Goal:** Create semantic color tokens for theming  
**Dependencies:** 1.2  
**Validation:** Visual inspection of colors in dev mode

**Steps:**
1. Update `web/src/styles/theme.css` with full color system:
```css
@layer base {
  :root {
    --background: 0 0% 100%;
    --foreground: 222.2 84% 4.9%;
    --card: 0 0% 100%;
    --card-foreground: 222.2 84% 4.9%;
    --primary: 221.2 83.2% 53.3%;
    --primary-foreground: 210 40% 98%;
    --secondary: 210 40% 96.1%;
    --secondary-foreground: 222.2 47.4% 11.2%;
    --muted: 210 40% 96.1%;
    --muted-foreground: 215.4 16.3% 46.9%;
    --accent: 210 40% 96.1%;
    --accent-foreground: 222.2 47.4% 11.2%;
    --destructive: 0 84.2% 60.2%;
    --destructive-foreground: 210 40% 98%;
    --border: 214.3 31.8% 91.4%;
    --input: 214.3 31.8% 91.4%;
    --ring: 221.2 83.2% 53.3%;
    --radius: 0.5rem;
  }

  .dark {
    --background: 222.2 84% 4.9%;
    --foreground: 210 40% 98%;
    --card: 222.2 84% 4.9%;
    --card-foreground: 210 40% 98%;
    --primary: 217.2 91.2% 59.8%;
    --primary-foreground: 222.2 47.4% 11.2%;
    --secondary: 217.2 32.6% 17.5%;
    --secondary-foreground: 210 40% 98%;
    --muted: 217.2 32.6% 17.5%;
    --muted-foreground: 215 20.2% 65.1%;
    --accent: 217.2 32.6% 17.5%;
    --accent-foreground: 210 40% 98%;
    --destructive: 0 62.8% 30.6%;
    --destructive-foreground: 210 40% 98%;
    --border: 217.2 32.6% 17.5%;
    --input: 217.2 32.6% 17.5%;
    --ring: 224.3 76.3% 48%;
  }
}
```
2. Update `tailwind.config.js` to use these variables:
```js
theme: {
  extend: {
    colors: {
      border: 'hsl(var(--border))',
      background: 'hsl(var(--background))',
      foreground: 'hsl(var(--foreground))',
      primary: {
        DEFAULT: 'hsl(var(--primary))',
        foreground: 'hsl(var(--primary-foreground))',
      },
      // ... add all color tokens
    },
    borderRadius: {
      lg: 'var(--radius)',
      md: 'calc(var(--radius) - 2px)',
      sm: 'calc(var(--radius) - 4px)',
    },
  },
},
```
3. Import theme.css in `src/index.css` (before custom CSS)
4. Commit: `feat(web): define CSS theme variables`

**Validation:**
```bash
npm run dev
# Inspect in DevTools: check :root and .dark CSS vars
```

---

### 1.6 Create ThemeProvider Context

**Goal:** Implement theme state management  
**Dependencies:** 1.5  
**Validation:** Theme toggles and persists

**Steps:**
1. Create `web/src/context/theme-provider.jsx`:
```jsx
import { createContext, useContext, useEffect, useState } from 'react'

const ThemeContext = createContext({
  theme: 'light',
  setTheme: () => {},
})

export function ThemeProvider({ children }) {
  const [theme, setTheme] = useState(() => {
    const saved = localStorage.getItem('theme')
    if (saved) return saved
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  })

  useEffect(() => {
    const root = window.document.documentElement
    root.classList.remove('light', 'dark')
    root.classList.add(theme)
    localStorage.setItem('theme', theme)
  }, [theme])

  return (
    <ThemeContext.Provider value={{ theme, setTheme }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme() {
  return useContext(ThemeContext)
}
```
2. Update `src/main.jsx` to wrap App in ThemeProvider:
```jsx
import { ThemeProvider } from './context/theme-provider'

<ThemeProvider>
  <App />
</ThemeProvider>
```
3. Test: Add console.log in App to verify theme value
4. Commit: `feat(web): add ThemeProvider context`

**Validation:**
```jsx
// In App.jsx temporarily:
const { theme } = useTheme()
console.log('Current theme:', theme)
// Check localStorage and <html class>
```

---

### 1.7 Add Base UI Components

**Goal:** Install foundational shadcn components  
**Dependencies:** 1.4, 1.5  
**Validation:** Components imported successfully

**Steps:**
1. Run: `npx shadcn@latest add button --yes`
2. Run: `npx shadcn@latest add card --yes`
3. Run: `npx shadcn@latest add badge --yes`
4. Run: `npx shadcn@latest add input --yes`
5. Run: `npx shadcn@latest add table --yes`
6. Run: `npx shadcn@latest add select --yes`
7. Run: `npx shadcn@latest add dialog --yes`
8. Run: `npx shadcn@latest add sheet --yes`
9. Verify all files in `src/components/ui/` compile
10. Commit: `feat(web): add shadcn base components`

**Validation:**
```bash
ls src/components/ui/*.jsx
# Should see: button.jsx, card.jsx, badge.jsx, input.jsx, table.jsx, select.jsx, dialog.jsx, sheet.jsx
npm run build  # Should succeed
```

---

## Phase 2: Layout Migration (2-3 days)

### 2.1 Install Sidebar Component

**Goal:** Add shadcn sidebar primitives  
**Dependencies:** 1.7  
**Validation:** Sidebar renders

**Steps:**
1. Run: `npx shadcn@latest add sidebar --yes`
2. Verify `src/components/ui/sidebar.jsx` exists
3. Install lucide-react icons: `npm install lucide-react`
4. Test import: `import { Sidebar } from '@/components/ui/sidebar'`
5. Commit: `feat(web): add sidebar component`

**Validation:**
```bash
npm run build
```

---

### 2.2 Create AppSidebar Component

**Goal:** Build custom sidebar with navigation items  
**Dependencies:** 2.1  
**Validation:** Sidebar displays all nav items

**Steps:**
1. Create `web/src/components/layout/app-sidebar.jsx`:
```jsx
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
} from '@/components/ui/sidebar'
import {
  LayoutDashboard,
  TrendingUp,
  LineChart,
  Activity,
  FileText,
  Layers,
  AlertTriangle,
  MemoryStick,
} from 'lucide-react'

export function AppSidebar({ activeTab, setActiveTab }) {
  const navItems = [
    { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
    { id: 'metadata-complexity', label: 'Metadata Complexity', icon: Layers },
    { id: 'metrics-overview', label: 'Metrics Overview', icon: TrendingUp },
    { id: 'active-series', label: 'Active Series', icon: Activity },
    { id: 'metrics', label: 'Metrics Details', icon: LineChart },
    { id: 'traces', label: 'Traces', icon: Activity },
    { id: 'logs', label: 'Logs', icon: FileText },
    { id: 'attributes', label: 'Attributes', icon: Layers },
    { id: 'noisy-neighbors', label: 'Noisy Neighbors', icon: AlertTriangle },
    { id: 'memory', label: 'Memory', icon: MemoryStick },
  ]

  return (
    <Sidebar>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              {navItems.map(item => (
                <SidebarMenuItem key={item.id}>
                  <SidebarMenuButton
                    onClick={() => setActiveTab(item.id)}
                    isActive={activeTab === item.id}
                  >
                    <item.icon />
                    <span>{item.label}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  )
}
```
2. Test: Import in App.jsx (don't render yet)
3. Commit: `feat(web): create AppSidebar component`

**Validation:**
```jsx
// Verify no import errors
import { AppSidebar } from '@/components/layout/app-sidebar'
```

---

### 2.3 Create AppHeader Component

**Goal:** Build header with title and theme toggle  
**Dependencies:** 1.6, 1.7  
**Validation:** Header renders with theme toggle

**Steps:**
1. Create `web/src/components/layout/app-header.jsx`:
```jsx
import { Button } from '@/components/ui/button'
import { useTheme } from '@/context/theme-provider'
import { Sun, Moon, Trash2 } from 'lucide-react'
import { useState } from 'react'

export function AppHeader({ onClearData }) {
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
            <Trash2 className="h-4 w-4" />
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
```
2. Test: Render in App.jsx temporarily
3. Verify theme toggle works
4. Commit: `feat(web): create AppHeader component`

**Validation:**
- Click theme toggle → theme switches
- Click Clear Data → confirmation appears

---

### 2.4 Integrate Sidebar and Header into App

**Goal:** Replace tabs with sidebar layout  
**Dependencies:** 2.2, 2.3  
**Validation:** Navigation works via sidebar

**Steps:**
1. Update `src/App.jsx`:
```jsx
import { SidebarProvider } from '@/components/ui/sidebar'
import { AppSidebar } from '@/components/layout/app-sidebar'
import { AppHeader } from '@/components/layout/app-header'

function App() {
  // Remove darkMode state (now in ThemeProvider)
  // Keep activeTab state for view routing

  return (
    <SidebarProvider>
      <div className="flex min-h-screen">
        <AppSidebar activeTab={activeTab} setActiveTab={setActiveTab} />
        <main className="flex-1">
          <AppHeader />
          <div className="container mx-auto p-6">
            {/* Existing view components */}
            {activeTab === 'dashboard' && <Dashboard />}
            {activeTab === 'metrics' && <MetricsView />}
            {/* ... etc */}
          </div>
        </main>
      </div>
    </SidebarProvider>
  )
}
```
2. Remove old `.tabs` CSS section
3. Remove old header JSX
4. Test all navigation items
5. Commit: `feat(web): integrate sidebar layout into App`

**Validation:**
- Click each sidebar item → correct view loads
- Sidebar is collapsible (click toggle)
- Theme toggle works
- Mobile: sidebar becomes overlay

---

### 2.5 Remove Legacy Tab Navigation

**Goal:** Clean up old CSS and code  
**Dependencies:** 2.4  
**Validation:** No visual regressions

**Steps:**
1. Remove from `index.css`:
   - `.tabs` class
   - `.tab` class
   - `.tab.active` class
   - Old `.header` styles (keep only what's needed)
2. Remove `darkMode` state from App.jsx
3. Remove `toggleDarkMode` function
4. Remove old dark mode useEffect
5. Verify no references to removed classes: `rg "\.tabs|\.tab" src/`
6. Commit: `refactor(web): remove legacy tab navigation`

**Validation:**
```bash
rg "\.tabs|className=\"tab" web/src/
# Should return 0 results
npm run build  # No errors
```

---

## Phase 3: Component Migration (3-4 days)

### 3.1 Migrate Dashboard Stats to Cards

**Goal:** Replace `.stat-card` with shadcn Card  
**Dependencies:** 2.4  
**Validation:** Dashboard looks correct in both themes

**Steps:**
1. Update `src/components/Dashboard.jsx`:
```jsx
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'

// Replace this:
<div className="stat-card">
  <div className="stat-value">{metrics}</div>
  <div className="stat-label">Metrics</div>
</div>

// With this:
<Card>
  <CardHeader className="pb-2">
    <CardTitle className="text-sm font-medium text-muted-foreground">
      Metrics
    </CardTitle>
  </CardHeader>
  <CardContent>
    <div className="text-2xl font-bold">{metrics}</div>
  </CardContent>
</Card>
```
2. Apply to all stat cards in Dashboard
3. Remove `.stat-card` CSS from index.css
4. Test visual appearance
5. Commit: `feat(web): migrate Dashboard stats to shadcn Card`

**Validation:**
- Stats look visually similar to before
- Dark mode works
- No layout shifts

---

### 3.2 Migrate Tables to shadcn Table Component

**Goal:** Replace custom `<table>` with shadcn Table  
**Dependencies:** 1.7  
**Validation:** Tables render correctly

**Steps:**
1. Update `src/components/MetricsView.jsx`:
```jsx
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

// Replace <table><thead><tbody> with:
<Table>
  <TableHeader>
    <TableRow>
      <TableHead>Name</TableHead>
      <TableHead>Type</TableHead>
      <TableHead>Cardinality</TableHead>
      <TableHead>Samples</TableHead>
    </TableRow>
  </TableHeader>
  <TableBody>
    {metrics.map(m => (
      <TableRow key={m.name}>
        <TableCell>{m.name}</TableCell>
        <TableCell>{m.type}</TableCell>
        <TableCell>{m.cardinality}</TableCell>
        <TableCell>{m.sample_count}</TableCell>
      </TableRow>
    ))}
  </TableBody>
</Table>
```
2. Apply to TracesView, LogsView, AttributesView
3. Remove custom `table`, `th`, `td` CSS from index.css
4. Commit: `feat(web): migrate tables to shadcn Table component`

**Validation:**
- All tables display data correctly
- Hover effects work
- Borders and spacing look good

---

### 3.3 Migrate Badges

**Goal:** Replace `.badge` classes with shadcn Badge  
**Dependencies:** 1.7  
**Validation:** Badges styled correctly

**Steps:**
1. Update all files using `.badge`:
```jsx
import { Badge } from '@/components/ui/badge'

// Replace:
<span className="badge high">High</span>

// With:
<Badge variant="destructive">High</Badge>
```
2. Map variants:
   - `.badge.high` → `variant="destructive"`
   - `.badge.medium` → `variant="secondary"` (with custom color)
   - `.badge.low` → `variant="outline"`
3. Remove `.badge` CSS from index.css
4. Commit: `feat(web): migrate badges to shadcn Badge component`

**Validation:**
- Badges display correctly
- Colors match intent (red=high, yellow=medium, gray=low)

---

### 3.4 Migrate Search and Filter Inputs

**Goal:** Replace custom inputs with shadcn components  
**Dependencies:** 1.7  
**Validation:** Filters work identically

**Steps:**
1. Update MetricsView, LogsView, etc:
```jsx
import { Input } from '@/components/ui/input'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'

// Replace .search-box:
<Input
  type="text"
  placeholder="Search metrics..."
  value={search}
  onChange={(e) => setSearch(e.target.value)}
  className="max-w-sm"
/>

// Replace <select>:
<Select value={filter} onValueChange={setFilter}>
  <SelectTrigger className="w-[180px]">
    <SelectValue placeholder="Filter by..." />
  </SelectTrigger>
  <SelectContent>
    <SelectItem value="all">All</SelectItem>
    <SelectItem value="high">High Cardinality</SelectItem>
  </SelectContent>
</Select>
```
2. Apply across all views with filters
3. Remove `.search-box`, `.filter-group` CSS
4. Commit: `feat(web): migrate search and filters to shadcn components`

**Validation:**
- Search works
- Filters update view
- Placeholder text appears

---

## Phase 4: Interactive Components (2-3 days)

### 4.1 Migrate Details Modal to Dialog

**Goal:** Replace Details view with shadcn Dialog  
**Dependencies:** 1.7  
**Validation:** Details opens/closes correctly

**Steps:**
1. Update `src/components/Details.jsx`:
```jsx
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'

export function Details({ type, name, onBack }) {
  return (
    <Dialog open={true} onOpenChange={() => onBack()}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>{type}: {name}</DialogTitle>
          <DialogDescription>
            Detailed metadata for this {type}
          </DialogDescription>
        </DialogHeader>
        {/* Existing detail content */}
      </DialogContent>
    </Dialog>
  )
}
```
2. Test opening/closing
3. Verify Escape key closes dialog
4. Commit: `feat(web): migrate Details to shadcn Dialog`

**Validation:**
- Click metric → details modal opens
- Click X or Escape → modal closes
- Click outside → modal closes

---

### 4.2 Migrate Deep-Dive Views to Sheet

**Goal:** Use Sheet (drawer) for ServiceExplorer, LogServiceDetails  
**Dependencies:** 1.7  
**Validation:** Drawers slide in/out

**Steps:**
1. Update `src/components/ServiceExplorer.jsx`:
```jsx
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from '@/components/ui/sheet'

export function ServiceExplorer({ serviceName, onBack }) {
  return (
    <Sheet open={true} onOpenChange={() => onBack()}>
      <SheetContent side="right" className="w-full sm:max-w-2xl overflow-y-auto">
        <SheetHeader>
          <SheetTitle>Service: {serviceName}</SheetTitle>
          <SheetDescription>
            Explore all telemetry for this service
          </SheetDescription>
        </SheetHeader>
        {/* Existing content */}
      </SheetContent>
    </Sheet>
  )
}
```
2. Apply to LogServiceDetails, TemplateDetails, LogPatternDetails
3. Commit: `feat(web): migrate detail views to shadcn Sheet`

**Validation:**
- Drawer slides in from right
- Scrollable when content overflows
- Closes on back button or outside click

---

### 4.3 Add Loading States

**Goal:** Show skeletons while data loads  
**Dependencies:** 1.7  
**Validation:** Skeleton appears during fetch

**Steps:**
1. Run: `npx shadcn@latest add skeleton --yes`
2. Create `src/components/loading-skeleton.jsx`:
```jsx
import { Skeleton } from '@/components/ui/skeleton'

export function TableSkeleton() {
  return (
    <div className="space-y-2">
      {[...Array(5)].map((_, i) => (
        <Skeleton key={i} className="h-12 w-full" />
      ))}
    </div>
  )
}

export function CardSkeleton() {
  return (
    <div className="space-y-2">
      <Skeleton className="h-4 w-[200px]" />
      <Skeleton className="h-8 w-[100px]" />
    </div>
  )
}
```
3. Use in views: `{loading ? <TableSkeleton /> : <Table>...</Table>}`
4. Commit: `feat(web): add loading skeleton states`

**Validation:**
- Skeleton appears while fetch is pending
- Transitions to content smoothly

---

### 4.4 Add Error Boundaries

**Goal:** Graceful error handling  
**Dependencies:** None  
**Validation:** Errors don't crash app

**Steps:**
1. Create `src/components/error-boundary.jsx`:
```jsx
import { Component } from 'react'
import { AlertTriangle } from 'lucide-react'

export class ErrorBoundary extends Component {
  state = { hasError: false, error: null }

  static getDerivedStateFromError(error) {
    return { hasError: true, error }
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex flex-col items-center justify-center p-8 text-center">
          <AlertTriangle className="h-12 w-12 text-destructive mb-4" />
          <h2 className="text-xl font-bold mb-2">Something went wrong</h2>
          <p className="text-muted-foreground mb-4">{this.state.error?.message}</p>
          <Button onClick={() => window.location.reload()}>Reload Page</Button>
        </div>
      )
    }
    return this.props.children
  }
}
```
2. Wrap App in ErrorBoundary (in main.jsx)
3. Test with intentional error
4. Commit: `feat(web): add error boundary`

**Validation:**
```jsx
// Test: throw new Error('Test') in a component
// Should show error UI instead of blank page
```

---

## Phase 5: Polish & Accessibility (1-2 days)

### 5.1 Add Keyboard Shortcuts

**Goal:** Power user keyboard navigation  
**Dependencies:** None  
**Validation:** Shortcuts work

**Steps:**
1. Create `src/hooks/use-keyboard-shortcuts.js`:
```jsx
import { useEffect } from 'react'

export function useKeyboardShortcuts(shortcuts) {
  useEffect(() => {
    const handler = (e) => {
      for (const [key, fn] of Object.entries(shortcuts)) {
        const [modifier, keyCode] = key.split('+')
        if (
          (modifier === 'cmd' && (e.metaKey || e.ctrlKey)) &&
          e.key.toLowerCase() === keyCode
        ) {
          e.preventDefault()
          fn()
        }
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [shortcuts])
}
```
2. Use in App.jsx:
```jsx
useKeyboardShortcuts({
  'cmd+k': () => setShowCommandPalette(true),
  'cmd+d': () => setActiveTab('dashboard'),
  'cmd+m': () => setActiveTab('metrics'),
})
```
3. Commit: `feat(web): add keyboard shortcuts`

**Validation:**
- Cmd/Ctrl+D → navigate to Dashboard
- Cmd/Ctrl+M → navigate to Metrics

---

### 5.2 Add ARIA Labels

**Goal:** Screen reader accessibility  
**Dependencies:** All component migrations  
**Validation:** Lighthouse a11y score ≥90

**Steps:**
1. Audit all interactive elements:
   - Buttons without text → add `aria-label`
   - Icons → add `aria-hidden="true"`
   - Forms → add `aria-describedby`
2. Example fixes:
```jsx
<Button size="icon" aria-label="Toggle theme">
  <Sun aria-hidden="true" />
</Button>

<Input
  aria-label="Search metrics"
  aria-describedby="search-help"
/>
<span id="search-help" className="sr-only">
  Type to filter metrics by name
</span>
```
3. Run: `npm run lighthouse` (create script if needed)
4. Commit: `a11y(web): add ARIA labels to all interactive elements`

**Validation:**
```bash
# Lighthouse accessibility audit
lighthouse http://localhost:3000 --only-categories=accessibility
# Target: score ≥90
```

---

### 5.3 Mobile Responsive Testing

**Goal:** Ensure all views work on mobile  
**Dependencies:** All phases  
**Validation:** Manual testing at 375px, 768px, 1024px

**Steps:**
1. Test each view at breakpoints:
   - 375px (mobile)
   - 768px (tablet)
   - 1024px (desktop)
2. Fix issues:
   - Tables: horizontal scroll wrapper
   - Cards: stack vertically
   - Sidebar: overlay mode on mobile
3. Add responsive classes:
```jsx
<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
  {/* Stats cards */}
</div>
```
4. Commit: `feat(web): improve mobile responsiveness`

**Validation:**
- Resize browser to 375px
- All content accessible (no overflow hidden)
- No horizontal scroll (except intentional tables)

---

### 5.4 Bundle Size Optimization

**Goal:** Keep bundle ≤300KB gzipped  
**Dependencies:** All migrations complete  
**Validation:** Build size check

**Steps:**
1. Run: `npm run build`
2. Check: `ls -lh dist/assets/*.js`
3. If over budget:
   - Lazy load views: `const MetricsView = lazy(() => import('./components/MetricsView'))`
   - Tree-shake Radix imports
   - Remove unused icons from lucide-react
4. Install bundle analyzer: `npm install -D rollup-plugin-visualizer`
5. Add to vite.config.js:
```js
import { visualizer } from 'rollup-plugin-visualizer'

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    visualizer({ filename: 'dist/stats.html' }),
  ],
})
```
6. Analyze and optimize
7. Commit: `perf(web): optimize bundle size`

**Validation:**
```bash
npm run build
gzip -c dist/assets/index-*.js | wc -c
# Should be ≤300KB (307200 bytes)
```

---

### 5.5 Final Integration Testing

**Goal:** Verify all features work end-to-end  
**Dependencies:** All previous tasks  
**Validation:** Checklist complete

**Checklist:**
- [ ] All 10 main views render without errors
- [ ] Navigation works (sidebar, breadcrumbs, back buttons)
- [ ] Dark mode toggle works across all views
- [ ] API calls unchanged (check network tab)
- [ ] Search/filter/sort work in tables
- [ ] Details/modal/drawer views open/close correctly
- [ ] Theme persists on page reload
- [ ] Sidebar collapse state persists
- [ ] Mobile: sidebar is overlay, header is responsive
- [ ] Lighthouse scores: Performance ≥80, Accessibility ≥90
- [ ] No console errors or warnings
- [ ] Bundle size ≤300KB gzipped

**Steps:**
1. Manual test each checklist item
2. Fix any issues found
3. Document known issues (if any) in CHANGELOG
4. Commit: `test(web): complete integration testing`

---

## Validation Commands

```bash
# Build check
npm run build

# Dev server
npm run dev

# Bundle size
npm run build && ls -lh dist/assets/

# Lighthouse (requires Chrome)
npx lighthouse http://localhost:3000 --view

# Code search
rg "className=\"(tabs|tab|stat-card|badge)" src/

# Verify imports
rg "from '@/components/ui'" src/
```

---

## Rollback Plan

If critical issues arise:

1. **Phase-level rollback:** `git revert <phase-commit-sha>`
2. **Feature flag:** Add `USE_NEW_UI=false` env var, conditionally render old/new
3. **CSS fallback:** Keep old CSS alongside new until Phase 5 complete

---

## Success Criteria

- [ ] All tasks completed and committed
- [ ] No regressions in functionality
- [ ] Lighthouse accessibility ≥90
- [ ] Bundle size ≤300KB gzipped
- [ ] Mobile viewport (≥375px) fully functional
- [ ] Dark mode works across all components
- [ ] Code review approved
- [ ] Production deployment successful
