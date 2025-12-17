package monitor

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// DomainStats 域名统计
type DomainStats struct {
	Domain          string
	Connections     int64
	BytesUp         int64
	BytesDown       int64
	TotalBytes      int64
	AvgLatency      time.Duration
	LastAccess      time.Time
	AccessTimes     []time.Time
	mu              sync.RWMutex
}

// TrafficTrend 流量趋势
type TrafficTrend struct {
	Timestamp   time.Time
	BytesUp     int64
	BytesDown   int64
	Connections int64
}

// AnomalyDetection 异常检测
type AnomalyDetection struct {
	Domain          string
	AnomalyType    string // "traffic_spike", "connection_anomaly", "latency_anomaly"
	Severity        string // "low", "medium", "high"
	DetectedAt      time.Time
	Value           float64
	ExpectedValue   float64
	Description     string
}

// TrafficAnalyzer 流量分析器
type TrafficAnalyzer struct {
	domainStats      map[string]*DomainStats
	trends           []TrafficTrend
	anomalies        []AnomalyDetection
	mu               sync.RWMutex
	trendWindow      time.Duration
	anomalyThreshold float64 // 异常阈值（倍数，如2.0表示2倍）
}

// NewTrafficAnalyzer 创建流量分析器
func NewTrafficAnalyzer(trendWindow time.Duration, anomalyThreshold float64) *TrafficAnalyzer {
	if trendWindow == 0 {
		trendWindow = 1 * time.Hour
	}
	if anomalyThreshold == 0 {
		anomalyThreshold = 2.0
	}

	return &TrafficAnalyzer{
		domainStats:      make(map[string]*DomainStats),
		trends:           make([]TrafficTrend, 0),
		anomalies:        make([]AnomalyDetection, 0),
		trendWindow:      trendWindow,
		anomalyThreshold: anomalyThreshold,
	}
}

// RecordDomainAccess 记录域名访问
func (ta *TrafficAnalyzer) RecordDomainAccess(domain string, bytesUp, bytesDown int64, latency time.Duration) {
	ta.mu.Lock()
	stat, ok := ta.domainStats[domain]
	if !ok {
		stat = &DomainStats{
			Domain:      domain,
			AccessTimes: make([]time.Time, 0),
		}
		ta.domainStats[domain] = stat
	}
	ta.mu.Unlock()

	stat.mu.Lock()
	atomic.AddInt64(&stat.Connections, 1)
	atomic.AddInt64(&stat.BytesUp, bytesUp)
	atomic.AddInt64(&stat.BytesDown, bytesDown)
	atomic.AddInt64(&stat.TotalBytes, bytesUp+bytesDown)
	stat.LastAccess = time.Now()
	stat.AccessTimes = append(stat.AccessTimes, time.Now())
	
	// 限制访问时间记录数量
	if len(stat.AccessTimes) > 1000 {
		stat.AccessTimes = stat.AccessTimes[1:]
	}
	
	// 更新平均延迟
	if latency > 0 {
		if stat.AvgLatency == 0 {
			stat.AvgLatency = latency
		} else {
			stat.AvgLatency = (stat.AvgLatency + latency) / 2
		}
	}
	stat.mu.Unlock()

	// 检测异常
	ta.detectAnomalies(domain, stat)
}

// detectAnomalies 检测异常
func (ta *TrafficAnalyzer) detectAnomalies(domain string, stat *DomainStats) {
	stat.mu.RLock()
	defer stat.mu.RUnlock()

	now := time.Now()
	
	// 检测流量突增
	if len(stat.AccessTimes) >= 10 {
		// 计算最近1分钟和之前1分钟的访问次数
		recentCount := 0
		previousCount := 0
		cutoff := now.Add(-2 * time.Minute)
		
		for _, t := range stat.AccessTimes {
			if t.After(cutoff) && t.Before(now.Add(-1*time.Minute)) {
				previousCount++
			} else if t.After(now.Add(-1 * time.Minute)) {
				recentCount++
			}
		}
		
		if previousCount > 0 && float64(recentCount)/float64(previousCount) > ta.anomalyThreshold {
			ta.mu.Lock()
			ta.anomalies = append(ta.anomalies, AnomalyDetection{
				Domain:        domain,
				AnomalyType:   "traffic_spike",
				Severity:      "medium",
				DetectedAt:    now,
				Value:         float64(recentCount),
				ExpectedValue: float64(previousCount),
				Description:   fmt.Sprintf("Traffic spike detected: %d requests in last minute vs %d in previous minute", recentCount, previousCount),
			})
			// 限制异常记录数量
			if len(ta.anomalies) > 100 {
				ta.anomalies = ta.anomalies[1:]
			}
			ta.mu.Unlock()
			
			logrus.Warnf("Anomaly detected for domain %s: traffic spike", domain)
		}
	}
}

