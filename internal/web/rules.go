package web

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"multiexit-proxy/internal/monitor"
	"multiexit-proxy/internal/proxy"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// generateRandomString 生成随机字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// getRules 获取所有规则
func (s *Server) getRules(w http.ResponseWriter, r *http.Request) {
	// 从代理服务器获取规则引擎
	var rules []*proxy.Rule
	if proxyServer, ok := s.proxyServer.(interface{ GetRuleEngine() *proxy.RuleEngine }); ok {
		if ruleEngine := proxyServer.GetRuleEngine(); ruleEngine != nil {
			rules = ruleEngine.ListRules()
		}
	}

	// 转换规则格式（从后端格式转为前端格式）
	type FrontendRule struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Priority    int      `json:"priority"`
		MatchDomain []string `json:"match_domain,omitempty"`
		MatchIP     []string `json:"match_ip,omitempty"`
		MatchPort   []int    `json:"match_port,omitempty"`
		TargetIP    string   `json:"target_ip,omitempty"`
		Action      string   `json:"action"`
		Enabled     bool     `json:"enabled"`
	}

	frontendRules := make([]FrontendRule, 0, len(rules))
	for _, rule := range rules {
		fr := FrontendRule{
			ID:       rule.ID,
			Name:     rule.Name,
			Priority: rule.Priority,
			Action:   rule.Action,
			Enabled:  rule.Enabled,
		}

		// 根据规则类型转换匹配条件
		switch rule.Type {
		case "domain":
			fr.MatchDomain = []string{rule.Pattern}
		case "ip":
			fr.MatchIP = []string{rule.Pattern}
		case "cidr":
			fr.MatchIP = []string{rule.Pattern}
		case "regex":
			// 正则表达式规则可能需要特殊处理
			fr.MatchDomain = []string{rule.Pattern}
		}

		if rule.Action == "use_ip" || rule.Action == "redirect" {
			fr.TargetIP = rule.TargetIP
		}

		frontendRules = append(frontendRules, fr)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"rules": frontendRules,
		"count": len(frontendRules),
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// addRule 添加规则
func (s *Server) addRule(w http.ResponseWriter, r *http.Request) {
	// 前端发送的规则格式
	type FrontendRule struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Priority    int      `json:"priority"`
		MatchDomain []string `json:"match_domain,omitempty"`
		MatchIP     []string `json:"match_ip,omitempty"`
		MatchPort   []int    `json:"match_port,omitempty"`
		TargetIP    string   `json:"target_ip,omitempty"`
		Action      string   `json:"action"`
		Enabled     bool     `json:"enabled"`
	}

	var frontendRule FrontendRule
	if err := json.NewDecoder(r.Body).Decode(&frontendRule); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 验证规则名称
	if frontendRule.Name == "" {
		http.Error(w, "rule name is required", http.StatusBadRequest)
		return
	}

	// 生成规则ID（如果不存在）
	ruleID := frontendRule.ID
	if ruleID == "" {
		// 使用时间戳和随机数生成唯一ID
		ruleID = fmt.Sprintf("rule-%d-%s", time.Now().UnixNano(), generateRandomString(8))
	}

	// 转换前端格式为后端格式
	rule := &proxy.Rule{
		ID:       ruleID,
		Name:     frontendRule.Name,
		Priority: frontendRule.Priority,
		Action:   frontendRule.Action,
		Enabled:  frontendRule.Enabled,
	}

	// 根据匹配条件确定规则类型
	if len(frontendRule.MatchDomain) > 0 {
		rule.Type = "domain"
		rule.Pattern = frontendRule.MatchDomain[0] // 简化处理，只取第一个
	} else if len(frontendRule.MatchIP) > 0 {
		rule.Type = "ip"
		rule.Pattern = frontendRule.MatchIP[0] // 简化处理，只取第一个
	} else {
		http.Error(w, "rule must have at least one match condition", http.StatusBadRequest)
		return
	}

	if frontendRule.Action == "use_ip" || frontendRule.Action == "redirect" {
		if frontendRule.TargetIP == "" {
			http.Error(w, "target_ip is required for use_ip or redirect action", http.StatusBadRequest)
			return
		}
		rule.TargetIP = frontendRule.TargetIP
	}

	// 添加到规则引擎
	if proxyServer, ok := s.proxyServer.(interface{ GetRuleEngine() *proxy.RuleEngine }); ok {
		if ruleEngine := proxyServer.GetRuleEngine(); ruleEngine != nil {
			if err := ruleEngine.AddRule(rule); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		} else {
			http.Error(w, "rule engine not enabled", http.StatusServiceUnavailable)
			return
		}
	} else {
		http.Error(w, "rule engine not available", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"rule":   frontendRule,
	}); err != nil {
		logrus.Errorf("Failed to encode add rule response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// updateRule 更新规则
func (s *Server) updateRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	// 前端发送的规则格式
	type FrontendRule struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Priority    int      `json:"priority"`
		MatchDomain []string `json:"match_domain,omitempty"`
		MatchIP     []string `json:"match_ip,omitempty"`
		MatchPort   []int    `json:"match_port,omitempty"`
		TargetIP    string   `json:"target_ip,omitempty"`
		Action      string   `json:"action"`
		Enabled     bool     `json:"enabled"`
	}

	var frontendRule FrontendRule
	if err := json.NewDecoder(r.Body).Decode(&frontendRule); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 转换前端格式为后端格式
	rule := &proxy.Rule{
		ID:       ruleID,
		Name:     frontendRule.Name,
		Priority: frontendRule.Priority,
		Action:   frontendRule.Action,
		Enabled:  frontendRule.Enabled,
	}

	// 根据匹配条件确定规则类型
	if len(frontendRule.MatchDomain) > 0 {
		rule.Type = "domain"
		rule.Pattern = frontendRule.MatchDomain[0]
	} else if len(frontendRule.MatchIP) > 0 {
		rule.Type = "ip"
		rule.Pattern = frontendRule.MatchIP[0]
	} else {
		http.Error(w, "rule must have at least one match condition", http.StatusBadRequest)
		return
	}

	if frontendRule.Action == "use_ip" || frontendRule.Action == "redirect" {
		if frontendRule.TargetIP == "" {
			http.Error(w, "target_ip is required for use_ip or redirect action", http.StatusBadRequest)
			return
		}
		rule.TargetIP = frontendRule.TargetIP
	}

	// 更新规则引擎
	if proxyServer, ok := s.proxyServer.(interface{ GetRuleEngine() *proxy.RuleEngine }); ok {
		if ruleEngine := proxyServer.GetRuleEngine(); ruleEngine != nil {
			if err := ruleEngine.UpdateRule(ruleID, rule); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		} else {
			http.Error(w, "rule engine not enabled", http.StatusServiceUnavailable)
			return
		}
	} else {
		http.Error(w, "rule engine not available", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"rule":   frontendRule,
	}); err != nil {
		logrus.Errorf("Failed to encode update rule response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// deleteRule 删除规则
func (s *Server) deleteRule(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["id"]

	// 从规则引擎删除
	if proxyServer, ok := s.proxyServer.(interface{ GetRuleEngine() *proxy.RuleEngine }); ok {
		if ruleEngine := proxyServer.GetRuleEngine(); ruleEngine != nil {
			if err := ruleEngine.RemoveRule(ruleID); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		} else {
			http.Error(w, "rule engine not enabled", http.StatusServiceUnavailable)
			return
		}
	} else {
		http.Error(w, "rule engine not available", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"id":     ruleID,
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// getTrafficAnalysis 获取流量分析数据
func (s *Server) getTrafficAnalysis(w http.ResponseWriter, r *http.Request) {
	if proxyServer, ok := s.proxyServer.(interface{ GetTrafficAnalyzer() *monitor.TrafficAnalyzer }); ok {
		if analyzer := proxyServer.GetTrafficAnalyzer(); analyzer != nil {
			// 解析时间范围参数
			timeRange := r.URL.Query().Get("range")
			var since time.Time
			switch timeRange {
			case "1h":
				since = time.Now().Add(-1 * time.Hour)
			case "6h":
				since = time.Now().Add(-6 * time.Hour)
			case "24h":
				since = time.Now().Add(-24 * time.Hour)
			case "7d":
				since = time.Now().Add(-7 * 24 * time.Hour)
			case "30d":
				since = time.Now().Add(-30 * 24 * time.Hour)
			default:
				since = time.Now().Add(-24 * time.Hour) // 默认24小时
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"domain_stats": analyzer.GetAllDomainStats(),
				"trends":       analyzer.GetTrends(since),
				"anomalies":    analyzer.GetAnomalies(since),
			}); err != nil {
				logrus.Errorf("Failed to encode traffic analysis: %v", err)
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}
			return
		}
	}

	http.Error(w, "Traffic analysis not enabled", http.StatusServiceUnavailable)
}

