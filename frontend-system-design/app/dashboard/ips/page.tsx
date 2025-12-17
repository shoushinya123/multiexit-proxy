"use client"

import { useState, useMemo } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Skeleton } from "@/components/ui/skeleton"
import { Plus, Trash2, RefreshCw, Activity, Clock, AlertTriangle } from "lucide-react"
import { useIPs, useConfig, useUpdateConfig, useStats } from "@/hooks/use-api"
import { useToast } from "@/hooks/use-toast"
import { Toaster } from "@/components/ui/toaster"
import { formatBytes, formatRelativeTime } from "@/lib/utils"
import type { IPInfo } from "@/lib/api/types"

export default function IPManagementPage() {
  const { toast } = useToast()
  const [newIP, setNewIP] = useState("")
  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [isDeleting, setIsDeleting] = useState<string | null>(null)

  // 获取IP列表（30秒刷新一次）
  const { data: ips, isLoading: ipsLoading, error: ipsError, refetch: refetchIPs } = useIPs(30000)
  
  // 获取配置（用于添加/删除IP）
  const { data: config, refetch: refetchConfig } = useConfig()
  
  // 更新配置的mutation
  const { mutate: updateConfig, isLoading: isUpdating } = useUpdateConfig({
    onSuccess: () => {
      toast({
        title: "操作成功",
        description: "IP列表已更新",
      })
      refetchIPs()
      refetchConfig()
      setIsDialogOpen(false)
      setNewIP("")
    },
    onError: (error) => {
      toast({
        title: "操作失败",
        description: error.message || "更新IP列表失败",
        variant: "destructive",
      })
    },
  })

  // 获取统计信息（用于显示IP的连接数、流量等）
  const { data: stats } = useStats(10000)

  // 处理添加IP
  const handleAddIP = () => {
    if (!newIP.trim()) {
      toast({
        title: "输入错误",
        description: "请输入有效的IP地址",
        variant: "destructive",
      })
      return
    }

    // 验证IP格式（简单验证）
    const ipRegex = /^(\d{1,3}\.){3}\d{1,3}$/
    if (!ipRegex.test(newIP.trim())) {
      toast({
        title: "输入错误",
        description: "请输入有效的IP地址格式",
        variant: "destructive",
      })
      return
    }

    if (!config) {
      toast({
        title: "错误",
        description: "无法获取配置信息",
        variant: "destructive",
      })
      return
    }

    // 检查IP是否已存在
    if (config.exit_ips?.includes(newIP.trim())) {
      toast({
        title: "操作失败",
        description: "该IP地址已存在",
        variant: "destructive",
      })
      return
    }

    // 更新配置，添加新IP
    const updatedConfig = {
      ...config,
      exit_ips: [...(config.exit_ips || []), newIP.trim()],
    }
    updateConfig(updatedConfig)
  }

  // 处理删除IP
  const handleDeleteIP = (ip: string) => {
    if (!config) {
      toast({
        title: "错误",
        description: "无法获取配置信息",
        variant: "destructive",
      })
      return
    }

    // 检查IP是否在配置的exit_ips中
    const exitIPs = config.exit_ips || []
    if (!exitIPs.includes(ip)) {
      toast({
        title: "无法删除",
        description: "该IP地址是自动检测的，不在配置列表中。如果启用了IP自动检测，请先禁用它。",
        variant: "destructive",
      })
      setIsDeleting(null)
      return
    }

    setIsDeleting(ip)
    const updatedConfig = {
      ...config,
      exit_ips: exitIPs.filter((existingIP) => existingIP !== ip),
    }
    updateConfig(updatedConfig)
  }

  // 合并IP信息和统计信息
  const ipListWithStats = useMemo(() => {
    if (!ips) return []
    
    const exitIPs = config?.exit_ips || []
    
    return ips.map((ipInfo: IPInfo) => {
      const ipStats = stats?.ip_stats?.[ipInfo.ip]
      const status = ipInfo.active ? "healthy" : "offline"
      const isInConfig = exitIPs.includes(ipInfo.ip)
      
      return {
        ip: ipInfo.ip,
        status,
        connections: ipStats?.connections || 0,
        activeConnections: ipStats?.active_conn || 0,
        traffic: formatBytes(ipStats?.total_bytes || 0),
        latency: ipStats?.avg_latency || "N/A",
        lastUsed: ipStats?.last_used ? formatRelativeTime(ipStats.last_used) : "从未使用",
        canDelete: isInConfig, // 只有配置中的IP才能删除
      }
    })
  }, [ips, stats, config])

  const getStatusColor = (status: string) => {
    switch (status) {
      case "healthy":
        return "bg-chart-2 text-white"
      case "warning":
        return "bg-chart-4 text-white"
      case "offline":
        return "bg-destructive text-white"
      default:
        return "bg-muted text-muted-foreground"
    }
  }

  const getStatusText = (status: string) => {
    switch (status) {
      case "healthy":
        return "健康"
      case "warning":
        return "警告"
      case "offline":
        return "离线"
      default:
        return "未知"
    }
  }

  const healthyCount = ipListWithStats.filter((ip) => ip.status === "healthy").length
  const offlineCount = ipListWithStats.filter((ip) => ip.status === "offline").length

  if (ipsLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <div className="grid gap-4 md:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-32" />
          ))}
        </div>
        <Skeleton className="h-96" />
      </div>
    )
  }

  if (ipsError) {
    return (
      <Alert variant="destructive">
        <AlertTriangle className="h-4 w-4" />
        <AlertDescription>{ipsError.message || "加载IP列表失败"}</AlertDescription>
      </Alert>
    )
  }

  return (
    <>
      <div className="space-y-6">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h1 className="text-3xl font-semibold tracking-tight">{"IP管理"}</h1>
            <p className="text-muted-foreground mt-1">{"管理和监控出口IP地址"}</p>
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              className="gap-2 bg-transparent"
              onClick={() => refetchIPs()}
              disabled={ipsLoading}
            >
              <RefreshCw className={`h-4 w-4 ${ipsLoading ? "animate-spin" : ""}`} />
              {"刷新状态"}
            </Button>
            <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
              <DialogTrigger asChild>
                <Button size="sm" className="gap-2 bg-primary hover:bg-primary/90">
                  <Plus className="h-4 w-4" />
                  {"添加IP"}
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>{"添加新IP地址"}</DialogTitle>
                  <DialogDescription>{"输入要添加到管理系统的IP地址"}</DialogDescription>
                </DialogHeader>
                <div className="space-y-4 py-4">
                  <div className="space-y-2">
                    <label htmlFor="ip" className="text-sm font-medium">
                      {"IP地址"}
                    </label>
                    <Input
                      id="ip"
                      placeholder="192.168.1.100"
                      value={newIP}
                      onChange={(e) => setNewIP(e.target.value)}
                      disabled={isUpdating}
                    />
                  </div>
                </div>
                <DialogFooter>
                  <Button variant="outline" onClick={() => setIsDialogOpen(false)} disabled={isUpdating}>
                    {"取消"}
                  </Button>
                  <Button onClick={handleAddIP} className="bg-primary hover:bg-primary/90" disabled={isUpdating}>
                    {isUpdating ? "添加中..." : "添加"}
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>
        </div>

        <div className="grid gap-4 md:grid-cols-3">
          <Card className="border-border/50">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium text-muted-foreground">{"总IP数量"}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-semibold">{ipListWithStats.length}</div>
            </CardContent>
          </Card>

          <Card className="border-border/50">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium text-muted-foreground">{"健康IP"}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-semibold text-chart-2">{healthyCount}</div>
            </CardContent>
          </Card>

          <Card className="border-border/50">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium text-muted-foreground">{"离线IP"}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-semibold text-destructive">{offlineCount}</div>
            </CardContent>
          </Card>
        </div>

        <Card className="border-border/50">
          <CardHeader>
            <CardTitle>{"IP列表"}</CardTitle>
            <CardDescription>{"查看和管理所有出口IP的状态和统计信息"}</CardDescription>
          </CardHeader>
          <CardContent>
            {ipListWithStats.length === 0 ? (
              <div className="text-center text-muted-foreground py-8">{"暂无IP地址"}</div>
            ) : (
              <div className="rounded-md border border-border/50">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{"IP地址"}</TableHead>
                      <TableHead>{"状态"}</TableHead>
                      <TableHead>{"连接数"}</TableHead>
                      <TableHead>{"流量"}</TableHead>
                      <TableHead>{"延迟"}</TableHead>
                      <TableHead>{"最后使用"}</TableHead>
                      <TableHead className="text-right">{"操作"}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {ipListWithStats.map((ip) => (
                      <TableRow key={ip.ip}>
                        <TableCell className="font-mono font-medium">{ip.ip}</TableCell>
                        <TableCell>
                          <Badge className={getStatusColor(ip.status)}>{getStatusText(ip.status)}</Badge>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            <Activity className="h-3 w-3 text-muted-foreground" />
                            {ip.connections.toLocaleString()}
                            {ip.activeConnections > 0 && (
                              <span className="text-xs text-muted-foreground">({ip.activeConnections} 活跃)</span>
                            )}
                          </div>
                        </TableCell>
                        <TableCell>{ip.traffic}</TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            <Clock className="h-3 w-3 text-muted-foreground" />
                            {ip.latency}
                          </div>
                        </TableCell>
                        <TableCell className="text-muted-foreground">{ip.lastUsed}</TableCell>
                        <TableCell className="text-right">
                          {ip.canDelete ? (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleDeleteIP(ip.ip)}
                              className="h-8 w-8 p-0 text-destructive hover:text-destructive hover:bg-destructive/10"
                              disabled={isDeleting === ip.ip || isUpdating}
                              title="删除IP地址"
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          ) : (
                            <span className="text-xs text-muted-foreground" title="该IP是自动检测的，无法删除">
                              自动检测
                            </span>
                          )}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
      <Toaster />
    </>
  )
}
