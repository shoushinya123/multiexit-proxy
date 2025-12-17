package main

import (
	"crypto/rand"
	"fmt"
	"runtime"
	"sync"
	"time"

	"multiexit-proxy/internal/protocol"
)

func main() {
	fmt.Println("==========================================")
	fmt.Println("  性能优化验证测试")
	fmt.Println("==========================================")
	fmt.Println()

	// 准备测试数据
	masterKey := make([]byte, 32)
	rand.Read(masterKey)

	testData := make([]byte, 32*1024) // 32KB
	rand.Read(testData)

	// 测试1: 加密性能对比
	fmt.Println("==========================================")
	fmt.Println("  测试1: 加密算法性能对比")
	fmt.Println("==========================================")
	fmt.Println()

	chaCha20Time, chaCha20Mem := benchmarkEncrypt(false, masterKey, testData, "ChaCha20-Poly1305")
	aesGCMTime, aesGCMMem := benchmarkEncrypt(true, masterKey, testData, "AES-GCM")

	fmt.Printf("\n性能提升:\n")
	fmt.Printf("  时间提升: %.2fx\n", float64(chaCha20Time)/float64(aesGCMTime))
	fmt.Printf("  内存减少: %.2f%%\n", (1.0-float64(aesGCMMem)/float64(chaCha20Mem))*100)

	// 测试2: 并发性能测试
	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println("  测试2: 并发加密性能")
	fmt.Println("==========================================")
	fmt.Println()

	concurrentChaCha20 := benchmarkConcurrentEncrypt(false, masterKey, testData, 100)
	concurrentAESGCM := benchmarkConcurrentEncrypt(true, masterKey, testData, 100)

	fmt.Printf("\n并发性能提升: %.2fx\n", float64(concurrentChaCha20)/float64(concurrentAESGCM))

	// 测试3: 连接级加密上下文测试
	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println("  测试3: 连接级加密上下文性能")
	fmt.Println("==========================================")
	fmt.Println()

	cipher, _ := protocol.NewCipher(masterKey, true)
	connCipherTime := benchmarkConnectionCipher(cipher, testData)

	fmt.Printf("连接级加密上下文时间: %v\n", connCipherTime)

	// 测试4: 内存使用统计
	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println("  测试4: 内存使用统计")
	fmt.Println("==========================================")
	fmt.Println()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// 创建100个连接级加密上下文
	for i := 0; i < 100; i++ {
		protocol.NewConnectionCipher(cipher)
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	fmt.Printf("100个连接级上下文内存占用: %d KB\n", (m2.Alloc-m1.Alloc)/1024)

	// 测试5: 不同数据大小性能
	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println("  测试5: 不同数据大小性能")
	fmt.Println("==========================================")
	fmt.Println()

	sizes := []int{1024, 8 * 1024, 32 * 1024, 64 * 1024}
	for _, size := range sizes {
		data := make([]byte, size)
		rand.Read(data)
		time, _ := benchmarkEncrypt(true, masterKey, data, fmt.Sprintf("%dKB", size/1024))
		fmt.Printf("  %dKB: %v\n", size/1024, time)
	}

	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println("  测试完成")
	fmt.Println("==========================================")
}

func benchmarkEncrypt(useAES bool, masterKey, data []byte, name string) (time.Duration, uint64) {
	cipher, _ := protocol.NewCipher(masterKey, useAES)

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	start := time.Now()
	iterations := 1000
	for i := 0; i < iterations; i++ {
		_, err := cipher.Encrypt(data)
		if err != nil {
			panic(err)
		}
	}
	elapsed := time.Since(start)

	runtime.GC()
	runtime.ReadMemStats(&m2)

	avgTime := elapsed / time.Duration(iterations)
	memUsed := m2.Alloc - m1.Alloc

	fmt.Printf("%s:\n", name)
	fmt.Printf("  平均时间: %v\n", avgTime)
	fmt.Printf("  吞吐量: %.2f MB/s\n", float64(len(data)*iterations)/1024/1024/elapsed.Seconds())
	fmt.Printf("  内存使用: %d KB\n", memUsed/1024)

	return avgTime, memUsed
}

func benchmarkConcurrentEncrypt(useAES bool, masterKey, data []byte, goroutines int) time.Duration {
	cipher, _ := protocol.NewCipher(masterKey, useAES)

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			connCipher := protocol.NewConnectionCipher(cipher)
			for j := 0; j < 100; j++ {
				_, err := connCipher.Encrypt(data)
				if err != nil {
					panic(err)
				}
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	fmt.Printf("并发(%d goroutines)总时间: %v\n", goroutines, elapsed)
	return elapsed
}

func benchmarkConnectionCipher(cipher *protocol.Cipher, data []byte) time.Duration {
	connCipher := protocol.NewConnectionCipher(cipher)

	start := time.Now()
	iterations := 1000
	for i := 0; i < iterations; i++ {
		_, err := connCipher.Encrypt(data)
		if err != nil {
			panic(err)
		}
	}
	elapsed := time.Since(start)

	return elapsed / time.Duration(iterations)
}

