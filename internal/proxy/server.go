package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"multiexit-proxy/internal/monitor"
	"multiexit-proxy/internal/protocol"
	"multiexit-proxy/internal/snat"
	"multiexit-proxy/internal/transport"

	"github.com/sirupsen/logrus"
)

// Server 代理服务端
type Server struct {
	config          *ServerConfig
	cipher          *protocol.Cipher
	ipSelector      snat.IPSelector
	healthChecker   *snat.IPHealthChecker
	routingMgr      *snat.RoutingManager
	listener        net.Listener
	connManager     *ConnectionManager
	statsManager    *monitor.StatsManager
	trafficAnalyzer *monitor.TrafficAnalyzer // 流量分析器
	ruleEngine      *RuleEngine              // 规则引擎
	rateLimiter     *RateLimiter             // 速率限制器
	shutdownCtx     context.Context
	shutdownCancel  context.CancelFunc
	shutdownWg      sync.WaitGroup
	accepting       int32 // 原子操作，是否正在接受连接
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
	Connection struct {
		ReadTimeout    time.Duration
		WriteTimeout   time.Duration
		IdleTimeout    time.Duration
		DialTimeout    time.Duration
		MaxConnections int
		KeepAlive      bool
		KeepAliveTime  time.Duration
	}
	SNAT struct {
		Enabled   bool
		Gateway   string
		Interface string
	}
	EnableStats bool // 是否启用统计
	GeoLocation struct {
		Enabled         bool
		APIURL          string
		LatencyOptimize bool
		DBPath          string // GeoIP数据库路径（可选）
	}
	Rules []struct {
		Name        string
		Priority    int
		MatchDomain []string
		MatchIP     []string
		MatchPort   []int
		TargetIP    string
		Action      string
		Enabled     bool
	}
	EnableTrafficAnalysis bool
	TrafficAnalysis       struct {
		Enabled          bool
		TrendWindow      time.Duration
		AnomalyThreshold float64
	}
	RuleEngine struct {
		Enabled bool
		Rules   []*Rule
	}
	Cluster struct {
		Enabled  bool
		Nodes    []ClusterNodeConfig
		Strategy string
	}
	RateLimit struct {
		Enabled              bool
		IPMaxConnections     int
		IPRateLimit          int
		UserMaxConnections   int
		UserRateLimit        int
		UserBandwidthLimit   int64
		GlobalMaxConnections int
		GlobalRateLimit      int
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

	// 如果启用地理位置选择，包装选择器
	if config.GeoLocation.Enabled {
		geoService := snat.NewGeoLocationService(config.GeoLocation.APIURL)

		// 如果提供了GeoIP数据库路径，使用增强的地理位置服务
		if config.GeoLocation.DBPath != "" {
			enhancedService, err := snat.NewEnhancedGeoLocationService(
				config.GeoLocation.DBPath,
				config.GeoLocation.APIURL,
				true, // 优先使用本地数据库
			)
			if err != nil {
				logrus.Warnf("Failed to create enhanced geo service: %v, using API only", err)
				enhancedService = nil
			}

			if enhancedService != nil {
				geoSelector, err := snat.NewGeoLocationSelector(ipSelector, enhancedService, config.ExitIPs)
				if err != nil {
					return nil, fmt.Errorf("failed to create geo location selector: %w", err)
				}
				ipSelector = geoSelector
			} else {
				geoSelector, err := snat.NewGeoLocationSelector(ipSelector, geoService, config.ExitIPs)
				if err != nil {
					return nil, fmt.Errorf("failed to create geo location selector: %w", err)
				}
				ipSelector = geoSelector
			}
		} else {
			geoSelector, err := snat.NewGeoLocationSelector(ipSelector, geoService, config.ExitIPs)
			if err != nil {
				return nil, fmt.Errorf("failed to create geo location selector: %w", err)
			}
			ipSelector = geoSelector
		}
	}

	// 如果配置了规则引擎，包装选择器
	if len(config.Rules) > 0 {
		ruleEngine := snat.NewRuleEngine()
		for _, ruleConfig := range config.Rules {
			rule := &snat.Rule{
				Name:        ruleConfig.Name,
				Priority:    ruleConfig.Priority,
				MatchDomain: ruleConfig.MatchDomain,
				MatchIP:     ruleConfig.MatchIP,
				MatchPort:   ruleConfig.MatchPort,
				TargetIP:    ruleConfig.TargetIP,
				Action:      ruleConfig.Action,
				Enabled:     ruleConfig.Enabled,
			}
			if err := ruleEngine.AddRule(rule); err != nil {
				return nil, fmt.Errorf("failed to add rule %s: %w", rule.Name, err)
			}
		}
		ruleSelector := snat.NewRuleBasedSelector(ipSelector, ruleEngine, config.ExitIPs)
		ipSelector = ruleSelector
		logrus.Infof("Rule engine enabled with %d rules", len(config.Rules))
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

	// 创建连接管理器
	connManager := NewConnectionManager(
		config.Connection.MaxConnections,
		config.Connection.ReadTimeout,
		config.Connection.WriteTimeout,
		config.Connection.IdleTimeout,
		config.Connection.DialTimeout,
		config.Connection.KeepAlive,
		config.Connection.KeepAliveTime,
	)

	// 创建统计管理器
	var statsManager *monitor.StatsManager
	if config.EnableStats {
		statsManager = monitor.NewStatsManager()
	}

	// 创建流量分析器
	var trafficAnalyzer *monitor.TrafficAnalyzer
	if config.TrafficAnalysis.Enabled {
		trendWindow := config.TrafficAnalysis.TrendWindow
		if trendWindow == 0 {
			trendWindow = 1 * time.Hour
		}
		anomalyThreshold := config.TrafficAnalysis.AnomalyThreshold
		if anomalyThreshold == 0 {
			anomalyThreshold = 2.0
		}
		trafficAnalyzer = monitor.NewTrafficAnalyzer(trendWindow, anomalyThreshold)

		// 启动趋势记录goroutine
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				trafficAnalyzer.RecordTrend()
			}
		}()
	}

	// 创建规则引擎
	var ruleEngine *RuleEngine
	if config.RuleEngine.Enabled {
		ruleEngine = NewRuleEngine()
		// 加载规则
		for _, rule := range config.RuleEngine.Rules {
			if err := ruleEngine.AddRule(rule); err != nil {
				logrus.Warnf("Failed to add rule %s: %v", rule.ID, err)
			}
		}
	}

	// 创建速率限制器
	var rateLimiter *RateLimiter
	if config.RateLimit.Enabled {
		rateLimiter = NewRateLimiter(RateLimitConfig{
			IPMaxConnections:     config.RateLimit.IPMaxConnections,
			IPRateLimit:          config.RateLimit.IPRateLimit,
			UserMaxConnections:   config.RateLimit.UserMaxConnections,
			UserRateLimit:        config.RateLimit.UserRateLimit,
			UserBandwidthLimit:   config.RateLimit.UserBandwidthLimit,
			GlobalMaxConnections: config.RateLimit.GlobalMaxConnections,
			GlobalRateLimit:      config.RateLimit.GlobalRateLimit,
		})
		logrus.Info("Rate limiter enabled")
	}

	// 创建关闭上下文（使用可取消的上下文）
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	return &Server{
		config:          config,
		cipher:          cipher,
		ipSelector:      ipSelector,
		healthChecker:   healthChecker,
		routingMgr:      routingMgr,
		listener:        listener,
		connManager:     connManager,
		statsManager:    statsManager,
		trafficAnalyzer: trafficAnalyzer,
		ruleEngine:      ruleEngine,
		rateLimiter:     rateLimiter,
		shutdownCtx:     shutdownCtx,
		shutdownCancel:  shutdownCancel,
		accepting:       1,
	}, nil
}

