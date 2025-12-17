"use client"

import { useState, useEffect } from "react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Activity, Network, TrendingUp, TrendingDown, Clock, AlertTriangle } from "lucide-react"
import { useStatus, useStats, useIPs, useTrafficAnalysis } from "@/hooks/use-api"
import { Skeleton } from "@/components/ui/skeleton"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { formatBytes } from "@/lib/utils"

export default function DashboardPage() {
  // 获取实时数据
  const { data: status, isLoading: statusLoading, error: statusError } = useStatus(5000)
  const { data: stats, isLoading: statsLoading, error: statsError } = useStats(5000)
  const { data: ips, isLoading: ipsLoading } = useIPs(30000) // IP状态30秒刷新一次
  // 流量分析可能未启用，禁用自动刷新避免503错误
  // 使用enabled选项，如果API返回503则禁用后续请求
  const [trafficEnabled, setTrafficEnabled] = useState(true)
  const { data: trafficData, error: trafficError } = useTrafficAnalysis(undefined, undefined, !trafficEnabled)
  
  // 如果流量分析返回503，禁用后续请求
  useEffect(() => {
    if (trafficError && trafficError.status === 503) {
      setTrafficEnabled(false)
    }
  }, [trafficError])

  // 计算统计数据
  const healthyIPs = ips?.filter((ip) => ip.active).length || 0
  const totalIPs = ips?.length || 0

  // 获取流量分析数据
  const topDomains = trafficData?.domain_stats
    ? Object.values(trafficData.domain_stats)
        .filter((stat) => stat && stat.total_bytes !== undefined)
        .sort((a, b) => (b.total_bytes || 0) - (a.total_bytes || 0))
        .slice(0, 4)
        .map((stat) => ({
          domain: stat.domain || "unknown",
          requests: stat.connections || 0,
          traffic: formatBytes(stat.total_bytes || 0),
        }))
    : []

  // 获取异常告警
  const recentAlerts = trafficData?.anomalies && Array.isArray(trafficData.anomalies)
    ? trafficData.anomalies
        .filter((anomaly) => anomaly && anomaly.detected_at)
        .slice(0, 3)
        .map((anomaly, index) => {
          let timeStr = "未知时间"
          try {
            if (anomaly.detected_at) {
              const date = new Date(anomaly.detected_at)
              if (!isNaN(date.getTime())) {
                timeStr = date.toLocaleString("zh-CN")
              }
            }
          } catch (e) {
            console.warn("Failed to parse date:", anomaly.detected_at, e)
          }
          return {
            id: index + 1,
            type: anomaly.severity === "high" ? "warning" : "info",
            message: anomaly.description || "未知异常",
            time: timeStr,
          }
        })
    : []

  if (statusLoading || statsLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {[1, 2, 3, 4].map((i) => (
            <Skeleton key={i} className="h-32" />
          ))}
        </div>
      </div>
    )
  }

  if (statusError || statsError) {
    return (
      <Alert variant="destructive">
        <AlertTriangle className="h-4 w-4" />
        <AlertDescription>
          {statusError?.message || statsError?.message || "加载数据失败"}
        </AlertDescription>
      </Alert>
    )
  }

  const displayStats = {
    totalConnections: stats?.total_connections || 0,
    activeConnections: stats?.active_connections || status?.connections || 0,
    upstreamTraffic: stats ? formatBytes(stats.bytes_up) : "0 B",
    downstreamTraffic: stats ? formatBytes(stats.bytes_down) : "0 B",
    averageLatency: "N/A", // 需要从IP统计中计算
    healthyIPs,
    totalIPs,
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-semibold tracking-tight">{"仪表板"}</h1>
        <p className="text-muted-foreground mt-1">{"系统运行状态概览"}</p>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card className="border-border/50">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">{"总连接数"}</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold">{displayStats.totalConnections.toLocaleString()}</div>
          </CardContent>
        </Card>

        <Card className="border-border/50">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">{"活跃连接"}</CardTitle>
            <Network className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold">{displayStats.activeConnections.toLocaleString()}</div>
          </CardContent>
        </Card>

        <Card className="border-border/50">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">{"总流量"}</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold">{`↑${displayStats.upstreamTraffic} / ↓${displayStats.downstreamTraffic}`}</div>
            <p className="text-xs text-muted-foreground mt-1">{"上行 / 下行"}</p>
          </CardContent>
        </Card>

        <Card className="border-border/50">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">{"IP状态"}</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold">{`${displayStats.healthyIPs}/${displayStats.totalIPs}`}</div>
            <div className="flex items-center gap-2 mt-1">
              <Badge variant="secondary" className="text-xs">
                {displayStats.healthyIPs === displayStats.totalIPs ? "全部健康" : "部分异常"}
              </Badge>
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        {/* IP Health Status */}
        <Card className="border-border/50">
          <CardHeader>
            <CardTitle className="text-base">{"IP健康状态"}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">{"健康IP"}</span>
                <div className="flex items-center gap-2">
                  <div className="h-2 w-32 bg-secondary rounded-full overflow-hidden">
                    <div
                      className="h-full bg-chart-2"
                      style={{
                        width: `${displayStats.totalIPs > 0 ? (displayStats.healthyIPs / displayStats.totalIPs) * 100 : 0}%`,
                      }}
                    />
                  </div>
                  <span className="text-sm font-medium">
                    {displayStats.healthyIPs} / {displayStats.totalIPs}
                  </span>
                </div>
              </div>

              <div className="pt-4 border-t border-border/50">
                <div className="flex items-center justify-between text-sm mb-2">
                  <span className="text-muted-foreground">{"状态分布"}</span>
                </div>
                <div className="flex gap-4">
                  <div className="flex items-center gap-2">
                    <div className="h-2 w-2 rounded-full bg-chart-2" />
                    <span className="text-xs text-muted-foreground">{`健康 (${displayStats.healthyIPs})`}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="h-2 w-2 rounded-full bg-chart-4" />
                    <span className="text-xs text-muted-foreground">
                      {`警告 (${displayStats.totalIPs - displayStats.healthyIPs})`}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Recent Alerts */}
        <Card className="border-border/50">
          <CardHeader>
            <CardTitle className="text-base">{"最近告警"}</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {recentAlerts.map((alert) => (
                <div
                  key={alert.id}
                  className="flex items-start gap-3 pb-3 border-b border-border/50 last:border-0 last:pb-0"
                >
                  <div className="mt-0.5">
                    <AlertTriangle
                      className={`h-4 w-4 ${alert.type === "warning" ? "text-chart-4" : "text-chart-1"}`}
                    />
                  </div>
                  <div className="flex-1 space-y-1">
                    <p className="text-sm leading-relaxed">{alert.message}</p>
                    <p className="text-xs text-muted-foreground">{alert.time}</p>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Top Domains */}
      <Card className="border-border/50">
        <CardHeader>
          <CardTitle className="text-base">{"热门域名"}</CardTitle>
        </CardHeader>
        <CardContent>
          {topDomains.length > 0 ? (
            <div className="space-y-3">
              {topDomains.map((item, index) => (
                <div
                  key={item.domain}
                  className="flex items-center justify-between pb-3 border-b border-border/50 last:border-0 last:pb-0"
                >
                  <div className="flex items-center gap-3">
                    <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 border border-primary/20">
                      <span className="text-xs font-semibold text-primary">{index + 1}</span>
                    </div>
                    <div>
                      <div className="text-sm font-medium font-mono">{item.domain}</div>
                      <div className="text-xs text-muted-foreground">{item.requests.toLocaleString()} 请求</div>
                    </div>
                  </div>
                  <div className="text-right">
                    <div className="text-sm font-medium">{item.traffic}</div>
                    <div className="text-xs text-muted-foreground">{"流量"}</div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center text-muted-foreground py-8">{"暂无流量数据"}</div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
