// +build linux

package proxy

import (
	"io"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// spliceCopy 使用splice进行零拷贝（仅Linux）
func spliceCopy(dst, src *os.File) (int64, error) {
	pipeR, pipeW, err := os.Pipe()
	if err != nil {
		return 0, err
	}
	defer pipeR.Close()
	defer pipeW.Close()

	var total int64
	bufSize := int64(32 * 1024) // 32KB

	for {
		// 从源文件读取到管道
		n1, err1 := unix.Splice(
			int(src.Fd()), nil,
			int(pipeW.Fd()), nil,
			int(bufSize),
			unix.SPLICE_F_MOVE|unix.SPLICE_F_NONBLOCK,
		)
		if n1 <= 0 {
			if err1 == unix.EAGAIN {
				// 没有数据可读
				continue
			}
			if err1 == nil {
				// EOF
				break
			}
			return total, err1
		}

		// 从管道写入到目标文件
		n2, err2 := unix.Splice(
			int(pipeR.Fd()), nil,
			int(dst.Fd()), nil,
			int(n1),
			unix.SPLICE_F_MOVE|unix.SPLICE_F_NONBLOCK,
		)
		if n2 <= 0 {
			if err2 == unix.EAGAIN {
				// 管道满，等待
				continue
			}
			return total, err2
		}

		total += int64(n2)
	}

	return total, nil
}

// spliceCopyConn 使用splice复制连接（需要TCPConn的文件描述符）
func spliceCopyConn(dst, src *os.File) error {
	_, err := spliceCopy(dst, src)
	return err
}

// getFileFromTCPConn 从TCP连接获取文件（Linux特定）
func getFileFromTCPConn(conn interface{}) (*os.File, error) {
	// 这是一个辅助函数，实际使用需要类型断言
	// 由于无法直接导入net.TCPConn，这里提供接口
	type fileConn interface {
		File() (*os.File, error)
	}
	
	if fc, ok := conn.(fileConn); ok {
		return fc.File()
	}
	return nil, syscall.EINVAL
}

// sendfileCopy 使用sendfile进行零拷贝（Linux）
func sendfileCopy(dst, src *os.File, offset *int64, count int64) (int64, error) {
	if offset == nil {
		offset = new(int64)
	}
	
	written, err := unix.Sendfile(int(dst.Fd()), int(src.Fd()), offset, int(count))
	return int64(written), err
}

// 注意：这些函数仅在Linux上可用
// 其他平台可以使用传统的io.Copy

