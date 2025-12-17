package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ServerConfig 服务端配置
type ServerConfig struct {
	Server struct {
		Listen string `yaml:"listen"`
		TLS    struct {
			Cert     string   `yaml:"cert"`
			Key      string   `yaml:"key"`
			SNIFake  bool     `yaml:"sni_fake"`
			FakeSNIs []string `yaml:"fake_snis"`
		} `yaml:"tls"`
	} `yaml:"server"`

	Auth struct {
		Method string `yaml:"method"`
		Key    string `yaml:"key"`
	} `yaml:"auth"`

	ExitIPs []string `yaml:"exit_ips"`

	Strategy struct {
		Type       string `yaml:"type"`
		PortRanges []struct {
			Range string `yaml:"range"`
			IP    string `yaml:"ip"`
		} `yaml:"port_ranges"`
	} `yaml:"strategy"`

	SNAT struct {
		Enabled   bool   `yaml:"enabled"`
		Gateway   string `yaml:"gateway"`
		Interface string `yaml:"interface"`
	} `yaml:"snat"`

	Logging struct {
		Level string `yaml:"level"`
		File  string `yaml:"file"`
	} `yaml:"logging"`

	Web struct {
		Enabled  bool   `yaml:"enabled"`
		Listen   string `yaml:"listen"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"web"`

	// Trojan协议支持
	Trojan struct {
		Enabled  bool   `yaml:"enabled"`
		Password string `yaml:"password"`
	} `yaml:"trojan"`

	// 健康检查配置
	HealthCheck struct {
		Enabled  bool   `yaml:"enabled"`
		Interval string `yaml:"interval"`
		Timeout  string `yaml:"timeout"`
	} `yaml:"health_check"`

	// IP自动检测配置
	IPDetection struct {
		Enabled   bool   `yaml:"enabled"`
		Interface string `yaml:"interface"` // 指定网络接口，为空则检测所有接口
	} `yaml:"ip_detection"`

	// 连接管理配置
	Connection struct {
		ReadTimeout    string `yaml:"read_timeout"`    // 读取超时
		WriteTimeout   string `yaml:"write_timeout"`   // 写入超时
		IdleTimeout    string `yaml:"idle_timeout"`    // 空闲超时
		DialTimeout    string `yaml:"dial_timeout"`    // 连接超时
		MaxConnections int    `yaml:"max_connections"` // 最大并发连接数
		KeepAlive      bool   `yaml:"keep_alive"`      // 启用TCP KeepAlive
		KeepAliveTime  string `yaml:"keep_alive_time"` // KeepAlive间隔
	} `yaml:"connection"`

	// 地理位置配置
	GeoLocation struct {
		Enabled         bool   `yaml:"enabled"`          // 启用地理位置选择
		APIURL          string `yaml:"api_url"`          // 地理位置API URL（可选，默认使用ip-api.com）
		LatencyOptimize bool   `yaml:"latency_optimize"` // 启用延迟优化
	} `yaml:"geo_location"`

	// 规则引擎配置
	Rules []struct {
		Name        string   `yaml:"name"`
		Priority    int      `yaml:"priority"`
		MatchDomain []string `yaml:"match_domain"`
		MatchIP     []string `yaml:"match_ip"`
		MatchPort   []int    `yaml:"match_port"`
		TargetIP    string   `yaml:"target_ip"`
		Action      string   `yaml:"action"`
		Enabled     bool     `yaml:"enabled"`
	} `yaml:"rules"`

	// 集群配置
	Cluster struct {
		Enabled        bool     `yaml:"enabled"`
		NodeID         string   `yaml:"node_id"`
		Nodes          []string `yaml:"nodes"`         // 其他节点地址列表
		LoadBalancer   string   `yaml:"load_balancer"` // "round_robin" or "least_connections"
		HealthInterval string   `yaml:"health_interval"`
	} `yaml:"cluster"`

	// 流量分析配置
	TrafficAnalysis struct {
		Enabled          bool    `yaml:"enabled"`
		TrendWindow      string  `yaml:"trend_window"`
		AnomalyThreshold float64 `yaml:"anomaly_threshold"`
	} `yaml:"traffic_analysis"`

	// 监控统计配置
	Monitor struct {
		Enabled bool `yaml:"enabled"` // 启用统计
	} `yaml:"monitor"`
}

// ClientConfig 客户端配置
type ClientConfig struct {
	Server struct {
		Address string `json:"address"`
		SNI     string `json:"sni"`
	} `json:"server"`

	Auth struct {
		Key string `json:"key"`
	} `json:"auth"`

	Local struct {
		SOCKS5 string `json:"socks5"`
		HTTP   string `json:"http"`
	} `json:"local"`

	Logging struct {
		Level string `json:"level"`
	} `json:"logging"`

	// 重连配置
	Reconnect struct {
		MaxRetries    int     `json:"max_retries"`    // 最大重试次数（0=无限重试）
		InitialDelay  string  `json:"initial_delay"`  // 初始延迟（如"1s"）
		MaxDelay      string  `json:"max_delay"`      // 最大延迟（如"5m"）
		BackoffFactor float64 `json:"backoff_factor"` // 退避因子（默认2.0）
		Jitter        bool    `json:"jitter"`         // 是否添加随机抖动
	} `json:"reconnect"`
}

// LoadServerConfig 加载服务端配置
func LoadServerConfig(path string) (*ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ServerConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// GetHealthCheckInterval 获取健康检查间隔
func (c *ServerConfig) GetHealthCheckInterval() time.Duration {
	if c.HealthCheck.Interval == "" {
		return 30 * time.Second
	}
	d, err := time.ParseDuration(c.HealthCheck.Interval)
	if err != nil {
		return 30 * time.Second
	}
	return d
}

// GetHealthCheckTimeout 获取健康检查超时
func (c *ServerConfig) GetHealthCheckTimeout() time.Duration {
	if c.HealthCheck.Timeout == "" {
		return 5 * time.Second
	}
	d, err := time.ParseDuration(c.HealthCheck.Timeout)
	if err != nil {
		return 5 * time.Second
	}
	return d
}

// GetReadTimeout 获取读取超时
func (c *ServerConfig) GetReadTimeout() time.Duration {
	if c.Connection.ReadTimeout == "" {
		return 30 * time.Second
	}
	d, err := time.ParseDuration(c.Connection.ReadTimeout)
	if err != nil {
		return 30 * time.Second
	}
	return d
}

// GetWriteTimeout 获取写入超时
func (c *ServerConfig) GetWriteTimeout() time.Duration {
	if c.Connection.WriteTimeout == "" {
		return 30 * time.Second
	}
	d, err := time.ParseDuration(c.Connection.WriteTimeout)
	if err != nil {
		return 30 * time.Second
	}
	return d
}

// GetIdleTimeout 获取空闲超时
func (c *ServerConfig) GetIdleTimeout() time.Duration {
	if c.Connection.IdleTimeout == "" {
		return 300 * time.Second // 5分钟
	}
	d, err := time.ParseDuration(c.Connection.IdleTimeout)
	if err != nil {
		return 300 * time.Second
	}
	return d
}

// GetDialTimeout 获取连接超时
func (c *ServerConfig) GetDialTimeout() time.Duration {
	if c.Connection.DialTimeout == "" {
		return 10 * time.Second
	}
	d, err := time.ParseDuration(c.Connection.DialTimeout)
	if err != nil {
		return 10 * time.Second
	}
	return d
}

// LoadClientConfig 加载客户端配置
func LoadClientConfig(path string) (*ClientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}
