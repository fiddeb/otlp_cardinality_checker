# Design: Gradual ShadcnUI Migration

**Change ID:** `migrate-shadcn-ui`

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     React Application                        │
├─────────────────────────────────────────────────────────────┤
│  App.jsx (Router State)                                     │
│    ├─ Layout Components (NEW)                               │
│    │   ├─ Sidebar (shadcn)                                  │
│    │   ├─ Header (shadcn)                                   │
│    │   └─ Main Content Area                                 │
│    │                                                         │
│    ├─ View Components (MIGRATED)                            │
│    │   ├─ Dashboard → use Card, Table                       │
│    │   ├─ MetricsView → use DataTable                       │
│    │   ├─ LogsView → use Card, Badge                        │
│    │   └─ ... (14 total views)                              │
│    │                                                         │
│    └─ UI Primitives (NEW)                                   │
│        ├─ components/ui/* (shadcn components)               │
│        ├─ Button, Card, Table, Badge                        │
│        ├─ Dialog, Drawer, Sheet                             │
│        └─ Select, Input, Switch                             │
├─────────────────────────────────────────────────────────────┤
│  Theme System                                               │
│    ├─ ThemeProvider (context)                               │
│    ├─ CSS Variables (light/dark)                            │
│    └─ Tailwind CSS v4 (utility classes)                     │
├─────────────────────────────────────────────────────────────┤
│  Backend API (UNCHANGED)                                    │
│    └─ /api/v1/* (existing endpoints)                        │
└─────────────────────────────────────────────────────────────┘
```

## Component Migration Strategy

### 1. Foundation Layer (Week 1)

**Install Dependencies:**
```json
{
  "dependencies": {
    "clsx": "^2.1.1",
    "class-variance-authority": "^0.7.0",
    "tailwind-merge": "^2.5.4",
    "@radix-ui/react-slot": "^1.1.0"
  },
  "devDependencies": {
    "tailwindcss": "^4.0.0",
    "@tailwindcss/vite": "^4.0.0-alpha.25",
    "typescript": "^5.6.0"
  }
}
```

**File Structure:**
```
web/
├── src/
│   ├── components/
│   │   ├── ui/               # shadcn components (copied)
│   │   │   ├── button.tsx
│   │   │   ├── card.tsx
│   │   │   ├── table.tsx
│   │   │   └── ...
│   │   ├── layout/           # NEW: Layout components
│   │   │   ├── app-sidebar.jsx
│   │   │   ├── app-header.jsx
│   │   │   └── main-content.jsx
│   │   └── [existing].jsx    # Existing components
│   ├── lib/
│   │   └── utils.ts          # cn() utility
│   ├── context/
│   │   └── theme-provider.jsx
│   └── styles/
│       ├── index.css         # Global styles
│       └── theme.css         # CSS variables
├── components.json           # shadcn config
└── tailwind.config.js        # Tailwind config
```

### 2. Theme System Design

**CSS Variables (theme.css):**
```css
@layer base {
  :root {
    /* Light mode */
    --background: 0 0% 100%;
    --foreground: 222.2 84% 4.9%;
    --card: 0 0% 100%;
    --card-foreground: 222.2 84% 4.9%;
    --primary: 221.2 83.2% 53.3%;
    --primary-foreground: 210 40% 98%;
    /* ... */
  }

  .dark {
    /* Dark mode */
    --background: 222.2 84% 4.9%;
    --foreground: 210 40% 98%;
    --card: 222.2 84% 4.9%;
    --card-foreground: 210 40% 98%;
    /* ... */
  }
}
```

**ThemeProvider Pattern:**
```jsx
// context/theme-provider.jsx
export function ThemeProvider({ children }) {
  const [theme, setTheme] = useState(() => {
    const saved = localStorage.getItem('theme')
    return saved || (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
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
```

### 3. Layout Components

**Sidebar Component (Phase 2):**
```jsx
// components/layout/app-sidebar.jsx
import { Sidebar, SidebarContent, SidebarGroup } from '@/components/ui/sidebar'
import { LayoutDashboard, LineChart, Activity, FileText } from 'lucide-react'

export function AppSidebar() {
  const navItems = [
    { title: 'Dashboard', icon: LayoutDashboard, href: '#dashboard' },
    { title: 'Metrics', icon: LineChart, href: '#metrics' },
    { title: 'Traces', icon: Activity, href: '#traces' },
    { title: 'Logs', icon: FileText, href: '#logs' },
  ]

  return (
    <Sidebar>
      <SidebarContent>
        <SidebarGroup>
          {navItems.map(item => (
            <SidebarMenuItem key={item.href} {...item} />
          ))}
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  )
}
```

**Main Layout Wrapper:**
```jsx
// App.jsx changes
import { SidebarProvider } from '@/components/ui/sidebar'
import { AppSidebar } from '@/components/layout/app-sidebar'
import { AppHeader } from '@/components/layout/app-header'

function App() {
  return (
    <ThemeProvider>
      <SidebarProvider>
        <div className="flex min-h-screen">
          <AppSidebar />
          <main className="flex-1">
            <AppHeader />
            <div className="container mx-auto p-6">
              {/* Existing view components */}
            </div>
          </main>
        </div>
      </SidebarProvider>
    </ThemeProvider>
  )
}
```

### 4. Component Mapping

**Current → shadcn Components:**

| Current                | Replacement                  | Phase |
|------------------------|------------------------------|-------|
| `.stat-card`          | `<Card>` + `<CardContent>`   | 2     |
| `<table>`             | `<Table>` + `<TableRow>`     | 3     |
| `.tab` / `.tabs`      | `<Tabs>` + `<TabsList>`      | 2     |
| `.badge`              | `<Badge>`                    | 3     |
| `.back-button`        | `<Button variant="ghost">`   | 2     |
| `.dark-mode-toggle`   | `<ThemeToggle>` (custom)     | 1     |
| Custom modals         | `<Dialog>` / `<Sheet>`       | 4     |
| `.search-box`         | `<Input>` + `<Search>`       | 3     |
| `.filter-group`       | `<Select>` + `<Label>`       | 3     |

### 5. Data Preservation Pattern

**Before (custom CSS):**
```jsx
<div className="stat-card">
  <div className="stat-value">{value}</div>
  <div className="stat-label">{label}</div>
</div>
```

**After (shadcn):**
```jsx
<Card>
  <CardHeader className="pb-2">
    <CardTitle className="text-sm font-medium text-muted-foreground">
      {label}
    </CardTitle>
  </CardHeader>
  <CardContent>
    <div className="text-2xl font-bold">{value}</div>
  </CardContent>
</Card>
```

**Key Principles:**
1. **Same data flow** - Props/state unchanged
2. **Same API calls** - useEffect hooks unchanged
3. **Same business logic** - Only rendering changes
4. **Progressive enhancement** - Old CSS coexists temporarily

### 6. Build Configuration

**Tailwind CSS v4 Setup:**
```js
// tailwind.config.js
export default {
  content: ['./index.html', './src/**/*.{js,jsx,ts,tsx}'],
  darkMode: ['class'],
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
        // ... rest from shadcn defaults
      },
    },
  },
  plugins: [],
}
```

**Vite Config (no changes needed):**
```js
// vite.config.js - already correct
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  // ... existing proxy config
})
```

### 7. TypeScript Adoption Path

**Gradual Migration:**
1. **Phase 1-2:** All new code in `.jsx` (no TS yet)
2. **Phase 3:** Add `tsconfig.json`, allow `.tsx` for new components
3. **Phase 4+:** Convert existing `.jsx` → `.tsx` as touched

**Config (when ready):**
```json
// tsconfig.json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "jsx": "react-jsx",
    "allowJs": true,
    "strict": false,  // Start permissive
    "esModuleInterop": true,
    "skipLibCheck": true,
    "paths": {
      "@/*": ["./src/*"]
    }
  }
}
```

### 8. Accessibility Improvements

**ARIA Labels:**
```jsx
// Before
<button className="dark-mode-toggle" onClick={toggleDarkMode}>
  {darkMode ? '☀️' : '🌙'}
