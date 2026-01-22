# Spec: Theme System

**Capability:** `theme-system`  
**Change:** `migrate-shadcn-ui`

## ADDED Requirements

### Requirement: ThemeProvider Context

**ID:** `TS-001`  
**Priority:** High  
**Category:** Theming

The application SHALL provide a ThemeProvider React context to manage theme state and apply theme classes to the document root.

#### Scenario: Initializing theme on app load

**Given** the application loads  
**When** ThemeProvider mounts  
**Then** it SHALL check `localStorage.getItem('theme')`  
**And** if a saved theme exists, it SHALL be applied  
**And** if no saved theme exists, it SHALL detect system preference via `window.matchMedia('(prefers-color-scheme: dark)')`  
**And** the appropriate class (`light` or `dark`) SHALL be added to `<html>`

#### Scenario: Programmatically changing theme

**Given** ThemeProvider is mounted  
**When** a component calls `setTheme('dark')`  
**Then** the `<html>` element SHALL:
- Remove `light` class if present
- Add `dark` class
- Update `localStorage` with `theme: 'dark'`  
**And** all CSS variables defined in `.dark` scope SHALL apply

#### Scenario: Listening to system theme changes

**Given** the theme is set to "system" mode  
**When** the OS theme changes (light ↔ dark)  
**Then** the application theme SHALL update automatically  
**And** the `<html>` class SHALL reflect the new system preference

---

### Requirement: Theme Toggle Component

**ID:** `TS-002`  
**Priority:** High  
**Category:** UI Component

The application SHALL provide a theme toggle button that cycles between light, dark, and system modes.

#### Scenario: Rendering the theme toggle

**Given** the header is rendered  
**When** the theme toggle button is displayed  
**Then** it SHALL show:
- Sun icon when dark mode is active
- Moon icon when light mode is active  
**And** the button SHALL have an accessible label: "Toggle theme"

#### Scenario: Toggling between themes

**Given** the current theme is "light"  
**When** the user clicks the toggle button  
**Then** the theme SHALL change to "dark"  
**And** the icon SHALL change to a Sun  

**Given** the current theme is "dark"  
**When** the user clicks the toggle button  
**Then** the theme SHALL change to "light"  
**And** the icon SHALL change to a Moon

#### Scenario: Visual feedback on toggle

**Given** the user clicks the theme toggle  
**When** the theme switches  
**Then** the transition SHALL be smooth (fade, ≤200ms)  
**And** the button SHALL briefly highlight to indicate the action  
**And** there SHALL be no flash of unstyled content (FOUC)

---

### Requirement: CSS Variables for Colors

**ID:** `TS-003`  
**Priority:** High  
**Category:** Design Tokens

The application SHALL define semantic color tokens as CSS custom properties, following the shadcn-ui color system.

#### Scenario: Defining light mode colors

**Given** `src/styles/theme.css` is loaded  
**When** the `:root` scope is evaluated  
**Then** the following CSS variables SHALL be defined:
```css
:root {
  --background: 0 0% 100%;           /* White */
  --foreground: 222.2 84% 4.9%;      /* Near black */
  --card: 0 0% 100%;                  /* White */
  --card-foreground: 222.2 84% 4.9%;  /* Near black */
  --primary: 221.2 83.2% 53.3%;       /* Blue */
  --primary-foreground: 210 40% 98%;  /* Off-white */
  --secondary: 210 40% 96.1%;         /* Light gray */
  --muted: 210 40% 96.1%;             /* Light gray */
  --accent: 210 40% 96.1%;            /* Light gray */
  --destructive: 0 84.2% 60.2%;       /* Red */
  --border: 214.3 31.8% 91.4%;        /* Light border */
  --input: 214.3 31.8% 91.4%;         /* Light border */
  --ring: 221.2 83.2% 53.3%;          /* Blue */
  --radius: 0.5rem;                    /* 8px */
}
```

#### Scenario: Defining dark mode colors

