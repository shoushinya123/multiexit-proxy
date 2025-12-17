package web

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"time"

	"multiexit-proxy/internal/database"

	"github.com/sirupsen/logrus"
)

// getHistoryStats 获取历史统计数据
func (s *Server) getHistoryStats(w http.ResponseWriter, r *http.Request) {
	if s.statsRepo == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	// 解析查询参数
	ip := r.URL.Query().Get("ip")
	limitStr := r.URL.Query().Get("limit")
	hoursStr := r.URL.Query().Get("hours")

	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			// 限制最大值为1000，防止过大查询
			if l > 1000 {
				l = 1000
			}
			limit = l
		}
	}

	hours := 24
	if hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 {
			// 限制最大值为720小时（30天），防止过大查询
			if h > 720 {
				h = 720
			}
			hours = h
		}
	}

	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	var response interface{}

	if ip != "" {
		// 获取特定IP的历史
		ipAddr := net.ParseIP(ip)
		if ipAddr == nil {
			http.Error(w, "Invalid IP address", http.StatusBadRequest)
			return
		}
		history, err := s.statsRepo.GetConnectionHistory(ipAddr, since, limit)
		if err != nil {
			logrus.Errorf("Failed to get connection history: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response = map[string]interface{}{
			"ip":      ip,
			"since":   since,
			"history": history,
		}
	} else {
		// 获取Top IPs
		topIPs, err := s.statsRepo.GetTopIPsByTraffic(limit)
		if err != nil {
			logrus.Errorf("Failed to get top IPs: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response = map[string]interface{}{
			"top_ips": topIPs,
			"limit":   limit,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logrus.Errorf("Failed to encode history stats response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// getHistoryTraffic 获取历史流量数据
func (s *Server) getHistoryTraffic(w http.ResponseWriter, r *http.Request) {
	if s.trafficRepo == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	// 解析查询参数
	domain := r.URL.Query().Get("domain")
	limitStr := r.URL.Query().Get("limit")
	hoursStr := r.URL.Query().Get("hours")
	rangeStr := r.URL.Query().Get("range") // "1h", "24h", "7d", "30d"

	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			// 限制最大值为1000，防止过大查询
			if l > 1000 {
				l = 1000
			}
			limit = l
		}
	}

	// 根据range参数计算时间范围
	var since time.Time
	if rangeStr != "" {
		switch rangeStr {
		case "1h":
			since = time.Now().Add(-1 * time.Hour)
		case "24h", "1d":
			since = time.Now().Add(-24 * time.Hour)
		case "7d":
			since = time.Now().Add(-7 * 24 * time.Hour)
		case "30d":
			since = time.Now().Add(-30 * 24 * time.Hour)
		default:
			since = time.Now().Add(-24 * time.Hour)
		}
	} else {
		hours := 24
		if hoursStr != "" {
			if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 {
				hours = h
			}
		}
		since = time.Now().Add(-time.Duration(hours) * time.Hour)
	}

	var response interface{}

	if domain != "" {
		// 获取特定域名的访问历史
		history, err := s.trafficRepo.GetDomainAccessHistory(domain, since, limit)
		if err != nil {
			logrus.Errorf("Failed to get domain access history: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response = map[string]interface{}{
			"domain":  domain,
			"since":   since,
			"history": history,
		}
	} else {
		// 获取流量趋势
		trends, err := s.trafficRepo.GetTrafficTrends(since, limit)
		if err != nil {
			logrus.Errorf("Failed to get traffic trends: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 获取Top域名
		topDomains, err := s.trafficRepo.GetTopDomainsByTraffic(limit)
		if err != nil {
			logrus.Errorf("Failed to get top domains: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 获取最近的异常
		anomalies, err := s.trafficRepo.GetRecentAnomalies(since, 50)
		if err != nil {
			logrus.Errorf("Failed to get recent anomalies: %v", err)
			// 不返回错误，只是不包含异常数据
			anomalies = []database.AnomalyRow{}
		}

		response = map[string]interface{}{
			"trends":       trends,
			"top_domains":  topDomains,
			"anomalies":    anomalies,
			"since":        since,
			"range":        rangeStr,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logrus.Errorf("Failed to encode history stats response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// getHistoryAnomalies 获取历史异常记录
func (s *Server) getHistoryAnomalies(w http.ResponseWriter, r *http.Request) {
	if s.trafficRepo == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	// 解析查询参数
	limitStr := r.URL.Query().Get("limit")
	hoursStr := r.URL.Query().Get("hours")
	severity := r.URL.Query().Get("severity") // "low", "medium", "high"

	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			// 限制最大值为1000，防止过大查询
			if l > 1000 {
				l = 1000
			}
			limit = l
		}
	}

	hours := 24
	if hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 {
			// 限制最大值为720小时（30天），防止过大查询
			if h > 720 {
				h = 720
			}
			hours = h
		}
	}

	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	anomalies, err := s.trafficRepo.GetRecentAnomalies(since, limit)
	if err != nil {
		logrus.Errorf("Failed to get anomalies: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 如果指定了严重程度，进行过滤
	if severity != "" {
		filtered := []database.AnomalyRow{}
		for _, a := range anomalies {
			if a.Severity == severity {
				filtered = append(filtered, a)
			}
		}
		anomalies = filtered
	}

	response := map[string]interface{}{
		"anomalies": anomalies,
		"since":     since,
		"severity":  severity,
		"count":     len(anomalies),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logrus.Errorf("Failed to encode history stats response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

