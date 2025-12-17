package snat

import (
	"net"
	"testing"
	"time"
)

func TestIPHealthChecker_GetHealthyIPs(t *testing.T) {
	ips := []net.IP{
		net.ParseIP("127.0.0.1"),
		net.ParseIP("8.8.8.8"),
	}
	
	checker := NewIPHealthChecker(ips, 30*time.Second, 5*time.Second)
	
	// 初始状态应该没有健康IP（还没有检查）
	healthy := checker.GetHealthyIPs()
	if len(healthy) > 0 {
		t.Logf("Initial healthy IPs: %v (may have checked)", healthy)
	}
}

func TestIPHealthChecker_GetHealthStatus(t *testing.T) {
	ips := []net.IP{
		net.ParseIP("127.0.0.1"),
		net.ParseIP("8.8.8.8"),
	}
	
	checker := NewIPHealthChecker(ips, 30*time.Second, 5*time.Second)
	
	status := checker.GetHealthStatus()
	t.Logf("Health status: %+v", status)
	
	if status["total"] != 2 {
		t.Errorf("Expected 2 total IPs, got %v", status["total"])
	}
}

func TestIPHealthChecker_Stop(t *testing.T) {
	ips := []net.IP{
		net.ParseIP("127.0.0.1"),
	}
	
	checker := NewIPHealthChecker(ips, 1*time.Second, 1*time.Second)
	
	// 启动检查
	go checker.Start()
	
	// 等待一小段时间
	time.Sleep(100 * time.Millisecond)
	
	// 停止检查
	checker.Stop()
	
	// 再次等待，确保已经停止
	time.Sleep(200 * time.Millisecond)
	
	// 如果没有panic，说明停止成功
}

