package proxy

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"multiexit-proxy/internal/protocol"
	"multiexit-proxy/internal/snat"
	"multiexit-proxy/internal/transport"
)

// Server 代理服务端
type Server struct {
	config        *ServerConfig
	cipher        *protocol.Cipher
	ipSelector    snat.IPSelector
	healthChecker *snat.IPHealthChecker
	routingMgr    *snat.RoutingManager
	listener      net.Listener
}

// ServerConfig 服务端配置（简化版，实际应从config包导入）
type ServerConfig struct {
	ListenAddr    string
	TLSConfig     *transport.ServerTLSConfig
	AuthKey       string
	ExitIPs       []string
	Strategy      string
	StrategyParam string // load_balanced策略参数：connections 或 traffic
	HealthCheck   struct {
		Enabled  bool
		Interval time.Duration
		Timeout  time.Duration
	}
	SNAT struct {
		Enabled   bool
		Gateway   string
		Interface string
	}
}

// NewServer 创建代理服务端
func NewServer(config *ServerConfig) (*Server, error) {
	// 创建加密器（优先使用AES-GCM硬件加速）
	masterKey := protocol.DeriveKeyFromPSK(config.AuthKey)
	cipher, err := protocol.NewCipher(masterKey, true) // true = 使用AES-GCM
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// 创建IP列表
	ipList := make([]net.IP, 0, len(config.ExitIPs))
	for _, ipStr := range config.ExitIPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return nil, fmt.Errorf("invalid IP address: %s", ipStr)
		}
		ipList = append(ipList, ip)
	}

	// 创建健康检查器（如果配置启用）
	var healthChecker *snat.IPHealthChecker
	if config.HealthCheck.Enabled && len(config.ExitIPs) > 0 {
		checkInterval := config.HealthCheck.Interval
		if checkInterval == 0 {
			checkInterval = 30 * time.Second
		}
		checkTimeout := config.HealthCheck.Timeout
		if checkTimeout == 0 {
			checkTimeout = 5 * time.Second
		}

		healthChecker = snat.NewIPHealthChecker(ipList, checkInterval, checkTimeout)
	}

	// 创建基础IP选择器
	var baseSelector snat.IPSelector
	switch config.Strategy {
	case "round_robin":
		baseSelector, err = snat.NewRoundRobinSelector(config.ExitIPs)
	case "destination_based":
		baseSelector, err = snat.NewDestinationBasedSelector(config.ExitIPs)
	case "load_balanced":
		strategyParam := config.StrategyParam
		if strategyParam == "" {
			strategyParam = "connections"
		}
		baseSelector, err = snat.NewLoadBalancedSelector(config.ExitIPs, strategyParam)
	default:
		baseSelector, err = snat.NewRoundRobinSelector(config.ExitIPs)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create IP selector: %w", err)
	}

	// 创建健康感知的IP选择器（包装基础选择器）
	var ipSelector snat.IPSelector
	if healthChecker != nil {
		ipSelector = snat.NewHealthAwareIPSelector(baseSelector, healthChecker, config.ExitIPs, config.Strategy, config.StrategyParam)
		// 启动健康检查
		go healthChecker.Start()
	} else {
		ipSelector = baseSelector
	}

	// 创建路由管理器
	var routingMgr *snat.RoutingManager
	if config.SNAT.Enabled {
		ips := make([]net.IP, 0, len(config.ExitIPs))
		for _, ipStr := range config.ExitIPs {
			ips = append(ips, net.ParseIP(ipStr))
		}
		routingMgr, err = snat.NewRoutingManager(ips, config.SNAT.Gateway, config.SNAT.Interface)
		if err != nil {
			return nil, fmt.Errorf("failed to create routing manager: %w", err)
		}

		// 设置路由规则
		if err := routingMgr.Setup(); err != nil {
			return nil, fmt.Errorf("failed to setup routing: %w", err)
		}
	}

	// 创建TLS监听器
	listener, err := transport.ListenTLS("tcp", config.ListenAddr, config.TLSConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	return &Server{
		config:        config,
		cipher:        cipher,
		ipSelector:    ipSelector,
		healthChecker: healthChecker,
		routingMgr:    routingMgr,
		listener:      listener,
	}, nil
}

