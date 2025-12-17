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
		Listen string `yaml:"listen" json:"listen"`
		TLS    struct {
			Cert     string   `yaml:"cert" json:"cert"`
			Key      string   `yaml:"key" json:"key"`
			SNIFake  bool     `yaml:"sni_fake" json:"sni_fake"`
			FakeSNIs []string `yaml:"fake_snis" json:"fake_snis"`
		} `yaml:"tls" json:"tls"`
	} `yaml:"server" json:"server"`

	Auth struct {
		Method string `yaml:"method" json:"method"`
		Key    string `yaml:"key" json:"key"`
	} `yaml:"auth" json:"auth"`

	ExitIPs []string `yaml:"exit_ips" json:"exit_ips"`

	Strategy struct {
		Type       string `yaml:"type" json:"type"`
		PortRanges []struct {
			Range string `yaml:"range" json:"range"`
			IP    string `yaml:"ip" json:"ip"`
		} `yaml:"port_ranges" json:"port_ranges"`
	} `yaml:"strategy" json:"strategy"`

	SNAT struct {
		Enabled   bool   `yaml:"enabled" json:"enabled"`
		Gateway   string `yaml:"gateway" json:"gateway"`
		Interface string `yaml:"interface" json:"interface"`
	} `yaml:"snat" json:"snat"`

	Logging struct {
		Level string `yaml:"level" json:"level"`
		File  string `yaml:"file" json:"file"`
	} `yaml:"logging" json:"logging"`

	Web struct {
		Enabled  bool   `yaml:"enabled" json:"enabled"`
		Listen   string `yaml:"listen" json:"listen"`
		Username string `yaml:"username" json:"username"`
		Password string `yaml:"password" json:"password"`
	} `yaml:"web" json:"web"`

	// Trojan协议支持
	Trojan struct {
		Enabled  bool   `yaml:"enabled" json:"enabled"`
		Password string `yaml:"password" json:"password"`
	} `yaml:"trojan" json:"trojan"`

	// 健康检查配置
	HealthCheck struct {
		Enabled  bool   `yaml:"enabled" json:"enabled"`
		Interval string `yaml:"interval" json:"interval"`
		Timeout  string `yaml:"timeout" json:"timeout"`
	} `yaml:"health_check" json:"health_check"`

	// IP自动检测配置
	IPDetection struct {
		Enabled   bool   `yaml:"enabled" json:"enabled"`
		Interface string `yaml:"interface" json:"interface"` // 指定网络接口，为空则检测所有接口
	} `yaml:"ip_detection" json:"ip_detection"`

	// 连接管理配置
	Connection struct {
		ReadTimeout    string `yaml:"read_timeout" json:"read_timeout"`       // 读取超时
		WriteTimeout   string `yaml:"write_timeout" json:"write_timeout"`     // 写入超时
		IdleTimeout    string `yaml:"idle_timeout" json:"idle_timeout"`       // 空闲超时
		DialTimeout    string `yaml:"dial_timeout" json:"dial_timeout"`       // 连接超时
		MaxConnections int    `yaml:"max_connections" json:"max_connections"` // 最大并发连接数
		KeepAlive      bool   `yaml:"keep_alive" json:"keep_alive"`           // 启用TCP KeepAlive
		KeepAliveTime  string `yaml:"keep_alive_time" json:"keep_alive_time"` // KeepAlive间隔
	} `yaml:"connection" json:"connection"`

	// 地理位置配置
	GeoLocation struct {
		Enabled         bool   `yaml:"enabled" json:"enabled"`                   // 启用地理位置选择
		APIURL          string `yaml:"api_url" json:"api_url"`                   // 地理位置API URL（可选，默认使用ip-api.com）
		LatencyOptimize bool   `yaml:"latency_optimize" json:"latency_optimize"` // 启用延迟优化
	} `yaml:"geo_location" json:"geo_location"`

	// 规则引擎配置
	Rules []struct {
		Name        string   `yaml:"name" json:"name"`
		Priority    int      `yaml:"priority" json:"priority"`
		MatchDomain []string `yaml:"match_domain" json:"match_domain"`
		MatchIP     []string `yaml:"match_ip" json:"match_ip"`
		MatchPort   []int    `yaml:"match_port" json:"match_port"`
		TargetIP    string   `yaml:"target_ip" json:"target_ip"`
		Action      string   `yaml:"action" json:"action"`
		Enabled     bool     `yaml:"enabled" json:"enabled"`
	} `yaml:"rules" json:"rules"`

	// 集群配置
	Cluster struct {
		Enabled        bool     `yaml:"enabled" json:"enabled"`
		NodeID         string   `yaml:"node_id" json:"node_id"`
		Nodes          []string `yaml:"nodes" json:"nodes"`                 // 其他节点地址列表
		LoadBalancer   string   `yaml:"load_balancer" json:"load_balancer"` // "round_robin" or "least_connections"
		HealthInterval string   `yaml:"health_interval" json:"health_interval"`
	} `yaml:"cluster" json:"cluster"`

	// 流量分析配置
	TrafficAnalysis struct {
		Enabled          bool    `yaml:"enabled" json:"enabled"`
		TrendWindow      string  `yaml:"trend_window" json:"trend_window"`
		AnomalyThreshold float64 `yaml:"anomaly_threshold" json:"anomaly_threshold"`
	} `yaml:"traffic_analysis" json:"traffic_analysis"`

	// 监控统计配置
	Monitor struct {
		Enabled bool `yaml:"enabled" json:"enabled"` // 启用统计
	} `yaml:"monitor" json:"monitor"`

	// 数据库配置
	Database struct {
		Enabled  bool   `yaml:"enabled" json:"enabled"`     // 启用数据库
		Host     string `yaml:"host" json:"host"`           // 数据库主机
		Port     int    `yaml:"port" json:"port"`           // 数据库端口
		Database string `yaml:"database" json:"database"`   // 数据库名
		User     string `yaml:"user" json:"user"`           // 用户名
		Password string `yaml:"password" json:"password"`   // 密码
		SSLMode  string `yaml:"ssl_mode" json:"ssl_mode"`   // SSL模式: disable, require, verify-ca, verify-full
		MaxConns int    `yaml:"max_conns" json:"max_conns"` // 最大连接数
		MaxIdle  int    `yaml:"max_idle" json:"max_idle"`   // 最大空闲连接数
	} `yaml:"database" json:"database"`
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
