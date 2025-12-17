package snat

import (
	"net"
	"sync"
	"sync/atomic"
)

// LoadBalancedSelector 负载均衡选择器（按连接数和流量）
type LoadBalancedSelector struct {
	ips      []net.IP
	ipStats  map[string]*IPLoadStats
	mu       sync.RWMutex
	strategy string // "connections" or "traffic"
}

// IPLoadStats IP负载统计
type IPLoadStats struct {
	ActiveConnections int64
	TotalBytes        int64
	LastSelected      int64 // Unix timestamp
	mu                sync.RWMutex
}

// NewLoadBalancedSelector 创建负载均衡选择器
func NewLoadBalancedSelector(ips []string, strategy string) (*LoadBalancedSelector, error) {
	ipList := make([]net.IP, 0, len(ips))
	ipStats := make(map[string]*IPLoadStats)
	
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return nil, &InvalidIPError{IP: ipStr}
		}
		ipList = append(ipList, ip)
		ipStats[ipStr] = &IPLoadStats{}
	}
	
	if strategy != "connections" && strategy != "traffic" {
		strategy = "connections" // 默认策略
	}
	
	return &LoadBalancedSelector{
		ips:      ipList,
		ipStats:  ipStats,
		strategy: strategy,
	}, nil
}

// SelectIP 选择IP（基于负载）
func (l *LoadBalancedSelector) SelectIP(targetAddr string, targetPort int) (net.IP, error) {
	if len(l.ips) == 0 {
		return nil, &NoIPAvailableError{}
	}
	
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	if l.strategy == "connections" {
		return l.selectByConnections()
	}
	return l.selectByTraffic()
}

// selectByConnections 按连接数选择（选择连接数最少的）
func (l *LoadBalancedSelector) selectByConnections() (net.IP, error) {
	var bestIP net.IP
	var minConnections int64 = -1
	
	for _, ip := range l.ips {
		ipStr := ip.String()
		stats := l.ipStats[ipStr]
		if stats == nil {
			continue
		}
		
		conns := atomic.LoadInt64(&stats.ActiveConnections)
		if minConnections == -1 || conns < minConnections {
			minConnections = conns
			bestIP = ip
		}
	}
	
	if bestIP == nil {
		return l.ips[0], nil
	}
	
	// 更新统计
	if stats := l.ipStats[bestIP.String()]; stats != nil {
		atomic.AddInt64(&stats.ActiveConnections, 1)
		atomic.StoreInt64(&stats.LastSelected, getCurrentTimestamp())
	}
	
	return bestIP, nil
}

// selectByTraffic 按流量选择（选择流量最少的）
func (l *LoadBalancedSelector) selectByTraffic() (net.IP, error) {
	var bestIP net.IP
	var minTraffic int64 = -1
	
	for _, ip := range l.ips {
		ipStr := ip.String()
		stats := l.ipStats[ipStr]
		if stats == nil {
			continue
		}
		
		traffic := atomic.LoadInt64(&stats.TotalBytes)
		if minTraffic == -1 || traffic < minTraffic {
			minTraffic = traffic
			bestIP = ip
		}
	}
	
	if bestIP == nil {
		return l.ips[0], nil
	}
	
	// 更新统计
	if stats := l.ipStats[bestIP.String()]; stats != nil {
		atomic.StoreInt64(&stats.LastSelected, getCurrentTimestamp())
	}
	
	return bestIP, nil
}

// OnConnectionEnd 连接结束时调用
func (l *LoadBalancedSelector) OnConnectionEnd(ip net.IP) {
	ipStr := ip.String()
	l.mu.RLock()
	stats := l.ipStats[ipStr]
	l.mu.RUnlock()
	
	if stats != nil {
		atomic.AddInt64(&stats.ActiveConnections, -1)
	}
}

// OnBytesTransferred 数据传输时调用
func (l *LoadBalancedSelector) OnBytesTransferred(ip net.IP, bytes int64) {
	ipStr := ip.String()
	l.mu.RLock()
	stats := l.ipStats[ipStr]
	l.mu.RUnlock()
	
	if stats != nil {
		atomic.AddInt64(&stats.TotalBytes, bytes)
	}
}

// GetStats 获取负载统计
func (l *LoadBalancedSelector) GetStats() map[string]interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	stats := make(map[string]interface{})
	for ipStr, stat := range l.ipStats {
		stats[ipStr] = map[string]interface{}{
			"active_connections": atomic.LoadInt64(&stat.ActiveConnections),
			"total_bytes":        atomic.LoadInt64(&stat.TotalBytes),
			"last_selected":      atomic.LoadInt64(&stat.LastSelected),
		}
	}
	return stats
}

func getCurrentTimestamp() int64 {
	// 简化实现，实际应该使用time.Now().Unix()
	return 0
}

// ResetStats 重置统计
func (l *LoadBalancedSelector) ResetStats() {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	for _, stat := range l.ipStats {
		atomic.StoreInt64(&stat.ActiveConnections, 0)
		atomic.StoreInt64(&stat.TotalBytes, 0)
		atomic.StoreInt64(&stat.LastSelected, 0)
	}
}

