package proxy

import "sync"

// bufferPool 全局buffer池，减少GC压力
var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 32*1024)
	},
}

