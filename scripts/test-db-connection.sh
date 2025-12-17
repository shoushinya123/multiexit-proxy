#!/bin/bash

# 测试数据库连接脚本

echo "=========================================="
echo "测试 PostgreSQL 数据库连接"
echo "=========================================="

# 测试连接
echo ""
echo "1. 测试数据库连接..."
docker exec multiexit-proxy-postgres pg_isready -U multiexit -d multiexit_proxy

if [ $? -eq 0 ]; then
    echo "✅ 数据库连接正常"
else
    echo "❌ 数据库连接失败"
    exit 1
fi

# 测试查询
echo ""
echo "2. 测试查询表结构..."
docker exec multiexit-proxy-postgres psql -U multiexit -d multiexit_proxy -c "
SELECT 
    table_name,
    (SELECT COUNT(*) FROM information_schema.columns WHERE table_name = t.table_name) as column_count
FROM information_schema.tables t
WHERE table_schema = 'public' 
AND table_type = 'BASE TABLE'
ORDER BY table_name;
"

# 测试视图
echo ""
echo "3. 测试视图..."
docker exec multiexit-proxy-postgres psql -U multiexit -d multiexit_proxy -c "
SELECT table_name 
FROM information_schema.views 
WHERE table_schema = 'public'
ORDER BY table_name;
"

# 测试统计概览
echo ""
echo "4. 测试统计概览视图..."
docker exec multiexit-proxy-postgres psql -U multiexit -d multiexit_proxy -c "SELECT * FROM stats_overview;"

echo ""
echo "=========================================="
echo "✅ 数据库测试完成"
echo "=========================================="



