package socks5

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

const (
	Version = 0x05

	MethodNoAuth       = 0x00
	MethodUsernamePass = 0x02
	MethodNoAcceptable = 0xFF

	CmdConnect = 0x01
	CmdBind    = 0x02
	CmdUDP     = 0x03

	AddrTypeIPv4   = 0x01
	AddrTypeIPv6   = 0x04
	AddrTypeDomain = 0x03

	ReplySuccess            = 0x00
	ReplyGeneralFailure     = 0x01
	ReplyConnectionRefused  = 0x05
	ReplyNetworkUnreachable = 0x03
	ReplyHostUnreachable    = 0x04
)

// Server SOCKS5服务器
type Server struct {
	DialFunc func(network, addr string) (net.Conn, error)
}

// NewServer 创建SOCKS5服务器
func NewServer(dialFunc func(network, addr string) (net.Conn, error)) *Server {
	return &Server{
		DialFunc: dialFunc,
	}
}

// HandleConn 处理连接
func (s *Server) HandleConn(conn net.Conn) error {
	defer conn.Close()

	// 协商认证方法
	if err := s.negotiateAuth(conn); err != nil {
		return err
	}

	// 处理请求
	return s.handleRequest(conn)
}

// negotiateAuth 协商认证方法
func (s *Server) negotiateAuth(conn net.Conn) error {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}

	if buf[0] != Version {
		return errors.New("invalid SOCKS version")
	}

	nMethods := int(buf[1])
	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return err
	}

	// 选择无认证方法
	response := []byte{Version, MethodNoAuth}
	if _, err := conn.Write(response); err != nil {
		return err
	}

	return nil
}

// handleRequest 处理请求
func (s *Server) handleRequest(conn net.Conn) error {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}

	if buf[0] != Version {
		return errors.New("invalid SOCKS version")
	}

	cmd := buf[1]
	switch cmd {
	case CmdConnect:
		// 处理TCP连接
		return s.handleConnect(conn)
	case CmdUDP:
		// 处理UDP关联请求
		return s.HandleUDPRequest(conn)
	default:
		s.sendReply(conn, ReplyGeneralFailure, nil, 0)
		return fmt.Errorf("unsupported command: %d", cmd)
	}
}

// handleConnect 处理TCP连接请求
func (s *Server) handleConnect(conn net.Conn) error {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}

	// 读取地址
	addrType := buf[3]
	var addr string
	var port uint16

	switch addrType {
	case AddrTypeIPv4:
		ip := make([]byte, 4)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return err
		}
		if _, err := io.ReadFull(conn, buf[:2]); err != nil {
			return err
		}
		port = binary.BigEndian.Uint16(buf[:2])
		addr = fmt.Sprintf("%s:%d", net.IP(ip).String(), port)

	case AddrTypeIPv6:
		ip := make([]byte, 16)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return err
		}
		if _, err := io.ReadFull(conn, buf[:2]); err != nil {
			return err
		}
		port = binary.BigEndian.Uint16(buf[:2])
		addr = fmt.Sprintf("[%s]:%d", net.IP(ip).String(), port)

	case AddrTypeDomain:
		if _, err := io.ReadFull(conn, buf[:1]); err != nil {
			return err
		}
		domainLen := int(buf[0])
		domain := make([]byte, domainLen)
		if _, err := io.ReadFull(conn, domain); err != nil {
			return err
		}
		if _, err := io.ReadFull(conn, buf[:2]); err != nil {
			return err
		}
		port = binary.BigEndian.Uint16(buf[:2])
		addr = fmt.Sprintf("%s:%d", string(domain), port)

	default:
		s.sendReply(conn, ReplyGeneralFailure, nil, 0)
		return fmt.Errorf("unsupported address type: %d", addrType)
	}

	// 连接目标
	targetConn, err := s.DialFunc("tcp", addr)
	if err != nil {
		s.sendReply(conn, ReplyConnectionRefused, nil, 0)
		return err
	}
	defer targetConn.Close()

	// 发送成功响应
	if err := s.sendReply(conn, ReplySuccess, nil, 0); err != nil {
		return err
	}

	// 转发数据
	go io.Copy(targetConn, conn)
	io.Copy(conn, targetConn)

	return nil
}

// sendReply 发送响应
func (s *Server) sendReply(conn net.Conn, reply byte, addr net.IP, port uint16) error {
	buf := make([]byte, 4)
	buf[0] = Version
	buf[1] = reply
	buf[2] = 0x00 // Reserved

	if addr == nil {
		buf[3] = AddrTypeIPv4
		buf = append(buf, make([]byte, 6)...)
	} else {
		if ip4 := addr.To4(); ip4 != nil {
			buf[3] = AddrTypeIPv4
			buf = append(buf, ip4...)
		} else {
			buf[3] = AddrTypeIPv6
			buf = append(buf, addr.To16()...)
		}
		portBuf := make([]byte, 2)
		binary.BigEndian.PutUint16(portBuf, port)
		buf = append(buf, portBuf...)
	}

	_, err := conn.Write(buf)
	return err
}