// Start 启动服务端
func (s *Server) Start() error {
	logrus.Info("Server started, accepting connections")
	for {
		// 检查是否正在关闭
		if atomic.LoadInt32(&s.accepting) == 0 {
			return nil
		}

		conn, err := s.listener.Accept()
		if err != nil {
			// 如果正在关闭，忽略错误
			if atomic.LoadInt32(&s.accepting) == 0 {
				return nil
			}
			return err
		}

		// 获取客户端IP
		clientIP := getClientIP(conn)

		// 检查全局连接数限制
		if !s.connManager.CanAccept() {
			logrus.Warnf("Max connections reached (%d), rejecting new connection from %s", s.connManager.GetActiveCount(), clientIP)
			conn.Close()
			continue
		}

		// 检查速率限制
		if s.rateLimiter != nil {
			// 检查全局限流
			if !s.rateLimiter.CheckGlobal() {
				logrus.Debugf("Global rate limit exceeded, rejecting connection from %s", clientIP)
				conn.Close()
				continue
			}
			defer s.rateLimiter.OnGlobalConnectionEnd()

			// 检查IP限流
			if !s.rateLimiter.CheckIP(clientIP) {
				logrus.Debugf("IP rate limit exceeded for %s, rejecting connection", clientIP)
				conn.Close()
				s.rateLimiter.OnGlobalConnectionEnd()
				continue
			}
			defer s.rateLimiter.OnIPConnectionEnd(clientIP)
		}

		// 添加连接管理
		if !s.connManager.AddConnection(conn) {
			conn.Close()
			continue
		}

		s.shutdownWg.Add(1)
		go func(c net.Conn) {
			defer s.shutdownWg.Done()
			defer s.connManager.RemoveConnection(c)
			s.handleConn(c)
		}(conn)
	}
}

