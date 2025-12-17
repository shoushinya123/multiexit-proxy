package config

import (
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

// ConfigWatcher 配置监听器
type ConfigWatcher struct {
	configPath string
	onUpdate   func(*ServerConfig) error
	watcher    *fsnotify.Watcher
	mu         sync.Mutex
	lastMod    time.Time
}

// NewConfigWatcher 创建配置监听器
func NewConfigWatcher(configPath string, onUpdate func(*ServerConfig) error) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	
	err = watcher.Add(configPath)
	if err != nil {
		watcher.Close()
		return nil, err
	}
	
	// 获取初始修改时间
	info, _ := os.Stat(configPath)
	var lastMod time.Time
	if info != nil {
		lastMod = info.ModTime()
	}
	
	return &ConfigWatcher{
		configPath: configPath,
		onUpdate:   onUpdate,
		watcher:    watcher,
		lastMod:    lastMod,
	}, nil
}

// Watch 开始监听配置变化
func (w *ConfigWatcher) Watch() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			
			if event.Op&fsnotify.Write == fsnotify.Write {
				w.handleConfigChange()
			}
			
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			logrus.Errorf("Config watcher error: %v", err)
		}
	}
}

// handleConfigChange 处理配置变化
func (w *ConfigWatcher) handleConfigChange() {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	// 检查文件是否真的被修改了（避免重复触发）
	info, err := os.Stat(w.configPath)
	if err != nil {
		logrus.Errorf("Failed to stat config file: %v", err)
		return
	}
	
	if info.ModTime().Equal(w.lastMod) || info.ModTime().Before(w.lastMod) {
		return
	}
	
	// 等待文件写入完成（简单延迟，生产环境应该更智能）
	time.Sleep(100 * time.Millisecond)
	
	// 重新加载配置
	config, err := LoadServerConfig(w.configPath)
	if err != nil {
		logrus.Errorf("Failed to reload config: %v", err)
		return
	}
	
	// 调用更新回调
	if w.onUpdate != nil {
		if err := w.onUpdate(config); err != nil {
			logrus.Errorf("Failed to apply config update: %v", err)
			return
		}
	}
	
	w.lastMod = info.ModTime()
	logrus.Info("Config reloaded successfully")
}

// Close 关闭监听器
func (w *ConfigWatcher) Close() error {
	if w.watcher != nil {
		return w.watcher.Close()
	}
	return nil
}



