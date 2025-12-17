package snat

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"sync"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// RoutingManager 路由管理器
type RoutingManager struct {
	ips      []net.IP
	gateway  net.IP
	iface    string
	ipToMark map[string]int
	markToIP map[int]net.IP
	mu       sync.Mutex // 保护并发访问
}

// NewRoutingManager 创建路由管理器
func NewRoutingManager(ips []net.IP, gateway, iface string) (*RoutingManager, error) {
	gwIP := net.ParseIP(gateway)
	if gwIP == nil {
		return nil, fmt.Errorf("invalid gateway IP: %s", gateway)
	}

	ipToMark := make(map[string]int)
	markToIP := make(map[int]net.IP)

	for i, ip := range ips {
		mark := i + 1
		ipToMark[ip.String()] = mark
		markToIP[mark] = ip
	}

	return &RoutingManager{
		ips:      ips,
		gateway:  gwIP,
		iface:    iface,
		ipToMark: ipToMark,
		markToIP: markToIP,
	}, nil
}

// Setup 设置路由规则
func (r *RoutingManager) Setup() error {
	// 为每个IP创建路由表和规则
	for i, ip := range r.ips {
		mark := i + 1
		table := 100 + i

		// 创建路由表
		cmd := exec.Command("ip", "route", "add", "default", "via", r.gateway.String(),
			"table", strconv.Itoa(table), "src", ip.String())
		if err := cmd.Run(); err != nil {
			// 如果路由已存在，记录警告但不返回错误
			logrus.Warnf("Failed to add route for %s (table %d): %v (may already exist)", ip.String(), table, err)
		} else {
			logrus.Debugf("Added route for %s in table %d", ip.String(), table)
		}

		// 创建路由规则
		cmd = exec.Command("ip", "rule", "add", "fwmark", strconv.Itoa(mark),
			"table", strconv.Itoa(table))
		if err := cmd.Run(); err != nil {
			// 如果规则已存在，记录警告但不返回错误
			logrus.Warnf("Failed to add rule for mark %d (table %d): %v (may already exist)", mark, table, err)
		} else {
			logrus.Debugf("Added rule for mark %d in table %d", mark, table)
		}

		// 创建SNAT规则
		cmd = exec.Command("iptables", "-t", "nat", "-A", "OUTPUT",
			"-m", "mark", "--mark", strconv.Itoa(mark),
			"-j", "SNAT", "--to-source", ip.String())
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add SNAT rule for %s: %w", ip.String(), err)
		}
	}

	return nil
}

// Cleanup 清理路由规则
func (r *RoutingManager) Cleanup() error {
	var cleanupErrors []error

	for i, ip := range r.ips {
		mark := i + 1
		table := 100 + i

		// 删除SNAT规则
		cmd := exec.Command("iptables", "-t", "nat", "-D", "OUTPUT",
			"-m", "mark", "--mark", strconv.Itoa(mark),
			"-j", "SNAT", "--to-source", ip.String())
		if err := cmd.Run(); err != nil {
			logrus.Warnf("Failed to delete SNAT rule for %s (mark %d): %v", ip.String(), mark, err)
			cleanupErrors = append(cleanupErrors, fmt.Errorf("SNAT rule cleanup for %s: %w", ip.String(), err))
		} else {
			logrus.Debugf("Deleted SNAT rule for %s (mark %d)", ip.String(), mark)
		}

		// 删除路由规则
		cmd = exec.Command("ip", "rule", "del", "fwmark", strconv.Itoa(mark),
			"table", strconv.Itoa(table))
		if err := cmd.Run(); err != nil {
			logrus.Warnf("Failed to delete rule for mark %d (table %d): %v", mark, table, err)
			cleanupErrors = append(cleanupErrors, fmt.Errorf("rule cleanup for mark %d: %w", mark, err))
		} else {
			logrus.Debugf("Deleted rule for mark %d (table %d)", mark, table)
		}

		// 删除路由表
		cmd = exec.Command("ip", "route", "del", "default", "via", r.gateway.String(),
			"table", strconv.Itoa(table), "src", ip.String())
		if err := cmd.Run(); err != nil {
			logrus.Warnf("Failed to delete route for %s (table %d): %v", ip.String(), table, err)
			cleanupErrors = append(cleanupErrors, fmt.Errorf("route cleanup for %s: %w", ip.String(), err))
		} else {
			logrus.Debugf("Deleted route for %s (table %d)", ip.String(), table)
		}
	}

	if len(cleanupErrors) > 0 {
		logrus.Warnf("Some cleanup operations failed, but continuing...")
		// 不返回错误，因为清理失败不应该阻止程序退出
	}

	return nil
}

// MarkConnection 标记连接
func (r *RoutingManager) MarkConnection(conn net.Conn, ip net.IP) error {
	mark, ok := r.ipToMark[ip.String()]
	if !ok {
		return fmt.Errorf("IP %s not found in routing manager", ip.String())
	}

	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return fmt.Errorf("connection is not TCP")
	}

	file, err := tcpConn.File()
	if err != nil {
		return fmt.Errorf("failed to get file descriptor: %w", err)
	}
	defer file.Close()

	fd := int(file.Fd())
	// SO_MARK is Linux-specific (36), use raw value for compatibility
	const SO_MARK = 36
	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, SO_MARK, mark); err != nil {
		return fmt.Errorf("failed to set SO_MARK: %w", err)
	}

	return nil
}

// GetMarkForIP 获取IP对应的标记
func (r *RoutingManager) GetMarkForIP(ip net.IP) (int, error) {
	mark, ok := r.ipToMark[ip.String()]
	if !ok {
		return 0, fmt.Errorf("IP %s not found", ip.String())
	}
	return mark, nil
}
