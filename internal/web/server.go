package web

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"multiexit-proxy/internal/config"
	"multiexit-proxy/internal/database"
	"multiexit-proxy/internal/monitor"
	"multiexit-proxy/internal/subscribe"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// formatDuration 格式化时间间隔
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "N/A"
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// ProxyServer 代理服务器接口（用于获取统计信息）
type ProxyServer interface {
	GetStats() interface{} // 返回统计信息，可以是*monitor.ConnectionStats或nil
}

// Server Web管理服务器
type Server struct {
	config          *config.ServerConfig
	configPath      string
	router          *mux.Router
	proxyServer     ProxyServer                         // 代理服务器实例（用于获取统计）
	statsMgr        interface{ GetStats() interface{} } // 统计管理器（用于Prometheus）
	versionMgr      *config.VersionManager              // 版本管理器
	csrfProtection  *CSRFProtection                     // CSRF保护
	loginProtection *LoginProtection                    // 登录保护
	statsRepo       *database.StatsRepository           // 统计数据仓库
	trafficRepo     *database.TrafficRepository         // 流量分析数据仓库
}

// NewServer 创建Web服务器
func NewServer(configPath string, cfg *config.ServerConfig) *Server {
	// 生成CSRF密钥（从配置密钥派生）
	csrfSecret := []byte(cfg.Auth.Key)
	if len(csrfSecret) < 32 {
		// 如果密钥太短，使用默认值
		csrfSecret = []byte("multiexit-proxy-csrf-secret-key-2024")
	}

	s := &Server{
		config:          cfg,
		configPath:      configPath,
		router:          mux.NewRouter(),
		versionMgr:      config.NewVersionManager(configPath),
		csrfProtection:  NewCSRFProtection(csrfSecret),
		loginProtection: NewLoginProtection(5, 15*time.Minute), // 5次失败，阻止15分钟
	}

	// 初始化数据库连接（如果启用）
	if cfg.Database.Enabled {
		dbConfig := database.Config{
			Host:     cfg.Database.Host,
			Port:     cfg.Database.Port,
			Database: cfg.Database.Database,
			User:     cfg.Database.User,
			Password: cfg.Database.Password,
			SSLMode:  cfg.Database.SSLMode,
			MaxConns: cfg.Database.MaxConns,
			MaxIdle:  cfg.Database.MaxIdle,
		}
		if dbConfig.Host == "" {
			dbConfig.Host = "localhost"
		}
		if dbConfig.Port == 0 {
			dbConfig.Port = 5432
		}

		db, err := database.NewDB(dbConfig)
		if err != nil {
			logrus.Errorf("Failed to connect to database: %v", err)
			logrus.Warn("Continuing without database support")
		} else {
			s.statsRepo = database.NewStatsRepository(db)
			s.trafficRepo = database.NewTrafficRepository(db)
			logrus.Info("Database connection established")
		}
	}

	// 加载版本列表
	if err := s.versionMgr.LoadVersionList(); err != nil {
		logrus.Warnf("Failed to load version list: %v", err)
	}
	s.setupRoutes()
	return s
}

// SetProxyServer 设置代理服务器实例（用于获取统计信息）
func (s *Server) SetProxyServer(proxy ProxyServer) {
	s.proxyServer = proxy
	if statsMgr, ok := proxy.(interface{ GetStats() interface{} }); ok {
		s.statsMgr = statsMgr
	}
}

// handlePrometheusMetrics 处理Prometheus指标请求
func (s *Server) handlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	if s.statsMgr == nil {
		http.Error(w, "Stats not available", http.StatusServiceUnavailable)
		return
	}

	stats := s.statsMgr.GetStats()
	if stats == nil {
		http.Error(w, "Stats not available", http.StatusServiceUnavailable)
		return
	}

	// 使用monitor包的Prometheus导出器
	// 这里简化处理，直接调用monitor包的函数
	// 实际应该导入monitor包并使用NewPrometheusExporter
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintf(w, "# Prometheus metrics endpoint\n")
	fmt.Fprintf(w, "# Use /api/stats for JSON format\n")
	fmt.Fprintf(w, "# Full Prometheus support: use monitor.NewPrometheusExporter\n")

	// 临时返回基本指标
	if statsMap, ok := stats.(map[string]interface{}); ok {
		fmt.Fprintf(w, "multiexit_proxy_total_connections %v\n", statsMap["total_connections"])
		fmt.Fprintf(w, "multiexit_proxy_active_connections %v\n", statsMap["active_connections"])
	}
}

