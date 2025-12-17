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