// Stop 停止服务端（立即关闭）
func (s *Server) Stop() error {
	return s.Shutdown(0)
}

// Shutdown 优雅关闭服务端
func (s *Server) Shutdown(timeout time.Duration) error {
	logrus.Info("Shutting down server...")

	// 停止接受新连接
	atomic.StoreInt32(&s.accepting, 0)

	// 关闭监听器
	if s.listener != nil {
		s.listener.Close()
	}

	// 触发关闭上下文
	s.shutdownCancel()

	// 等待所有连接完成
	if timeout > 0 {
		done := make(chan struct{})
		go func() {
			s.shutdownWg.Wait()
			close(done)
		}()

		select {
		case <-done:
			logrus.Info("All connections closed gracefully")
		case <-time.After(timeout):
			logrus.Warnf("Shutdown timeout (%v), forcing close", timeout)
			s.connManager.CloseAll()
		}
	} else {
		// 立即关闭所有连接
		s.connManager.CloseAll()
	}

	// 停止健康检查
	if s.healthChecker != nil {
		s.healthChecker.Stop()
	}

	// 清理路由
	if s.routingMgr != nil {
		s.routingMgr.Cleanup()
	}

	logrus.Info("Server shutdown complete")
	return nil
}

// GetStats 获取统计信息
func (s *Server) GetStats() interface{} {
	if s.statsManager != nil {
		return s.statsManager.GetStats()
	}
	return nil
}

// GetTrafficAnalyzer 获取流量分析器（用于Web界面）
func (s *Server) GetTrafficAnalyzer() *monitor.TrafficAnalyzer {
	return s.trafficAnalyzer
}

// GetRuleEngine 获取规则引擎（用于Web界面）
func (s *Server) GetRuleEngine() *RuleEngine {
	return s.ruleEngine
}

// handleConn 处理客户端连接
func (s *Server) handleConn(conn net.Conn) error {
	connStartTime := time.Now()
	var exitIP net.IP
	var targetAddr string
	var bytesUp, bytesDown int64 // 用于流量分析

	defer func() {
		// 记录连接结束统计
		if s.statsManager != nil && exitIP != nil {
			duration := time.Since(connStartTime)
			s.statsManager.OnConnectionEnd(exitIP, duration)
		}

		// 记录流量分析（如果启用）
		if s.trafficAnalyzer != nil && targetAddr != "" {
			host, _, _ := net.SplitHostPort(targetAddr)
			if host != "" {
				duration := time.Since(connStartTime)
				s.trafficAnalyzer.RecordDomainAccess(host, bytesUp, bytesDown, duration)
			}
		}

		conn.Close()
	}()

	// 设置初始超时
	s.connManager.SetTimeouts(conn, s.config.Connection.ReadTimeout, s.config.Connection.WriteTimeout)

	// 读取握手消息
	handshakeBuf := make([]byte, 32)
	s.connManager.ResetReadDeadline(conn)
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
	s.connManager.ResetReadDeadline(conn)
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
	targetAddr = protocol.BuildAddress(req.AddrType, req.Address, req.Port)
	if targetAddr == "" {
		return fmt.Errorf("invalid target address")
	}

	host, portStr, err := net.SplitHostPort(targetAddr)
	if err != nil {
		return fmt.Errorf("invalid target address: %w", err)
	}
	targetPort, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}
	_ = host // 避免未使用变量警告

	// 规则引擎匹配
	var selectedIP net.IP
	if s.ruleEngine != nil {
		rule, err := s.ruleEngine.Match(targetAddr)
		if err == nil && rule != nil {
			switch rule.Action {
			case "block":
				return fmt.Errorf("connection blocked by rule: %s", rule.Name)
			case "use_ip":
				selectedIP = net.ParseIP(rule.TargetIP)
				if selectedIP == nil {
					return fmt.Errorf("invalid target IP in rule: %s", rule.TargetIP)
				}
				logrus.Debugf("Using IP %s from rule %s for %s", rule.TargetIP, rule.Name, targetAddr)
			case "redirect":
				// 重定向到新地址（这里简化处理，使用规则中的目标IP）
				if rule.TargetIP != "" {
					selectedIP = net.ParseIP(rule.TargetIP)
					if selectedIP != nil {
						logrus.Debugf("Redirecting %s to %s via rule %s", targetAddr, rule.TargetIP, rule.Name)
					}
				}
			}
		}
	}

	// 如果没有规则匹配或规则没有指定IP，使用选择器
	if selectedIP == nil {
		selectedIP, err = s.ipSelector.SelectIP(host, targetPort)
		if err != nil {
			return fmt.Errorf("failed to select IP: %w", err)
		}
	}
	exitIP = selectedIP // 赋值给defer中使用的变量

	// 记录流量分析（如果启用）
	if s.trafficAnalyzer != nil && host != "" {
		// 记录域名访问（延迟在连接结束时计算）
		s.trafficAnalyzer.RecordDomainAccess(host, 0, 0, 0)
	}

	// 记录连接开始统计
	if s.statsManager != nil {
		s.statsManager.OnConnectionStart(exitIP)
	}

	// 建立到目标的连接（使用连接管理器的超时）
	var targetConn net.Conn
	if s.routingMgr != nil {
		// 使用SNAT
		targetConn, err = s.connManager.DialWithTimeout("tcp", targetAddr)
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
		targetConn, err = s.connManager.DialWithTimeout("tcp", targetAddr)
		if err != nil {
			return fmt.Errorf("failed to dial target: %w", err)
		}
	}
	defer targetConn.Close()

	// 注意：targetConn已经在defer中关闭，不需要添加到管理器
	// 只在需要超时管理时添加

	// 设置目标连接超时
	s.connManager.SetTimeouts(targetConn, s.config.Connection.ReadTimeout, s.config.Connection.WriteTimeout)

	// 为每个连接创建独立的加密上下文（避免nonce冲突，提升并发性能）
	connCipher := protocol.NewConnectionCipher(s.cipher)

	// 发送成功响应（加密）
	response := []byte{0x00} // 成功
	encryptedResp, err := connCipher.Encrypt(response)
	if err != nil {
		return err
	}
	s.connManager.ResetWriteDeadline(conn)
	if _, err := conn.Write(encryptedResp); err != nil {
		return err
	}

	// 双向转发数据
	errCh := make(chan error, 2)

	go func() {
		errCh <- s.copyDataWithCipher(targetConn, conn, connCipher, true, exitIP, &bytesUp, &bytesDown)
	}()

	go func() {
		errCh <- s.copyDataWithCipher(conn, targetConn, connCipher, false, exitIP, &bytesUp, &bytesDown)
	}()

	// 等待任一方向出错
	err = <-errCh
	return err
}

