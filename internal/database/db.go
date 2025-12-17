package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// DB 数据库连接
type DB struct {
	*sql.DB
}

// Config 数据库配置
type Config struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string // "disable", "require", "verify-ca", "verify-full"
	MaxConns int    // 最大连接数
	MaxIdle  int    // 最大空闲连接数
}

// NewDB 创建数据库连接
func NewDB(cfg Config) (*DB, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("database host is required")
	}
	if cfg.Database == "" {
		return nil, fmt.Errorf("database name is required")
	}
	if cfg.User == "" {
		return nil, fmt.Errorf("database user is required")
	}

	if cfg.Port == 0 {
		cfg.Port = 5432
	}
	if cfg.SSLMode == "" {
		cfg.SSLMode = "disable" // 默认禁用SSL（本地开发）
	}
	if cfg.MaxConns == 0 {
		cfg.MaxConns = 25
	}
	if cfg.MaxIdle == 0 {
		cfg.MaxIdle = 5
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxIdle)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logrus.Infof("Database connected: %s@%s:%d/%s", cfg.User, cfg.Host, cfg.Port, cfg.Database)

	return &DB{db}, nil
}

// Close 关闭数据库连接
func (db *DB) Close() error {
	if db.DB != nil {
		return db.DB.Close()
	}
	return nil
}

// HealthCheck 健康检查
func (db *DB) HealthCheck() error {
	return db.Ping()
}



