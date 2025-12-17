package monitor

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/sirupsen/logrus"
)

// PrometheusExporter Prometheus指标导出器
type PrometheusExporter struct {
	statsManager *StatsManager
	mu           sync.RWMutex
}

// NewPrometheusExporter 创建Prometheus导出器
func NewPrometheusExporter(statsManager *StatsManager) *PrometheusExporter {
	return &PrometheusExporter{
		statsManager: statsManager,
	}
}

// ServeHTTP 处理Prometheus指标请求
func (e *PrometheusExporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := e.statsManager.GetStats()
	e.mu.RLock()
	defer e.mu.RUnlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	// 导出连接统计
	fmt.Fprintf(w, "# HELP multiexit_proxy_total_connections Total number of connections\n")
	fmt.Fprintf(w, "# TYPE multiexit_proxy_total_connections counter\n")
	fmt.Fprintf(w, "multiexit_proxy_total_connections %d\n\n", stats.TotalConnections)

	fmt.Fprintf(w, "# HELP multiexit_proxy_active_connections Current active connections\n")
	fmt.Fprintf(w, "# TYPE multiexit_proxy_active_connections gauge\n")
	fmt.Fprintf(w, "multiexit_proxy_active_connections %d\n\n", stats.ActiveConnections)

	// 导出流量统计
	fmt.Fprintf(w, "# HELP multiexit_proxy_bytes_up Total bytes uploaded\n")
	fmt.Fprintf(w, "# TYPE multiexit_proxy_bytes_up counter\n")
	fmt.Fprintf(w, "multiexit_proxy_bytes_up %d\n\n", stats.BytesUp)

	fmt.Fprintf(w, "# HELP multiexit_proxy_bytes_down Total bytes downloaded\n")
	fmt.Fprintf(w, "# TYPE multiexit_proxy_bytes_down counter\n")
	fmt.Fprintf(w, "multiexit_proxy_bytes_down %d\n\n", stats.BytesDown)

	fmt.Fprintf(w, "# HELP multiexit_proxy_bytes_transferred Total bytes transferred\n")
	fmt.Fprintf(w, "# TYPE multiexit_proxy_bytes_transferred counter\n")
	fmt.Fprintf(w, "multiexit_proxy_bytes_transferred %d\n\n", stats.BytesTransferred)

	// 导出按IP的统计
	stats.mu.RLock()
	for ipStr, ipStat := range stats.IPStats {
		ipLabel := strconv.Quote(ipStr)
		
		fmt.Fprintf(w, "# HELP multiexit_proxy_ip_connections Connections per IP\n")
		fmt.Fprintf(w, "# TYPE multiexit_proxy_ip_connections counter\n")
		fmt.Fprintf(w, "multiexit_proxy_ip_connections{ip=%s} %d\n", ipLabel, ipStat.Connections)
		
		fmt.Fprintf(w, "# HELP multiexit_proxy_ip_active_connections Active connections per IP\n")
		fmt.Fprintf(w, "# TYPE multiexit_proxy_ip_active_connections gauge\n")
		fmt.Fprintf(w, "multiexit_proxy_ip_active_connections{ip=%s} %d\n", ipLabel, ipStat.ActiveConn)
		
		fmt.Fprintf(w, "# HELP multiexit_proxy_ip_bytes_up Bytes uploaded per IP\n")
		fmt.Fprintf(w, "# TYPE multiexit_proxy_ip_bytes_up counter\n")
		fmt.Fprintf(w, "multiexit_proxy_ip_bytes_up{ip=%s} %d\n", ipLabel, ipStat.BytesUp)
		
		fmt.Fprintf(w, "# HELP multiexit_proxy_ip_bytes_down Bytes downloaded per IP\n")
		fmt.Fprintf(w, "# TYPE multiexit_proxy_ip_bytes_down counter\n")
		fmt.Fprintf(w, "multiexit_proxy_ip_bytes_down{ip=%s} %d\n", ipLabel, ipStat.BytesDown)
		
		if ipStat.AvgLatency > 0 {
			fmt.Fprintf(w, "# HELP multiexit_proxy_ip_avg_latency Average latency per IP in seconds\n")
			fmt.Fprintf(w, "# TYPE multiexit_proxy_ip_avg_latency gauge\n")
			fmt.Fprintf(w, "multiexit_proxy_ip_avg_latency{ip=%s} %.3f\n", ipLabel, ipStat.AvgLatency.Seconds())
		}
	}
	stats.mu.RUnlock()
}

// RegisterMetrics 注册Prometheus指标端点
func RegisterMetrics(mux *http.ServeMux, statsManager *StatsManager) {
	exporter := NewPrometheusExporter(statsManager)
	mux.Handle("/metrics", exporter)
	logrus.Info("Prometheus metrics endpoint registered at /metrics")
}

