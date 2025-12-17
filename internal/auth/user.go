package auth

import (
	"crypto/subtle"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
)

// User 用户
type User struct {
	Username     string
	PasswordHash []byte
	RateLimit    int64 // bytes per second
	AllowedIPs   []net.IPNet
	CreatedAt    time.Time
	LastLogin    time.Time
}

// UserManager 用户管理器
type UserManager struct {
	users map[string]*User
	mu    sync.RWMutex
}

// NewUserManager 创建用户管理器
func NewUserManager() *UserManager {
	return &UserManager{
		users: make(map[string]*User),
	}
}

// AddUser 添加用户
func (m *UserManager) AddUser(username, password string, rateLimit int64, allowedIPs []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.users[username]; exists {
		return ErrUserExists
	}
	
	passwordHash := hashPassword(password)
	
	ipNets := make([]net.IPNet, 0, len(allowedIPs))
	for _, ipStr := range allowedIPs {
		_, ipNet, err := net.ParseCIDR(ipStr)
		if err != nil {
			// 尝试作为IP地址解析
			ip := net.ParseIP(ipStr)
			if ip == nil {
				continue
			}
			mask := net.CIDRMask(32, 32)
			if ip.To4() == nil {
				mask = net.CIDRMask(128, 128)
			}
			ipNet = &net.IPNet{IP: ip, Mask: mask}
		}
		ipNets = append(ipNets, *ipNet)
	}
	
	m.users[username] = &User{
		Username:     username,
		PasswordHash: passwordHash,
		RateLimit:    rateLimit,
		AllowedIPs:   ipNets,
		CreatedAt:    time.Now(),
	}
	
	return nil
}

// Authenticate 验证用户
func (m *UserManager) Authenticate(username, password string, clientIP net.IP) (*User, error) {
	m.mu.RLock()
	user, exists := m.users[username]
	m.mu.RUnlock()
	
	if !exists {
		return nil, ErrInvalidCredentials
	}
	
	// 验证密码
	if !verifyPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}
	
	// 检查IP白名单
	if len(user.AllowedIPs) > 0 {
		allowed := false
		for _, ipNet := range user.AllowedIPs {
			if ipNet.Contains(clientIP) {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, ErrIPNotAllowed
		}
	}
	
	// 更新最后登录时间
	m.mu.Lock()
	user.LastLogin = time.Now()
	m.mu.Unlock()
	
	return user, nil
}

// DeleteUser 删除用户
func (m *UserManager) DeleteUser(username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.users[username]; !exists {
		return ErrUserNotFound
	}
	
	delete(m.users, username)
	return nil
}

// GetUser 获取用户
func (m *UserManager) GetUser(username string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	user, exists := m.users[username]
	if !exists {
		return nil, ErrUserNotFound
	}
	
	// 返回副本避免并发修改
	return &User{
		Username:     user.Username,
		PasswordHash: append([]byte{}, user.PasswordHash...),
		RateLimit:    user.RateLimit,
		AllowedIPs:   append([]net.IPNet{}, user.AllowedIPs...),
		CreatedAt:    user.CreatedAt,
		LastLogin:    user.LastLogin,
	}, nil
}

// ListUsers 列出所有用户
func (m *UserManager) ListUsers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	usernames := make([]string, 0, len(m.users))
	for username := range m.users {
		usernames = append(usernames, username)
	}
	return usernames
}

// hashPassword 哈希密码（使用Argon2）
func hashPassword(password string) []byte {
	salt := make([]byte, 16)
	// 在生产环境中应该使用随机salt
	for i := range salt {
		salt[i] = byte(i)
	}
	
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return append(salt, hash...)
}

// verifyPassword 验证密码
func verifyPassword(password string, hash []byte) bool {
	if len(hash) < 16 {
		return false
	}
	
	salt := hash[:16]
	storedHash := hash[16:]
	
	computedHash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	
	return subtle.ConstantTimeCompare(storedHash, computedHash) == 1
}

// UpdateUserRateLimit 更新用户速率限制
func (m *UserManager) UpdateUserRateLimit(username string, rateLimit int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	user, exists := m.users[username]
	if !exists {
		return ErrUserNotFound
	}
	
	user.RateLimit = rateLimit
	return nil
}

