"use client"

import { useState } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Copy, RefreshCw, QrCode, Eye, EyeOff, AlertTriangle } from "lucide-react"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Skeleton } from "@/components/ui/skeleton"
import { useSubscriptionLink } from "@/hooks/use-api"
import { useToast } from "@/hooks/use-toast"
import { Toaster } from "@/components/ui/toaster"

export default function SubscriptionPage() {
  const { toast } = useToast()
  const [showToken, setShowToken] = useState(false)
  const [showQR, setShowQR] = useState(false)

  // 获取订阅链接
  const { data: subscriptionData, isLoading, error, refetch } = useSubscriptionLink()

  const handleCopyLink = () => {
    if (subscriptionData?.link) {
      navigator.clipboard.writeText(subscriptionData.link)
      toast({
        title: "已复制",
        description: "订阅链接已复制到剪贴板",
      })
    }
  }

  const handleCopyToken = () => {
    if (subscriptionData?.token) {
      navigator.clipboard.writeText(subscriptionData.token)
      toast({
        title: "已复制",
        description: "Token已复制到剪贴板",
      })
    }
  }

  const handleRefresh = () => {
    refetch()
    toast({
      title: "已刷新",
      description: "订阅链接已更新",
    })
  }

  if (isLoading && !subscriptionData) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <div className="grid gap-6 lg:grid-cols-2">
          <Skeleton className="h-64" />
          <Skeleton className="h-64" />
        </div>
        <Skeleton className="h-48" />
      </div>
    )
  }

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertTriangle className="h-4 w-4" />
        <AlertDescription>{error.message || "加载订阅信息失败"}</AlertDescription>
      </Alert>
    )
  }

  const configPreview = subscriptionData
    ? `{
  "version": "1",
  "servers": [
    {
      "type": "trojan",
      "address": "${subscriptionData.link.split("/")[2]?.split(":")[0] || "proxy.example.com"}",
      "port": 443,
      "password": "${subscriptionData.token}",
      "sni": "proxy.example.com"
    }
  ],
  "routing": {
    "domainStrategy": "AsIs"
  }
}`
    : ""

  return (
    <>
      <div className="space-y-6">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h1 className="text-3xl font-semibold tracking-tight">{"订阅管理"}</h1>
            <p className="text-muted-foreground mt-1">{"管理订阅链接和访问令牌"}</p>
          </div>
          <Button variant="outline" size="sm" className="gap-2 bg-transparent" onClick={handleRefresh} disabled={isLoading}>
            <RefreshCw className={`h-4 w-4 ${isLoading ? "animate-spin" : ""}`} />
            {"刷新"}
          </Button>
        </div>

        <div className="grid gap-6 lg:grid-cols-2">
          {/* Subscription Link */}
          <Card className="border-border/50">
            <CardHeader>
              <CardTitle>{"订阅链接"}</CardTitle>
              <CardDescription>{"用于客户端自动更新配置"}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="subscription-link">{"订阅URL"}</Label>
                <div className="flex gap-2">
                  <Input
                    id="subscription-link"
                    value={subscriptionData?.link || ""}
                    readOnly
                    className="font-mono text-sm"
                  />
                  <Button variant="outline" size="icon" onClick={handleCopyLink} className="shrink-0 bg-transparent">
                    <Copy className="h-4 w-4" />
                  </Button>
                </div>
              </div>

              <div className="flex gap-2">
                <Button
                  variant="outline"
                  onClick={() => setShowQR(!showQR)}
                  className="flex-1 gap-2 bg-transparent"
                  size="sm"
                >
                  <QrCode className="h-4 w-4" />
                  {showQR ? "隐藏二维码" : "显示二维码"}
                </Button>
              </div>

              {showQR && subscriptionData?.qr_code && (
                <div className="flex justify-center p-4 border border-border/50 rounded-lg bg-white">
                  <div className="h-48 w-48 flex items-center justify-center bg-muted/30 rounded">
                    <img src={subscriptionData.qr_code} alt="QR Code" className="h-full w-full object-contain" />
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Token Management */}
          <Card className="border-border/50">
            <CardHeader>
              <CardTitle>{"访问令牌"}</CardTitle>
              <CardDescription>{"用于订阅验证的访问令牌"}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="token">{"当前Token"}</Label>
                <div className="flex gap-2">
                  <Input
                    id="token"
                    type={showToken ? "text" : "password"}
                    value={subscriptionData?.token || ""}
                    readOnly
                    className="font-mono text-sm"
                  />
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={() => setShowToken(!showToken)}
                    className="shrink-0 bg-transparent"
                  >
                    {showToken ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </Button>
                  <Button variant="outline" size="icon" onClick={handleCopyToken} className="shrink-0 bg-transparent">
                    <Copy className="h-4 w-4" />
                  </Button>
                </div>
              </div>

              <div className="p-3 rounded-lg bg-muted/30 border border-border/50">
                <div className="flex items-start justify-between gap-2 mb-2">
                  <span className="text-sm font-medium">{"Token状态"}</span>
                  <Badge className="bg-chart-2 text-white">{"有效"}</Badge>
                </div>
                <div className="text-xs text-muted-foreground space-y-1">
                  <div>{"创建时间: " + new Date().toLocaleString("zh-CN")}</div>
                  <div>{"有效期: 永久"}</div>
                  <div>{"使用次数: 不限"}</div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Configuration Preview */}
        <Card className="border-border/50">
          <CardHeader>
            <CardTitle>{"配置预览"}</CardTitle>
            <CardDescription>{"客户端将接收到的配置示例"}</CardDescription>
          </CardHeader>
          <CardContent>
            {configPreview ? (
              <div className="rounded-lg bg-muted/30 border border-border/50 p-4">
                <pre className="text-xs font-mono overflow-x-auto text-foreground">{configPreview}</pre>
              </div>
            ) : (
              <div className="text-center text-muted-foreground py-8">{"暂无配置预览"}</div>
            )}
          </CardContent>
        </Card>
      </div>
      <Toaster />
    </>
  )
}
