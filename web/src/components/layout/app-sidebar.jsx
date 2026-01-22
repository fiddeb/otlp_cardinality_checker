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
  Database,
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
    { id: 'attributes', label: 'Attributes', icon: Database },
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
