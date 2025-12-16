package trojan

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"sync"

	"multiexit-proxy/internal/snat"
)

// Server Trojan服务器
type Server struct {
	password    string
	tlsConfig   *tls.Config
	ipSelector  snat.IPSelector
	routingMgr  *snat.RoutingManager
	listener    net.Listener
	connCounter sync.Map // 连接计数器
}

// ServerConfig Trojan服务器配置
type ServerConfig struct {
	ListenAddr string
	Password   string
	TLSConfig  *tls.Config
	IPSelector snat.IPSelector
	RoutingMgr *snat.RoutingManager
}

// NewServer 创建Trojan服务器
func NewServer(config *ServerConfig) (*Server, error) {
	// 创建TLS监听器
	listener, err := tls.Listen("tcp", config.ListenAddr, config.TLSConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	return &Server{
		password:   config.Password,
		tlsConfig:  config.TLSConfig,
		ipSelector: config.IPSelector,
		routingMgr: config.RoutingMgr,
		listener:   listener,
	}, nil
}

// Start 启动Trojan服务器
func (s *Server) Start() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return err
		}

		go s.handleConn(conn)
	}
}

// Stop 停止服务器
func (s *Server) Stop() error {
	return s.listener.Close()
}

// handleConn 处理Trojan连接
func (s *Server) handleConn(conn net.Conn) error {
	defer conn.Close()

	// 读取Trojan协议头（56字节密码哈希）
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}

	// 验证密码
	var headerHash Header
	copy(headerHash[:], header)
	if !VerifyHeader(headerHash, s.password) {
		// 密码错误，发送HTTP 400响应（伪装）
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return ErrInvalidPassword
	}

	// 解析连接请求
	req, err := ParseRequest(conn)
	if err != nil {
		return fmt.Errorf("failed to parse request: %w", err)
	}

	// 获取目标地址
	targetAddr := req.GetTargetAddr()

	// 选择出口IP
	host, portStr, _ := net.SplitHostPort(targetAddr)
	var port int
	fmt.Sscanf(portStr, "%d", &port)

	exitIP, err := s.ipSelector.SelectIP(host, port)
	if err != nil {
		return fmt.Errorf("failed to select IP: %w", err)
	}

	// 建立到目标的连接
	var targetConn net.Conn
	if s.routingMgr != nil {
		// 使用SNAT
		dialer := &net.Dialer{}
		targetConn, err = dialer.Dial("tcp", targetAddr)
		if err != nil {
			return fmt.Errorf("failed to dial target: %w", err)
		}

		// 标记连接
		if err := s.routingMgr.MarkConnection(targetConn, exitIP); err != nil {
			targetConn.Close()
			return fmt.Errorf("failed to mark connection: %w", err)
		}
	} else {
		// 不使用SNAT，直接连接
		targetConn, err = net.Dial("tcp", targetAddr)
		if err != nil {
			return fmt.Errorf("failed to dial target: %w", err)
		}
	}
	defer targetConn.Close()

	// 双向转发数据（Trojan协议没有响应头，直接转发）
	errCh := make(chan error, 2)

	go func() {
		_, err := io.Copy(targetConn, conn)
		errCh <- err
	}()

	go func() {
		_, err := io.Copy(conn, targetConn)
		errCh <- err
	}()

	// 等待任一方向结束
	err = <-errCh
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