</button>

// After
<Button
  variant="ghost"
  size="icon"
  onClick={toggleDarkMode}
  aria-label={darkMode ? 'Switch to light mode' : 'Switch to dark mode'}
>
  {darkMode ? <Sun className="h-5 w-5" /> : <Moon className="h-5 w-5" />}
</Button>
```

**Keyboard Navigation:**
- All interactive elements focusable
- Tab order logical
- Escape to close modals/drawers
- Enter/Space for button activation

### 9. Performance Considerations

**Code Splitting:**
```jsx
// Lazy load heavy views
const MetricsView = lazy(() => import('./components/MetricsView'))
const LogsView = lazy(() => import('./components/LogsView'))

// In App.jsx
<Suspense fallback={<LoadingSkeleton />}>
  {activeTab === 'metrics' && <MetricsView />}
</Suspense>
```

**Bundle Size Targets:**
- Current: ~150KB (gzipped)
- After migration: ≤300KB (gzipped)
- Acceptable: +150KB for UI library

**Optimization:**
- Tree-shake Radix UI components
- Use `lucide-react` icons (not `react-icons`)
- Lazy load views
- Minimize Tailwind generated CSS

### 10. Testing Strategy

**Visual Regression:**
```bash
# Take screenshots before migration
npm run screenshot:baseline

