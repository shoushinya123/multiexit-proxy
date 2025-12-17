// API客户端

import { API_ENDPOINTS } from "./endpoints"
import {
  getBasicAuthHeader,
  getAuthFromStorage,
  getCSRFToken,
  saveCSRFToken,
  clearAuthFromStorage,
} from "../auth"
import type {
  StatusResponse,
  IPInfo,
  ServerConfig,
  RulesResponse,
  ConnectionStats,
  TrafficAnalysisResponse,
  ConfigVersionsResponse,
  SubscriptionLinkResponse,
  Rule,
  ApiResponse,
} from "./types"

export class ApiError extends Error {
  constructor(
    public status: number,
    public statusText: string,
    public data?: any
  ) {
    // 提取错误消息
    const errorMessage = typeof data === 'string' ? data : (data?.message || data?.error || JSON.stringify(data) || statusText)
    super(`API Error: ${status} ${errorMessage}`)
    this.name = "ApiError"
    this.message = errorMessage
  }
}

class ApiClient {
  private baseURL: string

  constructor(baseURL: string = "") {
    this.baseURL = baseURL
  }

  // 获取认证头
  private getAuthHeaders(): HeadersInit {
    const headers: HeadersInit = {
      "Content-Type": "application/json",
    }

    // 添加Basic Auth
    const auth = getAuthFromStorage()
    if (auth) {
      headers["Authorization"] = getBasicAuthHeader(auth.username, auth.password)
      console.log("Auth headers created:", { username: auth.username, hasPassword: !!auth.password })
    } else {
      console.log("No auth found in storage")
    }

    // 添加CSRF Token（如果存在）
    const csrfToken = getCSRFToken()
    if (csrfToken) {
      headers["X-CSRF-Token"] = csrfToken
    }

    return headers
  }

  // 处理响应
  private async handleResponse<T>(response: Response): Promise<T> {
    // 尝试从响应头获取CSRF Token
    const csrfToken = response.headers.get("X-CSRF-Token")
    if (csrfToken) {
      saveCSRFToken(csrfToken)
    }
    
    // 对于配置请求，也尝试从响应体中获取CSRF token
    // 这是因为Next.js的rewrites可能不会转发自定义响应头

    // 处理401未授权
    if (response.status === 401) {
      clearAuthFromStorage()
      // 使用setTimeout避免在请求处理过程中立即重定向，让组件有机会处理错误
      if (typeof window !== "undefined") {
        setTimeout(() => {
          // 只在当前路径不是登录页时才重定向
          if (window.location.pathname !== "/") {
            window.location.href = "/"
          }
        }, 100)
      }
      throw new ApiError(401, "Unauthorized", "认证失败，请重新登录")
    }

    // 处理429 Too Many Requests（登录保护）
    if (response.status === 429) {
      throw new ApiError(429, "Too Many Requests", "登录尝试次数过多，请稍后再试")
    }

    // 处理503 Service Unavailable（服务不可用，但不应该清除认证）
    if (response.status === 503) {
      let errorData: any
      try {
        errorData = await response.json()
      } catch {
        errorData = await response.text()
      }
      throw new ApiError(503, "Service Unavailable", errorData)
    }

    if (!response.ok) {
      let errorData: any
      const contentType = response.headers.get("content-type")
      if (contentType && contentType.includes("application/json")) {
        try {
          errorData = await response.json()
        } catch {
          errorData = response.statusText
        }
      } else {
        try {
          const text = await response.text()
          errorData = text || response.statusText
        } catch {
          errorData = response.statusText
        }
      }
      // 提取错误消息
      const errorMessage = errorData?.message || errorData?.error || (typeof errorData === 'string' ? errorData : JSON.stringify(errorData)) || response.statusText
      throw new ApiError(response.status, response.statusText, errorMessage)
    }

    // 处理空响应
    const contentType = response.headers.get("content-type")
    if (!contentType || !contentType.includes("application/json")) {
      const text = await response.text()
      return text as any
    }

    return response.json()
  }

  // GET请求
  async get<T>(endpoint: string, options?: RequestInit): Promise<T> {
    const response = await fetch(`${this.baseURL}${endpoint}`, {
      method: "GET",
      headers: this.getAuthHeaders(),
      ...options,
    })

    return this.handleResponse<T>(response)
  }

