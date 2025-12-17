package database

import (
	"database/sql"
	"net"
	"time"

	"github.com/sirupsen/logrus"
)

// StatsRepository 统计数据仓库
type StatsRepository struct {
	db *DB
}

// NewStatsRepository 创建统计数据仓库
func NewStatsRepository(db *DB) *StatsRepository {
	return &StatsRepository{db: db}
}

// SaveConnectionStats 保存连接统计
func (r *StatsRepository) SaveConnectionStats(ip net.IP, totalConn, activeConn, bytesUp, bytesDown, totalBytes int64, avgLatency time.Duration, lastUsed time.Time) error {
	query := `
		INSERT INTO connection_stats (ip_address, total_connections, active_connections, bytes_up, bytes_down, total_bytes, avg_latency_ms, last_used)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (ip_address) 
		DO UPDATE SET 
			total_connections = EXCLUDED.total_connections,
			active_connections = EXCLUDED.active_connections,
			bytes_up = EXCLUDED.bytes_up,
			bytes_down = EXCLUDED.bytes_down,
			total_bytes = EXCLUDED.total_bytes,
			avg_latency_ms = EXCLUDED.avg_latency_ms,
			last_used = EXCLUDED.last_used,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.Exec(query, ip.String(), totalConn, activeConn, bytesUp, bytesDown, totalBytes, int(avgLatency.Milliseconds()), lastUsed)
	if err != nil {
		logrus.Errorf("Failed to save connection stats for %s: %v", ip.String(), err)
		return err
	}
	return nil
}

// SaveConnectionHistory 保存连接历史
func (r *StatsRepository) SaveConnectionHistory(ip net.IP, duration time.Duration, bytesTransferred int64, startedAt, endedAt time.Time, status string) error {
	query := `
		INSERT INTO connection_history (ip_address, connection_duration_ms, bytes_transferred, started_at, ended_at, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(query, ip.String(), int(duration.Milliseconds()), bytesTransferred, startedAt, endedAt, status)
	if err != nil {
		logrus.Errorf("Failed to save connection history for %s: %v", ip.String(), err)
		return err
	}
	return nil
}

// UpdateGlobalStats 更新全局统计
func (r *StatsRepository) UpdateGlobalStats(totalConn, activeConn, totalBytes, bytesUp, bytesDown int64) error {
	query := `
		UPDATE global_stats 
		SET 
			total_connections = $1,
			active_connections = $2,
			total_bytes_transferred = $3,
			total_bytes_up = $4,
			total_bytes_down = $5,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`
	_, err := r.db.Exec(query, totalConn, activeConn, totalBytes, bytesUp, bytesDown)
	if err != nil {
		logrus.Errorf("Failed to update global stats: %v", err)
		return err
	}
	return nil
}

// GetConnectionStats 获取连接统计
func (r *StatsRepository) GetConnectionStats(ip net.IP) (*ConnectionStatsRow, error) {
	query := `
		SELECT ip_address, total_connections, active_connections, bytes_up, bytes_down, total_bytes, avg_latency_ms, last_used
		FROM connection_stats
		WHERE ip_address = $1
	`
	row := r.db.QueryRow(query, ip.String())
	
	var stats ConnectionStatsRow
	err := row.Scan(&stats.IPAddress, &stats.TotalConnections, &stats.ActiveConnections, &stats.BytesUp, &stats.BytesDown, &stats.TotalBytes, &stats.AvgLatencyMs, &stats.LastUsed)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &stats, nil
}

// GetTopIPsByTraffic 获取Top IP流量排行
func (r *StatsRepository) GetTopIPsByTraffic(limit int) ([]ConnectionStatsRow, error) {
	query := `
		SELECT ip_address, total_connections, active_connections, bytes_up, bytes_down, total_bytes, avg_latency_ms, last_used
		FROM connection_stats
		ORDER BY total_bytes DESC
		LIMIT $1
	`
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []ConnectionStatsRow
	for rows.Next() {
		var s ConnectionStatsRow
		if err := rows.Scan(&s.IPAddress, &s.TotalConnections, &s.ActiveConnections, &s.BytesUp, &s.BytesDown, &s.TotalBytes, &s.AvgLatencyMs, &s.LastUsed); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetConnectionHistory 获取连接历史
func (r *StatsRepository) GetConnectionHistory(ip net.IP, since time.Time, limit int) ([]ConnectionHistoryRow, error) {
	query := `
		SELECT ip_address, connection_duration_ms, bytes_transferred, started_at, ended_at, status
		FROM connection_history
		WHERE ip_address = $1 AND started_at >= $2
		ORDER BY started_at DESC
		LIMIT $3
	`
	rows, err := r.db.Query(query, ip.String(), since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []ConnectionHistoryRow
	for rows.Next() {
		var h ConnectionHistoryRow
		if err := rows.Scan(&h.IPAddress, &h.DurationMs, &h.BytesTransferred, &h.StartedAt, &h.EndedAt, &h.Status); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, rows.Err()
}

// ConnectionStatsRow 连接统计行
type ConnectionStatsRow struct {
	IPAddress        string
	TotalConnections int64
	ActiveConnections int64
	BytesUp          int64
	BytesDown        int64
	TotalBytes       int64
	AvgLatencyMs     int
	LastUsed         time.Time
}

// ConnectionHistoryRow 连接历史行
type ConnectionHistoryRow struct {
	IPAddress        string
	DurationMs       int
	BytesTransferred int64
	StartedAt        time.Time
	EndedAt          sql.NullTime
	Status           string
}



