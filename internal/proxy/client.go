package proxy

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"multiexit-proxy/internal/protocol"
	"multiexit-proxy/internal/transport"

	"github.com/sirupsen/logrus"
)

// Client 代理客户端
type Client struct {
	config       *ClientConfig
	cipher       *protocol.Cipher
	serverConn   net.Conn
	reconnectMgr *ReconnectManager
	connPool     *ConnectionPool // 连接池（可选）
}

// ClientConfig 客户端配置
type ClientConfig struct {
	ServerAddr string
	SNI        string
	AuthKey    string
	LocalAddr  string
	Reconnect  *ReconnectConfig // 重连配置
	Pool       struct {
		Enabled     bool
		MaxSize     int
		MaxIdle     int
		IdleTimeout time.Duration
	}
}

// NewClient 创建代理客户端
func NewClient(config *ClientConfig) (*Client, error) {
	// 创建加密器（优先使用AES-GCM硬件加速）
	masterKey := protocol.DeriveKeyFromPSK(config.AuthKey)
	cipher, err := protocol.NewCipher(masterKey, true) // true = 使用AES-GCM
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// 创建重连管理器
	var reconnectMgr *ReconnectManager
	if config.Reconnect != nil {
		reconnectMgr = NewReconnectManager(*config.Reconnect)
	} else {
		// 使用默认配置
		reconnectMgr = NewReconnectManager(ReconnectConfig{
			MaxRetries:    0, // 无限重试
			InitialDelay:  1 * time.Second,
			MaxDelay:      5 * time.Minute,
			BackoffFactor: 2.0,
			Jitter:        true,
		})
	}

	// 创建连接池（如果启用）
	var connPool *ConnectionPool
	if config.Pool.Enabled {
		maxSize := config.Pool.MaxSize
		if maxSize == 0 {
			maxSize = 10 // 默认值
		}
		maxIdle := config.Pool.MaxIdle
		if maxIdle == 0 {
			maxIdle = 5 // 默认值
		}
		idleTimeout := config.Pool.IdleTimeout
		if idleTimeout == 0 {
			idleTimeout = 5 * time.Minute // 默认值
		}

		dialFunc := func() (net.Conn, error) {
			tlsConfig := &transport.ClientTLSConfig{
				SNI: config.SNI,
			}
			return transport.DialTLS("tcp", config.ServerAddr, tlsConfig)
		}

		connPool = NewConnectionPool(dialFunc, maxSize, maxIdle, idleTimeout)
		logrus.Info("Connection pool enabled for client")
	}

	return &Client{
		config:       config,
		cipher:       cipher,
		reconnectMgr: reconnectMgr,
		connPool:     connPool,
	}, nil
}

// Start 启动客户端
func (c *Client) Start() error {
	// 创建本地SOCKS5监听器
	listener, err := net.Listen("tcp", c.config.LocalAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", c.config.LocalAddr, err)
	}
	defer listener.Close()

	for {
		localConn, err := listener.Accept()
		if err != nil {
			continue
		}

		go c.handleLocalConn(localConn)
	}
}

// handleLocalConn 处理本地连接
func (c *Client) handleLocalConn(localConn net.Conn) error {
	defer localConn.Close()

	// 读取SOCKS5请求
	addr, err := c.readSOCKS5Request(localConn)
	if err != nil {
		return err
	}

	// 连接到服务端（带重连和连接池）
	var serverConn net.Conn
	// 使用可取消的上下文（支持超时和取消）
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 如果启用了连接池，优先使用连接池
	if c.connPool != nil {
		pooledConn, err := c.connPool.Get(ctx)
		if err == nil {
			serverConn = pooledConn
			defer func() {
				// 连接使用完毕后归还到池中
				if pooledConn, ok := serverConn.(*PooledConnection); ok {
					c.connPool.Return(pooledConn)
				} else {
					serverConn.Close()
				}
			}()
		} else {
			// 连接池获取失败，回退到直接连接
			logrus.Debugf("Failed to get connection from pool: %v, falling back to direct connection", err)
		}
	}

	// 如果没有使用连接池，使用重连管理器连接
	if serverConn == nil {
		err = c.reconnectMgr.Do(ctx, func() error {
			conn, err := c.connectToServer()
			if err != nil {
				return err
			}
			serverConn = conn
			return nil
		})
		if err != nil {
			c.sendSOCKS5Response(localConn, 0x05) // 连接失败
			return fmt.Errorf("failed to connect to server after retries: %w", err)
		}
		defer serverConn.Close()
	}

	// 为每个连接创建独立的加密上下文（避免nonce冲突，提升并发性能）
	connCipher := protocol.NewConnectionCipher(c.cipher)

	// 发送握手消息
	if err := c.sendHandshakeWithCipher(serverConn, connCipher); err != nil {
		c.sendSOCKS5Response(localConn, 0x05)
		return err
	}

	// 发送连接请求
	if err := c.sendConnectRequestWithCipher(serverConn, addr, connCipher); err != nil {
		c.sendSOCKS5Response(localConn, 0x05)
		return err
	}

	// 读取响应
	response, err := c.readResponseWithCipher(serverConn, connCipher)
	if err != nil || response[0] != 0x00 {
		c.sendSOCKS5Response(localConn, 0x05)
		return fmt.Errorf("server rejected connection")
	}

	// 发送SOCKS5成功响应
	if err := c.sendSOCKS5Response(localConn, 0x00); err != nil {
		return err
	}

	// 双向转发数据
	errCh := make(chan error, 2)

	go func() {
		errCh <- c.copyDataWithCipher(serverConn, localConn, connCipher, true)
	}()

	go func() {
		errCh <- c.copyDataWithCipher(localConn, serverConn, connCipher, false)
	}()

	err = <-errCh
	return err
}

