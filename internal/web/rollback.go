package web

import (
	"encoding/json"
	"net/http"

	"multiexit-proxy/internal/config"

	"github.com/sirupsen/logrus"
)

// rollbackConfig 回滚配置
func (s *Server) rollbackConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Version string `json:"version"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Version == "" {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}

	// 执行回滚
	if err := s.versionMgr.Rollback(req.Version); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 重新加载配置
	cfg, err := config.LoadServerConfig(s.configPath)
	if err != nil {
		logrus.Errorf("Failed to reload config after rollback: %v", err)
		http.Error(w, "rollback succeeded but failed to reload config", http.StatusInternalServerError)
		return
	}

	s.config = cfg

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"version": req.Version,
		"message": "Config rolled back successfully",
	}); err != nil {
		logrus.Errorf("Failed to encode rollback response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// listConfigVersions 列出配置版本
func (s *Server) listConfigVersions(w http.ResponseWriter, r *http.Request) {
	versions := s.versionMgr.ListVersions()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"versions": versions,
		"count":    len(versions),
	}); err != nil {
		logrus.Errorf("Failed to encode config versions response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

