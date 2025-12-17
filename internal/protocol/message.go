package protocol

import (
	"encoding/binary"
	"fmt"
	"net"
)

const (
	// 协议版本
	Version = 0x01

	// 消息类型
	MsgTypeHandshake = 0x01
	MsgTypeConnect   = 0x02
	MsgTypeData      = 0x03
	MsgTypeClose     = 0x04

	// 地址类型
	AddrTypeIPv4   = 0x01
	AddrTypeIPv6   = 0x02
	AddrTypeDomain = 0x03
)

// HandshakeMessage 握手消息 (32字节)
type HandshakeMessage struct {
	Version   uint8
	Method    uint8
	Reserved  uint16
	Nonce     [16]byte
	Timestamp int64
	HMAC      [4]byte
}

// ConnectRequest 连接请求
type ConnectRequest struct {
	Type     uint8
	AddrType uint8
	AddrLen  uint8
	Address  []byte
	Port     uint16
}

// DataMessage 数据消息
type DataMessage struct {
	Type     uint8
	StreamID uint32
	Length   uint16
	Data     []byte
	HMAC     [4]byte
}

// EncodeHandshake 编码握手消息
func EncodeHandshake(msg *HandshakeMessage) []byte {
	buf := make([]byte, 32)
	buf[0] = msg.Version
	buf[1] = msg.Method
	binary.BigEndian.PutUint16(buf[2:4], msg.Reserved)
	copy(buf[4:20], msg.Nonce[:])
	binary.BigEndian.PutUint64(buf[20:28], uint64(msg.Timestamp))
	copy(buf[28:32], msg.HMAC[:])
	return buf
}

// DecodeHandshake 解码握手消息
func DecodeHandshake(data []byte) (*HandshakeMessage, error) {
	if len(data) < 32 {
		return nil, ErrInvalidMessage
	}

	msg := &HandshakeMessage{
		Version:   data[0],
		Method:    data[1],
		Reserved:  binary.BigEndian.Uint16(data[2:4]),
		Timestamp: int64(binary.BigEndian.Uint64(data[20:28])),
	}
	copy(msg.Nonce[:], data[4:20])
	copy(msg.HMAC[:], data[28:32])

	return msg, nil
}

// EncodeConnectRequest 编码连接请求
func EncodeConnectRequest(req *ConnectRequest) []byte {
	addrLen := len(req.Address)
	totalLen := 1 + 1 + 1 + addrLen + 2 // type + addrType + addrLen + address + port

	buf := make([]byte, totalLen)
	buf[0] = req.Type
	buf[1] = req.AddrType
	buf[2] = uint8(addrLen)
	copy(buf[3:3+addrLen], req.Address)
	binary.BigEndian.PutUint16(buf[3+addrLen:3+addrLen+2], req.Port)

	return buf
}

// DecodeConnectRequest 解码连接请求
func DecodeConnectRequest(data []byte) (*ConnectRequest, error) {
	if len(data) < 5 {
		return nil, ErrInvalidMessage
	}

	req := &ConnectRequest{
		Type:     data[0],
		AddrType: data[1],
		AddrLen:  data[2],
	}

	addrStart := 3
	addrEnd := addrStart + int(req.AddrLen)
	if len(data) < addrEnd+2 {
		return nil, ErrInvalidMessage
	}

	req.Address = make([]byte, req.AddrLen)
	copy(req.Address, data[addrStart:addrEnd])
	req.Port = binary.BigEndian.Uint16(data[addrEnd : addrEnd+2])

	return req, nil
}

// EncodeDataMessage 编码数据消息
func EncodeDataMessage(msg *DataMessage) []byte {
	totalLen := 1 + 4 + 2 + len(msg.Data) + 4 // type + streamID + length + data + hmac
	buf := make([]byte, totalLen)

	offset := 0
	buf[offset] = msg.Type
	offset++
	binary.BigEndian.PutUint32(buf[offset:offset+4], msg.StreamID)
	offset += 4
	binary.BigEndian.PutUint16(buf[offset:offset+2], msg.Length)
	offset += 2
	copy(buf[offset:offset+len(msg.Data)], msg.Data)
	offset += len(msg.Data)
	copy(buf[offset:offset+4], msg.HMAC[:])

	return buf
}

// DecodeDataMessage 解码数据消息
func DecodeDataMessage(data []byte) (*DataMessage, error) {
	if len(data) < 7 {
		return nil, ErrInvalidMessage
	}

	msg := &DataMessage{
		Type:     data[0],
		StreamID: binary.BigEndian.Uint32(data[1:5]),
		Length:   binary.BigEndian.Uint16(data[5:7]),
	}

	if len(data) < 7+int(msg.Length)+4 {
		return nil, ErrInvalidMessage
	}

	msg.Data = make([]byte, msg.Length)
	copy(msg.Data, data[7:7+int(msg.Length)])
	copy(msg.HMAC[:], data[7+int(msg.Length):7+int(msg.Length)+4])

	return msg, nil
}

// ParseAddress 解析地址字符串为地址类型和字节
func ParseAddress(addr string) (uint8, []byte, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return 0, nil, err
	}

	// 尝试解析为IP
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			return AddrTypeIPv4, ip4, nil
		}
		return AddrTypeIPv6, ip.To16(), nil
	}

	// 域名
	return AddrTypeDomain, []byte(host), nil
}

// BuildAddress 构建地址字符串
func BuildAddress(addrType uint8, addr []byte, port uint16) string {
	var host string
	switch addrType {
	case AddrTypeIPv4:
		host = net.IP(addr).String()
	case AddrTypeIPv6:
		host = net.IP(addr).String()
	case AddrTypeDomain:
		host = string(addr)
	default:
		return ""
	}
	return net.JoinHostPort(host, fmt.Sprintf("%d", port))
}