// GetDomainStats 获取域名统计
func (ta *TrafficAnalyzer) GetDomainStats(domain string) *DomainStats {
	ta.mu.RLock()
	defer ta.mu.RUnlock()
	
	stat, ok := ta.domainStats[domain]
	if !ok {
		return nil
	}
	
	stat.mu.RLock()
	defer stat.mu.RUnlock()
	
	return &DomainStats{
		Domain:      stat.Domain,
		Connections: atomic.LoadInt64(&stat.Connections),
		BytesUp:     atomic.LoadInt64(&stat.BytesUp),
		BytesDown:   atomic.LoadInt64(&stat.BytesDown),
		TotalBytes:  atomic.LoadInt64(&stat.TotalBytes),
		AvgLatency:  stat.AvgLatency,
		LastAccess:  stat.LastAccess,
		AccessTimes: append([]time.Time{}, stat.AccessTimes...),
	}
}

// GetAllDomainStats 获取所有域名统计
func (ta *TrafficAnalyzer) GetAllDomainStats() map[string]*DomainStats {
	ta.mu.RLock()
	defer ta.mu.RUnlock()
	
	result := make(map[string]*DomainStats)
	for domain, stat := range ta.domainStats {
		stat.mu.RLock()
		result[domain] = &DomainStats{
			Domain:      stat.Domain,
			Connections: atomic.LoadInt64(&stat.Connections),
			BytesUp:     atomic.LoadInt64(&stat.BytesUp),
			BytesDown:   atomic.LoadInt64(&stat.BytesDown),
			TotalBytes:  atomic.LoadInt64(&stat.TotalBytes),
			AvgLatency:  stat.AvgLatency,
			LastAccess:  stat.LastAccess,
		}
		stat.mu.RUnlock()
	}
	
	return result
}

// GetTrends 获取流量趋势
func (ta *TrafficAnalyzer) GetTrends(since time.Time) []TrafficTrend {
	ta.mu.RLock()
	defer ta.mu.RUnlock()
	
	var filtered []TrafficTrend
	for _, trend := range ta.trends {
		if trend.Timestamp.After(since) {
			filtered = append(filtered, trend)
		}
	}
	
	return filtered
}

// RecordTrend 记录趋势数据点
func (ta *TrafficAnalyzer) RecordTrend() {
	ta.mu.Lock()
	defer ta.mu.Unlock()
	
	var totalBytesUp, totalBytesDown int64
	var totalConnections int64
	
	for _, stat := range ta.domainStats {
		totalBytesUp += atomic.LoadInt64(&stat.BytesUp)
		totalBytesDown += atomic.LoadInt64(&stat.BytesDown)
		totalConnections += atomic.LoadInt64(&stat.Connections)
	}
	
	trend := TrafficTrend{
		Timestamp:   time.Now(),
		BytesUp:     totalBytesUp,
		BytesDown:   totalBytesDown,
		Connections: totalConnections,
	}
	
	ta.trends = append(ta.trends, trend)
	
	// 限制趋势记录数量（保留最近N个）
	maxTrends := int(ta.trendWindow / (5 * time.Minute)) // 每5分钟一个数据点
	if len(ta.trends) > maxTrends {
		ta.trends = ta.trends[len(ta.trends)-maxTrends:]
	}
}

// GetAnomalies 获取异常检测结果
func (ta *TrafficAnalyzer) GetAnomalies(since time.Time) []AnomalyDetection {
	ta.mu.RLock()
	defer ta.mu.RUnlock()
	
	var filtered []AnomalyDetection
	for _, anomaly := range ta.anomalies {
		if anomaly.DetectedAt.After(since) {
			filtered = append(filtered, anomaly)
		}
	}
	
	return filtered
}

// StartTrendRecording 启动趋势记录（定期记录）
func (ta *TrafficAnalyzer) StartTrendRecording(interval time.Duration) {
	if interval == 0 {
		interval = 5 * time.Minute
	}
	
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			ta.RecordTrend()
		}
	}()
}

