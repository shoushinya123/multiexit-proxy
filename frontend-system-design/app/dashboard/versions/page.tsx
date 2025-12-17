"use client"

import { useState } from "react"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
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
import { RotateCcw, Eye, Clock, AlertTriangle } from "lucide-react"
import { useConfigVersions, useRollbackConfig } from "@/hooks/use-api"
import { useToast } from "@/hooks/use-toast"
import { Toaster } from "@/components/ui/toaster"
import { formatTimestamp } from "@/lib/utils"

export default function VersionsPage() {
  const { toast } = useToast()
  const [selectedVersion, setSelectedVersion] = useState<string | null>(null)
  const [isRollbackDialogOpen, setIsRollbackDialogOpen] = useState(false)
  const [rollbackVersion, setRollbackVersion] = useState<string | null>(null)

  // 获取配置版本列表
  const { data: versionsData, isLoading, error, refetch } = useConfigVersions()

  // 回滚配置的mutation
  const { mutate: rollbackConfig, isLoading: isRollingBack } = useRollbackConfig({
    onSuccess: () => {
      toast({
        title: "回滚成功",
        description: `已回滚到版本 ${rollbackVersion}`,
      })
      setIsRollbackDialogOpen(false)
      setRollbackVersion(null)
      refetch()
    },
    onError: (error) => {
      toast({
        title: "回滚失败",
        description: error.message || "回滚配置失败",
        variant: "destructive",
      })
    },
  })

  const handleRollback = (version: string) => {
    setRollbackVersion(version)
    setIsRollbackDialogOpen(true)
  }

  const confirmRollback = () => {
    if (rollbackVersion) {
      rollbackConfig(rollbackVersion)
    }
  }

  // 获取当前版本（第一个版本为当前版本）
  const currentVersion = versionsData?.versions && versionsData.versions.length > 0 
    ? versionsData.versions[0].version 
    : null

  if (isLoading && !versionsData) {
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
        <AlertDescription>{error.message || "加载版本列表失败"}</AlertDescription>
      </Alert>
    )
  }

  const versions = versionsData?.versions || []

  return (
    <>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight">{"版本回滚"}</h1>
          <p className="text-muted-foreground mt-1">{"查看配置历史版本并执行回滚操作"}</p>
        </div>

        <Card className="border-border/50">
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>{"配置版本历史"}</CardTitle>
                <CardDescription>{"所有配置变更的历史记录"}</CardDescription>
              </div>
              {currentVersion && (
                <Badge className="bg-primary/10 text-primary border-primary/20">{"当前版本: " + currentVersion}</Badge>
              )}
            </div>
          </CardHeader>
          <CardContent>
            {versions.length === 0 ? (
              <div className="text-center text-muted-foreground py-8">{"暂无版本历史"}</div>
            ) : (
              <div className="rounded-md border border-border/50">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{"版本号"}</TableHead>
                      <TableHead>{"创建时间"}</TableHead>
                      <TableHead>{"描述"}</TableHead>
                      <TableHead>{"校验和"}</TableHead>
                      <TableHead>{"状态"}</TableHead>
                      <TableHead className="text-right">{"操作"}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {versions.map((version, index) => {
                      const isCurrent = index === 0
                      return (
                        <TableRow key={version.version}>
                          <TableCell className="font-mono font-medium">{version.version}</TableCell>
                          <TableCell>
                            <div className="flex items-center gap-2">
                              <Clock className="h-3 w-3 text-muted-foreground" />
                              <span className="text-sm">{formatTimestamp(version.created_at)}</span>
                            </div>
                          </TableCell>
                          <TableCell className="max-w-md">
                            <span className="text-sm">{version.description || "无描述"}</span>
                          </TableCell>
                          <TableCell>
                            <code className="text-xs font-mono bg-muted/50 px-2 py-1 rounded">
                              {version.checksum.substring(0, 12)}...
                            </code>
                          </TableCell>
                          <TableCell>
                            {isCurrent ? (
                              <Badge className="bg-chart-2 text-white">{"当前版本"}</Badge>
                            ) : (
                              <Badge variant="outline">{"历史版本"}</Badge>
                            )}
                          </TableCell>
                          <TableCell className="text-right">
                            <div className="flex justify-end gap-1">
                              <Dialog>
                                <DialogTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => setSelectedVersion(version.version)}
                                    className="h-8 px-3 hover:bg-accent"
                                  >
                                    <Eye className="h-4 w-4 mr-1" />
                                    {"查看"}
                                  </Button>
                                </DialogTrigger>
                                <DialogContent>
                                  <DialogHeader>
                                    <DialogTitle>{"版本详情"}</DialogTitle>
                                    <DialogDescription>{`查看版本 ${version.version} 的详细信息`}</DialogDescription>
                                  </DialogHeader>
                                  <div className="space-y-4 py-4">
                                    <div className="grid grid-cols-2 gap-4">
                                      <div>
                                        <Label className="text-xs text-muted-foreground">{"版本号"}</Label>
                                        <p className="text-sm font-mono mt-1">{version.version}</p>
                                      </div>
                                      <div>
                                        <Label className="text-xs text-muted-foreground">{"校验和"}</Label>
                                        <p className="text-sm font-mono mt-1">{version.checksum}</p>
                                      </div>
                                    </div>
                                    <div>
                                      <Label className="text-xs text-muted-foreground">{"创建时间"}</Label>
                                      <p className="text-sm mt-1">{formatTimestamp(version.created_at)}</p>
                                    </div>
                                    <div>
                                      <Label className="text-xs text-muted-foreground">{"描述"}</Label>
                                      <p className="text-sm mt-1 leading-relaxed">{version.description || "无描述"}</p>
                                    </div>
                                    <div>
                                      <Label className="text-xs text-muted-foreground">{"备份路径"}</Label>
                                      <p className="text-sm font-mono mt-1 break-all">{version.backup_path}</p>
                                    </div>
                                  </div>
                                  <DialogFooter>
                                    <Button variant="outline" onClick={() => setSelectedVersion(null)}>
                                      {"关闭"}
                                    </Button>
                                  </DialogFooter>
                                </DialogContent>
                              </Dialog>

                              {!isCurrent && (
                                <Dialog open={isRollbackDialogOpen && rollbackVersion === version.version} onOpenChange={setIsRollbackDialogOpen}>
                                  <DialogTrigger asChild>
                                    <Button
                                      variant="ghost"
                                      size="sm"
                                      className="h-8 px-3 text-primary hover:text-primary hover:bg-primary/10"
                                      onClick={() => handleRollback(version.version)}
                                    >
                                      <RotateCcw className="h-4 w-4 mr-1" />
                                      {"回滚"}
                                    </Button>
                                  </DialogTrigger>
                                  <DialogContent>
                                    <DialogHeader>
                                      <DialogTitle>{"确认回滚"}</DialogTitle>
                                      <DialogDescription>
                                        {`您确定要回滚到版本 ${version.version} 吗？此操作将创建新的配置版本。`}
                                      </DialogDescription>
                                    </DialogHeader>
                                    <div className="py-4 space-y-2">
                                      <div className="p-3 rounded-lg bg-muted/30 border border-border/50">
                                        <p className="text-sm">
                                          <span className="font-medium">{"目标版本: "}</span>
                                          <span className="font-mono">{version.version}</span>
                                        </p>
                                        <p className="text-sm mt-1 text-muted-foreground">
                                          {version.description || "无描述"}
                                        </p>
                                        <p className="text-xs text-muted-foreground mt-2">
                                          {`创建时间: ${formatTimestamp(version.created_at)}`}
                                        </p>
                                      </div>
                                    </div>
                                    <DialogFooter>
                                      <Button
                                        variant="outline"
                                        onClick={() => {
                                          setIsRollbackDialogOpen(false)
                                          setRollbackVersion(null)
                                        }}
                                        disabled={isRollingBack}
                                      >
                                        {"取消"}
                                      </Button>
                                      <Button onClick={confirmRollback} className="bg-primary hover:bg-primary/90" disabled={isRollingBack}>
                                        {isRollingBack ? "回滚中..." : "确认回滚"}
                                      </Button>
                                    </DialogFooter>
                                  </DialogContent>
                                </Dialog>
                              )}
                            </div>
                          </TableCell>
                        </TableRow>
                      )
                    })}
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
