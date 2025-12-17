package proxy

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestReconnectManager_Do(t *testing.T) {
	// 测试成功情况
	manager := NewReconnectManager(ReconnectConfig{
		MaxRetries:    3,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
		Jitter:        false,
	})

	attempts := 0
	err := manager.Do(context.Background(), func() error {
		attempts++
		if attempts < 2 {
			return errors.New("simulated error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestReconnectManager_MaxRetries(t *testing.T) {
	manager := NewReconnectManager(ReconnectConfig{
		MaxRetries:    2,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
		Jitter:        false,
	})

	err := manager.Do(context.Background(), func() error {
		return errors.New("always fail")
	})

	if err == nil {
		t.Error("Expected error after max retries")
	}
	// maxRetries=2意味着：第1次尝试 + 2次重试 = 总共3次尝试
	// 所以retryCount应该是3（包括初始尝试）
	if manager.GetRetryCount() != 3 {
		t.Errorf("Expected 3 attempts (1 initial + 2 retries), got %d", manager.GetRetryCount())
	}
}

func TestReconnectManager_CalculateDelay(t *testing.T) {
	manager := NewReconnectManager(ReconnectConfig{
		InitialDelay:  1 * time.Second,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        false,
	})

	// 测试延迟计算
	delay1 := manager.calculateDelay(1)
	if delay1 != 1*time.Second {
		t.Errorf("Expected 1s delay, got %v", delay1)
	}

	delay2 := manager.calculateDelay(2)
	if delay2 != 2*time.Second {
		t.Errorf("Expected 2s delay, got %v", delay2)
	}

	delay3 := manager.calculateDelay(3)
	if delay3 != 4*time.Second {
		t.Errorf("Expected 4s delay, got %v", delay3)
	}

	// 测试最大延迟限制
	delay10 := manager.calculateDelay(10)
	if delay10 > 10*time.Second {
		t.Errorf("Expected delay <= 10s, got %v", delay10)
	}
}

func TestReconnectManager_Reset(t *testing.T) {
	manager := NewReconnectManager(ReconnectConfig{
		MaxRetries:  1,
		InitialDelay: 10 * time.Millisecond,
	})

	// 触发一次失败
	manager.Do(context.Background(), func() error {
		return errors.New("fail")
	})

	if manager.GetRetryCount() == 0 {
		t.Error("Expected retry count > 0")
	}

	// 重置
	manager.Reset()
	if manager.GetRetryCount() != 0 {
		t.Error("Expected retry count to be 0 after reset")
	}
}

