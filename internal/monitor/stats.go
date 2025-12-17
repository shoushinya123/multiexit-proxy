package monitor

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// ConnectionStats 连接统计
type ConnectionStats struct {
	TotalConnections    int64
	ActiveConnections   int64
	BytesTransferred    int64
	BytesUp             int64
	BytesDown           int64
	IPStats             map[string]*IPConnectionStats
	mu                  sync.RWMutex
}

// IPConnectionStats IP连接统计
type IPConnectionStats struct {
	Connections     int64
	ActiveConn      int64
	BytesUp         int64
	BytesDown       int64
	TotalBytes      int64
	AvgLatency      time.Duration
	LastUsed        time.Time
	ConnectionTimes []time.Duration
	mu              sync.RWMutex
}

// StatsManager 统计管理器
type StatsManager struct {
	stats *ConnectionStats
}

// NewStatsManager 创建统计管理器
func NewStatsManager() *StatsManager {
	return &StatsManager{
		stats: &ConnectionStats{
			IPStats: make(map[string]*IPConnectionStats),
		},
	}
}

// OnConnectionStart 连接开始
func (s *StatsManager) OnConnectionStart(ip net.IP) {
	ipStr := ip.String()
	atomic.AddInt64(&s.stats.TotalConnections, 1)
	atomic.AddInt64(&s.stats.ActiveConnections, 1)
	
	s.stats.mu.Lock()
	stat, ok := s.stats.IPStats[ipStr]
	if !ok {
		stat = &IPConnectionStats{
			ConnectionTimes: make([]time.Duration, 0, 100),
		}
		s.stats.IPStats[ipStr] = stat
	}
	s.stats.mu.Unlock()
	
	atomic.AddInt64(&stat.Connections, 1)
	atomic.AddInt64(&stat.ActiveConn, 1)
	stat.mu.Lock()
	stat.LastUsed = time.Now()
	stat.mu.Unlock()
}

// OnConnectionEnd 连接结束
func (s *StatsManager) OnConnectionEnd(ip net.IP, duration time.Duration) {
	atomic.AddInt64(&s.stats.ActiveConnections, -1)
	
	ipStr := ip.String()
	s.stats.mu.RLock()
	stat, ok := s.stats.IPStats[ipStr]
	s.stats.mu.RUnlock()
	
	if !ok {
		return
	}
	
	atomic.AddInt64(&stat.ActiveConn, -1)
	stat.mu.Lock()
	stat.ConnectionTimes = append(stat.ConnectionTimes, duration)
	if len(stat.ConnectionTimes) > 100 {
		stat.ConnectionTimes = stat.ConnectionTimes[1:]
	}
	// 计算平均延迟
	if len(stat.ConnectionTimes) > 0 {
		var total time.Duration
		for _, d := range stat.ConnectionTimes {
			total += d
		}
		stat.AvgLatency = total / time.Duration(len(stat.ConnectionTimes))
	}
	stat.mu.Unlock()
}

// OnBytesTransferred 数据传输
func (s *StatsManager) OnBytesTransferred(ip net.IP, up, down int64) {
	atomic.AddInt64(&s.stats.BytesTransferred, up+down)
	atomic.AddInt64(&s.stats.BytesUp, up)
	atomic.AddInt64(&s.stats.BytesDown, down)
	
	ipStr := ip.String()
	s.stats.mu.RLock()
	stat, ok := s.stats.IPStats[ipStr]
	s.stats.mu.RUnlock()
	
	if !ok {
		return
	}
	
	atomic.AddInt64(&stat.BytesUp, up)
	atomic.AddInt64(&stat.BytesDown, down)
	atomic.AddInt64(&stat.TotalBytes, up+down)
}

// GetStats 获取统计信息
func (s *StatsManager) GetStats() *ConnectionStats {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()
	
	// 创建副本避免并发问题
	ipStats := make(map[string]*IPConnectionStats)
	for k, v := range s.stats.IPStats {
		v.mu.RLock()
		ipStats[k] = &IPConnectionStats{
			Connections:     atomic.LoadInt64(&v.Connections),
			ActiveConn:      atomic.LoadInt64(&v.ActiveConn),
			BytesUp:         atomic.LoadInt64(&v.BytesUp),
			BytesDown:       atomic.LoadInt64(&v.BytesDown),
			TotalBytes:      atomic.LoadInt64(&v.TotalBytes),
			AvgLatency:      v.AvgLatency,
			LastUsed:        v.LastUsed,
			ConnectionTimes: append([]time.Duration{}, v.ConnectionTimes...),
		}
		v.mu.RUnlock()
	}
	
	return &ConnectionStats{
		TotalConnections:  atomic.LoadInt64(&s.stats.TotalConnections),
		ActiveConnections: atomic.LoadInt64(&s.stats.ActiveConnections),
		BytesTransferred:  atomic.LoadInt64(&s.stats.BytesTransferred),
		BytesUp:           atomic.LoadInt64(&s.stats.BytesUp),
		BytesDown:         atomic.LoadInt64(&s.stats.BytesDown),
		IPStats:           ipStats,
	}
}

// GetIPStats 获取特定IP的统计
func (s *StatsManager) GetIPStats(ip net.IP) *IPConnectionStats {
	ipStr := ip.String()
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()
	
	stat, ok := s.stats.IPStats[ipStr]
	if !ok {
		return nil
	}
	
	stat.mu.RLock()
	defer stat.mu.RUnlock()
	
	return &IPConnectionStats{
		Connections:     atomic.LoadInt64(&stat.Connections),
		ActiveConn:      atomic.LoadInt64(&stat.ActiveConn),
		BytesUp:         atomic.LoadInt64(&stat.BytesUp),
		BytesDown:       atomic.LoadInt64(&stat.BytesDown),
		TotalBytes:      atomic.LoadInt64(&stat.TotalBytes),
		AvgLatency:      stat.AvgLatency,
		LastUsed:        stat.LastUsed,
		ConnectionTimes: append([]time.Duration{}, stat.ConnectionTimes...),
	}
}

// Reset 重置统计
func (s *StatsManager) Reset() {
	atomic.StoreInt64(&s.stats.TotalConnections, 0)
	atomic.StoreInt64(&s.stats.ActiveConnections, 0)
	atomic.StoreInt64(&s.stats.BytesTransferred, 0)
	atomic.StoreInt64(&s.stats.BytesUp, 0)
	atomic.StoreInt64(&s.stats.BytesDown, 0)
	
	s.stats.mu.Lock()
	defer s.stats.mu.Unlock()
	
	for k := range s.stats.IPStats {
		delete(s.stats.IPStats, k)
	}
}

