package proxy

import (
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"time"

	"multiexit-proxy/internal/protocol"
	"multiexit-proxy/internal/transport"
)

// Client 代理客户端
type Client struct {
	config     *ClientConfig
	cipher     *protocol.Cipher
	serverConn net.Conn
}

// ClientConfig 客户端配置
type ClientConfig struct {
	ServerAddr string
	SNI        string
	AuthKey    string
	LocalAddr  string
}

// NewClient 创建代理客户端
func NewClient(config *ClientConfig) (*Client, error) {
	// 创建加密器
	masterKey := protocol.DeriveKeyFromPSK(config.AuthKey)
	cipher, err := protocol.NewCipher(masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	return &Client{
		config: config,
		cipher: cipher,
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

	// 连接到服务端
	serverConn, err := c.connectToServer()
	if err != nil {
		c.sendSOCKS5Response(localConn, 0x05) // 连接失败
		return err
	}
	defer serverConn.Close()

	// 发送握手消息
	if err := c.sendHandshake(serverConn); err != nil {
		c.sendSOCKS5Response(localConn, 0x05)
		return err
	}

	// 发送连接请求
	if err := c.sendConnectRequest(serverConn, addr); err != nil {
		c.sendSOCKS5Response(localConn, 0x05)
		return err
	}

	// 读取响应
	response, err := c.readResponse(serverConn)
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
		errCh <- c.copyData(serverConn, localConn, true)
	}()

	go func() {
		errCh <- c.copyData(localConn, serverConn, false)
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

// sendHandshake 发送握手消息
func (c *Client) sendHandshake(conn net.Conn) error {
	nonce := make([]byte, 16)
	rand.Read(nonce)

	handshake := &protocol.HandshakeMessage{
		Version:   protocol.Version,
		Method:    0x01,
		Reserved:  0x0000,
		Nonce:     [16]byte(nonce),
		Timestamp: time.Now().Unix(),
	}

	// 计算HMAC
	handshakeData := protocol.EncodeHandshake(handshake)
	handshake.HMAC = [4]byte(c.cipher.ComputeHMAC(handshakeData[:28]))

	// 加密并发送
	encrypted := protocol.EncodeHandshake(handshake)
	ciphertext, err := c.cipher.Encrypt(encrypted)
	if err != nil {
		return err
	}

	_, err = conn.Write(ciphertext)
	return err
}

// sendConnectRequest 发送连接请求
func (c *Client) sendConnectRequest(conn net.Conn, addr string) error {
	addrType, address, err := protocol.ParseAddress(addr)
	if err != nil {
		return err
	}

	_, portStr, _ := net.SplitHostPort(addr)
	var port uint16
	fmt.Sscanf(portStr, "%d", &port)

	req := &protocol.ConnectRequest{
		Type:     protocol.MsgTypeConnect,
		AddrType: addrType,
		AddrLen:  uint8(len(address)),
		Address:  address,
		Port:     port,
	}

	reqData := protocol.EncodeConnectRequest(req)
	ciphertext, err := c.cipher.Encrypt(reqData)
	if err != nil {
		return err
	}

	_, err = conn.Write(ciphertext)
	return err
}

// readResponse 读取响应
func (c *Client) readResponse(conn net.Conn) ([]byte, error) {
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	plaintext, err := c.cipher.Decrypt(buf[:n])
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

// copyData 复制数据并加密/解密
func (c *Client) copyData(dst, src net.Conn, encrypt bool) error {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			var data []byte
			if encrypt {
				// 加密数据
				encrypted, err := c.cipher.Encrypt(buf[:n])
				if err != nil {
					return err
				}
				data = encrypted
			} else {
				// 解密数据
				decrypted, err := c.cipher.Decrypt(buf[:n])
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

