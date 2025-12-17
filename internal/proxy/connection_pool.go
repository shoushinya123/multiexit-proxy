package proxy

import (
	"context"
	"net"
	"sync"
	"time"
)

// ConnectionPool 连接池
type ConnectionPool struct {
	dialFunc    func() (net.Conn, error)
	maxSize     int
	maxIdle     int
	idleTimeout time.Duration
	mu          sync.Mutex
	connections chan *PooledConnection
	activeCount int
	closed      bool
}

// PooledConnection 池化连接
type PooledConnection struct {
	conn        net.Conn
	pool        *ConnectionPool
	lastUsed    time.Time
	returned    bool
	mu          sync.Mutex
}

// NewConnectionPool 创建连接池
func NewConnectionPool(dialFunc func() (net.Conn, error), maxSize, maxIdle int, idleTimeout time.Duration) *ConnectionPool {
	pool := &ConnectionPool{
		dialFunc:    dialFunc,
		maxSize:     maxSize,
		maxIdle:     maxIdle,
		idleTimeout: idleTimeout,
		connections: make(chan *PooledConnection, maxIdle),
	}

	// 启动清理goroutine
	go pool.cleanupIdleConnections()

	return pool
}

// Get 从池中获取连接
func (p *ConnectionPool) Get(ctx context.Context) (*PooledConnection, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, ErrPoolClosed
	}

	// 尝试从池中获取空闲连接
	select {
	case conn := <-p.connections:
		conn.mu.Lock()
		if time.Since(conn.lastUsed) > p.idleTimeout {
			// 连接已过期，关闭并创建新连接
			conn.mu.Unlock()
			conn.conn.Close()
			p.activeCount--
			return p.createNewConnection(ctx)
		}
		conn.returned = false
		conn.lastUsed = time.Now()
		conn.mu.Unlock()
		return conn, nil
	default:
		// 池中没有空闲连接，创建新连接
		return p.createNewConnection(ctx)
	}
}

// createNewConnection 创建新连接
func (p *ConnectionPool) createNewConnection(ctx context.Context) (*PooledConnection, error) {
	if p.activeCount >= p.maxSize {
		return nil, ErrPoolExhausted
	}

	conn, err := p.dialFunc()
	if err != nil {
		return nil, err
	}

	p.activeCount++
	pooledConn := &PooledConnection{
		conn:     conn,
		pool:     p,
		lastUsed: time.Now(),
	}

	return pooledConn, nil
}

// Return 归还连接到池中
func (p *ConnectionPool) Return(conn *PooledConnection) {
	if conn == nil {
		return
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.returned {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		conn.conn.Close()
		p.activeCount--
		return
	}

	conn.returned = true
	conn.lastUsed = time.Now()

	// 尝试放回池中
	select {
	case p.connections <- conn:
		// 成功放回池中
	default:
		// 池已满，关闭连接
		conn.conn.Close()
		p.activeCount--
	}
}

// Close 关闭连接池
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	close(p.connections)

	// 关闭所有池中的连接
	for {
		select {
		case conn := <-p.connections:
			conn.conn.Close()
			p.activeCount--
		default:
			return nil
		}
	}
}

// cleanupIdleConnections 清理空闲连接
func (p *ConnectionPool) cleanupIdleConnections() {
	ticker := time.NewTicker(p.idleTimeout / 2)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return
		}
		p.mu.Unlock()

		// 清理过期的空闲连接
		now := time.Now()
		var expiredConns []*PooledConnection

		for {
			select {
			case conn := <-p.connections:
				conn.mu.Lock()
				if now.Sub(conn.lastUsed) > p.idleTimeout {
					expiredConns = append(expiredConns, conn)
				} else {
					// 放回去
					select {
					case p.connections <- conn:
					default:
						conn.conn.Close()
						p.mu.Lock()
						p.activeCount--
						p.mu.Unlock()
					}
				}
				conn.mu.Unlock()
			default:
				goto cleanup
			}
		}

	cleanup:
		// 关闭过期连接
		for _, conn := range expiredConns {
			conn.conn.Close()
			p.mu.Lock()
			p.activeCount--
			p.mu.Unlock()
		}
	}
}

// GetActiveCount 获取活跃连接数
func (p *ConnectionPool) GetActiveCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.activeCount
}

// GetIdleCount 获取空闲连接数
func (p *ConnectionPool) GetIdleCount() int {
	return len(p.connections)
}

var (
	ErrPoolClosed    = &PoolError{Message: "connection pool is closed"}
	ErrPoolExhausted = &PoolError{Message: "connection pool exhausted"}
)

// PoolError 连接池错误
type PoolError struct {
	Message string
}

func (e *PoolError) Error() string {
	return e.Message
}

// Close 关闭池化连接
func (c *PooledConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.returned {
		return nil
	}

	c.pool.mu.Lock()
	c.pool.activeCount--
	c.pool.mu.Unlock()

	return c.conn.Close()
}

// Read 读取数据
func (c *PooledConnection) Read(b []byte) (int, error) {
	return c.conn.Read(b)
}

// Write 写入数据
func (c *PooledConnection) Write(b []byte) (int, error) {
	return c.conn.Write(b)
}

// LocalAddr 返回本地地址
func (c *PooledConnection) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr 返回远程地址
func (c *PooledConnection) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// SetDeadline 设置截止时间
func (c *PooledConnection) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// SetReadDeadline 设置读截止时间
func (c *PooledConnection) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline 设置写截止时间
func (c *PooledConnection) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

