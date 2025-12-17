package web

import (
	"encoding/json"
	"net/http"
	"time"

	"multiexit-proxy/internal/monitor"
	"multiexit-proxy/internal/proxy"

	"github.com/gorilla/mux"
)

// getRules 获取所有规则
func (s *Server) getRules(w http.ResponseWriter, r *http.Request) {
	// 这里需要从proxyServer获取ruleEngine
	// 暂时返回空列表，需要添加接口
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"rules": []interface{}{},
		"count": 0,
	})
}

// addRule 添加规则
func (s *Server) addRule(w http.ResponseWriter, r *http.Request) {
	var rule proxy.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: 实际添加到ruleEngine
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"rule":   rule,
	})
}

// updateRule 更新规则
func (s *Server) updateRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	var rule proxy.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rule.ID = ruleID

	// TODO: 实际更新ruleEngine
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"rule":   rule,
	})
}

// deleteRule 删除规则
func (s *Server) deleteRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	// TODO: 实际从ruleEngine删除
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"id":     ruleID,
	})
}

// getTrafficAnalysis 获取流量分析数据
func (s *Server) getTrafficAnalysis(w http.ResponseWriter, r *http.Request) {
	if proxyServer, ok := s.proxyServer.(interface{ GetTrafficAnalyzer() *monitor.TrafficAnalyzer }); ok {
		if analyzer := proxyServer.GetTrafficAnalyzer(); analyzer != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"domain_stats": analyzer.GetAllDomainStats(),
				"trends":       analyzer.GetTrends(time.Now().Add(-24 * time.Hour)),
				"anomalies":    analyzer.GetAnomalies(time.Now().Add(-24 * time.Hour)),
			})
			return
		}
	}

	http.Error(w, "Traffic analysis not enabled", http.StatusServiceUnavailable)
}

