package trojan

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
)

// Client Trojan客户端
type Client struct {
	serverAddr string
	password   string
	tlsConfig  *tls.Config
}

// ClientConfig Trojan客户端配置
type ClientConfig struct {
	ServerAddr string
	Password   string
	TLSConfig  *tls.Config
}

// NewClient 创建Trojan客户端
func NewClient(config *ClientConfig) *Client {
	return &Client{
		serverAddr: config.ServerAddr,
		password:   config.Password,
		tlsConfig:  config.TLSConfig,
	}
}

// Dial 连接到Trojan服务器并建立到目标的连接
func (c *Client) Dial(targetAddr string) (net.Conn, error) {
	// 建立TLS连接到Trojan服务器
	conn, err := tls.Dial("tcp", c.serverAddr, c.tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	// 发送Trojan协议头（密码哈希）
	header := NewHeader(c.password)
	if _, err := conn.Write(header[:]); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to write header: %w", err)
	}

	// 构建并发送连接请求
	req, err := ParseTargetAddr(targetAddr, CmdConnect)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to parse target addr: %w", err)
	}

	requestData := BuildRequest(req)
	if _, err := conn.Write(requestData); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	return conn, nil
}

// HandleLocalConn 处理本地连接（用于SOCKS5等）
func (c *Client) HandleLocalConn(localConn net.Conn, targetAddr string) error {
	defer localConn.Close()

	// 连接到Trojan服务器
	trojanConn, err := c.Dial(targetAddr)
	if err != nil {
		return err
	}
	defer trojanConn.Close()

	// 双向转发数据
	errCh := make(chan error, 2)

	go func() {
		_, err := io.Copy(trojanConn, localConn)
		errCh <- err
	}()

	go func() {
		_, err := io.Copy(localConn, trojanConn)
		errCh <- err
	}()

	err = <-errCh
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

