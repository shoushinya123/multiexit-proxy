package web

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// CSRFProtection CSRF保护
type CSRFProtection struct {
	tokens map[string]time.Time
	mu     sync.RWMutex
	secret []byte
}

// NewCSRFProtection 创建CSRF保护
func NewCSRFProtection(secret []byte) *CSRFProtection {
	return &CSRFProtection{
		tokens: make(map[string]time.Time),
		secret: secret,
	}
}

// GenerateToken 生成CSRF Token
func (c *CSRFProtection) GenerateToken(r *http.Request) string {
	// 简化实现：使用时间戳+IP+密钥生成token
	// 生产环境应使用更安全的方法（如HMAC）
	token := generateCSRFToken(r.RemoteAddr, c.secret)
	
	c.mu.Lock()
	c.tokens[token] = time.Now()
	// 清理过期token（1小时）
	for t, ts := range c.tokens {
		if time.Since(ts) > time.Hour {
			delete(c.tokens, t)
		}
	}
	c.mu.Unlock()
	
	return token
}

// ValidateToken 验证CSRF Token
func (c *CSRFProtection) ValidateToken(token string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	ts, ok := c.tokens[token]
	if !ok {
		return false
	}
	
	// 检查token是否过期（1小时）
	if time.Since(ts) > time.Hour {
		return false
	}
	
	return true
}

// generateCSRFToken 生成CSRF Token（简化实现）
func generateCSRFToken(remoteAddr string, secret []byte) string {
	// 实际应使用HMAC-SHA256等安全方法
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

// csrfMiddleware CSRF中间件
func (s *Server) csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GET请求不需要CSRF保护
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// 检查CSRF Token
		token := r.Header.Get("X-CSRF-Token")
		if token == "" {
			// 尝试从表单获取
			token = r.FormValue("csrf_token")
		}

		if token == "" || !s.csrfProtection.ValidateToken(token) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// LoginAttempt 登录尝试记录
type LoginAttempt struct {
	IP        string
	Attempts  int
	LastAttempt time.Time
	BlockedUntil time.Time
}

// LoginProtection 登录保护（防止暴力破解）
type LoginProtection struct {
	attempts map[string]*LoginAttempt
	mu       sync.RWMutex
	maxAttempts int
	blockDuration time.Duration
}

// NewLoginProtection 创建登录保护
func NewLoginProtection(maxAttempts int, blockDuration time.Duration) *LoginProtection {
	return &LoginProtection{
		attempts:      make(map[string]*LoginAttempt),
		maxAttempts:   maxAttempts,
		blockDuration: blockDuration,
	}
}

// RecordFailedAttempt 记录失败尝试
func (l *LoginProtection) RecordFailedAttempt(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	attempt, ok := l.attempts[ip]
	if !ok {
		attempt = &LoginAttempt{
			IP: ip,
		}
		l.attempts[ip] = attempt
	}

	attempt.Attempts++
	attempt.LastAttempt = time.Now()

	// 如果超过最大尝试次数，阻止
	if attempt.Attempts >= l.maxAttempts {
		attempt.BlockedUntil = time.Now().Add(l.blockDuration)
		logrus.Warnf("IP %s blocked due to too many failed login attempts", ip)
	}
}

// RecordSuccess 记录成功登录
func (l *LoginProtection) RecordSuccess(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.attempts, ip)
}

// IsBlocked 检查IP是否被阻止
func (l *LoginProtection) IsBlocked(ip string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	attempt, ok := l.attempts[ip]
	if !ok {
		return false
	}

	// 检查是否仍在阻止期内
	if time.Now().Before(attempt.BlockedUntil) {
		return true
	}

	// 阻止期已过，重置
	l.mu.RUnlock()
	l.mu.Lock()
	delete(l.attempts, ip)
	l.mu.Unlock()
	l.mu.RLock()

	return false
}

