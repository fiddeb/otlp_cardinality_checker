# Spec: Design System Foundation

**Capability:** `design-system`  
**Change:** `migrate-shadcn-ui`

## ADDED Requirements

### Requirement: UI Component Library Integration

**ID:** `DS-001`  
**Priority:** High  
**Category:** Infrastructure

The web UI SHALL integrate the shadcn-ui component library built on Tailwind CSS v4 and RadixUI primitives to provide accessible, composable UI components.

#### Scenario: Installing shadcn-ui dependencies

**Given** a fresh web project directory  
**When** the developer runs `npm install`  
**Then** the following packages SHALL be installed:
- `tailwindcss@^4.0.0`
- `@tailwindcss/vite@^4.0.0`
- `clsx@^2.1.1`
- `class-variance-authority@^0.7.0`
- `tailwind-merge@^2.5.4`
- `@radix-ui/react-slot@^1.1.0`

**And** the build SHALL complete without errors

#### Scenario: Configuring shadcn-ui via CLI

**Given** the dependencies are installed  
**When** the developer runs `npx shadcn@latest init`  
**Then** a `components.json` file SHALL be created with:
```json
{
  "style": "default",
  "tsx": false,
  "tailwind": {
    "config": "tailwind.config.js",
    "css": "src/styles/index.css",
    "baseColor": "slate"
  },
  "aliases": {
    "components": "@/components",
    "utils": "@/lib",
    "ui": "@/components/ui"
  }
}
```

**And** the developer SHALL be able to add components via `npx shadcn@latest add <component>`

---

### Requirement: Theme System with CSS Variables

**ID:** `DS-002`  
**Priority:** High  
**Category:** Theming

The application SHALL use CSS custom properties for theming, supporting both light and dark modes with consistent color tokens across all components.

#### Scenario: Defining theme variables

**Given** the application is initialized  
**When** `src/styles/theme.css` is loaded  
**Then** CSS variables SHALL be defined for:
- `--background`, `--foreground`
- `--card`, `--card-foreground`
- `--primary`, `--primary-foreground`
- `--secondary`, `--muted`, `--accent`
- `--destructive`, `--border`, `--input`, `--ring`

**And** each variable SHALL have both `:root` (light) and `.dark` (dark) variants

#### Scenario: Switching between light and dark modes

**Given** the user is on any page  
**When** the user clicks the theme toggle button  
**Then** the `<html>` element SHALL toggle the `dark` class  
**And** all CSS variables SHALL update to dark mode values  
**And** the preference SHALL persist in `localStorage`  
**And** the visual transition SHALL be smooth (≤300ms)

#### Scenario: Respecting system theme preference

**Given** the user has no saved theme preference  
**When** the application loads  
**Then** the theme SHALL match `window.matchMedia('(prefers-color-scheme: dark)')`  
**And** the theme SHALL update if the system preference changes

---

### Requirement: Utility Class Composition

**ID:** `DS-003`  
**Priority:** Medium  
**Category:** Developer Experience

The project SHALL provide a `cn()` utility function for composing Tailwind classes with proper conflict resolution.

#### Scenario: Creating the cn() utility

**Given** the project structure includes `src/lib/utils.ts`  
**When** the file is created  
**Then** it SHALL export a `cn()` function:
```typescript
import { clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs) {
  return twMerge(clsx(inputs))
}
```

**And** the function SHALL handle conditional classes, arrays, and merge conflicts correctly

#### Scenario: Using cn() in components

**Given** a component needs conditional styling  
**When** the component uses `className={cn('base-class', condition && 'conditional-class', className)}`  
**Then** classes SHALL be properly merged without conflicts  
**And** later classes SHALL override earlier conflicting classes

---

### Requirement: Path Alias Configuration

**ID:** `DS-004`  
**Priority:** Medium  
**Category:** Configuration

The build system SHALL support the `@/` path alias for importing components and utilities from the `src/` directory.

#### Scenario: Configuring Vite path aliases

**Given** `vite.config.js` exists  
**When** the resolve.alias configuration includes:
```javascript
{
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
}
```
**Then** imports like `import { Button } from '@/components/ui/button'` SHALL resolve correctly

#### Scenario: Configuring TypeScript paths (optional)

**Given** the project later adopts TypeScript  
**When** `tsconfig.json` includes:
```json
{
  "compilerOptions": {
    "paths": {
      "@/*": ["./src/*"]
    }
  }
}
```
**Then** the IDE SHALL provide autocomplete for `@/` imports

---

### Requirement: Base UI Components

**ID:** `DS-005`  
**Priority:** High  
**Category:** Components

The project SHALL include foundational shadcn-ui components for buttons, cards, badges, and inputs in `src/components/ui/`.

#### Scenario: Adding the Button component

**Given** shadcn-ui is configured  
**When** the developer runs `npx shadcn@latest add button`  
**Then** `src/components/ui/button.jsx` SHALL be created  
**And** the component SHALL export `Button` and `buttonVariants`  
**And** the component SHALL support variants: `default`, `destructive`, `outline`, `secondary`, `ghost`, `link`  
**And** the component SHALL support sizes: `default`, `sm`, `lg`, `icon`

#### Scenario: Adding the Card component

**Given** shadcn-ui is configured  
**When** the developer runs `npx shadcn@latest add card`  
**Then** `src/components/ui/card.jsx` SHALL be created  
**And** the file SHALL export: `Card`, `CardHeader`, `CardTitle`, `CardDescription`, `CardContent`, `CardFooter`

#### Scenario: Adding the Badge component

**Given** shadcn-ui is configured  
**When** the developer runs `npx shadcn@latest add badge`  
**Then** `src/components/ui/badge.jsx` SHALL be created  
**And** the component SHALL support variants: `default`, `secondary`, `destructive`, `outline`

#### Scenario: Adding the Table component

**Given** shadcn-ui is configured  
**When** the developer runs `npx shadcn@latest add table`  
**Then** `src/components/ui/table.jsx` SHALL be created  
**And** the file SHALL export: `Table`, `TableHeader`, `TableBody`, `TableFooter`, `TableRow`, `TableHead`, `TableCell`, `TableCaption`

---

## MODIFIED Requirements

None (new capability)

## REMOVED Requirements

None
