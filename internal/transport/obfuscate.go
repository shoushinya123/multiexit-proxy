package transport

import (
	"math/rand"
	"time"
)

// AddPadding 添加随机padding
func AddPadding(data []byte) []byte {
	rand.Seed(time.Now().UnixNano())
	paddingSize := rand.Intn(128) // 0-127字节
	padding := make([]byte, paddingSize)
	rand.Read(padding)
	return append(data, padding...)
}

// AddRandomDelay 添加随机延迟
func AddRandomDelay() {
	rand.Seed(time.Now().UnixNano())
	delay := time.Duration(rand.Intn(100)) * time.Millisecond
	time.Sleep(delay)
}


