package security

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// DDoSProtector DDoS防护器
type DDoSProtector struct {
	maxConnectionsPerIP int
	connectionRateLimit int // 每秒连接数
	suspiciousIPs       map[string]*SuspiciousActivity
	ipConnections       map[string]*int64 // 使用指针以便原子操作
	ipConnectionTimes   map[string][]time.Time
	mu                  sync.RWMutex
	blockDuration       time.Duration
}

// SuspiciousActivity 可疑活动
type SuspiciousActivity struct {
	IP            net.IP
	FirstSeen     time.Time
	LastSeen      time.Time
	ConnectionCount int64
	BlockedUntil  time.Time
}

// NewDDoSProtector 创建DDoS防护器
func NewDDoSProtector(maxConnPerIP, rateLimit int, blockDuration time.Duration) *DDoSProtector {
	return &DDoSProtector{
		maxConnectionsPerIP: maxConnPerIP,
		connectionRateLimit: rateLimit,
		suspiciousIPs:       make(map[string]*SuspiciousActivity),
		ipConnections:       make(map[string]*int64),
		ipConnectionTimes:   make(map[string][]time.Time),
		blockDuration:       blockDuration,
	}
}

// CheckIP 检查IP是否允许连接
func (d *DDoSProtector) CheckIP(clientIP net.IP) bool {
	ipStr := clientIP.String()
	
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// 检查是否被阻止
	if activity, exists := d.suspiciousIPs[ipStr]; exists {
		if time.Now().Before(activity.BlockedUntil) {
			return false // IP被阻止
		}
		// 阻止时间已过，解除阻止
		delete(d.suspiciousIPs, ipStr)
	}
	
	// 检查连接数限制
	var connCount int64
	if countPtr, exists := d.ipConnections[ipStr]; exists {
		connCount = atomic.LoadInt64(countPtr)
	} else {
		var zero int64
		d.ipConnections[ipStr] = &zero
	}
	if connCount >= int64(d.maxConnectionsPerIP) {
		d.markSuspicious(clientIP)
		return false
	}
	
	// 检查连接速率
	now := time.Now()
	times := d.ipConnectionTimes[ipStr]
	
	// 清理1秒前的记录
	var recentTimes []time.Time
	for _, t := range times {
		if now.Sub(t) < time.Second {
			recentTimes = append(recentTimes, t)
		}
	}
	d.ipConnectionTimes[ipStr] = recentTimes
	
	// 检查速率限制
	if len(recentTimes) >= d.connectionRateLimit {
		d.markSuspicious(clientIP)
		return false
	}
	
	// 记录此次连接
	recentTimes = append(recentTimes, now)
	d.ipConnectionTimes[ipStr] = recentTimes
	
	// 原子操作增加连接数
	countPtr, exists := d.ipConnections[ipStr]
	if !exists {
		var zero int64
		countPtr = &zero
		d.ipConnections[ipStr] = countPtr
	}
	atomic.AddInt64(countPtr, 1)
	
	return true
}

// OnConnectionEnd 连接结束时调用
func (d *DDoSProtector) OnConnectionEnd(clientIP net.IP) {
	ipStr := clientIP.String()
	
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if countPtr, exists := d.ipConnections[ipStr]; exists {
		current := atomic.LoadInt64(countPtr)
		if current > 0 {
			atomic.AddInt64(countPtr, -1)
		}
	}
}

// markSuspicious 标记可疑IP
func (d *DDoSProtector) markSuspicious(clientIP net.IP) {
	ipStr := clientIP.String()
	
	activity, exists := d.suspiciousIPs[ipStr]
	if !exists {
		activity = &SuspiciousActivity{
			IP:            clientIP,
			FirstSeen:     time.Now(),
		}
	}
	
	activity.LastSeen = time.Now()
	activity.ConnectionCount++
	activity.BlockedUntil = time.Now().Add(d.blockDuration)
	
	d.suspiciousIPs[ipStr] = activity
}

// GetBlockedIPs 获取被阻止的IP列表
func (d *DDoSProtector) GetBlockedIPs() []net.IP {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	var blocked []net.IP
	now := time.Now()
	
	for _, activity := range d.suspiciousIPs {
		if now.Before(activity.BlockedUntil) {
			blocked = append(blocked, activity.IP)
		}
	}
	
	return blocked
}

// UnblockIP 解除IP阻止
func (d *DDoSProtector) UnblockIP(clientIP net.IP) {
	ipStr := clientIP.String()
	
	d.mu.Lock()
	defer d.mu.Unlock()
	
	delete(d.suspiciousIPs, ipStr)
	delete(d.ipConnections, ipStr)
	delete(d.ipConnectionTimes, ipStr)
}

// GetStats 获取防护统计
func (d *DDoSProtector) GetStats() map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	blockedCount := 0
	now := time.Now()
	
	for _, activity := range d.suspiciousIPs {
		if now.Before(activity.BlockedUntil) {
			blockedCount++
		}
	}
	
	return map[string]interface{}{
		"blocked_ips":      blockedCount,
		"max_connections":  d.maxConnectionsPerIP,
		"rate_limit":       d.connectionRateLimit,
		"block_duration":   d.blockDuration.String(),
	}
}

