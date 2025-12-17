-- MultiExit Proxy 数据库初始化脚本
-- 创建数据库表结构

-- 连接统计表
CREATE TABLE IF NOT EXISTS connection_stats (
    id SERIAL PRIMARY KEY,
    ip_address INET NOT NULL,
    total_connections BIGINT DEFAULT 0,
    active_connections BIGINT DEFAULT 0,
    bytes_up BIGINT DEFAULT 0,
    bytes_down BIGINT DEFAULT 0,
    total_bytes BIGINT DEFAULT 0,
    avg_latency_ms INTEGER DEFAULT 0,
    last_used TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(ip_address)
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_connection_stats_ip ON connection_stats(ip_address);
CREATE INDEX IF NOT EXISTS idx_connection_stats_updated ON connection_stats(updated_at);

-- 连接历史记录表
CREATE TABLE IF NOT EXISTS connection_history (
    id SERIAL PRIMARY KEY,
    ip_address INET NOT NULL,
    connection_duration_ms INTEGER,
    bytes_transferred BIGINT DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) DEFAULT 'completed' -- 'completed', 'failed', 'timeout'
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_connection_history_ip ON connection_history(ip_address);
CREATE INDEX IF NOT EXISTS idx_connection_history_started ON connection_history(started_at);
CREATE INDEX IF NOT EXISTS idx_connection_history_ended ON connection_history(ended_at);

-- 域名统计表
CREATE TABLE IF NOT EXISTS domain_stats (
    id SERIAL PRIMARY KEY,
    domain VARCHAR(255) NOT NULL,
    connections BIGINT DEFAULT 0,
    bytes_up BIGINT DEFAULT 0,
    bytes_down BIGINT DEFAULT 0,
    total_bytes BIGINT DEFAULT 0,
    avg_latency_ms INTEGER DEFAULT 0,
    last_access TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(domain)
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_domain_stats_domain ON domain_stats(domain);
CREATE INDEX IF NOT EXISTS idx_domain_stats_updated ON domain_stats(updated_at);
CREATE INDEX IF NOT EXISTS idx_domain_stats_last_access ON domain_stats(last_access);

-- 域名访问历史表
CREATE TABLE IF NOT EXISTS domain_access_history (
    id SERIAL PRIMARY KEY,
    domain VARCHAR(255) NOT NULL,
    bytes_up BIGINT DEFAULT 0,
    bytes_down BIGINT DEFAULT 0,
    latency_ms INTEGER,
    accessed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_domain_access_domain ON domain_access_history(domain);
CREATE INDEX IF NOT EXISTS idx_domain_access_time ON domain_access_history(accessed_at);

-- 流量趋势表
CREATE TABLE IF NOT EXISTS traffic_trends (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    bytes_up BIGINT DEFAULT 0,
    bytes_down BIGINT DEFAULT 0,
    total_bytes BIGINT DEFAULT 0,
    connections BIGINT DEFAULT 0
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_traffic_trends_timestamp ON traffic_trends(timestamp);

-- 异常检测表
CREATE TABLE IF NOT EXISTS anomaly_detections (
    id SERIAL PRIMARY KEY,
    domain VARCHAR(255),
    anomaly_type VARCHAR(50) NOT NULL, -- 'traffic_spike', 'connection_anomaly', 'latency_anomaly'
    severity VARCHAR(20) NOT NULL, -- 'low', 'medium', 'high'
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    value DOUBLE PRECISION,
    expected_value DOUBLE PRECISION,
    description TEXT
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_anomaly_domain ON anomaly_detections(domain);
CREATE INDEX IF NOT EXISTS idx_anomaly_detected_at ON anomaly_detections(detected_at);
CREATE INDEX IF NOT EXISTS idx_anomaly_type ON anomaly_detections(anomaly_type);
CREATE INDEX IF NOT EXISTS idx_anomaly_severity ON anomaly_detections(severity);

-- 全局统计表
CREATE TABLE IF NOT EXISTS global_stats (
    id SERIAL PRIMARY KEY,
    total_connections BIGINT DEFAULT 0,
    active_connections BIGINT DEFAULT 0,
    total_bytes_transferred BIGINT DEFAULT 0,
    total_bytes_up BIGINT DEFAULT 0,
    total_bytes_down BIGINT DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(id)
);

-- 插入初始全局统计记录
INSERT INTO global_stats (id, total_connections, active_connections, total_bytes_transferred, total_bytes_up, total_bytes_down)
VALUES (1, 0, 0, 0, 0, 0)
ON CONFLICT (id) DO NOTHING;

-- 创建更新时间触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为需要自动更新updated_at的表创建触发器
CREATE TRIGGER update_connection_stats_updated_at BEFORE UPDATE ON connection_stats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_domain_stats_updated_at BEFORE UPDATE ON domain_stats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_global_stats_updated_at BEFORE UPDATE ON global_stats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 创建清理旧数据的函数（可选，用于定期清理历史数据）
CREATE OR REPLACE FUNCTION cleanup_old_data(days_to_keep INTEGER DEFAULT 30)
RETURNS void AS $$
BEGIN
    -- 清理超过指定天数的连接历史
    DELETE FROM connection_history 
    WHERE ended_at < CURRENT_TIMESTAMP - (days_to_keep || ' days')::INTERVAL;
    
    -- 清理超过指定天数的域名访问历史
    DELETE FROM domain_access_history 
    WHERE accessed_at < CURRENT_TIMESTAMP - (days_to_keep || ' days')::INTERVAL;
    
    -- 清理超过指定天数的流量趋势（保留更长时间，如90天）
    DELETE FROM traffic_trends 
    WHERE timestamp < CURRENT_TIMESTAMP - (90 || ' days')::INTERVAL;
    
    -- 清理超过指定天数的异常检测记录（保留更长时间，如60天）
    DELETE FROM anomaly_detections 
    WHERE detected_at < CURRENT_TIMESTAMP - (60 || ' days')::INTERVAL;
END;
$$ LANGUAGE plpgsql;

-- 创建视图：实时统计概览
CREATE OR REPLACE VIEW stats_overview AS
SELECT 
    (SELECT total_connections FROM global_stats WHERE id = 1) as total_connections,
    (SELECT active_connections FROM global_stats WHERE id = 1) as active_connections,
    (SELECT total_bytes_transferred FROM global_stats WHERE id = 1) as total_bytes_transferred,
    (SELECT total_bytes_up FROM global_stats WHERE id = 1) as total_bytes_up,
    (SELECT total_bytes_down FROM global_stats WHERE id = 1) as total_bytes_down,
    (SELECT COUNT(*) FROM connection_stats) as total_ips,
    (SELECT COUNT(*) FROM domain_stats) as total_domains,
    (SELECT COUNT(*) FROM anomaly_detections WHERE detected_at > CURRENT_TIMESTAMP - INTERVAL '24 hours') as anomalies_24h;

-- 创建视图：Top IPs by traffic
CREATE OR REPLACE VIEW top_ips_by_traffic AS
SELECT 
    ip_address,
    total_bytes,
    bytes_up,
    bytes_down,
    total_connections,
    active_connections,
    avg_latency_ms,
    last_used
FROM connection_stats
ORDER BY total_bytes DESC
LIMIT 100;

-- 创建视图：Top Domains by traffic
CREATE OR REPLACE VIEW top_domains_by_traffic AS
SELECT 
    domain,
    total_bytes,
    bytes_up,
    bytes_down,
    connections,
    avg_latency_ms,
    last_access
FROM domain_stats
ORDER BY total_bytes DESC
LIMIT 100;

-- 创建视图：Recent anomalies
CREATE OR REPLACE VIEW recent_anomalies AS
SELECT 
    domain,
    anomaly_type,
    severity,
    detected_at,
    value,
    expected_value,
    description
FROM anomaly_detections
WHERE detected_at > CURRENT_TIMESTAMP - INTERVAL '7 days'
ORDER BY detected_at DESC;

-- 输出初始化完成信息
DO $$
BEGIN
    RAISE NOTICE 'Database initialization completed successfully!';
    RAISE NOTICE 'Database: multiexit_proxy';
    RAISE NOTICE 'User: multiexit';
    RAISE NOTICE 'Tables created: connection_stats, connection_history, domain_stats, domain_access_history, traffic_trends, anomaly_detections, global_stats';
    RAISE NOTICE 'Views created: stats_overview, top_ips_by_traffic, top_domains_by_traffic, recent_anomalies';
END $$;



