// API响应类型定义

export interface StatusResponse {
  running: boolean
  version: string
  connections: number
}

export interface IPInfo {
  ip: string
  active: boolean
}

export interface ServerConfig {
  server: {
    listen: string
    tls: {
      cert: string
      key: string
      sni_fake: boolean
      fake_snis: string[]
    }
  }
  auth: {
    method: string
    key: string
  }
  exit_ips: string[]
  strategy: {
    type: string
    port_ranges?: Array<{
      range: string
      ip: string
    }>
  }
  snat: {
    enabled: boolean
    gateway: string
    interface: string
  }
  logging: {
    level: string
    file: string
  }
  web: {
    enabled: boolean
    listen: string
    username: string
    password: string
  }
  health_check?: {
    enabled: boolean
    interval: string
    timeout: string
  }
  ip_detection?: {
    enabled: boolean
    interface: string
  }
  connection?: {
    read_timeout: string
    write_timeout: string
    idle_timeout: string
    dial_timeout: string
    max_connections: number
    keep_alive: boolean
    keep_alive_time: string
  }
  monitor?: {
    enabled: boolean
  }
  geo_location?: {
    enabled: boolean
    api_url: string
    latency_optimize: boolean
    db_path?: string
  }
  rules?: Rule[]
  traffic_analysis?: {
    enabled: boolean
    trend_window: string
    anomaly_threshold: number
  }
}

export interface Rule {
  id?: string
  name: string
  priority: number
  match_domain?: string[]
  match_ip?: string[]
  match_port?: number[]
  target_ip?: string
  action: "use_ip" | "block" | "redirect"
  enabled: boolean
}

export interface RulesResponse {
  rules: Rule[]
  count: number
}

export interface ConnectionStats {
  total_connections: number
  active_connections: number
  bytes_transferred: number
  bytes_up: number
  bytes_down: number
  ip_stats: Record<string, IPConnectionStats>
}

export interface IPConnectionStats {
  connections: number
  active_conn: number
  bytes_up: number
  bytes_down: number
  total_bytes: number
  avg_latency: string
  last_used: string
}

export interface DomainStats {
  domain: string
  connections: number
  bytes_up: number
  bytes_down: number
  total_bytes: number
  avg_latency: string
  last_access: string
}

export interface TrafficTrend {
  timestamp: string
  bytes_up: number
  bytes_down: number
  connections: number
}

export interface AnomalyDetection {
  domain: string
  anomaly_type: string
  severity: string
  detected_at: string
  value: number
  expected_value: number
  description: string
}

export interface TrafficAnalysisResponse {
  domain_stats: Record<string, DomainStats>
  trends: TrafficTrend[]
  anomalies: AnomalyDetection[]
}

export interface ConfigVersion {
  version: string
  backup_path: string
  created_at: string
  description: string
  checksum: string
}

export interface ConfigVersionsResponse {
  versions: ConfigVersion[]
  count: number
}

export interface SubscriptionLinkResponse {
  token: string
  link: string
  qr_code: string
}

export interface ApiResponse<T = any> {
  status?: string
  message?: string
  data?: T
  version?: string
}



