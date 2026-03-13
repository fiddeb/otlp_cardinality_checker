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
} from 'lucide-react'
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar'

const NAV_ITEMS = [
  { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboardIcon },
  { id: 'metadata-complexity', label: 'Metadata Complexity', icon: LayersIcon },
  { id: 'metrics-overview', label: 'Metrics Overview', icon: BarChart2Icon },
  { id: 'active-series', label: 'Active Series', icon: ActivityIcon },
  { id: 'metrics', label: 'Metrics Details', icon: BarChart3Icon },
  { id: 'traces', label: 'Traces', icon: GitBranchIcon },
  { id: 'trace-patterns', label: 'Trace Patterns', icon: NetworkIcon },
  { id: 'logs', label: 'Logs', icon: FileTextIcon },
  { id: 'attributes', label: 'Attributes', icon: TagIcon },
  { id: 'noisy-neighbors', label: 'Noisy Neighbors', icon: ZapIcon },
  { id: 'memory', label: 'Memory', icon: DatabaseIcon },
  { id: 'sessions', label: 'Sessions', icon: ClipboardListIcon },
]

export function AppSidebar({ activeTab, onNavigate }) {
  return (
    <Sidebar>
      <SidebarHeader>
        <div className="flex flex-col gap-1 px-2 py-1">
          <span className="text-sm font-semibold text-sidebar-foreground">OCC</span>
          <span className="text-xs text-muted-foreground">Cardinality Checker</span>
        </div>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarMenu>
            {NAV_ITEMS.map(({ id, label, icon: Icon }) => (
              <SidebarMenuItem key={id}>
                <SidebarMenuButton
                  isActive={activeTab === id}
                  onClick={() => onNavigate(id)}
                >
                  <Icon />
                  <span>{label}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            ))}
          </SidebarMenu>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  )
}
