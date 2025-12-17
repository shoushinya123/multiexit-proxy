// API端点常量定义

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "/api"

export const API_ENDPOINTS = {
  // 配置相关
  CONFIG: `${API_BASE}/config`,
  CONFIG_VERSIONS: `${API_BASE}/config/versions`,
  CONFIG_ROLLBACK: `${API_BASE}/config/rollback`,

  // IP管理
  IPS: `${API_BASE}/ips`,

  // 状态和统计
  STATUS: `${API_BASE}/status`,
  STATS: `${API_BASE}/stats`,
  METRICS: "/metrics", // Prometheus指标，不需要/api前缀

  // 规则引擎
  RULES: `${API_BASE}/rules`,
  RULE: (id: string) => `${API_BASE}/rules/${id}`,

  // 流量分析
  TRAFFIC: `${API_BASE}/traffic`,

  // 订阅功能
  SUBSCRIBE: `${API_BASE}/subscribe`,
  SUBSCRIPTION_LINK: `${API_BASE}/subscription/link`,
} as const



