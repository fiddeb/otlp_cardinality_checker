import {
  BarChart3Icon,
  BarChart2Icon,
  ActivityIcon,
  GitBranchIcon,
  NetworkIcon,
  FileTextIcon,
  TagIcon,
  ZapIcon,
  DatabaseIcon,
  LayersIcon,
  ClipboardListIcon,
  LayoutDashboardIcon,
  RadioIcon,
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
      { id: 'metadata-complexity', label: 'Metadata Complexity', icon: LayersIcon },
      { id: 'metrics-overview', label: 'Metrics Overview', icon: BarChart2Icon },
    ],
  },
  {
    label: 'Telemetry',
    items: [
      { id: 'active-series', label: 'Active Series', icon: ActivityIcon },
      { id: 'metrics', label: 'Metrics Details', icon: BarChart3Icon },
      { id: 'traces', label: 'Traces', icon: GitBranchIcon },
      { id: 'trace-patterns', label: 'Trace Patterns', icon: NetworkIcon },
      { id: 'logs', label: 'Logs', icon: FileTextIcon },
      { id: 'attributes', label: 'Attributes', icon: TagIcon },
    ],
  },
  {
    label: 'Analysis',
    items: [
      { id: 'noisy-neighbors', label: 'Noisy Neighbors', icon: ZapIcon },
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
        <div className="flex items-center gap-3 px-3 py-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-md bg-primary text-primary-foreground">
            <RadioIcon className="h-4 w-4" />
          </div>
          <div className="flex flex-col leading-none">
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
          OTLP Cardinality Checker
        </div>
      </SidebarFooter>
    </Sidebar>
  )
}
