package performance

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"multiexit-proxy/internal/protocol"
)

// LoadTestResult 负载测试结果
type LoadTestResult struct {
	TotalConnections int64
	SuccessfulConnections int64
	FailedConnections int64
	TotalBytes int64
	Duration time.Duration
	Throughput float64 // MB/s
	AvgLatency time.Duration
	MaxLatency time.Duration
	MinLatency time.Duration
}

// TestEncryptionLoad 加密性能负载测试
func TestEncryptionLoad(t *testing.T) {
	masterKey := make([]byte, 32)
	rand.Read(masterKey)

	cipher, err := protocol.NewCipher(masterKey, true)
	if err != nil {
		t.Fatalf("Failed to create cipher: %v", err)
	}

	concurrency := 100
	iterations := 1000
	dataSize := 32 * 1024 // 32KB

	start := time.Now()
	var wg sync.WaitGroup
	var totalBytes int64
	var totalOps int64

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			connCipher := protocol.NewConnectionCipher(cipher)
			data := make([]byte, dataSize)
			rand.Read(data)

			for j := 0; j < iterations; j++ {
				ciphertext, err := connCipher.Encrypt(data)
				if err != nil {
					t.Errorf("Encryption failed: %v", err)
					return
				}
				atomic.AddInt64(&totalBytes, int64(len(ciphertext)))
				atomic.AddInt64(&totalOps, 1)
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("Load Test Results:")
	t.Logf("  Concurrency: %d", concurrency)
	t.Logf("  Iterations per goroutine: %d", iterations)
	t.Logf("  Data size: %d KB", dataSize/1024)
	t.Logf("  Total operations: %d", atomic.LoadInt64(&totalOps))
	t.Logf("  Total bytes: %.2f MB", float64(atomic.LoadInt64(&totalBytes))/1024/1024)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Throughput: %.2f MB/s", float64(atomic.LoadInt64(&totalBytes))/1024/1024/duration.Seconds())
	t.Logf("  Ops/sec: %.0f", float64(atomic.LoadInt64(&totalOps))/duration.Seconds())
}

// BenchmarkEncryptionThroughput 加密吞吐量基准测试
func BenchmarkEncryptionThroughput(b *testing.B) {
	masterKey := make([]byte, 32)
	rand.Read(masterKey)

	cipher, err := protocol.NewCipher(masterKey, true)
	if err != nil {
		b.Fatalf("Failed to create cipher: %v", err)
	}

	sizes := []int{1024, 8 * 1024, 32 * 1024, 64 * 1024}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%dKB", size/1024), func(b *testing.B) {
			data := make([]byte, size)
			rand.Read(data)
			connCipher := protocol.NewConnectionCipher(cipher)

			b.ResetTimer()
			b.SetBytes(int64(size))
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := connCipher.Encrypt(data)
				if err != nil {
					b.Fatalf("Encryption failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkDecryptionThroughput 解密吞吐量基准测试
func BenchmarkDecryptionThroughput(b *testing.B) {
	masterKey := make([]byte, 32)
	rand.Read(masterKey)

	cipher, err := protocol.NewCipher(masterKey, true)
	if err != nil {
		b.Fatalf("Failed to create cipher: %v", err)
	}

	sizes := []int{1024, 8 * 1024, 32 * 1024, 64 * 1024}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%dKB", size/1024), func(b *testing.B) {
			data := make([]byte, size)
			rand.Read(data)
			connCipher := protocol.NewConnectionCipher(cipher)

			// 预加密数据
			ciphertexts := make([][]byte, b.N)
			for i := 0; i < b.N; i++ {
				ciphertexts[i], _ = connCipher.Encrypt(data)
			}

			b.ResetTimer()
			b.SetBytes(int64(size))
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := connCipher.Decrypt(ciphertexts[i])
				if err != nil {
					b.Fatalf("Decryption failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkConcurrentEncryption 并发加密基准测试
func BenchmarkConcurrentEncryption(b *testing.B) {
	masterKey := make([]byte, 32)
	rand.Read(masterKey)

	cipher, err := protocol.NewCipher(masterKey, true)
	if err != nil {
		b.Fatalf("Failed to create cipher: %v", err)
	}

	concurrencyLevels := []int{1, 10, 100, 1000}
	dataSize := 32 * 1024

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
			data := make([]byte, dataSize)
			rand.Read(data)

			b.ResetTimer()
			b.SetParallelism(concurrency)
			b.RunParallel(func(pb *testing.PB) {
				connCipher := protocol.NewConnectionCipher(cipher)
				for pb.Next() {
					_, err := connCipher.Encrypt(data)
					if err != nil {
						b.Fatalf("Encryption failed: %v", err)
					}
				}
			})
		})
	}
}

// TestMemoryUsage 内存使用测试
func TestMemoryUsage(t *testing.T) {
	masterKey := make([]byte, 32)
	rand.Read(masterKey)

	// 创建大量连接级加密上下文
	numConnections := 10000
	cipher, _ := protocol.NewCipher(masterKey, true)
	
	connCiphers := make([]*protocol.ConnectionCipher, numConnections)
	for i := 0; i < numConnections; i++ {
		connCiphers[i] = protocol.NewConnectionCipher(cipher)
	}

	// 估算内存使用
	// 每个ConnectionCipher约64KB buffer
	estimatedMemory := numConnections * 64 * 1024
	t.Logf("Created %d connection ciphers", numConnections)
	t.Logf("Estimated memory usage: %.2f MB", float64(estimatedMemory)/1024/1024)

	// 清理
	connCiphers = nil
}

// RunLoadTest 运行完整的负载测试
func RunLoadTest(ctx context.Context, config LoadTestConfig) (*LoadTestResult, error) {
	result := &LoadTestResult{
		MinLatency: time.Hour, // 初始化为大值
	}

	start := time.Now()
	var wg sync.WaitGroup
	var successful int64
	var failed int64
	var totalBytes int64
	latencies := make([]time.Duration, 0, config.NumConnections)

	for i := 0; i < config.NumConnections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			connStart := time.Now()
			// 模拟连接操作
			time.Sleep(10 * time.Millisecond)
			
			// 模拟数据传输
			bytes := int64(config.DataSize)
			atomic.AddInt64(&totalBytes, bytes)
			
			latency := time.Since(connStart)
			latencies = append(latencies, latency)
			
			atomic.AddInt64(&successful, 1)
		}()
	}

	wg.Wait()
	result.Duration = time.Since(start)
	result.SuccessfulConnections = atomic.LoadInt64(&successful)
	result.FailedConnections = atomic.LoadInt64(&failed)
	result.TotalBytes = atomic.LoadInt64(&totalBytes)
	result.TotalConnections = int64(config.NumConnections)

	// 计算统计信息
	if len(latencies) > 0 {
		var totalLatency time.Duration
		for _, lat := range latencies {
			totalLatency += lat
			if lat > result.MaxLatency {
				result.MaxLatency = lat
			}
			if lat < result.MinLatency {
				result.MinLatency = lat
			}
		}
		result.AvgLatency = totalLatency / time.Duration(len(latencies))
	}

	result.Throughput = float64(result.TotalBytes) / 1024 / 1024 / result.Duration.Seconds()

	return result, nil
}

// LoadTestConfig 负载测试配置
type LoadTestConfig struct {
	NumConnections int
	Concurrency    int
	DataSize       int
	Duration       time.Duration
}

