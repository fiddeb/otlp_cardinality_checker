# Spec: Layout Components

**Capability:** `layout-components`  
**Change:** `migrate-shadcn-ui`

## ADDED Requirements

### Requirement: Application Sidebar Navigation

**ID:** `LC-001`  
**Priority:** High  
**Category:** Navigation

The application SHALL provide a collapsible sidebar navigation component that replaces the horizontal tab bar with a vertical navigation menu.

#### Scenario: Rendering the sidebar with navigation items

**Given** the application is loaded  
**When** the sidebar is rendered  
**Then** it SHALL display navigation items for:
- Dashboard
- Metadata Complexity
- Metrics Overview
- Active Series
- Metrics Details
- Traces
- Logs
- Attributes
- Noisy Neighbors
- Memory

**And** each item SHALL have an appropriate icon (from lucide-react)  
**And** the current active tab SHALL be visually highlighted

#### Scenario: Collapsing and expanding the sidebar

**Given** the sidebar is visible  
**When** the user clicks the collapse toggle  
**Then** the sidebar SHALL animate to icon-only mode (width: 60px)  
**And** text labels SHALL be hidden  
**And** tooltips SHALL appear on hover showing the full label  
**And** the preference SHALL persist in `localStorage`

#### Scenario: Responsive sidebar on mobile

**Given** the viewport width is ≤768px  
**When** the application loads  
**Then** the sidebar SHALL be hidden by default  
**And** a hamburger menu button SHALL appear in the header  
**When** the user clicks the hamburger button  
**Then** the sidebar SHALL slide in as an overlay  
**And** clicking outside SHALL close the sidebar

---

### Requirement: Application Header

**ID:** `LC-002`  
**Priority:** High  
**Category:** Layout

The application SHALL provide a header component containing the app title, theme toggle, and admin actions.

#### Scenario: Rendering the header

**Given** the application is loaded  
**When** the header is visible  
**Then** it SHALL display:
- Application title: "OTLP Cardinality Checker"
- Subtitle: "Analyze metadata structure from OpenTelemetry signals"
- Theme toggle button (Sun/Moon icon)
- Clear Data button

**And** the header SHALL remain sticky at the top when scrolling

#### Scenario: Header actions

**Given** the header is rendered  
**When** the user clicks "Clear Data"  
**Then** a confirmation dialog SHALL appear  
**And** confirming SHALL call `POST /api/v1/admin/clear`  
**And** success SHALL reload the page

**When** the user clicks the theme toggle  
**Then** the theme SHALL switch between light/dark  
**And** the icon SHALL update accordingly

---

### Requirement: Main Content Area Layout

**ID:** `LC-003`  
**Priority:** Medium  
**Category:** Layout

The application SHALL wrap view components in a consistent main content container with proper spacing and max-width constraints.

#### Scenario: Content container structure

**Given** any view is rendered  
**When** the view is displayed  
**Then** it SHALL be wrapped in a container with:
- Horizontal padding: 24px
- Vertical padding: 24px
- Max-width: 1400px (centered on wide screens)
- Background: `--background` color

#### Scenario: Scroll behavior

**Given** content exceeds viewport height  
**When** the user scrolls  
**Then** the sidebar and header SHALL remain fixed  
**And** only the main content area SHALL scroll  
**And** scroll position SHALL be preserved when switching tabs

---

### Requirement: Breadcrumb Navigation

**ID:** `LC-004`  
**Priority:** Low  
**Category:** Navigation

The application SHALL display breadcrumb navigation when users navigate into detail views (Details, ServiceExplorer, TemplateDetails, etc.).

#### Scenario: Showing breadcrumbs in detail views

**Given** the user is viewing "Dashboard"  
**When** the user clicks on a service  
**Then** the breadcrumb SHALL show: `Dashboard > service.example`  
**And** clicking "Dashboard" SHALL navigate back

**Given** the user is in a LogPatternDetails view  
**Then** the breadcrumb SHALL show: `Logs > my-service > ERROR > <pattern>`  
**And** each segment SHALL be clickable to navigate up

#### Scenario: Breadcrumbs on main views

**Given** the user is on a main view (Dashboard, Metrics, Traces, Logs)  
**When** the view is rendered  
**Then** NO breadcrumb SHALL be displayed

---

### Requirement: SidebarProvider Context

**ID:** `LC-005`  
**Priority:** High  
**Category:** State Management

The application SHALL use a SidebarProvider context to manage sidebar open/collapsed state across components.

#### Scenario: Initializing sidebar state

**Given** the application loads  
**When** SidebarProvider mounts  
**Then** it SHALL read the saved state from `localStorage.getItem('sidebar:state')`  
**And** default to "expanded" if no saved state exists  
**And** default to "collapsed" on mobile viewports (≤768px)

#### Scenario: Sharing sidebar state

**Given** the SidebarProvider is mounted  
**When** any child component calls `useSidebar()`  
**Then** it SHALL receive:
- `open`: boolean (sidebar expanded/collapsed)
- `setOpen`: function to toggle state
- `isMobile`: boolean (viewport ≤768px)

**And** changes to `open` SHALL persist to `localStorage`

---

## MODIFIED Requirements

### Requirement: Tab Navigation (DEPRECATED)

**ID:** `NAV-001` (from existing system)  
**Status:** DEPRECATED  
**Replacement:** `LC-001` (Sidebar Navigation)

**Before:** Horizontal tab bar at top of page  
**After:** Vertical sidebar navigation  

**Migration:** The `.tabs` CSS class and tab state management SHALL be removed after sidebar is fully integrated. The `activeTab` state variable SHALL remain for routing logic but SHALL be controlled by sidebar clicks instead of tab clicks.

---

## REMOVED Requirements

None
