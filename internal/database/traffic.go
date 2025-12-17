package database

import (
	"database/sql"
	"time"

	"github.com/sirupsen/logrus"
)

// TrafficRepository 流量分析数据仓库
type TrafficRepository struct {
	db *DB
}

// NewTrafficRepository 创建流量分析数据仓库
func NewTrafficRepository(db *DB) *TrafficRepository {
	return &TrafficRepository{db: db}
}

// SaveDomainStats 保存域名统计
func (r *TrafficRepository) SaveDomainStats(domain string, connections, bytesUp, bytesDown, totalBytes int64, avgLatency time.Duration, lastAccess time.Time) error {
	query := `
		INSERT INTO domain_stats (domain, connections, bytes_up, bytes_down, total_bytes, avg_latency_ms, last_access)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (domain) 
		DO UPDATE SET 
			connections = EXCLUDED.connections,
			bytes_up = EXCLUDED.bytes_up,
			bytes_down = EXCLUDED.bytes_down,
			total_bytes = EXCLUDED.total_bytes,
			avg_latency_ms = EXCLUDED.avg_latency_ms,
			last_access = EXCLUDED.last_access,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.Exec(query, domain, connections, bytesUp, bytesDown, totalBytes, int(avgLatency.Milliseconds()), lastAccess)
	if err != nil {
		logrus.Errorf("Failed to save domain stats for %s: %v", domain, err)
		return err
	}
	return nil
}

// SaveDomainAccessHistory 保存域名访问历史
func (r *TrafficRepository) SaveDomainAccessHistory(domain string, bytesUp, bytesDown int64, latency time.Duration, accessedAt time.Time) error {
	query := `
		INSERT INTO domain_access_history (domain, bytes_up, bytes_down, latency_ms, accessed_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(query, domain, bytesUp, bytesDown, int(latency.Milliseconds()), accessedAt)
	if err != nil {
		logrus.Errorf("Failed to save domain access history for %s: %v", domain, err)
		return err
	}
	return nil
}

// SaveTrafficTrend 保存流量趋势
func (r *TrafficRepository) SaveTrafficTrend(timestamp time.Time, bytesUp, bytesDown, totalBytes, connections int64) error {
	query := `
		INSERT INTO traffic_trends (timestamp, bytes_up, bytes_down, total_bytes, connections)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(query, timestamp, bytesUp, bytesDown, totalBytes, connections)
	if err != nil {
		logrus.Errorf("Failed to save traffic trend: %v", err)
		return err
	}
	return nil
}

// SaveAnomaly 保存异常检测记录
func (r *TrafficRepository) SaveAnomaly(domain, anomalyType, severity string, value, expectedValue float64, description string, detectedAt time.Time) error {
	query := `
		INSERT INTO anomaly_detections (domain, anomaly_type, severity, detected_at, value, expected_value, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(query, domain, anomalyType, severity, detectedAt, value, expectedValue, description)
	if err != nil {
		logrus.Errorf("Failed to save anomaly: %v", err)
		return err
	}
	return nil
}

// GetTopDomainsByTraffic 获取Top域名流量排行
func (r *TrafficRepository) GetTopDomainsByTraffic(limit int) ([]DomainStatsRow, error) {
	query := `
		SELECT domain, connections, bytes_up, bytes_down, total_bytes, avg_latency_ms, last_access
		FROM domain_stats
		ORDER BY total_bytes DESC
		LIMIT $1
	`
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []DomainStatsRow
	for rows.Next() {
		var s DomainStatsRow
		if err := rows.Scan(&s.Domain, &s.Connections, &s.BytesUp, &s.BytesDown, &s.TotalBytes, &s.AvgLatencyMs, &s.LastAccess); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetTrafficTrends 获取流量趋势
func (r *TrafficRepository) GetTrafficTrends(since time.Time, limit int) ([]TrafficTrendRow, error) {
	query := `
		SELECT timestamp, bytes_up, bytes_down, total_bytes, connections
		FROM traffic_trends
		WHERE timestamp >= $1
		ORDER BY timestamp DESC
		LIMIT $2
	`
	rows, err := r.db.Query(query, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trends []TrafficTrendRow
	for rows.Next() {
		var t TrafficTrendRow
		if err := rows.Scan(&t.Timestamp, &t.BytesUp, &t.BytesDown, &t.TotalBytes, &t.Connections); err != nil {
			return nil, err
		}
		trends = append(trends, t)
	}
	return trends, rows.Err()
}

// GetRecentAnomalies 获取最近的异常
func (r *TrafficRepository) GetRecentAnomalies(since time.Time, limit int) ([]AnomalyRow, error) {
	query := `
		SELECT domain, anomaly_type, severity, detected_at, value, expected_value, description
		FROM anomaly_detections
		WHERE detected_at >= $1
		ORDER BY detected_at DESC
		LIMIT $2
	`
	rows, err := r.db.Query(query, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var anomalies []AnomalyRow
	for rows.Next() {
		var a AnomalyRow
		if err := rows.Scan(&a.Domain, &a.AnomalyType, &a.Severity, &a.DetectedAt, &a.Value, &a.ExpectedValue, &a.Description); err != nil {
			return nil, err
		}
		anomalies = append(anomalies, a)
	}
	return anomalies, rows.Err()
}

// GetDomainAccessHistory 获取域名访问历史
func (r *TrafficRepository) GetDomainAccessHistory(domain string, since time.Time, limit int) ([]DomainAccessHistoryRow, error) {
	query := `
		SELECT domain, bytes_up, bytes_down, latency_ms, accessed_at
		FROM domain_access_history
		WHERE domain = $1 AND accessed_at >= $2
		ORDER BY accessed_at DESC
		LIMIT $3
	`
	rows, err := r.db.Query(query, domain, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []DomainAccessHistoryRow
	for rows.Next() {
		var h DomainAccessHistoryRow
		if err := rows.Scan(&h.Domain, &h.BytesUp, &h.BytesDown, &h.LatencyMs, &h.AccessedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, rows.Err()
}

// DomainStatsRow 域名统计行
type DomainStatsRow struct {
	Domain       string
	Connections  int64
	BytesUp      int64
	BytesDown    int64
	TotalBytes   int64
	AvgLatencyMs int
	LastAccess   time.Time
}

// TrafficTrendRow 流量趋势行
type TrafficTrendRow struct {
	Timestamp   time.Time
	BytesUp     int64
	BytesDown   int64
	TotalBytes  int64
	Connections int64
}

// AnomalyRow 异常行
type AnomalyRow struct {
	Domain        sql.NullString
	AnomalyType   string
	Severity      string
	DetectedAt    time.Time
	Value         float64
	ExpectedValue float64
	Description   string
}

// DomainAccessHistoryRow 域名访问历史行
type DomainAccessHistoryRow struct {
	Domain     string
	BytesUp    int64
	BytesDown  int64
	LatencyMs  sql.NullInt64
	AccessedAt time.Time
}



