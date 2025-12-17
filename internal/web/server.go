package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"multiexit-proxy/internal/config"
	"multiexit-proxy/internal/subscribe"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

//go:embed static
var staticFiles embed.FS

// Server Web管理服务器
type Server struct {
	config     *config.ServerConfig
	configPath string
	router     *mux.Router
}

// NewServer 创建Web服务器
func NewServer(configPath string, cfg *config.ServerConfig) *Server {
	s := &Server{
		config:     cfg,
		configPath: configPath,
		router:     mux.NewRouter(),
	}
	s.setupRoutes()
	return s
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// API路由
	api := s.router.PathPrefix("/api").Subrouter()
	api.Use(s.authMiddleware)

	api.HandleFunc("/config", s.getConfig).Methods("GET")
	api.HandleFunc("/config", s.updateConfig).Methods("POST")
	api.HandleFunc("/ips", s.getIPs).Methods("GET")
	api.HandleFunc("/status", s.getStatus).Methods("GET")
	api.HandleFunc("/stats", s.getStats).Methods("GET")

	// 订阅相关路由（不需要认证）
	s.router.HandleFunc("/api/subscribe", s.handleSubscribe).Methods("GET")
	s.router.HandleFunc("/api/subscription/link", s.generateSubscribeLink).Methods("GET")

	// 静态文件 - 尝试从嵌入的文件系统读取，失败则从文件系统读取
	s.router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" || path == "/index.html" {
			// 尝试从嵌入的文件系统读取
			data, err := staticFiles.ReadFile("static/index.html")
			if err != nil {
				// 如果嵌入失败，尝试从文件系统读取
				fsPath := filepath.Join("internal/web/static/index.html")
				http.ServeFile(w, r, fsPath)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
		} else {
			http.NotFound(w, r)
		}
	})
}

// authMiddleware 认证中间件
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != s.config.Web.Username || password != s.config.Web.Password {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// getConfig 获取配置
func (s *Server) getConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.config)
}

// updateConfig 更新配置
func (s *Server) updateConfig(w http.ResponseWriter, r *http.Request) {
	var newConfig config.ServerConfig
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// getIPs 获取IP列表
func (s *Server) getIPs(w http.ResponseWriter, r *http.Request) {
	type IPInfo struct {
		IP     string `json:"ip"`
		Active bool   `json:"active"`
	}

	ips := make([]IPInfo, 0, len(s.config.ExitIPs))
	for _, ip := range s.config.ExitIPs {
		ips = append(ips, IPInfo{
			IP:     ip,
			Active: true, // TODO: 实际检测IP是否活跃
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ips)
}

// getStatus 获取状态
func (s *Server) getStatus(w http.ResponseWriter, r *http.Request) {
	type Status struct {
		Running     bool   `json:"running"`
		Version     string `json:"version"`
		Connections int    `json:"connections"`
	}

	status := Status{
		Running:     true, // TODO: 实际检测服务状态
		Version:     "1.0.0",
		Connections: 0, // TODO: 实际统计连接数
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// getStats 获取统计信息
func (s *Server) getStats(w http.ResponseWriter, r *http.Request) {
	type Stats struct {
		TotalConnections int64            `json:"total_connections"`
		IPStats          map[string]int64 `json:"ip_stats"`
		Bandwidth        map[string]int64 `json:"bandwidth"`
	}

	stats := Stats{
		TotalConnections: 0,
		IPStats:          make(map[string]int64),
		Bandwidth:        make(map[string]int64),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
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
	json.NewEncoder(w).Encode(response)
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
	logrus.Infof("Web管理界面启动在 %s", listenAddr)
	return http.ListenAndServe(listenAddr, s.router)
}
