"use client"

import type React from "react"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Shield, AlertCircle } from "lucide-react"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { saveAuthToStorage } from "@/lib/auth"
import { apiClient, ApiError } from "@/lib/api/client"

export default function LoginPage() {
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const router = useRouter()

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)
    setError(null)

    try {
      // 从表单元素直接获取值，避免React状态异步更新问题
      const formData = new FormData(e.target as HTMLFormElement)
      const loginUsername = formData.get("username") as string
      const loginPassword = formData.get("password") as string

      // 先临时保存认证信息，以便API调用时可以使用
      saveAuthToStorage({ username: loginUsername, password: loginPassword })
      console.log("Auth saved, attempting login with:", loginUsername)

      // 尝试调用一个需要认证的API来验证凭据
      // 使用getStatus API，因为它需要Basic Auth
      console.log("Calling apiClient.getStatus()...")
      const result = await apiClient.getStatus()
      console.log("API call successful:", result)

      // 认证成功，跳转到仪表板
      router.push("/dashboard")
    } catch (err) {
      // 如果认证失败，清除临时保存的认证信息
      if (err instanceof ApiError && (err.status === 401 || err.status === 429)) {
        localStorage.removeItem("auth_token")
        localStorage.removeItem("auth_username")
        localStorage.removeItem("auth_password")
      }

      if (err instanceof ApiError) {
        if (err.status === 401) {
          setError("用户名或密码错误")
        } else if (err.status === 429) {
          setError("登录尝试次数过多，请稍后再试")
        } else {
          setError(`登录失败: ${err.message}`)
        }
      } else {
        setError("登录失败，请检查网络连接")
      }
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background p-4">
      <Card className="w-full max-w-md border-border/50">
        <CardHeader className="space-y-2 text-center">
          <div className="flex justify-center mb-2">
            <div className="p-3 rounded-xl bg-primary/10 border border-primary/20">
              <Shield className="h-8 w-8 text-primary" />
            </div>
          </div>
          <CardTitle className="text-2xl font-semibold">Traffic Management</CardTitle>
          <CardDescription className="text-muted-foreground">{"登录到您的管理控制台"}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleLogin} className="space-y-4">
            {error && (
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
            <div className="space-y-2">
              <label htmlFor="username" className="text-sm font-medium text-foreground">
                {"用户名"}
              </label>
              <Input
                id="username"
                name="username"
                type="text"
                placeholder="admin"
                value={username}
                onChange={(e) => {
                  setUsername(e.target.value)
                  setError(null)
                }}
                required
                className="bg-background"
                disabled={isLoading}
              />
            </div>
            <div className="space-y-2">
              <label htmlFor="password" className="text-sm font-medium text-foreground">
                {"密码"}
              </label>
              <Input
                id="password"
                name="password"
                type="password"
                placeholder="••••••••"
                value={password}
                onChange={(e) => {
                  setPassword(e.target.value)
                  setError(null)
                }}
                required
                className="bg-background"
                disabled={isLoading}
              />
            </div>
            <Button type="submit" className="w-full bg-primary hover:bg-primary/90" disabled={isLoading}>
              {isLoading ? "登录中..." : "登录"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
