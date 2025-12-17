package logging

import (
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// SetupRotation 设置日志轮转
func SetupRotation(logFile string, maxSizeMB int, maxBackups int, maxAgeDays int, compress bool) error {
	if logFile == "" {
		return nil // 不设置文件输出
	}

	// 确保日志目录存在
	dir := filepath.Dir(logFile)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// 配置日志轮转
	logrus.SetOutput(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    maxSizeMB,    // 每个日志文件的最大大小（MB）
		MaxBackups: maxBackups,   // 保留的旧日志文件数量
		MaxAge:     maxAgeDays,   // 保留旧日志文件的天数
		Compress:   compress,     // 是否压缩旧日志文件
		LocalTime:  true,         // 使用本地时间
	})

	return nil
}

// SetupRotationWithDefaults 使用默认值设置日志轮转
func SetupRotationWithDefaults(logFile string) error {
	return SetupRotation(logFile, 100, 10, 30, true)
}

// RotateNow 立即轮转日志（手动触发）
func RotateNow(logFile string) error {
	if logFile == "" {
		return nil
	}

	// 重命名当前日志文件
	timestamp := time.Now().Format("20060102-150405")
	backupFile := logFile + "." + timestamp

	if err := os.Rename(logFile, backupFile); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	// 重新打开日志文件（lumberjack会自动处理）
	return nil
}

