package protocol

import (
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

const (
	KeySize   = 32 // ChaCha20-Poly1305密钥长度
	NonceSize = 12 // ChaCha20-Poly1305 nonce长度
	HMACSize  = 4  // HMAC长度（截断）
)

// Cipher 加密器
type Cipher struct {
	encKey []byte
	decKey []byte
	aead   cipher.AEAD
}

// NewCipher 创建新的加密器
func NewCipher(masterKey []byte) (*Cipher, error) {
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

	aead, err := chacha20poly1305.New(encKey)
	if err != nil {
		return nil, err
	}

	return &Cipher{
		encKey: encKey,
		decKey: decKey,
		aead:   aead,
	}, nil
}

// Encrypt 加密数据
func (c *Cipher) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := c.aead.Seal(nonce, nonce, plaintext, nil)
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

