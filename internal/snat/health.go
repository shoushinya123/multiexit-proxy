package snat

import (
	"context"
	"net"
	"sync"
	"time"
)

// IPHealthChecker IP健康检查器
type IPHealthChecker struct {
	ips           []net.IP
	checkInterval time.Duration
	timeout       time.Duration
	failedIPs     map[string]time.Time
	healthyIPs    map[string]bool
	failureCount  map[string]int // IP失败次数计数
	retryThreshold int          // 重试阈值（连续失败N次才标记为不健康）
	mu            sync.RWMutex
	stopCh        chan struct{}
	onIPFailed    func(ip net.IP)
	onIPRecovered func(ip net.IP)
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	IP      net.IP
	Healthy bool
	Latency time.Duration
	Error   error
}

// NewIPHealthChecker 创建IP健康检查器
func NewIPHealthChecker(ips []net.IP, checkInterval, timeout time.Duration) *IPHealthChecker {
	return &IPHealthChecker{
		ips:            ips,
		checkInterval:  checkInterval,
		timeout:        timeout,
		failedIPs:      make(map[string]time.Time),
		healthyIPs:     make(map[string]bool),
		failureCount:   make(map[string]int),
		retryThreshold: 3, // 默认连续失败3次才标记为不健康
		stopCh:         make(chan struct{}),
	}
}

// SetCallbacks 设置回调函数
func (h *IPHealthChecker) SetCallbacks(onFailed, onRecovered func(ip net.IP)) {
	h.onIPFailed = onFailed
	h.onIPRecovered = onRecovered
}

// Start 启动健康检查
func (h *IPHealthChecker) Start() {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	// 立即执行一次检查
	h.checkAll()

	for {
		select {
		case <-ticker.C:
			h.checkAll()
		case <-h.stopCh:
			return
		}
	}
}

// Stop 停止健康检查
func (h *IPHealthChecker) Stop() {
	close(h.stopCh)
}

// checkAll 检查所有IP
func (h *IPHealthChecker) checkAll() {
	var wg sync.WaitGroup
	results := make(chan HealthCheckResult, len(h.ips))

	for _, ip := range h.ips {
		wg.Add(1)
		go func(ip net.IP) {
			defer wg.Done()
			result := h.checkIP(ip)
			results <- result
		}(ip)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		h.updateHealth(result)
	}
}

// checkIP 检查单个IP
func (h *IPHealthChecker) checkIP(ip net.IP) HealthCheckResult {
	start := time.Now()
	
	// 使用TCP连接测试（更可靠）
		// 使用带超时的上下文进行健康检查
		ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	// 尝试连接常见的测试端口
	testPorts := []string{"80", "443", "53"}
	var lastErr error
	
	for _, port := range testPorts {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip.String(), port), h.timeout)
		if err == nil {
			conn.Close()
			latency := time.Since(start)
			return HealthCheckResult{
				IP:      ip,
				Healthy: true,
				Latency: latency,
			}
		}
		lastErr = err
		
		// 检查是否超时
		select {
		case <-ctx.Done():
			break
		default:
		}
	}

	// 如果所有端口都失败，尝试ICMP ping（需要root权限）
	if h.pingIP(ip) {
		return HealthCheckResult{
			IP:      ip,
			Healthy: true,
			Latency: time.Since(start),
		}
	}

	latency := time.Since(start)
	return HealthCheckResult{
		IP:      ip,
		Healthy: false,
		Latency: latency,
		Error:   lastErr,
	}
}

// pingIP 使用ICMP ping测试IP（简化版，实际可以使用golang.org/x/net/icmp）
func (h *IPHealthChecker) pingIP(ip net.IP) bool {
	// 简化实现：尝试连接DNS端口
	conn, err := net.DialTimeout("udp", net.JoinHostPort(ip.String(), "53"), h.timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// updateHealth 更新IP健康状态（带重试机制）
func (h *IPHealthChecker) updateHealth(result HealthCheckResult) {
	ipStr := result.IP.String()
	
	h.mu.Lock()
	defer h.mu.Unlock()

	wasHealthy := h.healthyIPs[ipStr]
	
	if result.Healthy {
		// IP健康，重置失败计数
		h.failureCount[ipStr] = 0
		h.healthyIPs[ipStr] = true
		delete(h.failedIPs, ipStr)
		
		// IP恢复
		if !wasHealthy && h.onIPRecovered != nil {
			h.onIPRecovered(result.IP)
		}
	} else {
		// IP失败，增加失败计数
		h.failureCount[ipStr]++
		
		// 只有连续失败达到阈值才标记为不健康
		if h.failureCount[ipStr] >= h.retryThreshold {
			h.healthyIPs[ipStr] = false
			h.failedIPs[ipStr] = time.Now()
			
			// IP故障（首次达到阈值时触发）
			if wasHealthy && h.onIPFailed != nil {
				h.onIPFailed(result.IP)
			}
		}
		// 如果失败次数未达到阈值，保持健康状态（允许临时故障）
	}
}

// IsHealthy 检查IP是否健康
func (h *IPHealthChecker) IsHealthy(ip net.IP) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	healthy, ok := h.healthyIPs[ip.String()]
	return ok && healthy
}

// GetHealthyIPs 获取所有健康的IP
func (h *IPHealthChecker) GetHealthyIPs() []net.IP {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	var healthy []net.IP
	for _, ip := range h.ips {
		if h.healthyIPs[ip.String()] {
			healthy = append(healthy, ip)
		}
	}
	return healthy
}

// GetFailedIPs 获取所有故障的IP
func (h *IPHealthChecker) GetFailedIPs() []net.IP {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	var failed []net.IP
	for ipStr := range h.failedIPs {
		ip := net.ParseIP(ipStr)
		if ip != nil {
			failed = append(failed, ip)
		}
	}
	return failed
}

// GetHealthStatus 获取健康状态统计
func (h *IPHealthChecker) GetHealthStatus() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	healthy := h.GetHealthyIPs()
	failed := h.GetFailedIPs()
	
	return map[string]interface{}{
		"total":   len(h.ips),
		"healthy": len(healthy),
		"failed":  len(failed),
		"healthy_ips": func() []string {
			var ips []string
			for _, ip := range healthy {
				ips = append(ips, ip.String())
			}
			return ips
		}(),
		"failed_ips": func() []string {
			var ips []string
			for _, ip := range failed {
				ips = append(ips, ip.String())
			}
			return ips
		}(),
	}
}

