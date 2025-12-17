"use client"
import { LayoutDashboard, Network, Settings, Activity, Filter, BarChart3, Link2, History, Shield } from "lucide-react"
import { usePathname } from "next/navigation"
import Link from "next/link"

import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarHeader,
  SidebarFooter,
} from "@/components/ui/sidebar"

const navItems = [
  {
    title: "仪表板",
    url: "/dashboard",
    icon: LayoutDashboard,
  },
  {
    title: "IP管理",
    url: "/dashboard/ips",
    icon: Network,
  },
  {
    title: "配置管理",
    url: "/dashboard/config",
    icon: Settings,
  },
  {
    title: "规则引擎",
    url: "/dashboard/rules",
    icon: Filter,
  },
  {
    title: "流量分析",
    url: "/dashboard/traffic",
    icon: Activity,
  },
  {
    title: "统计监控",
    url: "/dashboard/stats",
    icon: BarChart3,
  },
  {
    title: "订阅管理",
    url: "/dashboard/subscription",
    icon: Link2,
  },
  {
    title: "版本回滚",
    url: "/dashboard/versions",
    icon: History,
  },
]

export function AppSidebar() {
  const pathname = usePathname()

  return (
    <Sidebar>
      <SidebarHeader className="border-b border-sidebar-border px-4 py-3">
        <div className="flex items-center gap-2">
          <div className="p-1.5 rounded-lg bg-primary/10 border border-primary/20">
            <Shield className="h-5 w-5 text-primary" />
          </div>
          <div className="flex flex-col">
            <span className="text-sm font-semibold">Traffic Manager</span>
            <span className="text-xs text-muted-foreground">v1.0.0</span>
          </div>
        </div>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>{"导航"}</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {navItems.map((item) => (
                <SidebarMenuItem key={item.url}>
                  <SidebarMenuButton asChild isActive={pathname === item.url}>
                    <Link href={item.url}>
                      <item.icon className="h-4 w-4" />
                      <span>{item.title}</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter className="border-t border-sidebar-border p-4">
        <div className="text-xs text-muted-foreground">
          <div>{"系统运行正常"}</div>
          <div className="mt-1 flex items-center gap-1">
            <div className="h-1.5 w-1.5 rounded-full bg-chart-2 animate-pulse" />
            <span>{"在线"}</span>
          </div>
        </div>
      </SidebarFooter>
    </Sidebar>
  )
}
