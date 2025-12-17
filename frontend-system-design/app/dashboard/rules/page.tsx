"use client"

import { useState } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Skeleton } from "@/components/ui/skeleton"
import { Plus, Trash2, Edit, GripVertical, AlertTriangle } from "lucide-react"
import { Switch } from "@/components/ui/switch"
import { useRules, useAddRule, useUpdateRule, useDeleteRule } from "@/hooks/use-api"
import { useToast } from "@/hooks/use-toast"
import { Toaster } from "@/components/ui/toaster"
import type { Rule } from "@/lib/api/types"

export default function RulesPage() {
  const { toast } = useToast()
  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<Rule | null>(null)
  const [newRule, setNewRule] = useState<Partial<Rule>>({
    name: "",
    priority: 1,
    match_domain: [],
    match_ip: [],
    match_port: [],
    action: "use_ip",
    enabled: true,
  })

  // 获取规则列表
  const { data: rulesData, isLoading, error, refetch } = useRules()

  // 添加规则
  const { mutate: addRule, isLoading: isAdding } = useAddRule({
    onSuccess: () => {
      toast({
        title: "规则已添加",
        description: "新规则已成功添加",
      })
      setIsDialogOpen(false)
      resetForm()
      refetch()
    },
    onError: (error) => {
      toast({
        title: "添加失败",
        description: error.message || "添加规则失败",
        variant: "destructive",
      })
    },
  })

  // 更新规则
  const { mutate: updateRule, isLoading: isUpdating } = useUpdateRule({
    onSuccess: () => {
      toast({
        title: "规则已更新",
        description: "规则已成功更新",
      })
      setIsDialogOpen(false)
      setEditingRule(null)
      resetForm()
      refetch()
    },
    onError: (error) => {
      toast({
        title: "更新失败",
        description: error.message || "更新规则失败",
        variant: "destructive",
      })
    },
  })

  // 删除规则
  const { mutate: deleteRule } = useDeleteRule({
    onSuccess: () => {
      toast({
        title: "规则已删除",
        description: "规则已成功删除",
      })
      refetch()
    },
    onError: (error) => {
      toast({
        title: "删除失败",
        description: error.message || "删除规则失败",
        variant: "destructive",
      })
    },
  })

  const resetForm = () => {
    setNewRule({
      name: "",
      priority: (rulesData?.rules?.length || 0) + 1,
      match_domain: [],
      match_ip: [],
      match_port: [],
      action: "use_ip",
      enabled: true,
    })
  }

  const handleAddRule = () => {
    if (!newRule.name) {
      toast({
        title: "输入错误",
        description: "请输入规则名称",
        variant: "destructive",
      })
      return
    }

    if (
      (!newRule.match_domain || newRule.match_domain.length === 0) &&
      (!newRule.match_ip || newRule.match_ip.length === 0) &&
      (!newRule.match_port || newRule.match_port.length === 0)
    ) {
      toast({
        title: "输入错误",
        description: "至少需要设置一个匹配条件",
        variant: "destructive",
      })
      return
    }

    // 验证target_ip在use_ip或redirect时是否必需
    if ((newRule.action === "use_ip" || newRule.action === "redirect") && !newRule.target_ip) {
      toast({
        title: "输入错误",
        description: "使用IP或重定向操作需要指定目标IP",
        variant: "destructive",
      })
      return
    }

    // 生成规则ID（如果不存在）
    const ruleId = newRule.id || `rule-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`

    const rule: Rule = {
      id: ruleId,
      name: newRule.name,
      priority: newRule.priority || 1,
      match_domain: newRule.match_domain || [],
      match_ip: newRule.match_ip || [],
      match_port: newRule.match_port || [],
      action: newRule.action || "use_ip",
      enabled: newRule.enabled !== false,
      target_ip: newRule.target_ip,
    }
    addRule(rule)
  }

  const handleUpdateRule = () => {
    if (!editingRule || !editingRule.id) return

    if (!editingRule.name) {
      toast({
        title: "输入错误",
        description: "请输入规则名称",
        variant: "destructive",
      })
      return
    }

    updateRule({ id: editingRule.id, rule: editingRule })
  }

  const handleDeleteRule = (id: string) => {
    deleteRule(id)
  }

  const handleToggleRule = (rule: Rule) => {
    if (!rule.id) return
    const updatedRule = { ...rule, enabled: !rule.enabled }
    updateRule({ id: rule.id, rule: updatedRule })
  }

  const openEditDialog = (rule: Rule) => {
    setEditingRule(JSON.parse(JSON.stringify(rule)))
    setIsDialogOpen(true)
  }

  const openAddDialog = () => {
    resetForm()
    setEditingRule(null)
    setIsDialogOpen(true)
  }

  const getActionColor = (action: string) => {
    switch (action) {
      case "use_ip":
        return "bg-chart-2 text-white"
      case "block":
        return "bg-destructive text-white"
      case "redirect":
        return "bg-chart-1 text-white"
      default:
        return "bg-muted text-muted-foreground"
    }
  }

  const getActionText = (action: string) => {
    switch (action) {
      case "use_ip":
        return "使用IP"
      case "block":
        return "阻止"
      case "redirect":
        return "重定向"
      default:
        return action
    }
  }

  const addMatchValue = (type: "domain" | "ip" | "port", value: string) => {
    if (editingRule) {
      const updated = { ...editingRule }
      if (type === "domain") {
        updated.match_domain = [...(updated.match_domain || []), value]
      } else if (type === "ip") {
        updated.match_ip = [...(updated.match_ip || []), value]
      } else if (type === "port") {
        const port = parseInt(value)
        if (!isNaN(port)) {
          updated.match_port = [...(updated.match_port || []), port]
        }
      }
      setEditingRule(updated)
    } else {
      const updated = { ...newRule }
      if (type === "domain") {
        updated.match_domain = [...(updated.match_domain || []), value]
      } else if (type === "ip") {
        updated.match_ip = [...(updated.match_ip || []), value]
      } else if (type === "port") {
        const port = parseInt(value)
        if (!isNaN(port)) {
          updated.match_port = [...(updated.match_port || []), port]
        }
      }
      setNewRule(updated)
    }
  }

  const removeMatchValue = (type: "domain" | "ip" | "port", index: number) => {
    if (editingRule) {
      const updated = { ...editingRule }
      if (type === "domain" && updated.match_domain) {
        updated.match_domain = updated.match_domain.filter((_, i) => i !== index)
      } else if (type === "ip" && updated.match_ip) {
        updated.match_ip = updated.match_ip.filter((_, i) => i !== index)
      } else if (type === "port" && updated.match_port) {
        updated.match_port = updated.match_port.filter((_, i) => i !== index)
      }
      setEditingRule(updated)
    } else {
      const updated = { ...newRule }
      if (type === "domain" && updated.match_domain) {
        updated.match_domain = updated.match_domain.filter((_, i) => i !== index)
      } else if (type === "ip" && updated.match_ip) {
        updated.match_ip = updated.match_ip.filter((_, i) => i !== index)
      } else if (type === "port" && updated.match_port) {
        updated.match_port = updated.match_port.filter((_, i) => i !== index)
      }
      setNewRule(updated)
    }
  }

  const currentRule = editingRule || newRule

  if (isLoading && !rulesData) {
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
        <AlertDescription>{error.message || "加载规则列表失败"}</AlertDescription>
      </Alert>
    )
  }

  const rules = rulesData?.rules || []

  return (
    <>
      <div className="space-y-6">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h1 className="text-3xl font-semibold tracking-tight">{"规则引擎"}</h1>
            <p className="text-muted-foreground mt-1">{"配置流量匹配和处理规则"}</p>
          </div>
          <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
            <DialogTrigger asChild>
              <Button size="sm" className="gap-2 bg-primary hover:bg-primary/90" onClick={openAddDialog}>
                <Plus className="h-4 w-4" />
                {"添加规则"}
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-y-auto">
              <DialogHeader>
                <DialogTitle>{editingRule ? "编辑规则" : "添加新规则"}</DialogTitle>
                <DialogDescription>{"创建或编辑流量处理规则"}</DialogDescription>
              </DialogHeader>
              <div className="space-y-4 py-4">
                <div className="space-y-2">
                  <Label htmlFor="name">{"规则名称"}</Label>
                  <Input
                    id="name"
                    placeholder="例如: 允许API访问"
                    value={currentRule.name || ""}
                    onChange={(e) => {
                      if (editingRule) {
                        setEditingRule({ ...editingRule, name: e.target.value })
                      } else {
                        setNewRule({ ...newRule, name: e.target.value })
                      }
                    }}
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="priority">{"优先级"}</Label>
                    <Input
                      id="priority"
                      type="number"
                      value={currentRule.priority || 1}
                      onChange={(e) => {
                        if (editingRule) {
                          setEditingRule({ ...editingRule, priority: parseInt(e.target.value) || 1 })
                        } else {
                          setNewRule({ ...newRule, priority: parseInt(e.target.value) || 1 })
                        }
                      }}
                    />
                    <p className="text-xs text-muted-foreground">{"数字越小优先级越高"}</p>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="action">{"动作"}</Label>
                    <Select
                      value={currentRule.action}
                      onValueChange={(value: any) => {
                        if (editingRule) {
                          setEditingRule({ ...editingRule, action: value })
                        } else {
                          setNewRule({ ...newRule, action: value })
                        }
                      }}
                    >
                      <SelectTrigger id="action">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="use_ip">{"使用IP"}</SelectItem>
                        <SelectItem value="block">{"阻止"}</SelectItem>
                        <SelectItem value="redirect">{"重定向"}</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                {/* 域名匹配 */}
                <div className="space-y-2">
                  <Label>{"域名匹配"}</Label>
                  <div className="flex gap-2">
                    <Input
                      placeholder="*.example.com"
                      onKeyDown={(e) => {
                        if (e.key === "Enter" && e.currentTarget.value) {
                          addMatchValue("domain", e.currentTarget.value)
                          e.currentTarget.value = ""
                        }
                      }}
                    />
                    <Button
                      type="button"
                      variant="outline"
                      onClick={(e) => {
                        const input = e.currentTarget.previousElementSibling as HTMLInputElement
                        if (input?.value) {
                          addMatchValue("domain", input.value)
                          input.value = ""
                        }
                      }}
                    >
                      <Plus className="h-4 w-4" />
                    </Button>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {(currentRule.match_domain || []).map((domain, index) => (
                      <Badge key={index} variant="secondary" className="gap-1">
                        {domain}
                        <button
                          onClick={() => removeMatchValue("domain", index)}
                          className="ml-1 hover:text-destructive"
                        >
                          ×
                        </button>
                      </Badge>
                    ))}
                  </div>
                </div>

                {/* IP匹配 */}
                <div className="space-y-2">
                  <Label>{"IP匹配"}</Label>
                  <div className="flex gap-2">
                    <Input
                      placeholder="192.168.1.0/24"
                      onKeyDown={(e) => {
                        if (e.key === "Enter" && e.currentTarget.value) {
                          addMatchValue("ip", e.currentTarget.value)
                          e.currentTarget.value = ""
                        }
                      }}
                    />
                    <Button
                      type="button"
                      variant="outline"
                      onClick={(e) => {
                        const input = e.currentTarget.previousElementSibling as HTMLInputElement
                        if (input?.value) {
                          addMatchValue("ip", input.value)
                          input.value = ""
                        }
                      }}
                    >
                      <Plus className="h-4 w-4" />
                    </Button>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {(currentRule.match_ip || []).map((ip, index) => (
                      <Badge key={index} variant="secondary" className="gap-1">
                        {ip}
                        <button
                          onClick={() => removeMatchValue("ip", index)}
                          className="ml-1 hover:text-destructive"
                        >
                          ×
                        </button>
                      </Badge>
                    ))}
                  </div>
                </div>

                {/* 端口匹配 */}
                <div className="space-y-2">
                  <Label>{"端口匹配"}</Label>
                  <div className="flex gap-2">
                    <Input
                      type="number"
                      placeholder="443"
                      onKeyDown={(e) => {
                        if (e.key === "Enter" && e.currentTarget.value) {
                          addMatchValue("port", e.currentTarget.value)
                          e.currentTarget.value = ""
                        }
                      }}
                    />
                    <Button
                      type="button"
                      variant="outline"
                      onClick={(e) => {
                        const input = e.currentTarget.previousElementSibling as HTMLInputElement
                        if (input?.value) {
                          addMatchValue("port", input.value)
                          input.value = ""
                        }
                      }}
                    >
                      <Plus className="h-4 w-4" />
                    </Button>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {(currentRule.match_port || []).map((port, index) => (
                      <Badge key={index} variant="secondary" className="gap-1">
                        {port}
                        <button
                          onClick={() => removeMatchValue("port", index)}
                          className="ml-1 hover:text-destructive"
                        >
                          ×
                        </button>
                      </Badge>
                    ))}
                  </div>
                </div>

                {/* 目标IP（用于use_ip动作） */}
                {currentRule.action === "use_ip" && (
                  <div className="space-y-2">
                    <Label htmlFor="targetIP">{"目标IP"}</Label>
                    <Input
                      id="targetIP"
                      placeholder="1.2.3.4"
                      value={currentRule.target_ip || ""}
                      onChange={(e) => {
                        if (editingRule) {
                          setEditingRule({ ...editingRule, target_ip: e.target.value })
                        } else {
                          setNewRule({ ...newRule, target_ip: e.target.value })
                        }
                      }}
                    />
                  </div>
                )}

                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label>{"启用规则"}</Label>
                    <p className="text-sm text-muted-foreground">{"是否启用此规则"}</p>
                  </div>
                  <Switch
                    checked={currentRule.enabled !== false}
                    onCheckedChange={(checked) => {
                      if (editingRule) {
                        setEditingRule({ ...editingRule, enabled: checked })
                      } else {
                        setNewRule({ ...newRule, enabled: checked })
                      }
                    }}
                  />
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setIsDialogOpen(false)} disabled={isAdding || isUpdating}>
                  {"取消"}
                </Button>
                <Button
                  onClick={editingRule ? handleUpdateRule : handleAddRule}
                  className="bg-primary hover:bg-primary/90"
                  disabled={isAdding || isUpdating}
                >
                  {isAdding || isUpdating ? "处理中..." : editingRule ? "更新" : "添加"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>

        <Card className="border-border/50">
          <CardHeader>
            <CardTitle>{"规则列表"}</CardTitle>
            <CardDescription>{"按优先级排序，规则从上到下依次匹配"}</CardDescription>
          </CardHeader>
          <CardContent>
            {rules.length === 0 ? (
              <div className="text-center text-muted-foreground py-8">{"暂无规则"}</div>
            ) : (
              <div className="rounded-md border border-border/50">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-12"></TableHead>
                      <TableHead>{"规则名称"}</TableHead>
                      <TableHead>{"优先级"}</TableHead>
                      <TableHead>{"匹配条件"}</TableHead>
                      <TableHead>{"动作"}</TableHead>
                      <TableHead>{"状态"}</TableHead>
                      <TableHead className="text-right">{"操作"}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {rules
                      .sort((a, b) => (a.priority || 0) - (b.priority || 0))
                      .map((rule) => (
                        <TableRow key={rule.id || rule.name}>
                          <TableCell>
                            <GripVertical className="h-4 w-4 text-muted-foreground cursor-move" />
                          </TableCell>
                          <TableCell className="font-medium">{rule.name}</TableCell>
                          <TableCell>
                            <Badge variant="outline" className="font-mono">
                              {rule.priority}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            <div className="space-y-1">
                              {rule.match_domain && rule.match_domain.length > 0 && (
                                <div className="text-xs">
                                  <span className="text-muted-foreground">域名: </span>
                                  {rule.match_domain.join(", ")}
                                </div>
                              )}
                              {rule.match_ip && rule.match_ip.length > 0 && (
                                <div className="text-xs">
                                  <span className="text-muted-foreground">IP: </span>
                                  {rule.match_ip.join(", ")}
                                </div>
                              )}
                              {rule.match_port && rule.match_port.length > 0 && (
                                <div className="text-xs">
                                  <span className="text-muted-foreground">端口: </span>
                                  {rule.match_port.join(", ")}
                                </div>
                              )}
                              {(!rule.match_domain || rule.match_domain.length === 0) &&
                                (!rule.match_ip || rule.match_ip.length === 0) &&
                                (!rule.match_port || rule.match_port.length === 0) && (
                                  <div className="text-xs text-muted-foreground">{"无匹配条件"}</div>
                                )}
                            </div>
                          </TableCell>
                          <TableCell>
                            <Badge className={getActionColor(rule.action)}>{getActionText(rule.action)}</Badge>
                            {rule.action === "use_ip" && rule.target_ip && (
                              <div className="text-xs text-muted-foreground mt-1">{rule.target_ip}</div>
                            )}
                          </TableCell>
                          <TableCell>
                            <Switch checked={rule.enabled !== false} onCheckedChange={() => handleToggleRule(rule)} />
                          </TableCell>
                          <TableCell className="text-right">
                            <div className="flex justify-end gap-1">
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-8 w-8 p-0 hover:bg-accent"
                                onClick={() => openEditDialog(rule)}
                              >
                                <Edit className="h-4 w-4" />
                              </Button>
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => rule.id && handleDeleteRule(rule.id)}
                                className="h-8 w-8 p-0 text-destructive hover:text-destructive hover:bg-destructive/10"
                              >
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            </div>
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
