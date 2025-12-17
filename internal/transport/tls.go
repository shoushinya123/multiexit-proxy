package transport

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"time"

	utls "github.com/refraction-networking/utls"
)

var (
	// FakeSNIs 伪装的SNI列表
	FakeSNIs = []string{
		"cloudflare.com",
		"google.com",
		"github.com",
		"microsoft.com",
		"amazon.com",
		"facebook.com",
	}
)

// GetRandomSNI 获取随机SNI
func GetRandomSNI() string {
	rand.Seed(time.Now().UnixNano())
	return FakeSNIs[rand.Intn(len(FakeSNIs))]
}

// ClientTLSConfig 客户端TLS配置
type ClientTLSConfig struct {
	SNI string
}

// DialTLS 使用uTLS建立TLS连接（客户端）
func DialTLS(network, addr string, config *ClientTLSConfig) (net.Conn, error) {
	sni := config.SNI
	if sni == "" {
		sni = GetRandomSNI()
	}

	// 建立TCP连接
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}

	// 创建uTLS配置
	utlsConfig := &utls.Config{
		ServerName:         sni,
		InsecureSkipVerify: true, // 跳过证书验证（用于伪装）
		NextProtos:         []string{"h2", "http/1.1"},
	}

	// 创建uTLS连接
	utlsConn := utls.UClient(conn, utlsConfig, utls.HelloChrome_Auto)
	if err := utlsConn.Handshake(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("TLS handshake failed: %w", err)
	}

	return utlsConn, nil
}

// ServerTLSConfig 服务端TLS配置
type ServerTLSConfig struct {
	Cert    string
	Key     string
	SNIFake bool
}

// ListenTLS 监听TLS连接（服务端）
func ListenTLS(network, addr string, config *ServerTLSConfig) (net.Listener, error) {
	var tlsConfig *tls.Config

	if config.Cert != "" && config.Key != "" {
		// 使用提供的证书
		cert, err := tls.LoadX509KeyPair(config.Cert, config.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate: %w", err)
		}

		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"h2", "http/1.1"},
		}
	} else {
		// 生成自签名证书（用于测试）
		cert, err := generateSelfSignedCert()
		if err != nil {
			return nil, fmt.Errorf("failed to generate certificate: %w", err)
		}

		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"h2", "http/1.1"},
		}
	}

	// 创建TCP监听器
	listener, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}

	// 包装为TLS监听器
	return tls.NewListener(listener, tlsConfig), nil
}

// generateSelfSignedCert 生成自签名证书（用于测试）
func generateSelfSignedCert() (tls.Certificate, error) {
	// 这里应该使用crypto/x509生成证书
	// 为了简化，我们返回一个错误，提示需要提供证书
	return tls.Certificate{}, fmt.Errorf("certificate required, use Cert and Key in config")
}