// Start 启动服务端
func (s *Server) Start() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return err
		}

		go s.handleConn(conn)
	}
}

// Stop 停止服务端
func (s *Server) Stop() error {
	// 停止健康检查
	if s.healthChecker != nil {
		s.healthChecker.Stop()
	}

	if s.routingMgr != nil {
		s.routingMgr.Cleanup()
	}
	return s.listener.Close()
}

// handleConn 处理客户端连接
func (s *Server) handleConn(conn net.Conn) error {
	defer conn.Close()

	// 读取握手消息
	handshakeBuf := make([]byte, 32)
	if _, err := io.ReadFull(conn, handshakeBuf); err != nil {
		return err
	}

	// 解密握手消息
	decryptedHandshake, err := s.cipher.Decrypt(handshakeBuf)
	if err != nil {
		return fmt.Errorf("failed to decrypt handshake: %w", err)
	}

	handshake, err := protocol.DecodeHandshake(decryptedHandshake)
	if err != nil {
		return fmt.Errorf("failed to decode handshake: %w", err)
	}

	// 验证版本
	if handshake.Version != protocol.Version {
		return protocol.ErrInvalidVersion
	}

	// 验证时间戳（防重放攻击）
	now := time.Now().Unix()
	if abs(now-handshake.Timestamp) > 300 { // 5分钟窗口
		return fmt.Errorf("timestamp out of range")
	}

	// 验证HMAC
	if !s.cipher.VerifyHMAC(decryptedHandshake[:28], handshake.HMAC[:]) {
		return protocol.ErrAuthFailed
	}

	// 读取连接请求
	reqBuf := make([]byte, 1024)
	n, err := conn.Read(reqBuf)
	if err != nil {
		return err
	}

	decryptedReq, err := s.cipher.Decrypt(reqBuf[:n])
	if err != nil {
		return fmt.Errorf("failed to decrypt request: %w", err)
	}

	req, err := protocol.DecodeConnectRequest(decryptedReq)
	if err != nil {
		return fmt.Errorf("failed to decode request: %w", err)
	}

	// 构建目标地址
	targetAddr := protocol.BuildAddress(req.AddrType, req.Address, req.Port)
	if targetAddr == "" {
		return fmt.Errorf("invalid target address")
	}

	// 选择出口IP
	host, portStr, _ := net.SplitHostPort(targetAddr)
	targetPort, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	exitIP, err := s.ipSelector.SelectIP(host, targetPort)
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

	// 为每个连接创建独立的加密上下文（避免nonce冲突，提升并发性能）
	connCipher := protocol.NewConnectionCipher(s.cipher)

	// 发送成功响应（加密）
	response := []byte{0x00} // 成功
	encryptedResp, err := connCipher.Encrypt(response)
	if err != nil {
		return err
	}
	if _, err := conn.Write(encryptedResp); err != nil {
		return err
	}

	// 双向转发数据
	errCh := make(chan error, 2)

	go func() {
		errCh <- s.copyDataWithCipher(targetConn, conn, connCipher, true)
	}()

	go func() {
		errCh <- s.copyDataWithCipher(conn, targetConn, connCipher, false)
	}()

	// 等待任一方向出错
	err = <-errCh
	return err
}

// copyData 复制数据并加密/解密（使用buffer池优化，保留用于兼容）
func (s *Server) copyData(dst, src net.Conn, encrypt bool) error {
	connCipher := protocol.NewConnectionCipher(s.cipher)
	return s.copyDataWithCipher(dst, src, connCipher, encrypt)
}

// copyDataWithCipher 使用指定的加密上下文复制数据（连接级优化）
func (s *Server) copyDataWithCipher(dst, src net.Conn, cipher *protocol.ConnectionCipher, encrypt bool) error {
	// 从buffer池获取buffer
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)

	for {
		n, err := src.Read(buf)
		if n > 0 {
			var data []byte
			if encrypt {
				// 加密数据（使用连接级加密上下文）
				encrypted, err := cipher.Encrypt(buf[:n])
				if err != nil {
					return err
				}
				data = encrypted
			} else {
				// 解密数据
				decrypted, err := cipher.Decrypt(buf[:n])
				if err != nil {
					return err
				}
				data = decrypted
			}

			if _, err := dst.Write(data); err != nil {
				return err
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
