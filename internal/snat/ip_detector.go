package snat

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// IPDetector 公网IP检测器
type IPDetector struct {
	detectionServices []string
	timeout           time.Duration
}

// NewIPDetector 创建IP检测器
func NewIPDetector() *IPDetector {
	return &IPDetector{
		detectionServices: []string{
			"https://api.ipify.org",
			"https://ifconfig.me/ip",
			"https://icanhazip.com",
			"https://checkip.amazonaws.com",
			"https://api.ip.sb/ip",
		},
		timeout: 5 * time.Second,
	}
}

// DetectPublicIP 检测单个公网IP
func (d *IPDetector) DetectPublicIP() (string, error) {
	client := &http.Client{
		Timeout: d.timeout,
	}

	var lastErr error
	for _, service := range d.detectionServices {
		resp, err := client.Get(service)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("status code: %d", resp.StatusCode)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		ipStr := strings.TrimSpace(string(body))
		ip := net.ParseIP(ipStr)
		if ip != nil && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsMulticast() {
			return ipStr, nil
		}
	}

	return "", fmt.Errorf("failed to detect public IP: %w", lastErr)
}

// DetectLocalIPs 检测本机绑定的所有公网IP
func (d *IPDetector) DetectLocalIPs() ([]string, error) {
	var publicIPs []string

	// 获取所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get interfaces: %w", err)
	}

	for _, iface := range interfaces {
		// 跳过回环和未启用的接口
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// 获取接口的所有地址
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil {
				continue
			}

			// 检查是否是公网IP
			if !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsMulticast() && !ip.IsLinkLocalUnicast() {
				ipStr := ip.String()
				// 去重
				exists := false
				for _, existing := range publicIPs {
					if existing == ipStr {
						exists = true
						break
					}
				}
				if !exists {
					publicIPs = append(publicIPs, ipStr)
				}
			}
		}
	}

	return publicIPs, nil
}

// DetectAllPublicIPs 检测所有公网IP（包括本地绑定和通过API检测）
func (d *IPDetector) DetectAllPublicIPs() ([]string, error) {
	var allIPs []string

	// 1. 检测本地绑定的公网IP
	localIPs, err := d.DetectLocalIPs()
	if err == nil {
		allIPs = append(allIPs, localIPs...)
	}

	// 2. 通过API检测当前出口IP
	apiIP, err := d.DetectPublicIP()
	if err == nil {
		// 检查是否已存在
		exists := false
		for _, existing := range allIPs {
			if existing == apiIP {
				exists = true
				break
			}
		}
		if !exists {
			allIPs = append(allIPs, apiIP)
		}
	}

	if len(allIPs) == 0 {
		return nil, fmt.Errorf("no public IPs detected")
	}

	return allIPs, nil
}

// DetectByInterface 检测指定网络接口的公网IP
func (d *IPDetector) DetectByInterface(interfaceName string) ([]string, error) {
	var publicIPs []string

	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface %s: %w", interfaceName, err)
	}

	if iface.Flags&net.FlagUp == 0 {
		return nil, fmt.Errorf("interface %s is not up", interfaceName)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("failed to get addresses for interface %s: %w", interfaceName, err)
	}

	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}

		if ip == nil {
			continue
		}

		// 检查是否是公网IP
		if !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsMulticast() && !ip.IsLinkLocalUnicast() {
			publicIPs = append(publicIPs, ip.String())
		}
	}

	if len(publicIPs) == 0 {
		return nil, fmt.Errorf("no public IPs found on interface %s", interfaceName)
	}

	return publicIPs, nil
}

