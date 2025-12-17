// 认证工具函数

export interface AuthCredentials {
  username: string
  password: string
}

// 生成Basic Auth Header
export function getBasicAuthHeader(username: string, password: string): string {
  // 使用标准方式构造Basic Auth头
  const credentials = btoa(`${username}:${password}`)
  return `Basic ${credentials}`
}

// 从localStorage获取认证信息
export function getAuthFromStorage(): AuthCredentials | null {
  if (typeof window === "undefined") return null

  const username = localStorage.getItem("auth_username")
  const password = localStorage.getItem("auth_password")

  if (username && password) {
    return { username, password }
  }
  return null
}

// 保存认证信息到localStorage
export function saveAuthToStorage(credentials: AuthCredentials): void {
  if (typeof window === "undefined") return

  localStorage.setItem("auth_username", credentials.username)
  localStorage.setItem("auth_password", credentials.password)
  localStorage.setItem("auth_token", "authenticated") // 简单的认证标记
}

// 清除认证信息
export function clearAuthFromStorage(): void {
  if (typeof window === "undefined") return

  localStorage.removeItem("auth_username")
  localStorage.removeItem("auth_password")
  localStorage.removeItem("auth_token")
  localStorage.removeItem("csrf_token")
}

// 检查是否已认证
export function isAuthenticated(): boolean {
  if (typeof window === "undefined") return false
  return localStorage.getItem("auth_token") === "authenticated"
}

// 获取CSRF Token（从响应头或localStorage）
export function getCSRFToken(): string | null {
  if (typeof window === "undefined") return null
  return localStorage.getItem("csrf_token")
}

// 保存CSRF Token
export function saveCSRFToken(token: string): void {
  if (typeof window === "undefined") return
  localStorage.setItem("csrf_token", token)
}

