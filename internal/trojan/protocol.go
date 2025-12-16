package trojan

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

const (
	// Trojan协议常量
	MaxPacketSize = 8192
	HeaderSize    = 56 // SHA224哈希长度（两次SHA224拼接）
)

var (
	ErrInvalidPassword = errors.New("invalid trojan password")
	ErrInvalidHeader   = errors.New("invalid trojan header")
)

// Header Trojan协议头（56字节，两次SHA224拼接）
type Header [56]byte

// Command 命令类型
const (
	CmdConnect = 1
	CmdUDP     = 3
)

// Request Trojan连接请求
type Request struct {
	Command  byte   // 1=CONNECT, 3=UDP
	Address  net.IP
	Port     uint16
	Domain   string
	AddrType byte // 1=IPv4, 3=Domain, 4=IPv6
}

// NewHeader 创建Trojan协议头（密码的SHA224哈希）
// Trojan协议使用两次SHA224的结果拼接（28字节 + 28字节 = 56字节）
func NewHeader(password string) Header {
	// 计算第一次SHA224（28字节）
	h := sha256.New224()
	h.Write([]byte(password))
	hash1 := h.Sum(nil)
	
	// 计算第二次SHA224（28字节）：密码 + hash1
	h = sha256.New224()
	h.Write([]byte(password))
	h.Write(hash1)
	hash2 := h.Sum(nil)
	
	// 拼接两次哈希结果（56字节）
	var header Header
	copy(header[:28], hash1)
	copy(header[28:], hash2)
	return header
}

// VerifyHeader 验证Trojan协议头
func VerifyHeader(header Header, password string) bool {
	expected := NewHeader(password)
	return header == expected
}

// ParseRequest 解析Trojan连接请求
func ParseRequest(reader io.Reader) (*Request, error) {
	// 读取命令字节
	var cmd byte
	if err := binary.Read(reader, binary.BigEndian, &cmd); err != nil {
		return nil, err
	}

	req := &Request{
		Command: cmd,
	}

	switch cmd {
	case CmdConnect:
		// TCP连接
		return parseTCPRequest(reader, req)
	case CmdUDP:
		// UDP代理（暂不支持）
		return nil, errors.New("UDP not supported yet")
	default:
		return nil, errors.New("unknown command")
	}
}

// parseTCPRequest 解析TCP请求
func parseTCPRequest(reader io.Reader, req *Request) (*Request, error) {
	// 读取地址类型
	var addrType byte
	if err := binary.Read(reader, binary.BigEndian, &addrType); err != nil {
		return nil, err
	}

	req.AddrType = addrType

	switch addrType {
	case 1: // IPv4
		ip := make([]byte, 4)
		if _, err := io.ReadFull(reader, ip); err != nil {
			return nil, err
		}
		req.Address = net.IP(ip)

	case 3: // Domain
		var domainLen byte
		if err := binary.Read(reader, binary.BigEndian, &domainLen); err != nil {
			return nil, err
		}
		domain := make([]byte, domainLen)
		if _, err := io.ReadFull(reader, domain); err != nil {
			return nil, err
		}
		req.Domain = string(domain)
		// 解析域名获取IP（或保持域名）
		if ip := net.ParseIP(req.Domain); ip != nil {
			req.Address = ip
		}

	case 4: // IPv6
		ip := make([]byte, 16)
		if _, err := io.ReadFull(reader, ip); err != nil {
			return nil, err
		}
		req.Address = net.IP(ip)

	default:
		return nil, errors.New("invalid address type")
	}

	// 读取端口
	if err := binary.Read(reader, binary.BigEndian, &req.Port); err != nil {
		return nil, err
	}

	// 读取CRLF（Trojan协议要求）
	crlf := make([]byte, 2)
	if _, err := io.ReadFull(reader, crlf); err != nil {
		return nil, err
	}
	if crlf[0] != '\r' || crlf[1] != '\n' {
		return nil, errors.New("invalid CRLF")
	}

	return req, nil
}

// BuildRequest 构建Trojan请求
func BuildRequest(req *Request) []byte {
	buf := make([]byte, 0, 300)

	// 命令
	buf = append(buf, req.Command)

	// 地址类型和地址
	switch req.AddrType {
	case 1: // IPv4
		buf = append(buf, 1)
		buf = append(buf, req.Address.To4()...)

	case 3: // Domain
		buf = append(buf, 3)
		buf = append(buf, byte(len(req.Domain)))
		buf = append(buf, []byte(req.Domain)...)

	case 4: // IPv6
		buf = append(buf, 4)
		buf = append(buf, req.Address.To16()...)
	}

	// 端口
	portBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(portBuf, req.Port)
	buf = append(buf, portBuf...)

	// CRLF
	buf = append(buf, '\r', '\n')

	return buf
}

// GetTargetAddr 获取目标地址字符串
func (r *Request) GetTargetAddr() string {
	var host string
	if r.Domain != "" {
		host = r.Domain
	} else if r.Address != nil {
		host = r.Address.String()
	} else {
		return ""
	}
	return net.JoinHostPort(host, fmt.Sprintf("%d", r.Port))
}

// ParseTargetAddr 解析目标地址为Trojan请求
func ParseTargetAddr(addr string, cmd byte) (*Request, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	var port uint16
	fmt.Sscanf(portStr, "%d", &port)

	req := &Request{
		Command: cmd,
		Port:    port,
	}

	// 判断地址类型
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			req.AddrType = 1 // IPv4
			req.Address = ip4
		} else {
			req.AddrType = 4 // IPv6
			req.Address = ip.To16()
		}
	} else {
		req.AddrType = 3 // Domain
		req.Domain = host
	}

	return req, nil
}
