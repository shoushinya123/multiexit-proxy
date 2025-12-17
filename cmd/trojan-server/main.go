package main

import (
	"crypto/tls"
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"

	"multiexit-proxy/internal/config"
	"multiexit-proxy/internal/snat"
	"multiexit-proxy/internal/trojan"

	"github.com/sirupsen/logrus"
)

func main() {
	configPath := flag.String("config", "configs/server.yaml", "Path to server config file")
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadServerConfig(*configPath)
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Logging.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	// 检查Trojan是否启用
	if !cfg.Trojan.Enabled {
		logrus.Fatalf("Trojan protocol is not enabled in config")
	}

	if cfg.Trojan.Password == "" {
		logrus.Fatalf("Trojan password is required")
	}

	// 加载TLS证书
	var tlsConfig *tls.Config
	if cfg.Server.TLS.Cert != "" && cfg.Server.TLS.Key != "" {
		cert, err := tls.LoadX509KeyPair(cfg.Server.TLS.Cert, cfg.Server.TLS.Key)
		if err != nil {
			logrus.Fatalf("Failed to load TLS certificate: %v", err)
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"h2", "http/1.1"},
		}
	} else {
		logrus.Fatalf("TLS certificate is required for Trojan protocol")
	}

	// 创建IP选择器
	var ipSelector snat.IPSelector
	switch cfg.Strategy.Type {
	case "round_robin":
		ipSelector, err = snat.NewRoundRobinSelector(cfg.ExitIPs)
	case "destination_based":
		ipSelector, err = snat.NewDestinationBasedSelector(cfg.ExitIPs)
	default:
		ipSelector, err = snat.NewRoundRobinSelector(cfg.ExitIPs)
	}
	if err != nil {
		logrus.Fatalf("Failed to create IP selector: %v", err)
	}

	// 创建路由管理器
	var routingMgr *snat.RoutingManager
	if cfg.SNAT.Enabled {
		ips := make([]net.IP, 0, len(cfg.ExitIPs))
		for _, ipStr := range cfg.ExitIPs {
			ips = append(ips, net.ParseIP(ipStr))
		}
		routingMgr, err = snat.NewRoutingManager(ips, cfg.SNAT.Gateway, cfg.SNAT.Interface)
		if err != nil {
			logrus.Fatalf("Failed to create routing manager: %v", err)
		}

		if err := routingMgr.Setup(); err != nil {
			logrus.Fatalf("Failed to setup routing: %v", err)
		}
	}

	// 创建Trojan服务器
	serverConfig := &trojan.ServerConfig{
		ListenAddr: cfg.Server.Listen,
		Password:   cfg.Trojan.Password,
		TLSConfig:  tlsConfig,
		IPSelector: ipSelector,
		RoutingMgr: routingMgr,
	}

	server, err := trojan.NewServer(serverConfig)
	if err != nil {
		logrus.Fatalf("Failed to create Trojan server: %v", err)
	}

	// 处理信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		logrus.Info("Shutting down server...")
		if err := server.Stop(); err != nil {
			logrus.Errorf("Error stopping server: %v", err)
		}
		os.Exit(0)
	}()

	logrus.Infof("Trojan server starting on %s", cfg.Server.Listen)
	if err := server.Start(); err != nil {
		logrus.Fatalf("Server error: %v", err)
	}
}