// connectToServer 连接到服务端
func (c *Client) connectToServer() (net.Conn, error) {
	tlsConfig := &transport.ClientTLSConfig{
		SNI: c.config.SNI,
	}
	return transport.DialTLS("tcp", c.config.ServerAddr, tlsConfig)
}

// sendHandshake 发送握手消息（保留用于兼容）
func (c *Client) sendHandshake(conn net.Conn) error {
	return c.sendHandshakeWithCipher(conn, protocol.NewConnectionCipher(c.cipher))
}

// sendHandshakeWithCipher 使用指定的加密上下文发送握手消息
func (c *Client) sendHandshakeWithCipher(conn net.Conn, cipher *protocol.ConnectionCipher) error {
	nonce := make([]byte, 16)
	rand.Read(nonce)

	handshake := &protocol.HandshakeMessage{
		Version:   protocol.Version,
		Method:    0x01,
		Reserved:  0x0000,
		Nonce:     [16]byte(nonce),
		Timestamp: time.Now().Unix(),
	}

	// 计算HMAC（使用主cipher）
	handshakeData := protocol.EncodeHandshake(handshake)
	handshake.HMAC = [4]byte(c.cipher.ComputeHMAC(handshakeData[:28]))

	// 加密并发送（使用连接级加密上下文）
	encrypted := protocol.EncodeHandshake(handshake)
	ciphertext, err := cipher.Encrypt(encrypted)
	if err != nil {
		return err
	}

	_, err = conn.Write(ciphertext)
	return err
}

// sendConnectRequest 发送连接请求（保留用于兼容）
func (c *Client) sendConnectRequest(conn net.Conn, addr string) error {
	return c.sendConnectRequestWithCipher(conn, addr, protocol.NewConnectionCipher(c.cipher))
}

// sendConnectRequestWithCipher 使用指定的加密上下文发送连接请求
func (c *Client) sendConnectRequestWithCipher(conn net.Conn, addr string, cipher *protocol.ConnectionCipher) error {
	addrType, address, err := protocol.ParseAddress(addr)
	if err != nil {
		return err
	}

	_, portStr, _ := net.SplitHostPort(addr)
	portInt, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}
	port := uint16(portInt)

	req := &protocol.ConnectRequest{
		Type:     protocol.MsgTypeConnect,
		AddrType: addrType,
		AddrLen:  uint8(len(address)),
		Address:  address,
		Port:     port,
	}

	reqData := protocol.EncodeConnectRequest(req)
	ciphertext, err := cipher.Encrypt(reqData)
	if err != nil {
		return err
	}

	_, err = conn.Write(ciphertext)
	return err
}

// readResponse 读取响应（保留用于兼容）
func (c *Client) readResponse(conn net.Conn) ([]byte, error) {
	return c.readResponseWithCipher(conn, protocol.NewConnectionCipher(c.cipher))
}

// readResponseWithCipher 使用指定的加密上下文读取响应
func (c *Client) readResponseWithCipher(conn net.Conn, cipher *protocol.ConnectionCipher) ([]byte, error) {
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	plaintext, err := cipher.Decrypt(buf[:n])
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// readSOCKS5Request 读取SOCKS5请求（简化版）
func (c *Client) readSOCKS5Request(conn net.Conn) (string, error) {
	buf := make([]byte, 256)

	// 读取版本和方法
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return "", err
	}

	if buf[0] != 0x05 {
		return "", fmt.Errorf("invalid SOCKS version")
	}

	// 读取方法列表
	nMethods := int(buf[1])
	if _, err := io.ReadFull(conn, buf[:nMethods]); err != nil {
		return "", err
	}

	// 发送无认证响应
	conn.Write([]byte{0x05, 0x00})

	// 读取请求
	if _, err := io.ReadFull(conn, buf[:4]); err != nil {
		return "", err
	}

	if buf[0] != 0x05 || buf[1] != 0x01 {
		return "", fmt.Errorf("unsupported command")
	}

	// 读取地址
	addrType := buf[3]
	var addr string

	switch addrType {
	case 0x01: // IPv4
		if _, err := io.ReadFull(conn, buf[:6]); err != nil {
			return "", err
		}
		ip := net.IP(buf[:4])
		port := uint16(buf[4])<<8 | uint16(buf[5])
		addr = fmt.Sprintf("%s:%d", ip.String(), port)

	case 0x03: // Domain
		if _, err := io.ReadFull(conn, buf[:1]); err != nil {
			return "", err
		}
		domainLen := int(buf[0])
		domain := make([]byte, domainLen)
		if _, err := io.ReadFull(conn, domain); err != nil {
			return "", err
		}
		if _, err := io.ReadFull(conn, buf[:2]); err != nil {
			return "", err
		}
		port := uint16(buf[0])<<8 | uint16(buf[1])
		addr = fmt.Sprintf("%s:%d", string(domain), port)

	default:
		return "", fmt.Errorf("unsupported address type")
	}

	return addr, nil
}

// sendSOCKS5Response 发送SOCKS5响应
func (c *Client) sendSOCKS5Response(conn net.Conn, reply byte) error {
	response := []byte{0x05, reply, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	_, err := conn.Write(response)
	return err
}

// copyData 复制数据并加密/解密（使用buffer池优化，保留用于兼容）
func (c *Client) copyData(dst, src net.Conn, encrypt bool) error {
	return c.copyDataWithCipher(dst, src, protocol.NewConnectionCipher(c.cipher), encrypt)
}

// copyDataWithCipher 使用指定的加密上下文复制数据（连接级优化）
func (c *Client) copyDataWithCipher(dst, src net.Conn, cipher *protocol.ConnectionCipher, encrypt bool) error {
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
