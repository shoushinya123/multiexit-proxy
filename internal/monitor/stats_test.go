package monitor

import (
	"net"
	"testing"
	"time"
)

func TestStatsManager_OnConnectionStart(t *testing.T) {
	sm := NewStatsManager()
	ip := net.ParseIP("1.2.3.4")
	
	sm.OnConnectionStart(ip)
	
	stats := sm.GetStats()
	if stats.TotalConnections != 1 {
		t.Errorf("Expected 1 total connection, got %d", stats.TotalConnections)
	}
	if stats.ActiveConnections != 1 {
		t.Errorf("Expected 1 active connection, got %d", stats.ActiveConnections)
	}
}

func TestStatsManager_OnConnectionEnd(t *testing.T) {
	sm := NewStatsManager()
	ip := net.ParseIP("1.2.3.4")
	
	sm.OnConnectionStart(ip)
	sm.OnConnectionEnd(ip, 100*time.Millisecond)
	
	stats := sm.GetStats()
	if stats.ActiveConnections != 0 {
		t.Errorf("Expected 0 active connections, got %d", stats.ActiveConnections)
	}
	if stats.TotalConnections != 1 {
		t.Errorf("Expected 1 total connection, got %d", stats.TotalConnections)
	}
	
	ipStats := sm.GetIPStats(ip)
	if ipStats == nil {
		t.Error("Expected IP stats, got nil")
		return
	}
	if ipStats.ActiveConn != 0 {
		t.Errorf("Expected 0 active connections for IP, got %d", ipStats.ActiveConn)
	}
}

func TestStatsManager_OnBytesTransferred(t *testing.T) {
	sm := NewStatsManager()
	ip := net.ParseIP("1.2.3.4")
	
	sm.OnConnectionStart(ip)
	sm.OnBytesTransferred(ip, 1024, 2048)
	
	stats := sm.GetStats()
	if stats.BytesUp != 1024 {
		t.Errorf("Expected 1024 bytes up, got %d", stats.BytesUp)
	}
	if stats.BytesDown != 2048 {
		t.Errorf("Expected 2048 bytes down, got %d", stats.BytesDown)
	}
	if stats.BytesTransferred != 3072 {
		t.Errorf("Expected 3072 bytes transferred, got %d", stats.BytesTransferred)
	}
}

func TestStatsManager_Reset(t *testing.T) {
	sm := NewStatsManager()
	ip := net.ParseIP("1.2.3.4")
	
	sm.OnConnectionStart(ip)
	sm.OnBytesTransferred(ip, 1024, 2048)
	
	sm.Reset()
	
	stats := sm.GetStats()
	if stats.TotalConnections != 0 {
		t.Errorf("Expected 0 total connections after reset, got %d", stats.TotalConnections)
	}
	if stats.BytesTransferred != 0 {
		t.Errorf("Expected 0 bytes after reset, got %d", stats.BytesTransferred)
	}
}



