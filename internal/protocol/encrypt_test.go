package protocol

import (
	"crypto/rand"
	"testing"
)

func TestNewCipher(t *testing.T) {
	masterKey := make([]byte, KeySize)
	rand.Read(masterKey)
	
	cipher, err := NewCipher(masterKey, true)
	if err != nil {
		t.Fatalf("Failed to create cipher: %v", err)
	}
	
	if cipher == nil {
		t.Fatal("Cipher is nil")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	masterKey := make([]byte, KeySize)
	rand.Read(masterKey)
	
	cipher, err := NewCipher(masterKey, true)
	if err != nil {
		t.Fatalf("Failed to create cipher: %v", err)
	}
	
	plaintext := []byte("Hello, World!")
	
	// 加密
	ciphertext, err := cipher.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}
	
	if len(ciphertext) <= len(plaintext) {
		t.Error("Ciphertext should be longer than plaintext")
	}
	
	// 解密
	decrypted, err := cipher.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}
	
	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text mismatch: got %s, want %s", string(decrypted), string(plaintext))
	}
}

func TestConnectionCipher(t *testing.T) {
	masterKey := make([]byte, KeySize)
	rand.Read(masterKey)
	
	mainCipher, err := NewCipher(masterKey, true)
	if err != nil {
		t.Fatalf("Failed to create cipher: %v", err)
	}
	
	connCipher := NewConnectionCipher(mainCipher)
	
	plaintext := []byte("Test message")
	
	// 加密
	ciphertext, err := connCipher.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}
	
	// 解密
	decrypted, err := connCipher.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}
	
	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text mismatch")
	}
}

func TestHMAC(t *testing.T) {
	masterKey := make([]byte, KeySize)
	rand.Read(masterKey)
	
	cipher, err := NewCipher(masterKey, true)
	if err != nil {
		t.Fatalf("Failed to create cipher: %v", err)
	}
	
	data := []byte("test data")
	hmac1 := cipher.ComputeHMAC(data)
	hmac2 := cipher.ComputeHMAC(data)
	
	if len(hmac1) != HMACSize {
		t.Errorf("HMAC size mismatch: got %d, want %d", len(hmac1), HMACSize)
	}
	
	if !cipher.VerifyHMAC(data, hmac1) {
		t.Error("HMAC verification failed")
	}
	
	if !cipher.VerifyHMAC(data, hmac2) {
		t.Error("HMAC verification failed for second computation")
	}
	
	// 验证错误的HMAC应该失败
	wrongHMAC := make([]byte, HMACSize)
	if cipher.VerifyHMAC(data, wrongHMAC) {
		t.Error("HMAC verification should fail for wrong HMAC")
	}
}

