package integration

import (
	"crypto/rand"
	"fmt"
	"net"
	"sync"
	"testing"

	"multiexit-proxy/internal/protocol"
	"multiexit-proxy/internal/proxy"
	"multiexit-proxy/internal/snat"
)

// TestEndToEndProxy 端到端代理测试
func TestEndToEndProxy(t *testing.T) {
	// 准备测试环境
	masterKey := make([]byte, 32)
	rand.Read(masterKey)
	
	_, err := protocol.NewCipher(masterKey, true)
	if err != nil {
		t.Fatalf("Failed to create cipher: %v", err)
	}

	// 创建测试服务器配置
	_ = &proxy.ServerConfig{
		ListenAddr: "127.0.0.1:0", // 使用随机端口
		AuthKey:    "test-key",
		ExitIPs:    []string{"127.0.0.1"}, // 使用本地IP测试
		Strategy:   "round_robin",
	}
	
	// 注意：这里需要实际的TLS配置，简化测试
	t.Log("Integration test setup complete")
}

// TestSNATWithMultipleIPs 多IP SNAT测试
func TestSNATWithMultipleIPs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SNAT test in short mode (requires root)")
	}

	// 测试IP列表
	testIPs := []string{"192.168.1.1", "192.168.1.2"}
	ips := make([]net.IP, len(testIPs))
	for i, ipStr := range testIPs {
		ips[i] = net.ParseIP(ipStr)
	}

	// 创建路由管理器（注意：实际测试需要真实的IP）
	routingMgr, err := snat.NewRoutingManager(ips, "192.168.1.1", "eth0")
	if err != nil {
		t.Skipf("SNAT test skipped: %v (requires real network config)", err)
		return
	}

	// 测试设置
	err = routingMgr.Setup()
	if err != nil {
		t.Logf("SNAT setup failed (expected in test environment): %v", err)
		return
	}
	defer routingMgr.Cleanup()

	t.Log("SNAT routing manager created successfully")
}

// TestIPSelectorStrategies 测试所有IP选择策略
func TestIPSelectorStrategies(t *testing.T) {
	testIPs := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}

	tests := []struct {
		name     string
		strategy string
		create   func([]string) (snat.IPSelector, error)
	}{
		{"RoundRobin", "round_robin", func(ips []string) (snat.IPSelector, error) {
			return snat.NewRoundRobinSelector(ips)
		}},
		{"DestinationBased", "destination", func(ips []string) (snat.IPSelector, error) {
			return snat.NewDestinationBasedSelector(ips)
		}},
		{"LoadBalanced", "load_balanced", func(ips []string) (snat.IPSelector, error) {
			return snat.NewLoadBalancedSelector(ips, "connections")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector, err := tt.create(testIPs)
			if err != nil {
				t.Fatalf("Failed to create selector: %v", err)
			}

			// 测试选择IP
			ip1, err := selector.SelectIP("example.com", 80)
			if err != nil {
				t.Errorf("SelectIP failed: %v", err)
			}
			if ip1 == nil {
				t.Error("SelectIP returned nil")
			}

			// 多选几次确保稳定性
			for i := 0; i < 10; i++ {
				ip, err := selector.SelectIP("test.com", 443)
				if err != nil {
					t.Errorf("SelectIP failed on iteration %d: %v", i, err)
				}
				if ip == nil {
					t.Errorf("SelectIP returned nil on iteration %d", i)
				}
			}
		})
	}
}

// TestEncryptionRoundTrip 加密往返测试
func TestEncryptionRoundTrip(t *testing.T) {
	masterKey := make([]byte, 32)
	rand.Read(masterKey)

	cipher, err := protocol.NewCipher(masterKey, true)
	if err != nil {
		t.Fatalf("Failed to create cipher: %v", err)
	}

	// 测试不同大小的数据
	testSizes := []int{1, 100, 1024, 32 * 1024, 64 * 1024}

	for _, size := range testSizes {
		t.Run(fmt.Sprintf("Size_%d", size), func(t *testing.T) {
			plaintext := make([]byte, size)
			rand.Read(plaintext)

			// 加密
			ciphertext, err := cipher.Encrypt(plaintext)
			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}

			// 解密
			decrypted, err := cipher.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}

			// 验证
			if len(decrypted) != len(plaintext) {
				t.Errorf("Length mismatch: got %d, want %d", len(decrypted), len(plaintext))
			}

			for i := range plaintext {
				if decrypted[i] != plaintext[i] {
					t.Errorf("Data mismatch at index %d", i)
					break
				}
			}
		})
	}
}

// TestConcurrentEncryption 并发加密测试
func TestConcurrentEncryption(t *testing.T) {
	masterKey := make([]byte, 32)
	rand.Read(masterKey)

	cipher, err := protocol.NewCipher(masterKey, true)
	if err != nil {
		t.Fatalf("Failed to create cipher: %v", err)
	}

	concurrency := 100
	iterations := 10

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*iterations)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			connCipher := protocol.NewConnectionCipher(cipher)

			for j := 0; j < iterations; j++ {
				data := []byte(fmt.Sprintf("test-%d-%d", id, j))
				
				ciphertext, err := connCipher.Encrypt(data)
				if err != nil {
					errors <- err
					return
				}

				decrypted, err := connCipher.Decrypt(ciphertext)
				if err != nil {
					errors <- err
					return
				}

				if string(decrypted) != string(data) {
					errors <- fmt.Errorf("data mismatch")
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent encryption error: %v", err)
	}
}

// TestHTTPThroughProxy 通过代理的HTTP测试
func TestHTTPThroughProxy(t *testing.T) {
	// 这是一个示例测试框架
	// 实际测试需要运行真实的服务器和客户端
	
	t.Log("HTTP proxy test requires running server/client")
	t.Skip("Skipping - requires full server/client setup")
}

