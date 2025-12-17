package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

// FunctionGraph HTTP函数入口
// 根据华为云FunctionGraph文档：https://support.huaweicloud.com/devg-functiongraph/functiongraph_02_0603.html
func Handler(w http.ResponseWriter, r *http.Request) {
	router := mux.NewRouter()
	setupRoutes(router)
	router.ServeHTTP(w, r)
}

func setupRoutes(router *mux.Router) {
	// API路由
	api := router.PathPrefix("/api").Subrouter()

	// 订阅API（公开访问）
	api.HandleFunc("/subscribe", handleSubscribe).Methods("GET")
	api.HandleFunc("/subscription/link", handleGenerateLink).Methods("GET")

	// 配置管理API（需要认证）
	api.HandleFunc("/config", handleGetConfig).Methods("GET")
	api.HandleFunc("/config", handleUpdateConfig).Methods("POST")
	api.HandleFunc("/ips", handleGetIPs).Methods("GET")
	api.HandleFunc("/status", handleGetStatus).Methods("GET")

	// 静态文件
	router.PathPrefix("/").HandlerFunc(handleStatic)
}

// handleSubscribe 处理订阅请求
func handleSubscribe(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	// 从环境变量或OBS获取配置
	authKey := os.Getenv("AUTH_KEY")
	if token == "" || token != authKey {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// 从环境变量获取配置
	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = "your-server.com"
	}

	// 创建订阅配置
	subCfg := map[string]interface{}{
		"v":        "1.0",
		"server":   serverAddr + ":443",
		"sni":      "cloudflare.com",
		"key":      authKey,
		"ips":      []string{"1.2.3.4", "5.6.7.8"},
		"strategy": "round_robin",
		"remark":   "MultiExit Proxy",
	}

	// 编码为base64
	data, _ := json.Marshal(subCfg)
	encoded := base64.StdEncoding.EncodeToString(data)

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(encoded))
}

// handleGenerateLink 生成订阅链接
func handleGenerateLink(w http.ResponseWriter, r *http.Request) {
	// 需要认证
	username, password, ok := r.BasicAuth()
	webUser := os.Getenv("WEB_USERNAME")
	webPass := os.Getenv("WEB_PASSWORD")

	if !ok || username != webUser || password != webPass {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	authKey := os.Getenv("AUTH_KEY")
	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = r.Host
	}

	link := fmt.Sprintf("http://%s/api/subscribe?token=%s", serverAddr, authKey)

	response := map[string]string{
		"token":   authKey,
		"link":    link,
		"qr_code": fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=%s", link),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetConfig 获取配置
func handleGetConfig(w http.ResponseWriter, r *http.Request) {
	// TODO: 从OBS或环境变量读取配置
	config := map[string]interface{}{
		"exit_ips": []string{"1.2.3.4", "5.6.7.8"},
		"strategy": map[string]string{"type": "round_robin"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// handleUpdateConfig 更新配置
func handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	// TODO: 保存配置到OBS
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleGetIPs 获取IP列表
func handleGetIPs(w http.ResponseWriter, r *http.Request) {
	ips := []map[string]interface{}{
		{"ip": "1.2.3.4", "active": true},
		{"ip": "5.6.7.8", "active": true},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ips)
}

// handleGetStatus 获取状态
func handleGetStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"running":     true,
		"version":     "1.0.0",
		"connections": 0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleStatic 处理静态文件
func handleStatic(w http.ResponseWriter, r *http.Request) {
	// FunctionGraph中静态文件需要打包在函数包中
	// 或使用OBS存储静态文件
	http.NotFound(w, r)
}
