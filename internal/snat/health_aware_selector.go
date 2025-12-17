package snat

import (
	"net"
	"sync"
)

// HealthAwareIPSelector 健康感知的IP选择器包装器
// 自动过滤掉不健康的IP，只从健康的IP中选择
type HealthAwareIPSelector struct {
	baseSelector  IPSelector
	healthChecker *IPHealthChecker
	allIPs        []string              // 所有配置的IP
	healthyIPs    map[string]bool       // 当前健康的IP映射
	mu            sync.RWMutex
	selectorType  string                // 选择器类型：round_robin, destination_based, load_balanced
	strategy      string                // load_balanced的策略：connections 或 traffic
}

// NewHealthAwareIPSelector 创建健康感知的IP选择器
func NewHealthAwareIPSelector(baseSelector IPSelector, healthChecker *IPHealthChecker, allIPs []string, selectorType, strategy string) *HealthAwareIPSelector {
	selector := &HealthAwareIPSelector{
		baseSelector:  baseSelector,
		healthChecker: healthChecker,
		allIPs:        allIPs,
		healthyIPs:    make(map[string]bool),
		selectorType:  selectorType,
		strategy:      strategy,
	}

	// 初始化健康IP列表
	healthy := healthChecker.GetHealthyIPs()
	for _, ip := range healthy {
		selector.healthyIPs[ip.String()] = true
	}

	// 设置健康检查器的回调，当IP状态变化时更新选择器
	healthChecker.SetCallbacks(
		func(ip net.IP) {
			// IP故障回调
			selector.mu.Lock()
			delete(selector.healthyIPs, ip.String())
			selector.mu.Unlock()
			
			// 通知更新IP列表
			selector.updateBaseSelectorIPs()
		},
		func(ip net.IP) {
			// IP恢复回调
			selector.mu.Lock()
			selector.healthyIPs[ip.String()] = true
			selector.mu.Unlock()
			
			// 通知更新IP列表
			selector.updateBaseSelectorIPs()
		},
	)

	return selector
}

// updateBaseSelectorIPs 更新基础选择器的IP列表（只包含健康的IP）
func (h *HealthAwareIPSelector) updateBaseSelectorIPs() {
	h.mu.RLock()
	var healthyIPStrings []string
	for _, ipStr := range h.allIPs {
		if h.healthyIPs[ipStr] {
			healthyIPStrings = append(healthyIPStrings, ipStr)
		}
	}
	h.mu.RUnlock()

	if len(healthyIPStrings) == 0 {
		return // 没有健康的IP，保持原状态
	}

	// 根据基础选择器的类型更新IP列表
	var newSelector IPSelector
	var err error
	
	switch h.selectorType {
	case "round_robin":
		newSelector, err = NewRoundRobinSelector(healthyIPStrings)
	case "destination_based":
		newSelector, err = NewDestinationBasedSelector(healthyIPStrings)
	case "load_balanced":
		if h.strategy == "" {
			h.strategy = "connections"
		}
		newSelector, err = NewLoadBalancedSelector(healthyIPStrings, h.strategy)
	default:
		newSelector, err = NewRoundRobinSelector(healthyIPStrings)
	}
	
	if err == nil {
		h.mu.Lock()
		h.baseSelector = newSelector
		h.mu.Unlock()
	}
}

// SelectIP 选择IP（只从健康的IP中选择）
func (h *HealthAwareIPSelector) SelectIP(targetAddr string, targetPort int) (net.IP, error) {
	h.mu.RLock()
	healthyCount := len(h.healthyIPs)
	h.mu.RUnlock()

	if healthyCount == 0 {
		// 没有健康的IP，尝试从健康检查器重新获取
		healthy := h.healthChecker.GetHealthyIPs()
		if len(healthy) == 0 {
			return nil, &NoIPAvailableError{}
		}
		
		// 更新健康IP列表
		h.mu.Lock()
		h.healthyIPs = make(map[string]bool)
		for _, ip := range healthy {
			h.healthyIPs[ip.String()] = true
		}
		h.mu.Unlock()
		
		h.updateBaseSelectorIPs()
	}

	h.mu.RLock()
	selector := h.baseSelector
	h.mu.RUnlock()

	return selector.SelectIP(targetAddr, targetPort)
}

// GetHealthyIPs 获取当前健康的IP列表
func (h *HealthAwareIPSelector) GetHealthyIPs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var ips []string
	for ipStr, healthy := range h.healthyIPs {
		if healthy {
			ips = append(ips, ipStr)
		}
	}
	return ips
}

// IsHealthy 检查指定IP是否健康
func (h *HealthAwareIPSelector) IsHealthy(ip string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.healthyIPs[ip]
}

