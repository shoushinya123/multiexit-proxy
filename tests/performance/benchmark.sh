#!/bin/bash

# 性能基准测试脚本

echo "=========================================="
echo "  性能基准测试"
echo "=========================================="
echo ""

# 运行基准测试
echo "1. 加密性能基准测试..."
go test -bench=BenchmarkEncryptionThroughput -benchmem -benchtime=5s ./tests/performance/... 2>&1

echo ""
echo "2. 解密性能基准测试..."
go test -bench=BenchmarkDecryptionThroughput -benchmem -benchtime=5s ./tests/performance/... 2>&1

echo ""
echo "3. 并发加密基准测试..."
go test -bench=BenchmarkConcurrentEncryption -benchmem -benchtime=3s ./tests/performance/... 2>&1

echo ""
echo "4. 负载测试..."
go test -run=TestEncryptionLoad -timeout=60s ./tests/performance/... -v 2>&1

echo ""
echo "=========================================="
echo "  测试完成"
echo "=========================================="

