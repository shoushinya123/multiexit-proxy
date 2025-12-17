package proxy

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ReconnectManager 重连管理器
type ReconnectManager struct {
	maxRetries      int           // 最大重试次数（0=无限重试）
	initialDelay    time.Duration // 初始延迟
	maxDelay        time.Duration // 最大延迟
	backoffFactor   float64       // 退避因子（通常为2.0）
	jitter          bool          // 是否添加随机抖动
	mu              sync.RWMutex
	retryCount      int
	lastError       error
	lastRetryTime   time.Time
}

// ReconnectConfig 重连配置
type ReconnectConfig struct {
	MaxRetries    int           // 最大重试次数（0=无限重试）
	InitialDelay  time.Duration // 初始延迟（默认1秒）
	MaxDelay      time.Duration // 最大延迟（默认5分钟）
	BackoffFactor float64       // 退避因子（默认2.0）
	Jitter        bool          // 是否添加随机抖动（默认true）
}

// NewReconnectManager 创建重连管理器
func NewReconnectManager(config ReconnectConfig) *ReconnectManager {
	if config.InitialDelay == 0 {
		config.InitialDelay = 1 * time.Second
	}
	if config.MaxDelay == 0 {
		config.MaxDelay = 5 * time.Minute
	}
	if config.BackoffFactor == 0 {
		config.BackoffFactor = 2.0
	}
	if config.Jitter {
		config.Jitter = true
	}

	return &ReconnectManager{
		maxRetries:    config.MaxRetries,
		initialDelay:  config.InitialDelay,
		maxDelay:      config.MaxDelay,
		backoffFactor: config.BackoffFactor,
		jitter:        config.Jitter,
	}
}

// Do 执行带重连的操作
func (r *ReconnectManager) Do(ctx context.Context, fn func() error) error {
	r.mu.Lock()
	r.retryCount = 0
	r.lastError = nil
	r.mu.Unlock()

	for {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 执行操作
		err := fn()
		if err == nil {
			// 成功，重置重试计数
			r.mu.Lock()
			r.retryCount = 0
			r.lastError = nil
			r.mu.Unlock()
			return nil
		}

		// 操作失败，记录错误
		r.mu.Lock()
		r.lastError = err
		r.retryCount++
		currentRetry := r.retryCount
		r.mu.Unlock()

		// 检查是否超过最大重试次数
		// 注意：currentRetry是当前尝试次数（包括第一次），maxRetries是最大重试次数
		// 如果maxRetries=2，允许：第1次（初始）+ 第2次重试 = 总共2次尝试
		if r.maxRetries > 0 && currentRetry > r.maxRetries {
			return fmt.Errorf("max retries (%d) exceeded after %d attempts, last error: %w", r.maxRetries, currentRetry, err)
		}

		// 计算延迟时间（指数退避）
		delay := r.calculateDelay(currentRetry)

		logrus.Warnf("Connection failed (attempt %d/%s): %v, retrying in %v",
			currentRetry,
			r.getMaxRetriesString(),
			err,
			delay,
		)

		// 等待延迟时间
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			r.mu.Lock()
			r.lastRetryTime = time.Now()
			r.mu.Unlock()
		}
	}
}

// calculateDelay 计算延迟时间（指数退避）
func (r *ReconnectManager) calculateDelay(retryCount int) time.Duration {
	// 指数退避：delay = initialDelay * (backoffFactor ^ (retryCount - 1))
	delay := float64(r.initialDelay) * math.Pow(r.backoffFactor, float64(retryCount-1))

	// 限制最大延迟（防止无限增长）
	if delay > float64(r.maxDelay) {
		delay = float64(r.maxDelay)
	}

	// 添加随机抖动（±20%）
	if r.jitter {
		jitter := delay * 0.2 * (0.5 - float64(time.Now().UnixNano()%100)/100.0)
		delay += jitter
		// 确保抖动后不超过最大延迟
		if delay > float64(r.maxDelay) {
			delay = float64(r.maxDelay)
		}
	}

	return time.Duration(delay)
}

// getMaxRetriesString 获取最大重试次数的字符串表示
func (r *ReconnectManager) getMaxRetriesString() string {
	if r.maxRetries == 0 {
		return "∞"
	}
	return fmt.Sprintf("%d", r.maxRetries)
}

// GetRetryCount 获取当前重试次数
func (r *ReconnectManager) GetRetryCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.retryCount
}

// GetLastError 获取最后一次错误
func (r *ReconnectManager) GetLastError() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastError
}

// GetLastRetryTime 获取最后一次重试时间
func (r *ReconnectManager) GetLastRetryTime() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastRetryTime
}

// Reset 重置重连管理器
func (r *ReconnectManager) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.retryCount = 0
	r.lastError = nil
	r.lastRetryTime = time.Time{}
}