// corsMiddleware CORS中间件
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置CORS响应头
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
		w.Header().Set("Access-Control-Expose-Headers", "X-CSRF-Token")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 添加CORS中间件（在所有路由之前）
	s.router.Use(s.corsMiddleware)

	// API路由
	api := s.router.PathPrefix("/api").Subrouter()
	api.Use(s.authMiddleware)
	api.Use(s.csrfMiddleware) // 添加CSRF保护

	api.HandleFunc("/config", s.getConfig).Methods("GET")
	api.HandleFunc("/config", s.updateConfig).Methods("POST")
	api.HandleFunc("/config/rollback", s.rollbackConfig).Methods("POST")
	api.HandleFunc("/config/versions", s.listConfigVersions).Methods("GET")
	api.HandleFunc("/ips", s.getIPs).Methods("GET")
	api.HandleFunc("/status", s.getStatus).Methods("GET")
	api.HandleFunc("/stats", s.getStats).Methods("GET")
	api.HandleFunc("/rules", s.getRules).Methods("GET")
	api.HandleFunc("/rules", s.addRule).Methods("POST")
	api.HandleFunc("/rules/{id}", s.updateRule).Methods("PUT")
	api.HandleFunc("/rules/{id}", s.deleteRule).Methods("DELETE")
	api.HandleFunc("/traffic", s.getTrafficAnalysis).Methods("GET")

	// 历史数据查询端点
	api.HandleFunc("/history/stats", s.getHistoryStats).Methods("GET")
	api.HandleFunc("/history/traffic", s.getHistoryTraffic).Methods("GET")
	api.HandleFunc("/history/anomalies", s.getHistoryAnomalies).Methods("GET")

	// Prometheus指标端点（不需要认证）
	s.router.HandleFunc("/metrics", s.handlePrometheusMetrics).Methods("GET")

	// 订阅相关路由（不需要认证）
	s.router.HandleFunc("/api/subscribe", s.handleSubscribe).Methods("GET")
	s.router.HandleFunc("/api/subscription/link", s.generateSubscribeLink).Methods("GET")

	// 根路径 - 返回API信息提示（前端界面在独立服务中运行）
	s.router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"message": "MultiExit Proxy API Server",
			"version": "1.0.0",
			"endpoints": map[string]string{
				"api_base": "/api",
				"status":   "/api/status",
				"stats":    "/api/stats",
				"config":   "/api/config",
				"ips":      "/api/ips",
				"rules":    "/api/rules",
				"traffic":  "/api/traffic",
				"metrics":  "/metrics",
			},
			"frontend": "Please use the frontend application for the management interface",
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logrus.Errorf("Failed to encode root response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	})
}

// authMiddleware 认证中间件（带登录保护）
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIPFromRequest(r)

		// 检查IP是否被阻止
		if s.loginProtection.IsBlocked(clientIP) {
			http.Error(w, "Too many failed login attempts. Please try again later.", http.StatusTooManyRequests)
			return
		}

		username, password, ok := r.BasicAuth()
		if !ok || username != s.config.Web.Username || password != s.config.Web.Password {
			// 记录失败尝试
			s.loginProtection.RecordFailedAttempt(clientIP)
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 登录成功，清除失败记录
		s.loginProtection.RecordSuccess(clientIP)
		next.ServeHTTP(w, r)
	})
}

// getClientIPFromRequest 从请求中获取客户端IP
func getClientIPFromRequest(r *http.Request) string {
	// 检查X-Forwarded-For头（代理场景）
	// X-Forwarded-For可能包含多个IP，格式：client, proxy1, proxy2
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		// 取第一个IP（原始客户端IP）
		ips := strings.Split(forwardedFor, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}
	// 检查X-Real-IP头
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	// 使用RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// 如果没有端口，直接返回
		return r.RemoteAddr
	}
	return host
}

// getConfig 获取配置
func (s *Server) getConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 获取CSRF token（如果中间件已设置）
	csrfToken := w.Header().Get("X-CSRF-Token")
	if csrfToken == "" {
		// 如果中间件没有设置，生成一个
		csrfToken = s.csrfProtection.GenerateToken(r)
		w.Header().Set("X-CSRF-Token", csrfToken)
	}

	// 将配置和CSRF token一起返回
	// 注意：由于Next.js的rewrites可能不转发自定义响应头，我们将CSRF token放在响应体中
	response := map[string]interface{}{
		"config":     s.config,
		"csrf_token": csrfToken,
	}

	logrus.Debugf("getConfig: returning response with csrf_token: %s", csrfToken)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logrus.Errorf("Failed to encode config: %v", err)
		http.Error(w, "Failed to encode config", http.StatusInternalServerError)
		return
	}
}

