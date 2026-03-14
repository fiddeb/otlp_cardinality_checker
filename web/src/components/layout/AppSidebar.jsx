import {
  BarChart3Icon,
  BarChart2Icon,
  ChartNoAxesCombinedIcon,
  GitBranchIcon,
  GitGraphIcon,
  FileTextIcon,
  SearchAlertIcon,
  CircleAlertIcon,
  DatabaseIcon,
  LayersIcon,
  ClipboardListIcon,
  LayoutDashboardIcon,
  SearchCodeIcon,
} from 'lucide-react'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from '@/components/ui/sidebar'

const NAV_GROUPS = [
  {
    label: 'Overview',
    items: [
      { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboardIcon },
    ],
  },
  {
    label: 'Telemetry',
    items: [
      { id: 'metrics-overview', label: 'Metrics Overview', icon: BarChart2Icon },
      { id: 'metrics', label: 'Metrics Details', icon: BarChart3Icon },
      { id: 'active-series', label: 'Active Series', icon: ChartNoAxesCombinedIcon },
      { id: 'traces', label: 'Traces', icon: GitBranchIcon },
      { id: 'trace-patterns', label: 'Trace Patterns', icon: GitGraphIcon },
      { id: 'logs', label: 'Logs', icon: FileTextIcon },
      { id: 'attributes', label: 'Attributes', icon: () => <SearchAlertIcon className="scale-x-[-1]" /> },
    ],
  },
  {
    label: 'Analysis',
    items: [
      { id: 'noisy-neighbors', label: 'Noisy Neighbors', icon: CircleAlertIcon },
      { id: 'metadata-complexity', label: 'Metadata Complexity', icon: LayersIcon },
    ],
  },
  {
    label: 'System',
    items: [
      { id: 'memory', label: 'Memory', icon: DatabaseIcon },
      { id: 'sessions', label: 'Sessions', icon: ClipboardListIcon },
    ],
  },
]

export function AppSidebar({ activeTab, onNavigate }) {
  return (
    <Sidebar collapsible="icon">
      <SidebarHeader>
        <div className="flex items-center gap-3 px-3 py-3 group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-0">
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-primary text-primary-foreground">
            <SearchCodeIcon className="h-4 w-4 scale-x-[-1]" />
          </div>
          <div className="flex flex-col leading-none group-data-[collapsible=icon]:hidden">
            <span className="font-semibold text-sm text-sidebar-foreground">OCC</span>
            <span className="text-[10px] text-muted-foreground">Cardinality Checker</span>
          </div>
        </div>
      </SidebarHeader>
      <SidebarContent>
        {NAV_GROUPS.map(({ label, items }) => (
          <SidebarGroup key={label}>
            <SidebarGroupLabel>{label}</SidebarGroupLabel>
            <SidebarMenu>
              {items.map(({ id, label: itemLabel, icon: Icon }) => (
                <SidebarMenuItem key={id}>
                  <SidebarMenuButton
                    isActive={activeTab === id}
                    tooltip={itemLabel}
                    onClick={() => onNavigate(id)}
                  >
                    <Icon />
                    <span>{itemLabel}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroup>
        ))}
      </SidebarContent>
      <SidebarRail />
      <SidebarFooter>
        <div className="px-3 py-3 text-xs text-muted-foreground group-data-[collapsible=icon]:hidden">
          <div>OTLP Cardinality Checker</div>
          <div className="mt-1 text-muted-foreground/60">© Fredrik Berggren 2025</div>
        </div>
      </SidebarFooter>
    </Sidebar>
  )
}
