"use client"

import { useState, useEffect } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { Switch } from "@/components/ui/switch"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Skeleton } from "@/components/ui/skeleton"
import { Save, RotateCcw, AlertTriangle, Plus, Trash2 } from "lucide-react"
import { useConfig, useUpdateConfig } from "@/hooks/use-api"
import { useToast } from "@/hooks/use-toast"
import { Toaster } from "@/components/ui/toaster"
import type { ServerConfig } from "@/lib/api/types"

export default function ConfigPage() {
  const { toast } = useToast()
  const [config, setConfig] = useState<ServerConfig | null>(null)
  const [originalConfig, setOriginalConfig] = useState<ServerConfig | null>(null)

  // 获取配置
  const { data: configData, isLoading, error, refetch } = useConfig()

  // 更新配置的mutation
  const { mutate: updateConfig, isLoading: isUpdating } = useUpdateConfig({
    onSuccess: () => {
      toast({
        title: "配置已保存",
        description: "系统配置已成功更新",
      })
      refetch()
    },
    onError: (error) => {
      toast({
        title: "保存失败",
        description: error.message || "更新配置失败",
        variant: "destructive",
      })
    },
  })

  // 初始化配置对象，确保所有必需的嵌套对象都存在
  const initializeConfig = (data: ServerConfig): ServerConfig => {
    return {
      ...data,
      server: {
        listen: data.server?.listen || "",
        tls: {
          cert: data.server?.tls?.cert || "",
          key: data.server?.tls?.key || "",
          sni_fake: data.server?.tls?.sni_fake || false,
          fake_snis: data.server?.tls?.fake_snis || [],
        },
        ...data.server,
      },
      auth: {
        method: data.auth?.method || "",
        key: data.auth?.key || "",
        ...data.auth,
      },
      strategy: {
        type: data.strategy?.type || "",
        port_ranges: data.strategy?.port_ranges || [],
        ...data.strategy,
      },
      snat: {
        enabled: data.snat?.enabled || false,
        gateway: data.snat?.gateway || "",
        interface: data.snat?.interface || "",
        ...data.snat,
      },
      logging: {
        level: data.logging?.level || "",
        file: data.logging?.file || "",
        ...data.logging,
      },
      exit_ips: data.exit_ips || [],
      web: data.web || {
        enabled: false,
        listen: "",
        username: "",
        password: "",
      },
      health_check: data.health_check,
      ip_detection: data.ip_detection,
      connection: data.connection,
      monitor: data.monitor,
      geo_location: data.geo_location,
      rules: data.rules,
      traffic_analysis: data.traffic_analysis,
    }
  }

  // 当配置数据加载时，更新本地状态
  useEffect(() => {
    if (configData) {
      const initializedConfig = initializeConfig(configData)
      setConfig(initializedConfig)
      setOriginalConfig(JSON.parse(JSON.stringify(initializedConfig))) // 深拷贝
    }
  }, [configData])

  const handleSave = () => {
    if (!config) return
    updateConfig(config)
  }

  const handleReset = () => {
    if (originalConfig) {
      setConfig(JSON.parse(JSON.stringify(originalConfig)))
      toast({
        title: "配置已重置",
        description: "配置已恢复到原始值",
      })
    }
  }

  const updateConfigField = (path: string[], value: any) => {
    if (!config) return
    const newConfig = JSON.parse(JSON.stringify(config))
    let current: any = newConfig
    for (let i = 0; i < path.length - 1; i++) {
      if (!current[path[i]]) {
        current[path[i]] = {}
      }
      current = current[path[i]]
    }
    current[path[path.length - 1]] = value
    setConfig(newConfig)
  }

  const addFakeSNI = () => {
    if (!config) return
    const newConfig = JSON.parse(JSON.stringify(config))
    if (!newConfig.server) {
      newConfig.server = { listen: "", tls: { cert: "", key: "", sni_fake: false, fake_snis: [] } }
    }
    if (!newConfig.server.tls) {
      newConfig.server.tls = { cert: "", key: "", sni_fake: false, fake_snis: [] }
    }
    if (!newConfig.server.tls.fake_snis) {
      newConfig.server.tls.fake_snis = []
    }
    newConfig.server.tls.fake_snis.push("")
    setConfig(newConfig)
  }

  const removeFakeSNI = (index: number) => {
    if (!config) return
    const newConfig = JSON.parse(JSON.stringify(config))
    if (newConfig.server?.tls?.fake_snis) {
      newConfig.server.tls.fake_snis.splice(index, 1)
      setConfig(newConfig)
    }
  }

  const addPortRange = () => {
    if (!config) return
    const newConfig = JSON.parse(JSON.stringify(config))
    if (!newConfig.strategy) {
      newConfig.strategy = { type: "", port_ranges: [] }
    }
    if (!newConfig.strategy.port_ranges) {
      newConfig.strategy.port_ranges = []
    }
    newConfig.strategy.port_ranges.push({ range: "", ip: "" })
    setConfig(newConfig)
  }

  const removePortRange = (index: number) => {
    if (!config) return
    const newConfig = JSON.parse(JSON.stringify(config))
    if (newConfig.strategy?.port_ranges) {
      newConfig.strategy.port_ranges.splice(index, 1)
      setConfig(newConfig)
    }
  }

  if (isLoading && !config) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-96" />
      </div>
    )
  }

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertTriangle className="h-4 w-4" />
        <AlertDescription>{error.message || "加载配置失败"}</AlertDescription>
      </Alert>
    )
  }

  if (!config) {
    return (
      <Alert>
        <AlertTriangle className="h-4 w-4" />
        <AlertDescription>{"无法加载配置"}</AlertDescription>
      </Alert>
    )
  }

  return (
    <>
      <div className="space-y-6">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h1 className="text-3xl font-semibold tracking-tight">{"配置管理"}</h1>
            <p className="text-muted-foreground mt-1">{"管理系统配置和参数"}</p>
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={handleReset}
              className="gap-2 bg-transparent"
              disabled={isUpdating}
            >
              <RotateCcw className="h-4 w-4" />
              {"重置"}
            </Button>
            <Button size="sm" onClick={handleSave} className="gap-2 bg-primary hover:bg-primary/90" disabled={isUpdating}>
              <Save className="h-4 w-4" />
              {isUpdating ? "保存中..." : "保存配置"}
            </Button>
          </div>
        </div>

        <Tabs defaultValue="server" className="space-y-4">
          <TabsList className="bg-muted/50">
            <TabsTrigger value="server">{"服务器配置"}</TabsTrigger>
            <TabsTrigger value="auth">{"认证配置"}</TabsTrigger>
            <TabsTrigger value="strategy">{"策略配置"}</TabsTrigger>
            <TabsTrigger value="network">{"网络配置"}</TabsTrigger>
            <TabsTrigger value="logging">{"日志配置"}</TabsTrigger>
            <TabsTrigger value="advanced">{"高级配置"}</TabsTrigger>
          </TabsList>

          {/* 服务器配置 */}
          <TabsContent value="server" className="space-y-4">
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>{"服务器设置"}</CardTitle>
                <CardDescription>{"配置服务器监听地址和TLS证书"}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="serverListen">{"监听地址"}</Label>
                  <Input
                    id="serverListen"
                    value={config.server?.listen || ""}
                    onChange={(e) => updateConfigField(["server", "listen"], e.target.value)}
                    placeholder=":443"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="tlsCert">{"TLS证书路径"}</Label>
                  <Input
                    id="tlsCert"
                    value={config.server?.tls?.cert || ""}
                    onChange={(e) => updateConfigField(["server", "tls", "cert"], e.target.value)}
                    placeholder="/path/to/cert.pem"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="tlsKey">{"TLS密钥路径"}</Label>
                  <Input
                    id="tlsKey"
                    value={config.server?.tls?.key || ""}
                    onChange={(e) => updateConfigField(["server", "tls", "key"], e.target.value)}
                    placeholder="/path/to/key.pem"
                  />
                </div>
                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label>{"启用SNI伪装"}</Label>
                    <p className="text-sm text-muted-foreground">{"启用TLS SNI伪装"}</p>
                  </div>
                  <Switch
                    checked={config.server?.tls?.sni_fake || false}
                    onCheckedChange={(checked) => updateConfigField(["server", "tls", "sni_fake"], checked)}
                  />
                </div>
                {config.server?.tls?.sni_fake && (
                  <div className="space-y-2">
                    <div className="flex items-center justify-between">
                      <Label>{"伪装SNI列表"}</Label>
                      <Button variant="outline" size="sm" onClick={addFakeSNI} className="gap-2">
                        <Plus className="h-4 w-4" />
                        {"添加"}
                      </Button>
                    </div>
                    {(config.server?.tls?.fake_snis || []).map((sni, index) => (
                      <div key={index} className="flex gap-2">
                        <Input
                          value={sni}
                          onChange={(e) => {
                            const newSNIs = [...(config.server?.tls?.fake_snis || [])]
                            newSNIs[index] = e.target.value
                            updateConfigField(["server", "tls", "fake_snis"], newSNIs)
                          }}
                          placeholder="cloudflare.com"
                        />
                        <Button variant="outline" size="icon" onClick={() => removeFakeSNI(index)}>
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          {/* 认证配置 */}
          <TabsContent value="auth" className="space-y-4">
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>{"认证设置"}</CardTitle>
                <CardDescription>{"配置认证方法和密钥"}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="authMethod">{"认证方法"}</Label>
                  <Input
                    id="authMethod"
                    value={config.auth?.method || ""}
                    onChange={(e) => updateConfigField(["auth", "method"], e.target.value)}
                    placeholder="psk"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="authKey">{"认证密钥"}</Label>
                  <Textarea
                    id="authKey"
                    value={config.auth?.key || ""}
                    onChange={(e) => updateConfigField(["auth", "key"], e.target.value)}
                    placeholder="your-secret-key-here"
                    rows={3}
                    className="font-mono text-sm"
                  />
                  <p className="text-xs text-muted-foreground">{"用于客户端认证的密钥，请妥善保管"}</p>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          {/* 策略配置 */}
          <TabsContent value="strategy" className="space-y-4">
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>{"IP选择策略"}</CardTitle>
                <CardDescription>{"配置出口IP选择策略"}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="strategyType">{"策略类型"}</Label>
                  <Input
                    id="strategyType"
                    value={config.strategy?.type || ""}
                    onChange={(e) => updateConfigField(["strategy", "type"], e.target.value)}
                    placeholder="round_robin"
                  />
                  <p className="text-xs text-muted-foreground">{"可选值: round_robin, port_based, destination_based"}</p>
                </div>
                {config.strategy?.type === "port_based" && (
                  <div className="space-y-2">
                    <div className="flex items-center justify-between">
                      <Label>{"端口范围映射"}</Label>
                      <Button variant="outline" size="sm" onClick={addPortRange} className="gap-2">
                        <Plus className="h-4 w-4" />
                        {"添加"}
                      </Button>
                    </div>
                    {(config.strategy?.port_ranges || []).map((range, index) => (
                      <div key={index} className="flex gap-2">
                        <Input
                          value={range.range}
                          onChange={(e) => {
                            const newRanges = [...(config.strategy?.port_ranges || [])]
                            newRanges[index].range = e.target.value
                            updateConfigField(["strategy", "port_ranges"], newRanges)
                          }}
                          placeholder="0-32767"
                        />
                        <Input
                          value={range.ip}
                          onChange={(e) => {
                            const newRanges = [...(config.strategy?.port_ranges || [])]
                            newRanges[index].ip = e.target.value
                            updateConfigField(["strategy", "port_ranges"], newRanges)
                          }}
                          placeholder="1.2.3.4"
                        />
                        <Button variant="outline" size="icon" onClick={() => removePortRange(index)}>
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          {/* 网络配置 */}
          <TabsContent value="network" className="space-y-4">
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>{"SNAT配置"}</CardTitle>
                <CardDescription>{"源地址转换设置"}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label>{"启用SNAT"}</Label>
                    <p className="text-sm text-muted-foreground">{"启用源地址转换功能"}</p>
                  </div>
                  <Switch
                    checked={config.snat?.enabled || false}
                    onCheckedChange={(checked) => updateConfigField(["snat", "enabled"], checked)}
                  />
                </div>
                {config.snat?.enabled && (
                  <>
                    <div className="space-y-2">
                      <Label htmlFor="snatGateway">{"网关地址"}</Label>
                      <Input
                        id="snatGateway"
                        value={config.snat?.gateway || ""}
                        onChange={(e) => updateConfigField(["snat", "gateway"], e.target.value)}
                        placeholder="192.168.1.1"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="snatInterface">{"网络接口"}</Label>
                      <Input
                        id="snatInterface"
                        value={config.snat?.interface || ""}
                        onChange={(e) => updateConfigField(["snat", "interface"], e.target.value)}
                        placeholder="eth0"
                      />
                    </div>
                  </>
                )}
              </CardContent>
            </Card>

            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>{"连接管理"}</CardTitle>
                <CardDescription>{"连接超时和保活设置"}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {config.connection && (
                  <>
                    <div className="space-y-2">
                      <Label htmlFor="readTimeout">{"读取超时"}</Label>
                      <Input
                        id="readTimeout"
                        value={config.connection.read_timeout || ""}
                        onChange={(e) => updateConfigField(["connection", "read_timeout"], e.target.value)}
                        placeholder="30s"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="writeTimeout">{"写入超时"}</Label>
                      <Input
                        id="writeTimeout"
                        value={config.connection.write_timeout || ""}
                        onChange={(e) => updateConfigField(["connection", "write_timeout"], e.target.value)}
                        placeholder="30s"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="idleTimeout">{"空闲超时"}</Label>
                      <Input
                        id="idleTimeout"
                        value={config.connection.idle_timeout || ""}
                        onChange={(e) => updateConfigField(["connection", "idle_timeout"], e.target.value)}
                        placeholder="300s"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="dialTimeout">{"连接超时"}</Label>
                      <Input
                        id="dialTimeout"
                        value={config.connection.dial_timeout || ""}
                        onChange={(e) => updateConfigField(["connection", "dial_timeout"], e.target.value)}
                        placeholder="10s"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="maxConnections">{"最大连接数"}</Label>
                      <Input
                        id="maxConnections"
                        type="number"
                        value={config.connection.max_connections || 0}
                        onChange={(e) => updateConfigField(["connection", "max_connections"], parseInt(e.target.value) || 0)}
                        placeholder="1000"
                      />
                    </div>
                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label>{"保持连接"}</Label>
                        <p className="text-sm text-muted-foreground">{"启用TCP Keep-Alive"}</p>
                      </div>
                      <Switch
                        checked={config.connection.keep_alive || false}
                        onCheckedChange={(checked) => updateConfigField(["connection", "keep_alive"], checked)}
                      />
                    </div>
                    {config.connection.keep_alive && (
                      <div className="space-y-2">
                        <Label htmlFor="keepAliveTime">{"Keep-Alive间隔"}</Label>
                        <Input
                          id="keepAliveTime"
                          value={config.connection.keep_alive_time || ""}
                          onChange={(e) => updateConfigField(["connection", "keep_alive_time"], e.target.value)}
                          placeholder="30s"
                        />
                      </div>
                    )}
                  </>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          {/* 日志配置 */}
          <TabsContent value="logging" className="space-y-4">
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>{"日志设置"}</CardTitle>
                <CardDescription>{"配置日志级别和输出"}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="logLevel">{"日志级别"}</Label>
                  <Input
                    id="logLevel"
                    value={config.logging?.level || ""}
                    onChange={(e) => updateConfigField(["logging", "level"], e.target.value)}
                    placeholder="info"
                  />
                  <p className="text-xs text-muted-foreground">{"可选值: debug, info, warn, error"}</p>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="logFile">{"日志文件路径"}</Label>
                  <Input
                    id="logFile"
                    value={config.logging?.file || ""}
                    onChange={(e) => updateConfigField(["logging", "file"], e.target.value)}
                    placeholder="/var/log/proxy.log"
                  />
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          {/* 高级配置 */}
          <TabsContent value="advanced" className="space-y-4">
            {/* 健康检查 */}
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>{"健康检查"}</CardTitle>
                <CardDescription>{"IP健康检查配置"}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label>{"启用健康检查"}</Label>
                    <p className="text-sm text-muted-foreground">{"启用IP健康检查功能"}</p>
                  </div>
                  <Switch
                    checked={config.health_check?.enabled || false}
                    onCheckedChange={(checked) => {
                      if (!config.health_check) {
                        updateConfigField(["health_check"], { enabled: checked, interval: "30s", timeout: "5s" })
                      } else {
                        updateConfigField(["health_check", "enabled"], checked)
                      }
                    }}
                  />
                </div>
                {config.health_check?.enabled && (
                  <>
                    <div className="space-y-2">
                      <Label htmlFor="healthCheckInterval">{"检查间隔"}</Label>
                      <Input
                        id="healthCheckInterval"
                        value={config.health_check.interval || ""}
                        onChange={(e) => updateConfigField(["health_check", "interval"], e.target.value)}
                        placeholder="30s"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="healthCheckTimeout">{"检查超时"}</Label>
                      <Input
                        id="healthCheckTimeout"
                        value={config.health_check.timeout || ""}
                        onChange={(e) => updateConfigField(["health_check", "timeout"], e.target.value)}
                        placeholder="5s"
                      />
                    </div>
                  </>
                )}
              </CardContent>
            </Card>

            {/* IP检测 */}
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>{"IP自动检测"}</CardTitle>
                <CardDescription>{"自动检测出口IP配置"}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label>{"启用IP检测"}</Label>
                    <p className="text-sm text-muted-foreground">{"启用自动检测出口IP"}</p>
                  </div>
                  <Switch
                    checked={config.ip_detection?.enabled || false}
                    onCheckedChange={(checked) => {
                      if (!config.ip_detection) {
                        updateConfigField(["ip_detection"], { enabled: checked, interface: "" })
                      } else {
                        updateConfigField(["ip_detection", "enabled"], checked)
                      }
                    }}
                  />
                </div>
                {config.ip_detection?.enabled && (
                  <div className="space-y-2">
                    <Label htmlFor="ipDetectionInterface">{"网络接口"}</Label>
                    <Input
                      id="ipDetectionInterface"
                      value={config.ip_detection.interface || ""}
                      onChange={(e) => updateConfigField(["ip_detection", "interface"], e.target.value)}
                      placeholder="eth0 (留空检测所有接口)"
                    />
                  </div>
                )}
              </CardContent>
            </Card>

            {/* 监控配置 */}
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>{"监控统计"}</CardTitle>
                <CardDescription>{"启用监控和统计功能"}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label>{"启用监控"}</Label>
                    <p className="text-sm text-muted-foreground">{"启用系统监控和统计"}</p>
                  </div>
                  <Switch
                    checked={config.monitor?.enabled || false}
                    onCheckedChange={(checked) => {
                      if (!config.monitor) {
                        updateConfigField(["monitor"], { enabled: checked })
                      } else {
                        updateConfigField(["monitor", "enabled"], checked)
                      }
                    }}
                  />
                </div>
              </CardContent>
            </Card>

            {/* 地理位置配置 */}
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>{"地理位置选择"}</CardTitle>
                <CardDescription>{"基于地理位置的IP选择"}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label>{"启用地理位置选择"}</Label>
                    <p className="text-sm text-muted-foreground">{"启用基于地理位置的IP选择"}</p>
                  </div>
                  <Switch
                    checked={config.geo_location?.enabled || false}
                    onCheckedChange={(checked) => {
                      if (!config.geo_location) {
                        updateConfigField(["geo_location"], { enabled: checked, api_url: "", latency_optimize: false })
                      } else {
                        updateConfigField(["geo_location", "enabled"], checked)
                      }
                    }}
                  />
                </div>
                {config.geo_location?.enabled && (
                  <>
                    <div className="space-y-2">
                      <Label htmlFor="geoLocationAPI">{"API URL"}</Label>
                      <Input
                        id="geoLocationAPI"
                        value={config.geo_location.api_url || ""}
                        onChange={(e) => updateConfigField(["geo_location", "api_url"], e.target.value)}
                        placeholder="留空使用默认API"
                      />
                    </div>
                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label>{"延迟优化"}</Label>
                        <p className="text-sm text-muted-foreground">{"启用延迟优化"}</p>
                      </div>
                      <Switch
                        checked={config.geo_location.latency_optimize || false}
                        onCheckedChange={(checked) => updateConfigField(["geo_location", "latency_optimize"], checked)}
                      />
                    </div>
                  </>
                )}
              </CardContent>
            </Card>

            {/* 流量分析配置 */}
            <Card className="border-border/50">
              <CardHeader>
                <CardTitle>{"流量分析"}</CardTitle>
                <CardDescription>{"流量分析和异常检测配置"}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label>{"启用流量分析"}</Label>
                    <p className="text-sm text-muted-foreground">{"启用流量分析和异常检测"}</p>
                  </div>
                  <Switch
                    checked={config.traffic_analysis?.enabled || false}
                    onCheckedChange={(checked) => {
                      if (!config.traffic_analysis) {
                        updateConfigField(["traffic_analysis"], {
                          enabled: checked,
                          trend_window: "1h",
                          anomaly_threshold: 2.0,
                        })
                      } else {
                        updateConfigField(["traffic_analysis", "enabled"], checked)
                      }
                    }}
                  />
                </div>
                {config.traffic_analysis?.enabled && (
                  <>
                    <div className="space-y-2">
                      <Label htmlFor="trendWindow">{"趋势窗口"}</Label>
                      <Input
                        id="trendWindow"
                        value={config.traffic_analysis.trend_window || ""}
                        onChange={(e) => updateConfigField(["traffic_analysis", "trend_window"], e.target.value)}
                        placeholder="1h"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="anomalyThreshold">{"异常阈值"}</Label>
                      <Input
                        id="anomalyThreshold"
                        type="number"
                        step="0.1"
                        value={config.traffic_analysis.anomaly_threshold || 2.0}
                        onChange={(e) =>
                          updateConfigField(["traffic_analysis", "anomaly_threshold"], parseFloat(e.target.value) || 2.0)
                        }
                        placeholder="2.0"
                      />
                    </div>
                  </>
                )}
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
      <Toaster />
    </>
  )
}
