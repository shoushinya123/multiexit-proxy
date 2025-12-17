package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// ValidateServerConfig 验证服务端配置
func ValidateServerConfig(cfg *ServerConfig) error {
	var errors []error

	// 验证监听地址
	if cfg.Server.Listen == "" {
		errors = append(errors, fmt.Errorf("server.listen is required"))
	} else {
		if _, _, err := net.SplitHostPort(cfg.Server.Listen); err != nil {
			errors = append(errors, fmt.Errorf("invalid server.listen address: %w", err))
		}
	}

	// 验证TLS配置
	if cfg.Server.TLS.Cert == "" {
		errors = append(errors, fmt.Errorf("server.tls.cert is required"))
	} else {
		if _, err := os.Stat(cfg.Server.TLS.Cert); err != nil {
			errors = append(errors, fmt.Errorf("TLS cert file not found: %s: %w", cfg.Server.TLS.Cert, err))
		}
	}

	if cfg.Server.TLS.Key == "" {
		errors = append(errors, fmt.Errorf("server.tls.key is required"))
	} else {
		if _, err := os.Stat(cfg.Server.TLS.Key); err != nil {
			errors = append(errors, fmt.Errorf("TLS key file not found: %s: %w", cfg.Server.TLS.Key, err))
		}
	}

	// 验证认证密钥
	if cfg.Auth.Key == "" {
		errors = append(errors, fmt.Errorf("auth.key is required"))
	} else if len(cfg.Auth.Key) < 16 {
		errors = append(errors, fmt.Errorf("auth.key must be at least 16 characters"))
	}

	// 验证出口IP
	if len(cfg.ExitIPs) == 0 && !cfg.IPDetection.Enabled {
		errors = append(errors, fmt.Errorf("exit_ips is required when ip_detection is disabled"))
	} else {
		for i, ipStr := range cfg.ExitIPs {
			if ip := net.ParseIP(ipStr); ip == nil {
				errors = append(errors, fmt.Errorf("invalid exit IP at index %d: %s", i, ipStr))
			}
		}
	}

	// 验证策略类型
	validStrategies := map[string]bool{
		"round_robin":        true,
		"port_based":        true,
		"destination_based": true,
		"load_balanced":     true,
	}
	if cfg.Strategy.Type != "" && !validStrategies[cfg.Strategy.Type] {
		errors = append(errors, fmt.Errorf("invalid strategy type: %s", cfg.Strategy.Type))
	}

	// 验证SNAT配置
	if cfg.SNAT.Enabled {
		if cfg.SNAT.Gateway == "" {
			errors = append(errors, fmt.Errorf("snat.gateway is required when snat.enabled is true"))
		} else {
			if ip := net.ParseIP(cfg.SNAT.Gateway); ip == nil {
				errors = append(errors, fmt.Errorf("invalid snat.gateway: %s", cfg.SNAT.Gateway))
			}
		}
		if cfg.SNAT.Interface == "" {
			errors = append(errors, fmt.Errorf("snat.interface is required when snat.enabled is true"))
		}
	}

	// 验证健康检查配置
	if cfg.HealthCheck.Enabled {
		if cfg.HealthCheck.Interval != "" {
			if _, err := time.ParseDuration(cfg.HealthCheck.Interval); err != nil {
				errors = append(errors, fmt.Errorf("invalid health_check.interval: %w", err))
			}
		}
		if cfg.HealthCheck.Timeout != "" {
			if _, err := time.ParseDuration(cfg.HealthCheck.Timeout); err != nil {
				errors = append(errors, fmt.Errorf("invalid health_check.timeout: %w", err))
			}
		}
	}

	// 验证连接配置
	if cfg.Connection.ReadTimeout != "" {
		if _, err := time.ParseDuration(cfg.Connection.ReadTimeout); err != nil {
			errors = append(errors, fmt.Errorf("invalid connection.read_timeout: %w", err))
		}
	}
	if cfg.Connection.WriteTimeout != "" {
		if _, err := time.ParseDuration(cfg.Connection.WriteTimeout); err != nil {
			errors = append(errors, fmt.Errorf("invalid connection.write_timeout: %w", err))
		}
	}
	if cfg.Connection.IdleTimeout != "" {
		if _, err := time.ParseDuration(cfg.Connection.IdleTimeout); err != nil {
			errors = append(errors, fmt.Errorf("invalid connection.idle_timeout: %w", err))
		}
	}
	if cfg.Connection.DialTimeout != "" {
		if _, err := time.ParseDuration(cfg.Connection.DialTimeout); err != nil {
			errors = append(errors, fmt.Errorf("invalid connection.dial_timeout: %w", err))
		}
	}
	if cfg.Connection.MaxConnections < 0 {
		errors = append(errors, fmt.Errorf("connection.max_connections must be >= 0"))
	}

	// 验证日志配置
	if cfg.Logging.Level != "" {
		if _, err := logrus.ParseLevel(cfg.Logging.Level); err != nil {
			errors = append(errors, fmt.Errorf("invalid logging.level: %w", err))
		}
	}
	if cfg.Logging.File != "" {
		dir := filepath.Dir(cfg.Logging.File)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				errors = append(errors, fmt.Errorf("failed to create log directory: %w", err))
			}
		}
	}

	// 验证Web配置
	if cfg.Web.Enabled {
		if cfg.Web.Listen == "" {
			errors = append(errors, fmt.Errorf("web.listen is required when web.enabled is true"))
		} else {
			if _, _, err := net.SplitHostPort(cfg.Web.Listen); err != nil {
				errors = append(errors, fmt.Errorf("invalid web.listen address: %w", err))
			}
		}
		if cfg.Web.Username == "" {
			errors = append(errors, fmt.Errorf("web.username is required when web.enabled is true"))
		}
		if cfg.Web.Password == "" {
			errors = append(errors, fmt.Errorf("web.password is required when web.enabled is true"))
		}
	}

	// 返回所有错误
	if len(errors) > 0 {
		errMsg := "configuration validation failed:\n"
		for i, err := range errors {
			errMsg += fmt.Sprintf("  %d. %v\n", i+1, err)
		}
		return fmt.Errorf(errMsg)
	}

	return nil
}