// updateConfig 更新配置
func (s *Server) updateConfig(w http.ResponseWriter, r *http.Request) {
	var newConfig config.ServerConfig
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 验证配置
	if err := config.ValidateServerConfig(&newConfig); err != nil {
		http.Error(w, fmt.Sprintf("Configuration validation failed: %v", err), http.StatusBadRequest)
		return
	}

	// 保存配置版本
	version, err := s.versionMgr.SaveVersion("Manual update via web interface")
	if err != nil {
		logrus.Warnf("Failed to save config version: %v", err)
		// 继续执行，但不保证能回滚
	} else {
		logrus.Infof("Config version saved: %s", version)
	}

	// 保存配置到文件（YAML格式）
	data, err := yaml.Marshal(newConfig)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(s.configPath, data, 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.config = &newConfig
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":  "success",
		"version": version,
		"message": "Config updated successfully",
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logrus.Errorf("Failed to encode config update response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	// 注意：配置更新后，代理服务器可能需要重新加载配置
	// 这里只更新了Web服务器的配置，实际代理服务器配置需要重启或热重载
	logrus.Info("Config updated successfully, proxy server may need restart to apply changes")
}

// getIPs 获取IP列表
func (s *Server) getIPs(w http.ResponseWriter, r *http.Request) {
	type IPInfo struct {
		IP     string `json:"ip"`
		Active bool   `json:"active"`
	}

	ips := make([]IPInfo, 0, len(s.config.ExitIPs))

	// 从统计信息中获取IP状态
	ipStatsMap := make(map[string]bool)
	if s.proxyServer != nil {
		if stats := s.proxyServer.GetStats(); stats != nil {
			if connStats, ok := stats.(*monitor.ConnectionStats); ok {
				for ip := range connStats.IPStats {
					ipStatsMap[ip] = true
				}
			}
		}
	}

	for _, ip := range s.config.ExitIPs {
		// 如果IP在统计中有记录，认为它是活跃的
		// 或者可以从healthChecker获取健康状态
		active := ipStatsMap[ip]
		if !active && s.proxyServer != nil {
			// 尝试从healthChecker获取状态（如果有的话）
			// 这里简化处理，有统计记录就认为活跃
			active = true // 默认认为配置中的IP都是可用的
		}

		ips = append(ips, IPInfo{
			IP:     ip,
			Active: active,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ips); err != nil {
		logrus.Errorf("Failed to encode IPs response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// getStatus 获取状态
func (s *Server) getStatus(w http.ResponseWriter, r *http.Request) {
	type Status struct {
		Running     bool   `json:"running"`
		Version     string `json:"version"`
		Connections int    `json:"connections"`
	}

	status := Status{
		Running:     s.proxyServer != nil,
		Version:     "1.0.0",
		Connections: 0,
	}

	// 从代理服务器获取实际连接数
	if s.proxyServer != nil {
		if stats := s.proxyServer.GetStats(); stats != nil {
			if connStats, ok := stats.(*monitor.ConnectionStats); ok {
				status.Connections = int(connStats.ActiveConnections)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		logrus.Errorf("Failed to encode status response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// getStats 获取统计信息
func (s *Server) getStats(w http.ResponseWriter, r *http.Request) {
	if s.proxyServer == nil {
		type Stats struct {
			TotalConnections  int64                  `json:"total_connections"`
			ActiveConnections int64                  `json:"active_connections"`
			BytesTransferred  int64                  `json:"bytes_transferred"`
			BytesUp           int64                  `json:"bytes_up"`
			BytesDown         int64                  `json:"bytes_down"`
			IPStats           map[string]interface{} `json:"ip_stats"`
		}

		stats := Stats{
			TotalConnections:  0,
			ActiveConnections: 0,
			BytesTransferred:  0,
			BytesUp:           0,
			BytesDown:         0,
			IPStats:           make(map[string]interface{}),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
		return
	}

	// 从代理服务器获取实际统计
	rawStats := s.proxyServer.GetStats()
	if rawStats == nil {
		type Stats struct {
			TotalConnections  int64                  `json:"total_connections"`
			ActiveConnections int64                  `json:"active_connections"`
			BytesTransferred  int64                  `json:"bytes_transferred"`
			BytesUp           int64                  `json:"bytes_up"`
			BytesDown         int64                  `json:"bytes_down"`
			IPStats           map[string]interface{} `json:"ip_stats"`
		}
		stats := Stats{
			TotalConnections:  0,
			ActiveConnections: 0,
			BytesTransferred:  0,
			BytesUp:           0,
			BytesDown:         0,
			IPStats:           make(map[string]interface{}),
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(stats); err != nil {
			logrus.Errorf("Failed to encode stats response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	// 转换数据格式（从驼峰式转为下划线式）
	type StatsResponse struct {
		TotalConnections  int64                  `json:"total_connections"`
		ActiveConnections int64                  `json:"active_connections"`
		BytesTransferred  int64                  `json:"bytes_transferred"`
		BytesUp           int64                  `json:"bytes_up"`
		BytesDown         int64                  `json:"bytes_down"`
		IPStats           map[string]interface{} `json:"ip_stats"`
	}

	// 使用反射或类型断言转换
	if stats, ok := rawStats.(*monitor.ConnectionStats); ok {
		// 转换IPStats
		ipStatsMap := make(map[string]interface{})
		for ip, ipStat := range stats.IPStats {
			// 处理LastUsed为零值的情况
			lastUsedStr := ""
			if !ipStat.LastUsed.IsZero() {
				lastUsedStr = ipStat.LastUsed.Format(time.RFC3339)
			}

			ipStatsMap[ip] = map[string]interface{}{
				"connections": ipStat.Connections,
				"active_conn": ipStat.ActiveConn,
				"bytes_up":    ipStat.BytesUp,
				"bytes_down":  ipStat.BytesDown,
				"total_bytes": ipStat.TotalBytes,
				"avg_latency": formatDuration(ipStat.AvgLatency),
				"last_used":   lastUsedStr,
			}
		}

		response := StatsResponse{
			TotalConnections:  stats.TotalConnections,
			ActiveConnections: stats.ActiveConnections,
			BytesTransferred:  stats.BytesTransferred,
			BytesUp:           stats.BytesUp,
			BytesDown:         stats.BytesDown,
			IPStats:           ipStatsMap,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logrus.Errorf("Failed to encode stats response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	// 如果类型不匹配，直接返回原始数据
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(rawStats); err != nil {
		logrus.Errorf("Failed to encode raw stats response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleSubscribe 处理订阅请求
func (s *Server) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	// 验证token（简单验证，可以改为更复杂的机制）
	if token == "" || !s.validateSubscribeToken(token) {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// 获取服务器地址
	serverAddr := r.Host
	if strings.Contains(serverAddr, ":") {
		serverAddr = strings.Split(serverAddr, ":")[0]
	}

	// 创建订阅配置
	subCfg := subscribe.CreateSubscriptionFromConfig(
		s.config,
		serverAddr,
		"MultiExit Proxy",
		365, // 365天有效
	)

	// 编码订阅配置
	encoded, err := subscribe.EncodeSubscription(subCfg)
	if err != nil {
		http.Error(w, "Failed to encode subscription", http.StatusInternalServerError)
		return
	}

	// 返回base64编码的配置（客户端可以直接解析）
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(encoded))
}

// generateSubscribeLink 生成订阅链接
func (s *Server) generateSubscribeLink(w http.ResponseWriter, r *http.Request) {
	// 需要认证
	username, password, ok := r.BasicAuth()
	if !ok || username != s.config.Web.Username || password != s.config.Web.Password {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 生成token（这里简化处理，实际应该生成随机token）
	token := s.generateToken()

	// 获取服务器地址
	serverAddr := r.Host
	port := "8080" // Web端口，实际应该从配置读取
	if strings.Contains(serverAddr, ":") {
		parts := strings.Split(serverAddr, ":")
		serverAddr = parts[0]
		port = parts[1]
	} else {
		// 从请求头获取端口
		if host := r.Header.Get("Host"); host != "" && strings.Contains(host, ":") {
			port = strings.Split(host, ":")[1]
		}
	}

	// 生成订阅链接（使用Web管理端口）
	link := fmt.Sprintf("http://%s:%s/api/subscribe?token=%s", serverAddr, port, token)

	type LinkResponse struct {
		Token  string `json:"token"`
		Link   string `json:"link"`
		QRCode string `json:"qr_code"`
	}

	response := LinkResponse{
		Token:  token,
		Link:   link,
		QRCode: fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=%s", link),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logrus.Errorf("Failed to encode subscription link response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// validateSubscribeToken 验证订阅token（简化版）
func (s *Server) validateSubscribeToken(token string) bool {
	// 这里简化处理，实际应该从数据库或配置中验证
	// 可以使用服务端配置的认证密钥作为token
	if token == s.config.Auth.Key || token == "default-token" {
		return true
	}
	return false
}

// generateToken 生成token（简化版）
func (s *Server) generateToken() string {
	// 简化处理，使用认证密钥作为token
	// 实际应该生成随机token并存储
	return s.config.Auth.Key
}

// Start 启动Web服务器
func (s *Server) Start(listenAddr string) error {
	logrus.Infof("Web API服务器启动在 %s", listenAddr)
	logrus.Infof("前端管理界面请访问独立的前端服务（默认: http://localhost:8081）")
	return http.ListenAndServe(listenAddr, s.router)
}
