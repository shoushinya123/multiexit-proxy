# PostgreSQL 数据库配置说明

## 📋 数据库信息

### 连接信息
- **主机**: `localhost`
- **端口**: `5432`
- **数据库名**: `multiexit_proxy`
- **用户名**: `multiexit`
- **密码**: `multiexit123`

### 连接字符串
```
postgresql://multiexit:multiexit123@localhost:5432/multiexit_proxy
```

## 📊 数据库表结构

### 核心表（7个）

1. **connection_stats** - IP连接统计
   - 存储每个IP的连接数、流量、延迟等统计信息
   - 主键：`id`
   - 唯一索引：`ip_address`

2. **connection_history** - 连接历史记录
   - 记录每次连接的详细信息
   - 包括连接时长、传输字节数、状态等

3. **domain_stats** - 域名统计
   - 存储每个域名的访问统计
   - 包括连接数、流量、延迟等

4. **domain_access_history** - 域名访问历史
   - 记录每次域名访问的详细信息

5. **traffic_trends** - 流量趋势
   - 按时间点记录流量趋势数据

6. **anomaly_detections** - 异常检测记录
   - 存储检测到的流量异常信息

7. **global_stats** - 全局统计
   - 存储系统级别的总体统计数据

### 视图（4个）

1. **stats_overview** - 统计概览
   - 提供系统整体统计信息的快速视图

2. **top_ips_by_traffic** - Top IP流量排行
   - 按流量排序的前100个IP

3. **top_domains_by_traffic** - Top域名流量排行
   - 按流量排序的前100个域名

4. **recent_anomalies** - 最近异常
   - 最近7天的异常检测记录

## 🚀 常用命令

### 启动数据库
```bash
docker start multiexit-proxy-postgres
```

### 停止数据库
```bash
docker stop multiexit-proxy-postgres
```

### 查看日志
```bash
docker logs -f multiexit-proxy-postgres
```

### 进入数据库命令行
```bash
docker exec -it multiexit-proxy-postgres psql -U multiexit -d multiexit_proxy
```

### 测试连接
```bash
./scripts/test-db-connection.sh
```

### 查看表结构
```bash
docker exec multiexit-proxy-postgres psql -U multiexit -d multiexit_proxy -c "\dt"
```

### 查看统计概览
```bash
docker exec multiexit-proxy-postgres psql -U multiexit -d multiexit_proxy -c "SELECT * FROM stats_overview;"
```

## 🔧 数据清理

数据库提供了自动清理旧数据的函数：

```sql
-- 清理30天前的历史数据
SELECT cleanup_old_data(30);

-- 清理60天前的历史数据
SELECT cleanup_old_data(60);
```

## 📝 示例查询

### 查看Top 10 IP流量
```sql
SELECT * FROM top_ips_by_traffic LIMIT 10;
```

### 查看Top 10域名流量
```sql
SELECT * FROM top_domains_by_traffic LIMIT 10;
```

### 查看最近24小时的异常
```sql
SELECT * FROM recent_anomalies 
WHERE detected_at > NOW() - INTERVAL '24 hours'
ORDER BY detected_at DESC;
```

### 查看最近1小时的流量趋势
```sql
SELECT * FROM traffic_trends 
WHERE timestamp > NOW() - INTERVAL '1 hour'
ORDER BY timestamp DESC;
```

## 🔐 安全建议

⚠️ **生产环境使用前，请务必修改默认密码！**

修改密码方法：
```bash
docker exec -it multiexit-proxy-postgres psql -U multiexit -d multiexit_proxy -c "ALTER USER multiexit WITH PASSWORD 'your_strong_password';"
```

## 📦 数据持久化

数据库数据存储在Docker卷 `postgres_data` 中，即使容器删除，数据也会保留。

查看卷信息：
```bash
docker volume inspect postgres_data
```

备份数据库：
```bash
docker exec multiexit-proxy-postgres pg_dump -U multiexit multiexit_proxy > backup.sql
```

恢复数据库：
```bash
docker exec -i multiexit-proxy-postgres psql -U multiexit -d multiexit_proxy < backup.sql
```