// ValidateClientConfig 验证客户端配置
func ValidateClientConfig(cfg *ClientConfig) error {
	var errors []error

	// 验证服务器地址
	if cfg.Server.Address == "" {
		errors = append(errors, fmt.Errorf("server.address is required"))
	} else {
		if _, _, err := net.SplitHostPort(cfg.Server.Address); err != nil {
			errors = append(errors, fmt.Errorf("invalid server.address: %w", err))
		}
	}

	// 验证认证密钥
	if cfg.Auth.Key == "" {
		errors = append(errors, fmt.Errorf("auth.key is required"))
	}

	// 验证本地监听地址
	if cfg.Local.SOCKS5 == "" && cfg.Local.HTTP == "" {
		errors = append(errors, fmt.Errorf("at least one of local.socks5 or local.http is required"))
	}
	if cfg.Local.SOCKS5 != "" {
		if _, _, err := net.SplitHostPort(cfg.Local.SOCKS5); err != nil {
			errors = append(errors, fmt.Errorf("invalid local.socks5 address: %w", err))
		}
	}
	if cfg.Local.HTTP != "" {
		if _, _, err := net.SplitHostPort(cfg.Local.HTTP); err != nil {
			errors = append(errors, fmt.Errorf("invalid local.http address: %w", err))
		}
	}

	// 验证日志级别
	if cfg.Logging.Level != "" {
		if _, err := logrus.ParseLevel(cfg.Logging.Level); err != nil {
			errors = append(errors, fmt.Errorf("invalid logging.level: %w", err))
		}
	}

	if len(errors) > 0 {
		errMsg := "configuration validation failed:\n"
		for i, err := range errors {
			errMsg += fmt.Sprintf("  %d. %v\n", i+1, err)
		}
		return fmt.Errorf(errMsg)
	}

	return nil
}



