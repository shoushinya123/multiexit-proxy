package protocol

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync/atomic"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

const (
	KeySize   = 32 // AES-256/ChaCha20-Poly1305密钥长度
	NonceSize = 12 // Nonce长度
	HMACSize  = 4  // HMAC长度（截断）
)

// Cipher 加密器
type Cipher struct {
	encKey        []byte
	decKey        []byte
	aead          cipher.AEAD
	useAES        bool
	nonceCounter  uint64 // 原子计数器，用于生成nonce
	ciphertextBuf []byte // 预分配的加密输出buffer
}

// NewCipher 创建新的加密器
// useAES: true使用AES-GCM（硬件加速），false使用ChaCha20-Poly1305
func NewCipher(masterKey []byte, useAES bool) (*Cipher, error) {
	if len(masterKey) != KeySize {
		return nil, fmt.Errorf("invalid key size: %d, expected %d", len(masterKey), KeySize)
	}

	// 使用HKDF派生加密密钥
	encKey := make([]byte, KeySize)
	decKey := make([]byte, KeySize)

	hkdfReader := hkdf.New(sha256.New, masterKey, nil, []byte("multiexit-proxy-enc"))
	if _, err := io.ReadFull(hkdfReader, encKey); err != nil {
		return nil, err
	}

	hkdfReader = hkdf.New(sha256.New, masterKey, nil, []byte("multiexit-proxy-dec"))
	if _, err := io.ReadFull(hkdfReader, decKey); err != nil {
		return nil, err
	}

	var aead cipher.AEAD
	var err error

	if useAES {
		// 尝试使用AES-GCM（支持硬件加速）
		block, err := aes.NewCipher(encKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create AES cipher: %w", err)
		}
		aead, err = cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("failed to create GCM: %w", err)
		}
	} else {
		// 使用ChaCha20-Poly1305
		aead, err = chacha20poly1305.New(encKey)
		if err != nil {
			return nil, err
		}
	}

	// 预分配加密输出buffer（64KB，足够大多数场景）
	ciphertextBuf := make([]byte, 64*1024)

	return &Cipher{
		encKey:        encKey,
		decKey:        decKey,
		aead:          aead,
		useAES:        useAES,
		ciphertextBuf: ciphertextBuf,
	}, nil
}

// Encrypt 加密数据（使用计数器nonce，避免随机数生成开销）
func (c *Cipher) Encrypt(plaintext []byte) ([]byte, error) {
	// 使用原子计数器生成nonce（更高效）
	counter := atomic.AddUint64(&c.nonceCounter, 1)
	nonce := make([]byte, NonceSize)
	binary.BigEndian.PutUint64(nonce[0:8], counter)
	// 使用counter的低4字节填充剩余空间
	binary.BigEndian.PutUint32(nonce[8:12], uint32(counter>>32))

	// 计算输出大小
	outputSize := NonceSize + len(plaintext) + c.aead.Overhead()

	// 如果预分配的buffer不够，重新分配
	var output []byte
	if cap(c.ciphertextBuf) >= outputSize {
		output = c.ciphertextBuf[:outputSize]
	} else {
		output = make([]byte, outputSize)
	}

	// 将nonce放在开头
	copy(output[:NonceSize], nonce)

	// 加密数据
	ciphertext := c.aead.Seal(output[:NonceSize], nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt 解密数据
func (c *Cipher) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < NonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce := ciphertext[:NonceSize]
	ciphertext = ciphertext[NonceSize:]

	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// ComputeHMAC 计算HMAC
func (c *Cipher) ComputeHMAC(data []byte) []byte {
	mac := hmac.New(sha256.New, c.encKey)
	mac.Write(data)
	hash := mac.Sum(nil)
	return hash[:HMACSize] // 截断为4字节
}

// VerifyHMAC 验证HMAC
func (c *Cipher) VerifyHMAC(data []byte, expectedHMAC []byte) bool {
	computed := c.ComputeHMAC(data)
	return hmac.Equal(computed, expectedHMAC)
}

// DeriveKeyFromPSK 从PSK派生密钥
func DeriveKeyFromPSK(psk string) []byte {
	hash := sha256.Sum256([]byte(psk))
	return hash[:]
}

// ConnectionCipher 连接级加密上下文（复用主cipher，但维护独立的nonce计数器）
type ConnectionCipher struct {
	cipher        *Cipher
	nonceCounter  uint64
	ciphertextBuf []byte
}

// NewConnectionCipher 创建连接级加密上下文
func NewConnectionCipher(c *Cipher) *ConnectionCipher {
	return &ConnectionCipher{
		cipher:        c,
		ciphertextBuf: make([]byte, 64*1024),
	}
}

// Encrypt 使用连接级nonce计数器加密
func (cc *ConnectionCipher) Encrypt(plaintext []byte) ([]byte, error) {
	cc.nonceCounter++
	nonce := make([]byte, NonceSize)
	binary.BigEndian.PutUint64(nonce[0:8], cc.nonceCounter)
	binary.BigEndian.PutUint32(nonce[8:12], uint32(cc.nonceCounter>>32))

	outputSize := NonceSize + len(plaintext) + cc.cipher.aead.Overhead()
	var output []byte
	if cap(cc.ciphertextBuf) >= outputSize {
		output = cc.ciphertextBuf[:outputSize]
	} else {
		output = make([]byte, outputSize)
	}

	copy(output[:NonceSize], nonce)
	ciphertext := cc.cipher.aead.Seal(output[:NonceSize], nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt 解密（复用主cipher的方法）
func (cc *ConnectionCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	return cc.cipher.Decrypt(ciphertext)
}
