package proxy

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// RateLimiter 速率限制器
type RateLimiter struct {
	// IP限流
	ipLimits      map[string]*IPLimit
	ipMu          sync.RWMutex
	
	// 用户限流
	userLimits    map[string]*UserLimit
	userMu        sync.RWMutex
	
	// 全局限流
	globalLimit   *GlobalLimit
}

// IPLimit IP限流配置
type IPLimit struct {
	MaxConnections int           // 最大并发连接数
	RateLimit      int           // 每秒连接数
	Connections    int64         // 当前连接数（原子操作）
	LastConnTime   time.Time     // 最后连接时间
	ConnTimes      []time.Time   // 连接时间记录
	mu             sync.RWMutex
}

// UserLimit 用户限流配置
type UserLimit struct {
	MaxConnections int           // 最大并发连接数
	RateLimit      int           // 每秒连接数
	BandwidthLimit int64         // 带宽限制（字节/秒）
	Connections    int64         // 当前连接数（原子操作）
	BytesTransferred int64       // 已传输字节数（原子操作）
	LastReset      time.Time     // 上次重置时间
	mu             sync.RWMutex
}

// GlobalLimit 全局限流配置
type GlobalLimit struct {
	MaxConnections int64         // 全局最大连接数
	RateLimit      int           // 全局每秒连接数
	Connections    int64         // 当前连接数（原子操作）
	ConnTimes      []time.Time   // 连接时间记录
	mu             sync.RWMutex
}

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	// IP限流
	IPMaxConnections int           // 每个IP最大连接数
	IPRateLimit      int           // 每个IP每秒连接数
	
	// 用户限流
	UserMaxConnections int         // 每个用户最大连接数
	UserRateLimit      int         // 每个用户每秒连接数
	UserBandwidthLimit int64       // 每个用户带宽限制（字节/秒）
	
	// 全局限流
	GlobalMaxConnections int       // 全局最大连接数
	GlobalRateLimit      int       // 全局每秒连接数
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		ipLimits: make(map[string]*IPLimit),
		userLimits: make(map[string]*UserLimit),
		globalLimit: &GlobalLimit{
			MaxConnections: int64(config.GlobalMaxConnections),
			RateLimit:      config.GlobalRateLimit,
			ConnTimes:      make([]time.Time, 0),
		},
	}
}

// CheckIP 检查IP是否允许连接
func (rl *RateLimiter) CheckIP(clientIP net.IP) bool {
	ipStr := clientIP.String()
	
	rl.ipMu.Lock()
	limit, exists := rl.ipLimits[ipStr]
	if !exists {
		limit = &IPLimit{
			MaxConnections: 10, // 默认值
			RateLimit:      5,  // 默认值
			ConnTimes:      make([]time.Time, 0),
		}
		rl.ipLimits[ipStr] = limit
	}
	rl.ipMu.Unlock()

	limit.mu.Lock()
	defer limit.mu.Unlock()

	// 检查连接数限制
	currentConn := atomic.LoadInt64(&limit.Connections)
	if currentConn >= int64(limit.MaxConnections) {
		logrus.Debugf("IP %s exceeded max connections (%d)", ipStr, limit.MaxConnections)
		return false
	}

	// 检查速率限制
	now := time.Now()
	// 清理1秒前的记录
	var recentTimes []time.Time
	for _, t := range limit.ConnTimes {
		if now.Sub(t) < time.Second {
			recentTimes = append(recentTimes, t)
		}
	}
	limit.ConnTimes = recentTimes

	if len(recentTimes) >= limit.RateLimit {
		logrus.Debugf("IP %s exceeded rate limit (%d/sec)", ipStr, limit.RateLimit)
		return false
	}

	// 记录此次连接
	limit.ConnTimes = append(recentTimes, now)
	atomic.AddInt64(&limit.Connections, 1)
	return true
}

// OnIPConnectionEnd IP连接结束时调用
func (rl *RateLimiter) OnIPConnectionEnd(clientIP net.IP) {
	ipStr := clientIP.String()
	rl.ipMu.RLock()
	limit, exists := rl.ipLimits[ipStr]
	rl.ipMu.RUnlock()

	if exists {
		current := atomic.LoadInt64(&limit.Connections)
		if current > 0 {
			atomic.AddInt64(&limit.Connections, -1)
		}
	}
}

