package proxy

import (
	"net"
	"testing"
	"time"
)

func TestConnectionManager_CanAccept(t *testing.T) {
	cm := NewConnectionManager(10, 30*time.Second, 30*time.Second, 300*time.Second, 10*time.Second, true, 30*time.Second)
	
	if !cm.CanAccept() {
		t.Error("Should be able to accept connections")
	}
	
	// 测试连接数限制
	cm.maxConnections = 1
	conn1, _ := net.Pipe()
	cm.AddConnection(conn1)
	
	if cm.CanAccept() {
		t.Error("Should not be able to accept more connections")
	}
	
	cm.RemoveConnection(conn1)
	if !cm.CanAccept() {
		t.Error("Should be able to accept connections after removing one")
	}
}

func TestConnectionManager_AddRemove(t *testing.T) {
	cm := NewConnectionManager(10, 30*time.Second, 30*time.Second, 300*time.Second, 10*time.Second, true, 30*time.Second)
	
	conn, _ := net.Pipe()
	
	// 添加连接
	cm.AddConnection(conn)
	if cm.GetActiveCount() != 1 {
		t.Errorf("Expected 1 active connection, got %d", cm.GetActiveCount())
	}
	
	// 移除连接
	cm.RemoveConnection(conn)
	if cm.GetActiveCount() != 0 {
		t.Errorf("Expected 0 active connections, got %d", cm.GetActiveCount())
	}
}

func TestConnectionManager_Timeouts(t *testing.T) {
	cm := NewConnectionManager(10, 5*time.Second, 5*time.Second, 10*time.Second, 5*time.Second, true, 30*time.Second)
	
	conn, _ := net.Pipe()
	defer conn.Close()
	
	// 测试设置超时
	cm.SetTimeouts(conn, 5*time.Second, 5*time.Second)
	
	// 测试重置超时
	cm.ResetReadDeadline(conn)
	cm.ResetWriteDeadline(conn)
}

