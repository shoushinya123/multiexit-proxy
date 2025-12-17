package proxy

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// ConnectionManager 连接管理器
type ConnectionManager struct {
	maxConnections int64
	activeCount    int64
	connections    sync.Map // map[net.Conn]context.CancelFunc
	mu             sync.RWMutex
	readTimeout    time.Duration
	writeTimeout   time.Duration
	idleTimeout    time.Duration
	dialTimeout    time.Duration
	keepAlive      bool
	keepAliveTime  time.Duration
}

// NewConnectionManager 创建连接管理器
func NewConnectionManager(maxConnections int, readTimeout, writeTimeout, idleTimeout, dialTimeout time.Duration, keepAlive bool, keepAliveTime time.Duration) *ConnectionManager {
	return &ConnectionManager{
		maxConnections: int64(maxConnections),
		readTimeout:    readTimeout,
		writeTimeout:   writeTimeout,
		idleTimeout:    idleTimeout,
		dialTimeout:    dialTimeout,
		keepAlive:      keepAlive,
		keepAliveTime:  keepAliveTime,
	}
}

// CanAccept 检查是否可以接受新连接
func (cm *ConnectionManager) CanAccept() bool {
	current := atomic.LoadInt64(&cm.activeCount)
	max := atomic.LoadInt64(&cm.maxConnections)
	if max <= 0 {
		return true // 无限制
	}
	return current < max
}

// AddConnection 添加连接
func (cm *ConnectionManager) AddConnection(conn net.Conn) bool {
	if !cm.CanAccept() {
		return false
	}

	// 设置TCP KeepAlive
	if tcpConn, ok := conn.(*net.TCPConn); ok && cm.keepAlive {
		tcpConn.SetKeepAlive(true)
		if cm.keepAliveTime > 0 {
			tcpConn.SetKeepAlivePeriod(cm.keepAliveTime)
		}
	}

	// 创建取消上下文（使用可取消的上下文，支持超时）
	ctx, cancel := context.WithCancel(context.Background())
	cm.connections.Store(conn, cancel)
	atomic.AddInt64(&cm.activeCount, 1)

	// 启动空闲超时检测
	if cm.idleTimeout > 0 {
		go cm.monitorIdleTimeout(conn, ctx)
	}

	return true
}

// RemoveConnection 移除连接
func (cm *ConnectionManager) RemoveConnection(conn net.Conn) {
	if cancel, ok := cm.connections.LoadAndDelete(conn); ok {
		if cancelFunc, ok := cancel.(context.CancelFunc); ok {
			cancelFunc()
		}
		atomic.AddInt64(&cm.activeCount, -1)
	}
}

// monitorIdleTimeout 监控空闲超时
func (cm *ConnectionManager) monitorIdleTimeout(conn net.Conn, ctx context.Context) {
	if cm.idleTimeout <= 0 {
		return
	}
	
	ticker := time.NewTicker(cm.idleTimeout / 2)
	defer ticker.Stop()

	conn.SetReadDeadline(time.Now().Add(cm.idleTimeout))

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 检查连接是否超时（通过检查读取超时）
			// 如果连接有活动，读取超时会被重置，这里主要做定期检查
			now := time.Now()
			// 简单的检查：如果连接一直空闲，会在读取时超时
			// 这里主要是定期检查上下文是否取消
			if ctx.Err() != nil {
				return
			}
			// 重新设置读取超时
			conn.SetReadDeadline(now.Add(cm.idleTimeout))
		}
	}
}

// SetTimeouts 设置连接超时
func (cm *ConnectionManager) SetTimeouts(conn net.Conn, readTimeout, writeTimeout time.Duration) {
	if readTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(readTimeout))
	}
	if writeTimeout > 0 {
		conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	}
}

// ResetReadDeadline 重置读取超时
func (cm *ConnectionManager) ResetReadDeadline(conn net.Conn) {
	if cm.readTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(cm.readTimeout))
	}
	if cm.idleTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(cm.idleTimeout))
	}
}

// ResetWriteDeadline 重置写入超时
func (cm *ConnectionManager) ResetWriteDeadline(conn net.Conn) {
	if cm.writeTimeout > 0 {
		conn.SetWriteDeadline(time.Now().Add(cm.writeTimeout))
	}
}

// DialWithTimeout 使用超时连接
func (cm *ConnectionManager) DialWithTimeout(network, address string) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout: cm.dialTimeout,
	}
	if cm.keepAlive {
		dialer.KeepAlive = cm.keepAliveTime
	}
	return dialer.Dial(network, address)
}

// GetActiveCount 获取当前活跃连接数
func (cm *ConnectionManager) GetActiveCount() int64 {
	return atomic.LoadInt64(&cm.activeCount)
}

// CloseAll 关闭所有连接
func (cm *ConnectionManager) CloseAll() {
	cm.connections.Range(func(key, value interface{}) bool {
		if conn, ok := key.(net.Conn); ok {
			conn.Close()
		}
		if cancel, ok := value.(context.CancelFunc); ok {
			cancel()
		}
		return true
	})
	cm.connections = sync.Map{}
	atomic.StoreInt64(&cm.activeCount, 0)
}

