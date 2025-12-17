"use client"

import type React from "react"

import { useEffect } from "react"
import { useRouter } from "next/navigation"
import { SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar"
import { AppSidebar } from "@/components/app-sidebar"
import { Button } from "@/components/ui/button"
import { LogOut } from "lucide-react"
import { clearAuthFromStorage, isAuthenticated } from "@/lib/auth"

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const router = useRouter()

  useEffect(() => {
    // 只在客户端检查认证状态，避免服务端渲染时的问题
    if (typeof window === "undefined") return
    
    // 使用setTimeout避免在渲染过程中立即重定向
    const checkAuth = setTimeout(() => {
      if (!isAuthenticated()) {
        router.push("/")
      }
    }, 0)
    
    return () => clearTimeout(checkAuth)
  }, [router])

  const handleLogout = () => {
    clearAuthFromStorage()
    router.push("/")
  }

  return (
    <SidebarProvider>
      <div className="flex min-h-screen w-full">
        <AppSidebar />
        <div className="flex-1 flex flex-col">
          <header className="sticky top-0 z-10 flex h-14 items-center gap-4 border-b border-border bg-card/50 backdrop-blur-sm px-4 lg:px-6">
            <SidebarTrigger />
            <div className="flex-1" />
            <Button variant="outline" size="sm" onClick={handleLogout} className="gap-2 bg-transparent">
              <LogOut className="h-4 w-4" />
              <span className="hidden sm:inline">{"退出登录"}</span>
            </Button>
          </header>
          <main className="flex-1 p-4 lg:p-6">{children}</main>
        </div>
      </div>
    </SidebarProvider>
  )
}
