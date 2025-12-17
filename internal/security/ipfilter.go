package security

import (
	"net"
	"sync"
)

// IPFilter IP过滤器
type IPFilter struct {
	whitelist []net.IPNet
	blacklist []net.IPNet
	mu        sync.RWMutex
}

// NewIPFilter 创建IP过滤器
func NewIPFilter() *IPFilter {
	return &IPFilter{
		whitelist: make([]net.IPNet, 0),
		blacklist: make([]net.IPNet, 0),
	}
}

// AddWhitelist 添加白名单
func (f *IPFilter) AddWhitelist(cidr string) error {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.whitelist = append(f.whitelist, *ipNet)
	return nil
}

// AddBlacklist 添加黑名单
func (f *IPFilter) AddBlacklist(cidr string) error {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.blacklist = append(f.blacklist, *ipNet)
	return nil
}

// IsAllowed 检查IP是否允许
func (f *IPFilter) IsAllowed(ip net.IP) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	// 先检查黑名单
	for _, ipNet := range f.blacklist {
		if ipNet.Contains(ip) {
			return false
		}
	}
	
	// 如果有白名单，只允许白名单中的IP
	if len(f.whitelist) > 0 {
		for _, ipNet := range f.whitelist {
			if ipNet.Contains(ip) {
				return true
			}
		}
		return false
	}
	
	// 没有白名单时，只要不在黑名单中就允许
	return true
}

// ClearWhitelist 清空白名单
func (f *IPFilter) ClearWhitelist() {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.whitelist = f.whitelist[:0]
}

// ClearBlacklist 清空黑名单
func (f *IPFilter) ClearBlacklist() {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.blacklist = f.blacklist[:0]
}

// GetWhitelist 获取白名单
func (f *IPFilter) GetWhitelist() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	result := make([]string, len(f.whitelist))
	for i, ipNet := range f.whitelist {
		result[i] = ipNet.String()
	}
	return result
}

// GetBlacklist 获取黑名单
func (f *IPFilter) GetBlacklist() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	result := make([]string, len(f.blacklist))
	for i, ipNet := range f.blacklist {
		result[i] = ipNet.String()
	}
	return result
}



