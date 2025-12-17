package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ConfigVersion 配置版本信息
type ConfigVersion struct {
	Version     string    `json:"version"`
	BackupPath  string    `json:"backup_path"`
	CreatedAt   time.Time `json:"created_at"`
	Description string    `json:"description"`
	Checksum    string    `json:"checksum"`
}

// VersionManager 版本管理器
type VersionManager struct {
	configPath string
	versions   []ConfigVersion
	mu         sync.RWMutex
}

// NewVersionManager 创建版本管理器
func NewVersionManager(configPath string) *VersionManager {
	return &VersionManager{
		configPath: configPath,
		versions:   make([]ConfigVersion, 0),
	}
}

// SaveVersion 保存配置版本
func (vm *VersionManager) SaveVersion(description string) (string, error) {
	// 创建备份
	backupPath, err := BackupConfig(vm.configPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	// 计算校验和
	checksum, err := vm.calculateChecksum(vm.configPath)
	if err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// 创建版本信息
	version := ConfigVersion{
		Version:     time.Now().Format("20060102-150405"),
		BackupPath:  backupPath,
		CreatedAt:   time.Now(),
		Description: description,
		Checksum:    checksum,
	}

	vm.mu.Lock()
	vm.versions = append(vm.versions, version)
	// 按时间排序（最新的在前）
	sort.Slice(vm.versions, func(i, j int) bool {
		return vm.versions[i].CreatedAt.After(vm.versions[j].CreatedAt)
	})
	vm.mu.Unlock()

	// 保存版本列表
	if err := vm.saveVersionList(); err != nil {
		logrus.Warnf("Failed to save version list: %v", err)
	}

	// 自动清理旧版本（保留最近10个）
	if err := vm.CleanOldVersions(10); err != nil {
		logrus.Warnf("Failed to clean old versions: %v", err)
	}

	logrus.Infof("Config version saved: %s (%s)", version.Version, description)
	return version.Version, nil
}

// ListVersions 列出所有版本
func (vm *VersionManager) ListVersions() []ConfigVersion {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	// 返回副本
	versions := make([]ConfigVersion, len(vm.versions))
	copy(versions, vm.versions)
	return versions
}

// Rollback 回滚到指定版本
func (vm *VersionManager) Rollback(version string) error {
	vm.mu.RLock()
	var targetVersion *ConfigVersion
	for _, v := range vm.versions {
		if v.Version == version {
			targetVersion = &v
			break
		}
	}
	vm.mu.RUnlock()

	if targetVersion == nil {
		return fmt.Errorf("version %s not found", version)
	}

	// 验证备份文件是否存在
	if _, err := os.Stat(targetVersion.BackupPath); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	// 验证校验和
	checksum, err := vm.calculateChecksum(targetVersion.BackupPath)
	if err != nil {
		return fmt.Errorf("failed to verify checksum: %w", err)
	}

	if checksum != targetVersion.Checksum {
		logrus.Warnf("Checksum mismatch for version %s, proceeding anyway", version)
	}

	// 恢复配置
	if err := RestoreConfig(targetVersion.BackupPath, vm.configPath); err != nil {
		return fmt.Errorf("failed to restore config: %w", err)
	}

	logrus.Infof("Config rolled back to version %s (%s)", version, targetVersion.Description)
	return nil
}

// RollbackToLatest 回滚到最新版本
func (vm *VersionManager) RollbackToLatest() error {
	vm.mu.RLock()
	if len(vm.versions) == 0 {
		vm.mu.RUnlock()
		return fmt.Errorf("no versions available")
	}
	latestVersion := vm.versions[0].Version
	vm.mu.RUnlock()

	return vm.Rollback(latestVersion)
}

// calculateChecksum 计算文件校验和（简化版，使用文件大小和修改时间）
func (vm *VersionManager) calculateChecksum(filePath string) (string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	// 读取文件内容计算MD5
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// 使用简单的哈希（实际应该使用MD5或SHA256）
	hash := fmt.Sprintf("%x", len(data))
	hash += fmt.Sprintf("-%d", info.ModTime().Unix())
	return hash, nil
}

// saveVersionList 保存版本列表
func (vm *VersionManager) saveVersionList() error {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	versionFile := vm.configPath + ".versions.json"
	data, err := json.MarshalIndent(vm.versions, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(versionFile, data, 0644)
}

// LoadVersionList 加载版本列表（公开方法）
func (vm *VersionManager) LoadVersionList() error {
	versionFile := vm.configPath + ".versions.json"
	data, err := os.ReadFile(versionFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，使用空列表
		}
		return err
	}

	vm.mu.Lock()
	defer vm.mu.Unlock()

	if err := json.Unmarshal(data, &vm.versions); err != nil {
		return err
	}

	// 清理旧版本（保留最近10个）
	vm.CleanOldVersions(10)

	return nil
}

// CleanOldVersions 清理旧版本（保留最近N个）
func (vm *VersionManager) CleanOldVersions(keepCount int) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if len(vm.versions) <= keepCount {
		return nil
	}

	// 按时间排序（最新的在前）
	sort.Slice(vm.versions, func(i, j int) bool {
		return vm.versions[i].CreatedAt.After(vm.versions[j].CreatedAt)
	})

	// 删除旧版本文件
	toDelete := vm.versions[keepCount:]
	for _, version := range toDelete {
		if _, err := os.Stat(version.BackupPath); err == nil {
			if err := os.Remove(version.BackupPath); err != nil {
				logrus.Warnf("Failed to delete old backup %s: %v", version.BackupPath, err)
			}
		}
	}

	// 更新版本列表
	vm.versions = vm.versions[:keepCount]

	// 保存更新后的版本列表
	return vm.saveVersionList()
}

