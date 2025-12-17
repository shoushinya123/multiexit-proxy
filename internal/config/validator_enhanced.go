package config

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// ValidateServerConfigEnhanced 增强的服务端配置验证
func ValidateServerConfigEnhanced(cfg *ServerConfig) error {
	// 先执行基础验证
	if err := ValidateServerConfig(cfg); err != nil {
		return err
	}

	var errors []error

	// 验证IP可达性
	if len(cfg.ExitIPs) > 0 {
		logrus.Info("Validating exit IPs reachability...")
		for i, ipStr := range cfg.ExitIPs {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				errors = append(errors, fmt.Errorf("invalid exit IP at index %d: %s", i, ipStr))
				continue
			}

			// 检查IP是否可达（ping测试）
			if err := checkIPReachability(ip); err != nil {
				logrus.Warnf("Exit IP %s may not be reachable: %v", ipStr, err)
				// 警告但不阻止启动
			} else {
				logrus.Debugf("Exit IP %s is reachable", ipStr)
			}
		}
	}

	// 验证TLS证书有效性
	if cfg.Server.TLS.Cert != "" && cfg.Server.TLS.Key != "" {
		logrus.Info("Validating TLS certificate...")
		if err := validateTLSCertificate(cfg.Server.TLS.Cert, cfg.Server.TLS.Key); err != nil {
			errors = append(errors, fmt.Errorf("TLS certificate validation failed: %w", err))
		} else {
			logrus.Info("TLS certificate is valid")
		}
	}

	if len(errors) > 0 {
		errMsg := "enhanced configuration validation failed:\n"
		for i, err := range errors {
			errMsg += fmt.Sprintf("  %d. %v\n", i+1, err)
		}
		return fmt.Errorf(errMsg)
	}

	return nil
}

// checkIPReachability 检查IP是否可达
func checkIPReachability(ip net.IP) error {
	// 尝试建立TCP连接到常见端口（如80或443）
	// 这里简化处理，只检查IP格式和基本连通性
	testPorts := []int{80, 443, 22}
	
	for _, port := range testPorts {
		addr := fmt.Sprintf("%s:%d", ip.String(), port)
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			return nil // 至少有一个端口可达
		}
	}

	// 如果所有端口都不可达，返回警告但不报错
	// 因为IP可能只是配置了但还没有绑定服务
	return fmt.Errorf("IP %s may not be reachable on common ports", ip.String())
}

// validateTLSCertificate 验证TLS证书有效性
func validateTLSCertificate(certPath, keyPath string) error {
	// 读取证书文件
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("failed to read certificate file: %w", err)
	}

	// 解析证书
	block, _ := pem.Decode(certData)
	if block == nil {
		return fmt.Errorf("failed to decode certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// 检查证书是否过期
	now := time.Now()
	if now.After(cert.NotAfter) {
		return fmt.Errorf("certificate expired on %s", cert.NotAfter.Format(time.RFC3339))
	}

	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificate not valid until %s", cert.NotBefore.Format(time.RFC3339))
	}

	// 验证证书和密钥匹配
	certificate, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return fmt.Errorf("certificate and key mismatch: %w", err)
	}

	if len(certificate.Certificate) == 0 {
		return fmt.Errorf("no certificates found in key pair")
	}

	logrus.Debugf("Certificate valid until: %s", cert.NotAfter.Format(time.RFC3339))
	return nil
}