  // POST请求
  async post<T>(endpoint: string, data?: any, options?: RequestInit): Promise<T> {
    const response = await fetch(`${this.baseURL}${endpoint}`, {
      method: "POST",
      headers: this.getAuthHeaders(),
      body: data ? JSON.stringify(data) : undefined,
      ...options,
    })

    return this.handleResponse<T>(response)
  }

  // PUT请求
  async put<T>(endpoint: string, data?: any, options?: RequestInit): Promise<T> {
    const response = await fetch(`${this.baseURL}${endpoint}`, {
      method: "PUT",
      headers: this.getAuthHeaders(),
      body: data ? JSON.stringify(data) : undefined,
      ...options,
    })

    return this.handleResponse<T>(response)
  }

  // DELETE请求
  async delete<T>(endpoint: string, options?: RequestInit): Promise<T> {
    const response = await fetch(`${this.baseURL}${endpoint}`, {
      method: "DELETE",
      headers: this.getAuthHeaders(),
      ...options,
    })

    return this.handleResponse<T>(response)
  }

  // ============ API方法 ============

  // 获取系统状态
  async getStatus(): Promise<StatusResponse> {
    return this.get<StatusResponse>(API_ENDPOINTS.STATUS)
  }

  // 获取统计信息
  async getStats(): Promise<ConnectionStats> {
    return this.get<ConnectionStats>(API_ENDPOINTS.STATS)
  }

  // 获取IP列表
  async getIPs(): Promise<IPInfo[]> {
    return this.get<IPInfo[]>(API_ENDPOINTS.IPS)
  }

  // 获取配置
  async getConfig(): Promise<ServerConfig> {
    const response = await this.get<any>(API_ENDPOINTS.CONFIG)
    // 如果响应包含csrf_token，保存它
    if (response && response.csrf_token) {
      saveCSRFToken(response.csrf_token)
      return response.config || response
    }
    return response
  }

  // 更新配置
  async updateConfig(config: ServerConfig): Promise<ApiResponse> {
    return this.post<ApiResponse>(API_ENDPOINTS.CONFIG, config)
  }

  // 获取配置版本列表
  async getConfigVersions(): Promise<ConfigVersionsResponse> {
    return this.get<ConfigVersionsResponse>(API_ENDPOINTS.CONFIG_VERSIONS)
  }

  // 回滚配置
  async rollbackConfig(version: string): Promise<ApiResponse> {
    return this.post<ApiResponse>(API_ENDPOINTS.CONFIG_ROLLBACK, { version })
  }

  // 获取所有规则
  async getRules(): Promise<RulesResponse> {
    return this.get<RulesResponse>(API_ENDPOINTS.RULES)
  }

  // 添加规则
  async addRule(rule: Rule): Promise<ApiResponse<Rule>> {
    return this.post<ApiResponse<Rule>>(API_ENDPOINTS.RULES, rule)
  }

  // 更新规则
  async updateRule(id: string, rule: Rule): Promise<ApiResponse<Rule>> {
    return this.put<ApiResponse<Rule>>(API_ENDPOINTS.RULE(id), rule)
  }

  // 删除规则
  async deleteRule(id: string): Promise<ApiResponse> {
    return this.delete<ApiResponse>(API_ENDPOINTS.RULE(id))
  }

  // 获取流量分析数据
  async getTrafficAnalysis(timeRange?: string): Promise<TrafficAnalysisResponse> {
    const url = timeRange ? `${API_ENDPOINTS.TRAFFIC}?range=${timeRange}` : API_ENDPOINTS.TRAFFIC
    return this.get<TrafficAnalysisResponse>(url)
  }

  // 生成订阅链接
  async generateSubscriptionLink(): Promise<SubscriptionLinkResponse> {
    return this.get<SubscriptionLinkResponse>(API_ENDPOINTS.SUBSCRIPTION_LINK)
  }

  // 获取订阅配置（无需认证）
  async getSubscribeConfig(token: string): Promise<string> {
    const response = await fetch(`${this.baseURL}${API_ENDPOINTS.SUBSCRIBE}?token=${token}`)
    if (!response.ok) {
      throw new ApiError(response.status, response.statusText)
    }
    return response.text()
  }
}

// 导出单例
export const apiClient = new ApiClient()

