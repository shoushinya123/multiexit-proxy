package subscribe

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	sub "multiexit-proxy/internal/subscribe"
)

// ClientConfig 客户端配置结构
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

// Client 订阅客户端
type Client struct {
	subscriptionURL string
	config          *sub.SubscriptionConfig
}

// NewClient 创建订阅客户端
func NewClient(subscriptionURL string) *Client {
	return &Client{
		subscriptionURL: subscriptionURL,
	}
}

// FetchSubscription 获取订阅配置
func (c *Client) FetchSubscription() (*sub.SubscriptionConfig, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(c.subscriptionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subscription server returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscription data: %w", err)
	}

	// 解码base64
	cfg, err := sub.DecodeSubscription(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode subscription: %w", err)
	}

	// 验证订阅
	if err := sub.VerifySubscription(cfg); err != nil {
		return nil, err
	}

	c.config = cfg
	return cfg, nil
}

// ToClientConfig 转换为客户端配置
func (c *Client) ToClientConfig() (*ClientConfig, error) {
	if c.config == nil {
		return nil, fmt.Errorf("subscription not fetched")
	}

	cfg := &ClientConfig{
		Server: struct {
			Address string `json:"address"`
			SNI     string `json:"sni"`
		}{
			Address: c.config.ServerAddr,
			SNI:     c.config.SNI,
		},
		Auth: struct {
			Key string `json:"key"`
		}{
			Key: c.config.AuthKey,
		},
		Local: struct {
			SOCKS5 string `json:"socks5"`
			HTTP   string `json:"http"`
		}{
			SOCKS5: "127.0.0.1:1080",
			HTTP:   "127.0.0.1:8080",
		},
		Logging: struct {
			Level string `json:"level"`
		}{
			Level: "info",
		},
	}

	return cfg, nil
}

// GetConfigJSON 获取JSON格式的配置
func (c *Client) GetConfigJSON() ([]byte, error) {
	cfg, err := c.ToClientConfig()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(cfg, "", "  ")
}
