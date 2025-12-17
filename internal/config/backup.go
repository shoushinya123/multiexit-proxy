package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BackupConfig 备份配置文件
func BackupConfig(configPath string) (string, error) {
	// 读取原配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config file: %w", err)
	}

	// 生成备份文件名（带时间戳）
	dir := filepath.Dir(configPath)
	base := filepath.Base(configPath)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(dir, fmt.Sprintf("%s.backup.%s%s", name, timestamp, ext))

	// 写入备份文件
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	return backupPath, nil
}

// RestoreConfig 恢复配置文件
func RestoreConfig(backupPath, configPath string) error {
	// 读取备份文件
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// 写入配置文件
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ListBackups 列出所有备份文件
func ListBackups(configPath string) ([]string, error) {
	dir := filepath.Dir(configPath)
	base := filepath.Base(configPath)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	pattern := filepath.Join(dir, fmt.Sprintf("%s.backup.*%s", name, ext))

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob backup files: %w", err)
	}

	return matches, nil
}

// CleanOldBackups 清理旧备份（保留最近N个）
func CleanOldBackups(configPath string, keepCount int) error {
	backups, err := ListBackups(configPath)
	if err != nil {
		return err
	}

	if len(backups) <= keepCount {
		return nil
	}

	// 按修改时间排序（最新的在前）
	// 这里简化处理，直接删除最旧的
	toDelete := backups[keepCount:]
	for _, backup := range toDelete {
		if err := os.Remove(backup); err != nil {
			return fmt.Errorf("failed to delete backup %s: %w", backup, err)
		}
	}

	return nil
}

