"use client"

import { useState, useMemo } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Switch } from "@/components/ui/switch"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Skeleton } from "@/components/ui/skeleton"
import { RefreshCw, Activity, TrendingUp, Clock, Network, AlertTriangle } from "lucide-react"
import { useStats } from "@/hooks/use-api"
import { formatBytes } from "@/lib/utils"
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@/components/ui/chart"
import { CartesianGrid, XAxis, YAxis, Line, LineChart } from "recharts"

export default function StatsPage() {
  const [autoRefresh, setAutoRefresh] = useState(true)

  // 获取统计信息（10秒刷新一次）
  const { data: stats, isLoading, error, refetch } = useStats(autoRefresh ? 10000 : undefined)

  // 生成连接数趋势数据（模拟，实际应该从后端获取）
  const connectionTrendData = useMemo(() => {
    if (!stats) return []
    // 这里应该从后端获取历史数据，暂时使用当前数据模拟
    const now = new Date()
    const data = []
    for (let i = 5; i >= 0; i--) {
      const time = new Date(now.getTime() - i * 4 * 60 * 60 * 1000) // 每4小时一个点
      data.push({
        time: time.toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" }),
        connections: Math.floor((stats.total_connections || 0) * (0.8 + Math.random() * 0.4)),
      })
    }
    return data
  }, [stats])

  // 处理IP统计列表
  const ipStatsList = useMemo(() => {
    if (!stats?.ip_stats) return []
    
    return Object.entries(stats.ip_stats)
      .map(([ip, ipStats]: [string, any]) => ({
        ip,
        connections: ipStats.connections || 0,
        activeConnections: ipStats.active_conn || 0,
        upstream: formatBytes(ipStats.bytes_up || 0),
        downstream: formatBytes(ipStats.bytes_down || 0),
        latency: ipStats.avg_latency || "N/A",
        status: ipStats.active_conn > 0 ? "healthy" : ipStats.connections > 0 ? "warning" : "offline",
      }))
      .sort((a, b) => b.connections - a.connections)
  }, [stats])

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

  // 计算平均延迟
  const avgLatency = useMemo(() => {
    if (!stats?.ip_stats) return "N/A"
    const latencies = Object.values(stats.ip_stats)
      .map((ipStats: any) => {
        const latency = ipStats.avg_latency
        if (!latency || latency === "N/A") return null
        const match = latency.match(/(\d+(?:\.\d+)?)\s*(ms|s)/)
        if (!match) return null
        const value = parseFloat(match[1])
        const unit = match[2]
        return unit === "s" ? value * 1000 : value
      })
      .filter((v): v is number => v !== null)
    
    if (latencies.length === 0) return "N/A"
    const avg = latencies.reduce((a, b) => a + b, 0) / latencies.length
    return `${avg.toFixed(0)}ms`
  }, [stats])

  if (isLoading && !stats) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {[1, 2, 3, 4].map((i) => (
            <Skeleton key={i} className="h-32" />
          ))}
        </div>
        <Skeleton className="h-96" />
      </div>
    )
  }

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertTriangle className="h-4 w-4" />
        <AlertDescription>{error.message || "加载统计数据失败"}</AlertDescription>
      </Alert>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight">{"统计监控"}</h1>
          <p className="text-muted-foreground mt-1">{"实时系统统计和性能监控"}</p>
        </div>
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <span className="text-sm text-muted-foreground">{"自动刷新"}</span>
            <Switch checked={autoRefresh} onCheckedChange={setAutoRefresh} />
          </div>
          <Button variant="outline" size="sm" className="gap-2 bg-transparent" onClick={() => refetch()}>
            <RefreshCw className={`h-4 w-4 ${isLoading ? "animate-spin" : ""}`} />
            {"刷新"}
          </Button>
        </div>
      </div>

      {/* Real-time Stats */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card className="border-border/50">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">{"总连接数"}</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold">{(stats?.total_connections || 0).toLocaleString()}</div>
            <div className="flex items-center gap-1 mt-1">
              <Badge variant="secondary" className="text-xs">
                {"累计"}
              </Badge>
            </div>
          </CardContent>
        </Card>

        <Card className="border-border/50">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">{"活跃连接"}</CardTitle>
            <Network className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold">{(stats?.active_connections || 0).toLocaleString()}</div>
            <div className="flex items-center gap-1 mt-1">
              <Badge variant="secondary" className="text-xs">
                {"实时"}
              </Badge>
            </div>
          </CardContent>
        </Card>

        <Card className="border-border/50">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">{"总流量"}</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold">{formatBytes(stats?.bytes_transferred || 0)}</div>
            <p className="text-xs text-muted-foreground mt-1">
              {`↑${formatBytes(stats?.bytes_up || 0)} / ↓${formatBytes(stats?.bytes_down || 0)}`}
            </p>
          </CardContent>
        </Card>

        <Card className="border-border/50">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">{"平均延迟"}</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold">{avgLatency}</div>
            <div className="flex items-center gap-2 mt-1">
              <Badge variant="secondary" className="text-xs">
                {"良好"}
              </Badge>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Connection Trend */}
      <Card className="border-border/50">
        <CardHeader>
          <CardTitle>{"连接数趋势"}</CardTitle>
          <CardDescription>{"实时连接数随时间变化"}</CardDescription>
        </CardHeader>
        <CardContent>
          {connectionTrendData.length > 0 ? (
            <ChartContainer
              config={{
                connections: {
                  label: "连接数",
                  color: "hsl(var(--chart-1))",
                },
              }}
              className="h-[250px]"
            >
              <LineChart data={connectionTrendData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-border/50" />
                <XAxis dataKey="time" className="text-xs" />
                <YAxis className="text-xs" />
                <ChartTooltip content={<ChartTooltipContent />} />
                <Line
                  type="monotone"
                  dataKey="connections"
                  stroke="hsl(var(--chart-1))"
                  strokeWidth={2}
                  dot={{ fill: "hsl(var(--chart-1))", r: 4 }}
                />
              </LineChart>
            </ChartContainer>
          ) : (
            <div className="text-center text-muted-foreground py-8">{"暂无趋势数据"}</div>
          )}
        </CardContent>
      </Card>

      {/* IP Statistics */}
      <Card className="border-border/50">
        <CardHeader>
          <CardTitle>{"按IP统计"}</CardTitle>
          <CardDescription>{"各出口IP的详细统计信息"}</CardDescription>
        </CardHeader>
        <CardContent>
          {ipStatsList.length === 0 ? (
            <div className="text-center text-muted-foreground py-8">{"暂无IP统计数据"}</div>
          ) : (
            <div className="rounded-md border border-border/50">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{"IP地址"}</TableHead>
                    <TableHead>{"连接数"}</TableHead>
                    <TableHead>{"上行流量"}</TableHead>
                    <TableHead>{"下行流量"}</TableHead>
                    <TableHead>{"延迟"}</TableHead>
                    <TableHead>{"状态"}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {ipStatsList.map((stat) => (
                    <TableRow key={stat.ip}>
                      <TableCell className="font-mono font-medium">{stat.ip}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <Activity className="h-3 w-3 text-muted-foreground" />
                          {stat.connections.toLocaleString()}
                          {stat.activeConnections > 0 && (
                            <span className="text-xs text-muted-foreground">({stat.activeConnections} 活跃)</span>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>{stat.upstream}</TableCell>
                      <TableCell>{stat.downstream}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <Clock className="h-3 w-3 text-muted-foreground" />
                          {stat.latency}
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge className={getStatusColor(stat.status)}>{getStatusText(stat.status)}</Badge>
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
  )
}
