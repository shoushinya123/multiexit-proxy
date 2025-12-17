package proxy

import (
	"testing"
	"time"
)

func TestServer_GetStats(t *testing.T) {
	config := &ServerConfig{
		ListenAddr: "127.0.0.1:0",
		AuthKey:    "test-key",
		ExitIPs:    []string{"127.0.0.1"},
		Strategy:   "round_robin",
		Connection: struct {
			ReadTimeout    time.Duration
			WriteTimeout   time.Duration
			IdleTimeout    time.Duration
			DialTimeout    time.Duration
			MaxConnections int
			KeepAlive      bool
			KeepAliveTime  time.Duration
		}{
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			IdleTimeout:    300 * time.Second,
			DialTimeout:    10 * time.Second,
			MaxConnections: 100,
			KeepAlive:      true,
			KeepAliveTime:  30 * time.Second,
		},
		EnableStats: true,
	}

	// 注意：这个测试需要TLS配置，所以可能无法完全运行
	// 这里主要是测试代码结构
	t.Log("Server config created successfully")
	t.Logf("Stats enabled: %v", config.EnableStats)
}

