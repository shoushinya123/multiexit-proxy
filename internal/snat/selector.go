package snat

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
)

// IPSelector IP选择策略接口
type IPSelector interface {
	SelectIP(targetAddr string, targetPort int) (net.IP, error)
}

// RoundRobinSelector 轮询选择器
type RoundRobinSelector struct {
	ips     []net.IP
	current int
	mu      sync.Mutex
}

// NewRoundRobinSelector 创建轮询选择器
func NewRoundRobinSelector(ips []string) (*RoundRobinSelector, error) {
	ipList := make([]net.IP, 0, len(ips))
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return nil, &InvalidIPError{IP: ipStr}
		}
		ipList = append(ipList, ip)
	}

	return &RoundRobinSelector{
		ips:     ipList,
		current: 0,
	}, nil
}

// SelectIP 选择IP（轮询）
func (r *RoundRobinSelector) SelectIP(targetAddr string, targetPort int) (net.IP, error) {
	if len(r.ips) == 0 {
		return nil, &NoIPAvailableError{}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	ip := r.ips[r.current]
	r.current = (r.current + 1) % len(r.ips)
	return ip, nil
}

// PortBasedSelector 按端口选择器
type PortBasedSelector struct {
	portRanges []PortRange
}

// PortRange 端口范围
type PortRange struct {
	Start int
	End   int
	IP    net.IP
}

// NewPortBasedSelector 创建按端口选择器
func NewPortBasedSelector(portRanges []struct {
	Range string
	IP    string
}) (*PortBasedSelector, error) {
	ranges := make([]PortRange, 0, len(portRanges))
	for _, pr := range portRanges {
		var start, end int
		if _, err := fmt.Sscanf(pr.Range, "%d-%d", &start, &end); err != nil {
			return nil, fmt.Errorf("invalid port range: %s", pr.Range)
		}

		ip := net.ParseIP(pr.IP)
		if ip == nil {
			return nil, &InvalidIPError{IP: pr.IP}
		}

		ranges = append(ranges, PortRange{
			Start: start,
			End:   end,
			IP:    ip,
		})
	}

	return &PortBasedSelector{
		portRanges: ranges,
	}, nil
}

// SelectIP 选择IP（按端口）
func (p *PortBasedSelector) SelectIP(targetAddr string, targetPort int) (net.IP, error) {
	for _, pr := range p.portRanges {
		if targetPort >= pr.Start && targetPort <= pr.End {
			return pr.IP, nil
		}
	}

	// 默认返回第一个IP
	if len(p.portRanges) > 0 {
		return p.portRanges[0].IP, nil
	}

	return nil, &NoIPAvailableError{}
}

// DestinationBasedSelector 按目标地址选择器
type DestinationBasedSelector struct {
	ips []net.IP
}

// NewDestinationBasedSelector 创建按目标地址选择器
func NewDestinationBasedSelector(ips []string) (*DestinationBasedSelector, error) {
	ipList := make([]net.IP, 0, len(ips))
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return nil, &InvalidIPError{IP: ipStr}
		}
		ipList = append(ipList, ip)
	}

	return &DestinationBasedSelector{
		ips: ipList,
	}, nil
}

// SelectIP 选择IP（按目标地址哈希）
func (d *DestinationBasedSelector) SelectIP(targetAddr string, targetPort int) (net.IP, error) {
	if len(d.ips) == 0 {
		return nil, &NoIPAvailableError{}
	}

	// 使用目标地址和端口计算哈希
	key := fmt.Sprintf("%s:%d", targetAddr, targetPort)
	hash := sha256.Sum256([]byte(key))
	index := int(binary.BigEndian.Uint64(hash[:8])) % len(d.ips)

	return d.ips[index], nil
}

