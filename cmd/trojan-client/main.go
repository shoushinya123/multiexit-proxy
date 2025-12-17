package main

import (
	"crypto/tls"
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"

	"multiexit-proxy/internal/config"
	"multiexit-proxy/internal/trojan"
	"multiexit-proxy/pkg/socks5"

	"github.com/sirupsen/logrus"
)

func main() {
	configPath := flag.String("config", "configs/client.json", "Path to client config file")
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadClientConfig(*configPath)
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Logging.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	// 创建TLS配置
	tlsConfig := &tls.Config{
		ServerName:         cfg.Server.SNI,
		InsecureSkipVerify: true, // 可以根据需要改为false并配置CA
		NextProtos:         []string{"h2", "http/1.1"},
	}

	// 创建Trojan客户端
	trojanClient := trojan.NewClient(&trojan.ClientConfig{
		ServerAddr: cfg.Server.Address,
		Password:   cfg.Auth.Key, // 使用auth.key作为Trojan密码
		TLSConfig:  tlsConfig,
	})

	// 创建SOCKS5服务器
	socks5Server := socks5.NewServer(func(network, addr string) (net.Conn, error) {
		return trojanClient.Dial(addr)
	})

	// 创建本地监听器
	listener, err := net.Listen("tcp", cfg.Local.SOCKS5)
	if err != nil {
		logrus.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	// 处理信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		logrus.Info("Shutting down client...")
		os.Exit(0)
	}()

	logrus.Infof("Trojan client starting, listening on %s", cfg.Local.SOCKS5)
	for {
		conn, err := listener.Accept()
		if err != nil {
			logrus.Errorf("Accept error: %v", err)
			continue
		}

		go socks5Server.HandleConn(conn)
	}
}


