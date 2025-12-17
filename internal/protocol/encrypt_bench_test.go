package protocol

import (
	"crypto/rand"
	"fmt"
	"testing"
)

// BenchmarkEncrypt_ChaCha20 测试ChaCha20加密性能（优化前）
func BenchmarkEncrypt_ChaCha20(b *testing.B) {
	masterKey := make([]byte, KeySize)
	rand.Read(masterKey)

	cipher, err := NewCipher(masterKey, false) // 使用ChaCha20
	if err != nil {
		b.Fatalf("Failed to create cipher: %v", err)
	}

	data := make([]byte, 32*1024) // 32KB数据
	rand.Read(data)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := cipher.Encrypt(data)
		if err != nil {
			b.Fatalf("Encrypt failed: %v", err)
		}
	}
}

// BenchmarkEncrypt_AESGCM 测试AES-GCM加密性能（优化后）
func BenchmarkEncrypt_AESGCM(b *testing.B) {
	masterKey := make([]byte, KeySize)
	rand.Read(masterKey)

	cipher, err := NewCipher(masterKey, true) // 使用AES-GCM
	if err != nil {
		b.Fatalf("Failed to create cipher: %v", err)
	}

	data := make([]byte, 32*1024) // 32KB数据
	rand.Read(data)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := cipher.Encrypt(data)
		if err != nil {
			b.Fatalf("Encrypt failed: %v", err)
		}
	}
}

// BenchmarkDecrypt_ChaCha20 测试ChaCha20解密性能
func BenchmarkDecrypt_ChaCha20(b *testing.B) {
	masterKey := make([]byte, KeySize)
	rand.Read(masterKey)

	cipher, err := NewCipher(masterKey, false)
	if err != nil {
		b.Fatalf("Failed to create cipher: %v", err)
	}

	data := make([]byte, 32*1024)
	rand.Read(data)
	ciphertext, _ := cipher.Encrypt(data)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := cipher.Decrypt(ciphertext)
		if err != nil {
			b.Fatalf("Decrypt failed: %v", err)
		}
	}
}

// BenchmarkDecrypt_AESGCM 测试AES-GCM解密性能
func BenchmarkDecrypt_AESGCM(b *testing.B) {
	masterKey := make([]byte, KeySize)
	rand.Read(masterKey)

	cipher, err := NewCipher(masterKey, true)
	if err != nil {
		b.Fatalf("Failed to create cipher: %v", err)
	}

	data := make([]byte, 32*1024)
	rand.Read(data)
	ciphertext, _ := cipher.Encrypt(data)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := cipher.Decrypt(ciphertext)
		if err != nil {
			b.Fatalf("Decrypt failed: %v", err)
		}
	}
}

// BenchmarkConnectionCipher 测试连接级加密上下文性能
func BenchmarkConnectionCipher(b *testing.B) {
	masterKey := make([]byte, KeySize)
	rand.Read(masterKey)

	mainCipher, _ := NewCipher(masterKey, true)
	connCipher := NewConnectionCipher(mainCipher)

	data := make([]byte, 32*1024)
	rand.Read(data)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := connCipher.Encrypt(data)
		if err != nil {
			b.Fatalf("Encrypt failed: %v", err)
		}
	}
}

// BenchmarkDifferentSizes 测试不同数据大小的性能
func BenchmarkEncrypt_DifferentSizes(b *testing.B) {
	masterKey := make([]byte, KeySize)
	rand.Read(masterKey)

	cipher, _ := NewCipher(masterKey, true)

	sizes := []int{1 * 1024, 8 * 1024, 32 * 1024, 64 * 1024}

	for _, size := range sizes {
		data := make([]byte, size)
		rand.Read(data)

		b.Run(fmt.Sprintf("size_%dKB", size/1024), func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := cipher.Encrypt(data)
				if err != nil {
					b.Fatalf("Encrypt failed: %v", err)
				}
			}
		})
	}
}