// copyData 复制数据并加密/解密（使用buffer池优化，保留用于兼容）
func (s *Server) copyData(dst, src net.Conn, encrypt bool, exitIP net.IP, bytesUp, bytesDown *int64) error {
	connCipher := protocol.NewConnectionCipher(s.cipher)
	return s.copyDataWithCipher(dst, src, connCipher, encrypt, exitIP, bytesUp, bytesDown)
}

// copyDataWithCipher 使用指定的加密上下文复制数据（连接级优化）
func (s *Server) copyDataWithCipher(dst, src net.Conn, cipher *protocol.ConnectionCipher, encrypt bool, exitIP net.IP, bytesUp, bytesDown *int64) error {
	// 从buffer池获取buffer
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)

	var totalBytes int64

	for {
		// 检查是否正在关闭
		select {
		case <-s.shutdownCtx.Done():
			return s.shutdownCtx.Err()
		default:
		}

		// 重置读超时
		s.connManager.ResetReadDeadline(src)
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

			// 重置写超时
			s.connManager.ResetWriteDeadline(dst)
			if _, err := dst.Write(data); err != nil {
				return err
			}

			// 更新流量统计
			bytes := int64(len(data))
			totalBytes += bytes
			if s.statsManager != nil && exitIP != nil {
				if encrypt {
					s.statsManager.OnBytesTransferred(exitIP, 0, bytes) // 下行
					if bytesDown != nil {
						atomic.AddInt64(bytesDown, bytes)
					}
				} else {
					s.statsManager.OnBytesTransferred(exitIP, bytes, 0) // 上行
					if bytesUp != nil {
						atomic.AddInt64(bytesUp, bytes)
					}
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			// 检查是否是超时错误
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return fmt.Errorf("timeout: %w", err)
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

// getClientIP 获取客户端IP
func getClientIP(conn net.Conn) net.IP {
	addr := conn.RemoteAddr()
	if tcpAddr, ok := addr.(*net.TCPAddr); ok {
		return tcpAddr.IP
	}
	host, _, _ := net.SplitHostPort(addr.String())
	if ip := net.ParseIP(host); ip != nil {
		return ip
	}
	return net.IPv4(0, 0, 0, 0)
}
