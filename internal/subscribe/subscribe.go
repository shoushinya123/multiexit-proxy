package subscribe

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"multiexit-proxy/internal/config"
)

// SubscriptionConfig 订阅配置
type SubscriptionConfig struct {
	Version    string   `json:"v"`
	ServerAddr string   `json:"server"`
	SNI        string   `json:"sni"`
	AuthKey    string   `json:"key"`
	ExitIPs    []string `json:"ips"`
	Strategy   string   `json:"strategy"`
	Remark     string   `json:"remark"`
	ExpiresAt  int64    `json:"expires"`
}

// GenerateSubscriptionLink 生成订阅链接
func GenerateSubscriptionLink(serverAddr string, port int, token string) string {
	return fmt.Sprintf("http://%s:%d/api/subscribe?token=%s", serverAddr, port, token)
}

// EncodeSubscription 编码订阅配置为base64字符串
func EncodeSubscription(cfg *SubscriptionConfig) (string, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// DecodeSubscription 解码订阅配置
func DecodeSubscription(encoded string) (*SubscriptionConfig, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	var cfg SubscriptionConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// CreateSubscriptionFromConfig 从服务端配置创建订阅配置
func CreateSubscriptionFromConfig(cfg *config.ServerConfig, serverAddr string, remark string, expiresDays int) *SubscriptionConfig {
	expiresAt := time.Now().AddDate(0, 0, expiresDays).Unix()

	// 选择第一个SNI作为默认值
	sni := "cloudflare.com"
	if len(cfg.Server.TLS.FakeSNIs) > 0 {
		sni = cfg.Server.TLS.FakeSNIs[0]
	}

	return &SubscriptionConfig{
		Version:    "1.0",
		ServerAddr: serverAddr,
		SNI:        sni,
		AuthKey:    cfg.Auth.Key,
		ExitIPs:    cfg.ExitIPs,
		Strategy:   cfg.Strategy.Type,
		Remark:     remark,
		ExpiresAt:  expiresAt,
	}
}

// VerifySubscription 验证订阅是否有效
func VerifySubscription(cfg *SubscriptionConfig) error {
	if cfg.ExpiresAt > 0 && time.Now().Unix() > cfg.ExpiresAt {
		return fmt.Errorf("subscription expired")
	}
	return nil
}