// CheckUser 检查用户是否允许连接
func (rl *RateLimiter) CheckUser(username string) bool {
	rl.userMu.Lock()
	limit, exists := rl.userLimits[username]
	if !exists {
		limit = &UserLimit{
			MaxConnections: 10, // 默认值
			RateLimit:      5,  // 默认值
			LastReset:      time.Now(),
		}
		rl.userLimits[username] = limit
	}
	rl.userMu.Unlock()

	limit.mu.Lock()
	defer limit.mu.Unlock()

	// 检查连接数限制
	currentConn := atomic.LoadInt64(&limit.Connections)
	if currentConn >= int64(limit.MaxConnections) {
		logrus.Debugf("User %s exceeded max connections (%d)", username, limit.MaxConnections)
		return false
	}

	atomic.AddInt64(&limit.Connections, 1)
	return true
}

// OnUserConnectionEnd 用户连接结束时调用
func (rl *RateLimiter) OnUserConnectionEnd(username string) {
	rl.userMu.RLock()
	limit, exists := rl.userLimits[username]
	rl.userMu.RUnlock()

	if exists {
		current := atomic.LoadInt64(&limit.Connections)
		if current > 0 {
			atomic.AddInt64(&limit.Connections, -1)
		}
	}
}

// CheckGlobal 检查全局是否允许连接
func (rl *RateLimiter) CheckGlobal() bool {
	rl.globalLimit.mu.Lock()
	defer rl.globalLimit.mu.Unlock()

	// 检查全局连接数限制
	currentConn := atomic.LoadInt64(&rl.globalLimit.Connections)
	if rl.globalLimit.MaxConnections > 0 && currentConn >= rl.globalLimit.MaxConnections {
		logrus.Debugf("Global max connections exceeded (%d)", rl.globalLimit.MaxConnections)
		return false
	}

	// 检查全局速率限制
	now := time.Now()
	// 清理1秒前的记录
	var recentTimes []time.Time
	for _, t := range rl.globalLimit.ConnTimes {
		if now.Sub(t) < time.Second {
			recentTimes = append(recentTimes, t)
		}
	}
	rl.globalLimit.ConnTimes = recentTimes

	if len(recentTimes) >= rl.globalLimit.RateLimit {
		logrus.Debugf("Global rate limit exceeded (%d/sec)", rl.globalLimit.RateLimit)
		return false
	}

	// 记录此次连接
	rl.globalLimit.ConnTimes = append(recentTimes, now)
	atomic.AddInt64(&rl.globalLimit.Connections, 1)
	return true
}

// OnGlobalConnectionEnd 全局连接结束时调用
func (rl *RateLimiter) OnGlobalConnectionEnd() {
	current := atomic.LoadInt64(&rl.globalLimit.Connections)
	if current > 0 {
		atomic.AddInt64(&rl.globalLimit.Connections, -1)
	}
}

// SetIPLimit 设置IP限流配置
func (rl *RateLimiter) SetIPLimit(ipStr string, maxConnections, rateLimit int) {
	rl.ipMu.Lock()
	defer rl.ipMu.Unlock()

	limit, exists := rl.ipLimits[ipStr]
	if !exists {
		limit = &IPLimit{
			ConnTimes: make([]time.Time, 0),
		}
		rl.ipLimits[ipStr] = limit
	}

	limit.MaxConnections = maxConnections
	limit.RateLimit = rateLimit
}

// SetUserLimit 设置用户限流配置
func (rl *RateLimiter) SetUserLimit(username string, maxConnections, rateLimit int, bandwidthLimit int64) {
	rl.userMu.Lock()
	defer rl.userMu.Unlock()

	limit, exists := rl.userLimits[username]
	if !exists {
		limit = &UserLimit{
			LastReset: time.Now(),
		}
		rl.userLimits[username] = limit
	}

	limit.MaxConnections = maxConnections
	limit.RateLimit = rateLimit
	limit.BandwidthLimit = bandwidthLimit
}

// GetStats 获取限流统计
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.ipMu.RLock()
	ipCount := len(rl.ipLimits)
	rl.ipMu.RUnlock()

	rl.userMu.RLock()
	userCount := len(rl.userLimits)
	rl.userMu.RUnlock()

	rl.globalLimit.mu.RLock()
	globalConn := atomic.LoadInt64(&rl.globalLimit.Connections)
	rl.globalLimit.mu.RUnlock()

	return map[string]interface{}{
		"ip_limits_count":    ipCount,
		"user_limits_count":  userCount,
		"global_connections":  globalConn,
		"global_max":         rl.globalLimit.MaxConnections,
		"global_rate_limit":  rl.globalLimit.RateLimit,
	}
}



