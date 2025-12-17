// React Hooks for API调用

import { useState, useEffect, useCallback, useRef } from "react"
import { apiClient, ApiError } from "@/lib/api/client"
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
} from "@/lib/api/types"

// 通用数据获取Hook
export function useQuery<T>(
  queryFn: () => Promise<T>,
  options?: {
    enabled?: boolean
    refetchInterval?: number
    onSuccess?: (data: T) => void
    onError?: (error: ApiError) => void
  }
) {
  const [data, setData] = useState<T | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<ApiError | null>(null)

  const enabled = options?.enabled !== false

  // 使用 useRef 存储最新的 queryFn 和 options，避免依赖项变化导致无限循环
  const queryFnRef = useRef(queryFn)
  const optionsRef = useRef(options)

  // 更新 ref 的值，但不触发重新渲染
  useEffect(() => {
    queryFnRef.current = queryFn
    optionsRef.current = options
  })

  const fetchData = useCallback(async () => {
    if (!enabled) return

    setIsLoading(true)
    setError(null)

    try {
      // 使用 ref 中的最新值
      const result = await queryFnRef.current()
      setData(result)
      optionsRef.current?.onSuccess?.(result)
    } catch (err) {
      const apiError = err instanceof ApiError ? err : new ApiError(500, "Unknown Error", err)
      setError(apiError)
      optionsRef.current?.onError?.(apiError)
    } finally {
      setIsLoading(false)
    }
  }, [enabled]) // 只依赖 enabled，避免无限循环

  useEffect(() => {
    fetchData()
  }, [fetchData])

  // 单独处理自动刷新，避免错误时无限重试
  useEffect(() => {
    if (options?.refetchInterval && enabled) {
      const interval = setInterval(() => {
        // 只在没有错误或错误不是503/401时才自动重试
        // 401 Unauthorized 和 503 Service Unavailable 都应该停止重试
        setError((currentError) => {
          if (!currentError || (currentError instanceof ApiError && currentError.status !== 503 && currentError.status !== 401)) {
            fetchData()
          }
          return currentError
        })
      }, options.refetchInterval)
      return () => clearInterval(interval)
    }
  }, [fetchData, options?.refetchInterval, enabled])

  return { data, isLoading, error, refetch: fetchData }
}

// 通用数据更新Hook
export function useMutation<TData, TVariables = void>(
  mutationFn: (variables: TVariables) => Promise<TData>,
  options?: {
    onSuccess?: (data: TData) => void
    onError?: (error: ApiError) => void
  }
) {
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<ApiError | null>(null)

  const mutate = useCallback(
    async (variables: TVariables) => {
      setIsLoading(true)
      setError(null)

      try {
        const result = await mutationFn(variables)
        options?.onSuccess?.(result)
        return result
      } catch (err) {
        const apiError = err instanceof ApiError ? err : new ApiError(500, "Unknown Error", err)
        setError(apiError)
        options?.onError?.(apiError)
        throw apiError
      } finally {
        setIsLoading(false)
      }
    },
    [mutationFn, options]
  )

  return { mutate, isLoading, error }
}

// ============ 具体API Hooks ============

// 获取系统状态
export function useStatus(refetchInterval?: number) {
  return useQuery(() => apiClient.getStatus(), { refetchInterval })
}

// 获取统计信息
export function useStats(refetchInterval?: number) {
  return useQuery(() => apiClient.getStats(), { refetchInterval })
}

// 获取IP列表
export function useIPs(refetchInterval?: number) {
  return useQuery(() => apiClient.getIPs(), { refetchInterval })
}

// 获取配置
export function useConfig() {
  return useQuery(() => apiClient.getConfig())
}

// 更新配置
export function useUpdateConfig() {
  return useMutation((config: ServerConfig) => apiClient.updateConfig(config))
}

// 获取配置版本列表
export function useConfigVersions() {
  return useQuery(() => apiClient.getConfigVersions())
}

// 回滚配置
export function useRollbackConfig() {
  return useMutation((version: string) => apiClient.rollbackConfig(version))
}

// 获取规则列表
export function useRules() {
  return useQuery(() => apiClient.getRules())
}

// 添加规则
export function useAddRule() {
  return useMutation((rule: Rule) => apiClient.addRule(rule))
}

// 更新规则
export function useUpdateRule() {
  return useMutation(({ id, rule }: { id: string; rule: Rule }) =>
    apiClient.updateRule(id, rule)
  )
}

// 删除规则
export function useDeleteRule() {
  return useMutation((id: string) => apiClient.deleteRule(id))
}

// 获取流量分析
export function useTrafficAnalysis(timeRange?: string, refetchInterval?: number, enabled: boolean = true) {
  return useQuery(() => apiClient.getTrafficAnalysis(timeRange), { 
    refetchInterval,
    enabled,
    onError: (error) => {
      // 503错误不阻止页面渲染，只是禁用后续请求
      if (error.status === 503) {
        console.warn("Traffic analysis not enabled, disabling feature")
      }
    }
  })
}

// 生成订阅链接
export function useSubscriptionLink() {
  return useQuery(() => apiClient.generateSubscriptionLink())
}

