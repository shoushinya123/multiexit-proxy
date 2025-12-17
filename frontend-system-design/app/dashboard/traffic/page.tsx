"use client"

import { useState, useEffect, useMemo } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Skeleton } from "@/components/ui/skeleton"
import { Download, TrendingUp, TrendingDown, AlertCircle } from "lucide-react"
import { useTrafficAnalysis } from "@/hooks/use-api"
import { formatBytes } from "@/lib/utils"
import { ChartContainer, ChartTooltip, ChartTooltipContent } from "@/components/ui/chart"
import { Area, AreaChart, CartesianGrid, XAxis, YAxis, ResponsiveContainer } from "recharts"

export default function TrafficPage() {
  const [timeRange, setTimeRange] = useState("24h")

  // 获取流量分析数据（根据时间范围刷新）
  const [trafficEnabled, setTrafficEnabled] = useState(true)
  const { data: trafficData, isLoading, error } = useTrafficAnalysis(timeRange, trafficEnabled ? 30000 : undefined, trafficEnabled)
  
  // 如果返回503，禁用自动刷新
  useEffect(() => {
    if (error && error.status === 503) {
      setTrafficEnabled(false)
    }
  }, [error])

  // 处理流量趋势数据
  const trendData = useMemo(() => {
    if (!trafficData?.trends || !Array.isArray(trafficData.trends)) return []
    
    return trafficData.trends
      .filter((trend) => trend && trend.timestamp)
      .map((trend) => ({
        time: new Date(trend.timestamp).toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" }),
        upstream: Math.round((trend.bytes_up || 0) / 1024 / 1024), // 转换为MB
        downstream: Math.round((trend.bytes_down || 0) / 1024 / 1024), // 转换为MB
        connections: trend.connections || 0,
      }))
  }, [trafficData])

  // 处理域名统计数据
  const domainStatsList = useMemo(() => {
    if (!trafficData?.domain_stats || typeof trafficData.domain_stats !== "object") return []
    
    return Object.entries(trafficData.domain_stats)
      .filter(([_, stats]) => stats && typeof stats === "object")
      .map(([domain, stats]: [string, any]) => ({
        domain: domain || "unknown",
        requests: stats?.connections || 0,
        traffic: formatBytes(stats?.total_bytes || 0),
        avgLatency: stats?.avg_latency || "N/A",
        trend: "up", // 实际应该从历史数据计算趋势
      }))
      .sort((a, b) => {
        // 按流量排序
        const aStats = trafficData.domain_stats?.[a.domain]
        const bStats = trafficData.domain_stats?.[b.domain]
        const aBytes = (aStats?.total_bytes || 0)
        const bBytes = (bStats?.total_bytes || 0)
        return bBytes - aBytes
      })
      .slice(0, 20) // 只显示前20个
  }, [trafficData])

  // 处理异常检测数据
  const anomaliesList = useMemo(() => {
    if (!trafficData?.anomalies || !Array.isArray(trafficData.anomalies)) return []
    
    return trafficData.anomalies
      .filter((anomaly) => anomaly && anomaly.severity)
      .sort((a, b) => {
        const severityOrder = { high: 3, medium: 2, low: 1 }
        return (severityOrder[b.severity as keyof typeof severityOrder] || 0) - 
               (severityOrder[a.severity as keyof typeof severityOrder] || 0)
      })
      .slice(0, 10) // 只显示前10个
  }, [trafficData])

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case "high":
        return "bg-destructive text-white"
      case "medium":
        return "bg-chart-4 text-white"
      case "low":
        return "bg-chart-1 text-white"
      default:
        return "bg-muted text-muted-foreground"
    }
  }

  const getSeverityText = (severity: string) => {
    switch (severity) {
      case "high":
        return "高"
      case "medium":
        return "中"
      case "low":
        return "低"
      default:
        return severity
    }
  }

  if (isLoading && !trafficData) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-96" />
        <Skeleton className="h-96" />
      </div>
    )
  }

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertCircle className="h-4 w-4" />
        <AlertDescription>{error.message || "加载流量分析数据失败"}</AlertDescription>
      </Alert>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight">{"流量分析"}</h1>
          <p className="text-muted-foreground mt-1">{"实时流量监控和分析"}</p>
        </div>
        <div className="flex gap-2">
          <Select value={timeRange} onValueChange={setTimeRange}>
            <SelectTrigger className="w-32 bg-transparent">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="1h">{"最近1小时"}</SelectItem>
              <SelectItem value="24h">{"最近24小时"}</SelectItem>
              <SelectItem value="7d">{"最近7天"}</SelectItem>
              <SelectItem value="30d">{"最近30天"}</SelectItem>
            </SelectContent>
          </Select>
          <Button variant="outline" size="sm" className="gap-2 bg-transparent">
            <Download className="h-4 w-4" />
            {"导出数据"}
          </Button>
        </div>
      </div>

      {/* Traffic Chart */}
      <Card className="border-border/50">
        <CardHeader>
          <CardTitle>{"流量趋势"}</CardTitle>
          <CardDescription>{"上行和下行流量随时间变化"}</CardDescription>
        </CardHeader>
        <CardContent>
          {trendData.length > 0 ? (
            <ChartContainer
              config={{
                upstream: {
                  label: "上行流量",
                  color: "hsl(var(--chart-1))",
                },
                downstream: {
                  label: "下行流量",
                  color: "hsl(var(--chart-2))",
                },
              }}
              className="h-[300px]"
            >
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={trendData}>
                  <CartesianGrid strokeDasharray="3 3" className="stroke-border/50" />
                  <XAxis dataKey="time" className="text-xs" />
                  <YAxis className="text-xs" />
                  <ChartTooltip content={<ChartTooltipContent />} />
                  <Area
                    type="monotone"
                    dataKey="upstream"
                    stackId="1"
                    stroke="hsl(var(--chart-1))"
                    fill="hsl(var(--chart-1))"
                    fillOpacity={0.6}
                    name="上行流量 (MB)"
                  />
                  <Area
                    type="monotone"
                    dataKey="downstream"
                    stackId="2"
                    stroke="hsl(var(--chart-2))"
                    fill="hsl(var(--chart-2))"
                    fillOpacity={0.6}
                    name="下行流量 (MB)"
                  />
                </AreaChart>
              </ResponsiveContainer>
            </ChartContainer>
          ) : (
            <div className="text-center text-muted-foreground py-8">{"暂无流量趋势数据"}</div>
          )}
        </CardContent>
      </Card>

      {/* Domain Statistics */}
      <Card className="border-border/50">
        <CardHeader>
          <CardTitle>{"域名统计"}</CardTitle>
          <CardDescription>{"按域名统计的访问量和流量"}</CardDescription>
        </CardHeader>
        <CardContent>
          {domainStatsList.length === 0 ? (
            <div className="text-center text-muted-foreground py-8">{"暂无域名统计数据"}</div>
          ) : (
            <div className="rounded-md border border-border/50">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{"域名"}</TableHead>
                    <TableHead>{"请求数"}</TableHead>
                    <TableHead>{"总流量"}</TableHead>
                    <TableHead>{"平均延迟"}</TableHead>
                    <TableHead>{"趋势"}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {domainStatsList.map((stat) => (
                    <TableRow key={stat.domain}>
                      <TableCell className="font-mono font-medium">{stat.domain}</TableCell>
                      <TableCell>{stat.requests.toLocaleString()}</TableCell>
                      <TableCell>{stat.traffic}</TableCell>
                      <TableCell>{stat.avgLatency}</TableCell>
                      <TableCell>
                        {stat.trend === "up" ? (
                          <div className="flex items-center gap-1 text-chart-2">
                            <TrendingUp className="h-4 w-4" />
                            <span className="text-sm">{"上升"}</span>
                          </div>
                        ) : (
                          <div className="flex items-center gap-1 text-destructive">
                            <TrendingDown className="h-4 w-4" />
                            <span className="text-sm">{"下降"}</span>
                          </div>
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

      {/* Anomaly Detection */}
      <Card className="border-border/50">
        <CardHeader>
          <CardTitle>{"异常检测"}</CardTitle>
          <CardDescription>{"自动检测的异常流量模式"}</CardDescription>
        </CardHeader>
        <CardContent>
          {anomaliesList.length === 0 ? (
            <div className="text-center text-muted-foreground py-8">{"暂无异常检测数据"}</div>
          ) : (
            <div className="space-y-3">
              {anomaliesList.map((anomaly, index) => (
                <div
                  key={index}
                  className="flex items-start gap-3 p-3 rounded-lg border border-border/50 bg-card"
                >
                  <AlertCircle className="h-5 w-5 mt-0.5 text-chart-4" />
                  <div className="flex-1 space-y-1">
                    <div className="flex items-center gap-2">
                      <Badge className={getSeverityColor(anomaly.severity)}>
                        {getSeverityText(anomaly.severity)}
                      </Badge>
                      <span className="text-sm font-medium">{anomaly.description}</span>
                    </div>
                    <div className="text-xs text-muted-foreground space-y-1">
                      <div>{`域名: ${anomaly.domain}`}</div>
                      <div>{`类型: ${anomaly.anomaly_type}`}</div>
                      <div>{`检测时间: ${new Date(anomaly.detected_at).toLocaleString("zh-CN")}`}</div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