# Compare after each phase
npm run screenshot:compare
```

**Functional Testing:**
- All 14 views render without errors
- Dark mode toggle works
- API calls unchanged (network tab inspection)
- Navigation state preserved

**Accessibility Testing:**
```bash
# Lighthouse CI
lighthouse http://localhost:3000 --preset=desktop --only-categories=accessibility

# axe-core checks
npm run test:a11y
```

## Migration Checklist

### Phase 1: Foundation ✅
- [ ] Install dependencies (Tailwind, shadcn, Radix)
- [ ] Configure `components.json`
- [ ] Set up CSS variables in `theme.css`
- [ ] Create `ThemeProvider`
- [ ] Copy base UI components (Button, Card, Badge)

### Phase 2: Layout ✅
- [ ] Create `AppSidebar` component
- [ ] Create `AppHeader` component
- [ ] Wrap App in `SidebarProvider`
- [ ] Migrate tab navigation to Sidebar
- [ ] Test responsive collapse

### Phase 3: Data Display ✅
- [ ] Migrate Dashboard stats to `Card`
- [ ] Convert metrics table to shadcn `Table`
- [ ] Add search with shadcn `Input`
- [ ] Implement filters with `Select`
- [ ] Add `Badge` for severity/cardinality

### Phase 4: Interactive ✅
- [ ] Migrate Details modal to `Dialog`
- [ ] Convert deep-dive views to `Sheet`
- [ ] Add loading skeletons
- [ ] Implement error states

### Phase 5: Polish ✅
- [ ] Add keyboard shortcuts
- [ ] ARIA labels on all controls
- [ ] Mobile responsive test (≥375px)
- [ ] Lighthouse audit (≥90 a11y score)
- [ ] Bundle size check

## Rollback Plan

If critical issues found:

1. **Git revert** - Each phase is a separate commit
2. **Feature flag** - `USE_SHADCN_UI=false` env var
3. **CSS fallback** - Keep `index.css` until Phase 5 complete
4. **Component isolation** - Old and new components can coexist

## Success Metrics

1. **Visual parity** - All views look ≥ as good as before
2. **Performance** - No regression in load time
3. **Accessibility** - Score ≥90 (from current ~60)
4. **Code reduction** - 30% less custom CSS
5. **User feedback** - No complaints about broken flows
