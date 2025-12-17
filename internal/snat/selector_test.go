package snat

import (
	"testing"
)

func TestRoundRobinSelector(t *testing.T) {
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}
	selector, err := NewRoundRobinSelector(ips)
	if err != nil {
		t.Fatalf("Failed to create selector: %v", err)
	}
	
	// 测试轮询
	selected1, _ := selector.SelectIP("example.com", 80)
	selected2, _ := selector.SelectIP("example.com", 80)
	selector.SelectIP("example.com", 80) // 跳过第三个
	selected4, _ := selector.SelectIP("example.com", 80)
	
	if selected1.String() == selected2.String() {
		t.Error("Round robin should select different IPs")
	}
	
	if selected4.String() != selected1.String() {
		t.Error("Round robin should cycle back to first IP")
	}
}

func TestDestinationBasedSelector(t *testing.T) {
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}
	selector, err := NewDestinationBasedSelector(ips)
	if err != nil {
		t.Fatalf("Failed to create selector: %v", err)
	}
	
	// 相同目标应该返回相同IP
	selected1, _ := selector.SelectIP("example.com", 80)
	selected2, _ := selector.SelectIP("example.com", 80)
	
	if selected1.String() != selected2.String() {
		t.Error("Destination-based selector should return same IP for same destination")
	}
	
	// 不同目标可能返回不同IP（注意：不保证一定不同，因为哈希可能相同）
	selector.SelectIP("google.com", 443)
}

func TestSelectorWithEmptyIPs(t *testing.T) {
	selector, err := NewRoundRobinSelector([]string{})
	if err != nil {
		// 允许返回错误或nil
		return
	}
	
	_, err = selector.SelectIP("example.com", 80)
	if err == nil {
		t.Error("Selector should return error for empty IP list")
	}
}

func TestSelectorWithInvalidIP(t *testing.T) {
	_, err := NewRoundRobinSelector([]string{"invalid.ip.address"})
	if err == nil {
		t.Error("Selector should return error for invalid IP")
	}
}