**Given** `src/styles/theme.css` is loaded  
**When** the `.dark` scope is evaluated  
**Then** the following CSS variables SHALL override light mode:
```css
.dark {
  --background: 222.2 84% 4.9%;       /* Near black */
  --foreground: 210 40% 98%;          /* Off-white */
  --card: 222.2 84% 4.9%;             /* Near black */
  --card-foreground: 210 40% 98%;     /* Off-white */
  --primary: 217.2 91.2% 59.8%;       /* Bright blue */
  --primary-foreground: 222.2 47.4% 11.2%; /* Dark */
  --secondary: 217.2 32.6% 17.5%;     /* Dark gray */
  --muted: 217.2 32.6% 17.5%;         /* Dark gray */
  --accent: 217.2 32.6% 17.5%;        /* Dark gray */
  --destructive: 0 62.8% 30.6%;       /* Dark red */
  --border: 217.2 32.6% 17.5%;        /* Dark border */
  --input: 217.2 32.6% 17.5%;         /* Dark border */
  --ring: 224.3 76.3% 48%;            /* Blue */
}
```

#### Scenario: Using color variables in components

**Given** a component needs a background color  
**When** the component uses `className="bg-background text-foreground"`  
**Then** Tailwind SHALL generate:
```css
.bg-background {
  background-color: hsl(var(--background));
}
.text-foreground {
  color: hsl(var(--foreground));
}
```
**And** the color SHALL automatically adapt to light/dark mode

---

### Requirement: Smooth Theme Transitions

**ID:** `TS-004`  
**Priority:** Medium  
**Category:** UX

The application SHALL provide smooth visual transitions when switching themes to avoid jarring flashes.

#### Scenario: Preventing flash of unstyled content (FOUC)

**Given** the user loads the page with a saved dark theme  
**When** the page initially renders  
**Then** the dark theme SHALL apply BEFORE the first paint  
**And** there SHALL be no visible flash of light mode

**Implementation:** Inline blocking script in `<head>` to set theme class before render

#### Scenario: Transitioning theme colors

**Given** the user switches from light to dark  
**When** the theme changes  
**Then** color properties SHALL transition smoothly over 200ms  
**And** the transition SHALL use an easing function (ease-in-out)  

**Implementation:**
```css
* {
  transition: background-color 0.2s ease, color 0.2s ease;
}
```

---

### Requirement: Theme Persistence

**ID:** `TS-005`  
**Priority:** High  
**Category:** State Management

The application SHALL persist the user's theme preference across sessions using localStorage.

#### Scenario: Saving theme preference

**Given** the user changes the theme to "dark"  
**When** the theme is updated  
**Then** `localStorage.setItem('theme', 'dark')` SHALL be called  
**And** the value SHALL persist after page reload

#### Scenario: Loading saved theme

**Given** the user previously saved "dark" theme  
**When** the application loads  
**Then** `localStorage.getItem('theme')` SHALL return "dark"  
**And** the dark theme SHALL be applied  
**And** the theme toggle SHALL show the correct icon (Sun)

#### Scenario: Clearing theme preference

**Given** the user has a saved theme  
**When** the user clears browser data (localStorage)  
**Then** the application SHALL fall back to system preference  
**And** the theme SHALL match `prefers-color-scheme` media query

---

## MODIFIED Requirements

### Requirement: Dark Mode Toggle (MIGRATED)

**ID:** `UI-DARK-001` (from existing system)  
**Status:** MIGRATED  
**Replacement:** `TS-002` (Theme Toggle Component)

**Before:** Custom dark mode toggle with emoji icons (☀️/🌙) and manual class toggling on `<body>`  
**After:** shadcn-ui based theme toggle with Lucide React icons and context-based state management

**Changes:**
- Replace `body.dark-mode` class with `html.dark` class
- Replace emoji icons with `<Sun />` and `<Moon />` from lucide-react
- Move state management from component state to ThemeProvider context
- Add system theme detection (new feature)

---

## REMOVED Requirements

None
