package config

import (
	"encoding/json"
	"fmt"
	"os"

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

