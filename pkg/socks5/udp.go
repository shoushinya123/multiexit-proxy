package socks5

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"time"
)

// UDPRelay UDP中继服务器
type UDPRelay struct {
	conn       net.PacketConn
	timeout    time.Duration
	associations map[string]*UDPAssociation
}

// UDPAssociation UDP关联
type UDPAssociation struct {
	clientAddr net.Addr
	created    time.Time
}

// NewUDPRelay 创建UDP中继服务器
func NewUDPRelay(listenAddr string, timeout time.Duration) (*UDPRelay, error) {
	conn, err := net.ListenPacket("udp", listenAddr)
	if err != nil {
		return nil, err
	}

	relay := &UDPRelay{
		conn:       conn,
		timeout:    timeout,
		associations: make(map[string]*UDPAssociation),
	}

	go relay.cleanupAssociations()

	return relay, nil
}

// HandleUDPRequest 处理UDP请求
func (s *Server) HandleUDPRequest(conn net.Conn) error {
	// 获取客户端地址用于UDP关联
	clientAddr := conn.RemoteAddr()
	
	// 创建UDP监听器（绑定到随机端口）
	udpConn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		s.sendReply(conn, ReplyGeneralFailure, nil, 0)
		return err
	}
	defer udpConn.Close()

	// 获取UDP监听地址
	localAddr := udpConn.LocalAddr()
	udpAddr, ok := localAddr.(*net.UDPAddr)
	if !ok {
		udpConn.Close()
		return fmt.Errorf("failed to get UDP address")
	}
	
	// 发送UDP ASSOCIATE响应
	if err := s.sendReply(conn, ReplySuccess, udpAddr.IP, uint16(udpAddr.Port)); err != nil {
		return err
	}

	// 创建UDP中继
	relay := &UDPRelay{
		conn:       udpConn,
		timeout:    5 * time.Minute,
		associations: make(map[string]*UDPAssociation),
	}
	relay.associations[clientAddr.String()] = &UDPAssociation{
		clientAddr: clientAddr,
		created:    time.Now(),
	}

	// 启动UDP中继处理
	return relay.Serve(s.DialFunc)
}

// Serve 启动UDP中继服务
func (r *UDPRelay) Serve(dialFunc func(network, addr string) (net.Conn, error)) error {
	buf := make([]byte, 65507) // UDP最大包大小

	for {
		// 设置读取超时
		r.conn.SetReadDeadline(time.Now().Add(r.timeout))
		
		n, clientAddr, err := r.conn.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return err
		}

		// 解析SOCKS5 UDP请求
		if n < 6 {
			continue
		}

		// 跳过RSV和FRAG字段（前3字节）
		addrType := buf[3]
		var targetAddr string
		var dataOffset int

		switch addrType {
		case AddrTypeIPv4:
			if n < 10 {
				continue
			}
			ip := net.IP(buf[4:8])
			port := binary.BigEndian.Uint16(buf[8:10])
			targetAddr = fmt.Sprintf("%s:%d", ip.String(), port)
			dataOffset = 10

		case AddrTypeIPv6:
			if n < 22 {
				continue
			}
			ip := net.IP(buf[4:20])
			port := binary.BigEndian.Uint16(buf[20:22])
			targetAddr = fmt.Sprintf("[%s]:%d", ip.String(), port)
			dataOffset = 22

		case AddrTypeDomain:
			if n < 5 {
				continue
			}
			domainLen := int(buf[4])
			if n < 5+domainLen+2 {
				continue
			}
			domain := string(buf[5 : 5+domainLen])
			port := binary.BigEndian.Uint16(buf[5+domainLen : 5+domainLen+2])
			targetAddr = fmt.Sprintf("%s:%d", domain, port)
			dataOffset = 5 + domainLen + 2

		default:
			continue
		}

		// 获取数据部分
		data := buf[dataOffset:n]

		// 转发数据到目标
		go r.forwardUDP(dialFunc, clientAddr, targetAddr, data)
	}
}

// forwardUDP 转发UDP数据包
func (r *UDPRelay) forwardUDP(dialFunc func(network, addr string) (net.Conn, error), clientAddr net.Addr, targetAddr string, data []byte) {
	// 连接到目标（UDP实际上不需要连接，但需要获取UDPConn）
	conn, err := net.Dial("udp", targetAddr)
	if err != nil {
		return
	}
	defer conn.Close()

	// 发送数据到目标
	_, err = conn.Write(data)
	if err != nil {
		return
	}

	// 读取响应
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	respBuf := make([]byte, 65507)
	n, err := conn.Read(respBuf)
	if err != nil {
		return
	}

	// 构建UDP响应包
	udpResp := r.buildUDPResponse(clientAddr, targetAddr, respBuf[:n])
	
	// 发送响应回客户端
	r.conn.WriteTo(udpResp, clientAddr)
}

// buildUDPResponse 构建UDP响应包
func (r *UDPRelay) buildUDPResponse(clientAddr net.Addr, targetAddr string, data []byte) []byte {
	// 解析目标地址
	host, portStr, err := net.SplitHostPort(targetAddr)
	if err != nil {
		return nil
	}

	portInt, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil
	}
	port := uint16(portInt)

	var resp []byte
	resp = append(resp, 0x00, 0x00, 0x00) // RSV + FRAG

	// 添加地址
	ip := net.ParseIP(host)
	if ip == nil {
		// 域名
		resp = append(resp, AddrTypeDomain)
		resp = append(resp, byte(len(host)))
		resp = append(resp, []byte(host)...)
	} else if ip4 := ip.To4(); ip4 != nil {
		// IPv4
		resp = append(resp, AddrTypeIPv4)
		resp = append(resp, ip4...)
	} else {
		// IPv6
		resp = append(resp, AddrTypeIPv6)
		resp = append(resp, ip.To16()...)
	}

	// 添加端口
	portBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(portBuf, port)
	resp = append(resp, portBuf...)

	// 添加数据
	resp = append(resp, data...)

	return resp
}

// cleanupAssociations 清理过期的UDP关联
func (r *UDPRelay) cleanupAssociations() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		for key, assoc := range r.associations {
			if now.Sub(assoc.created) > r.timeout {
				delete(r.associations, key)
			}
		}
	}
}

