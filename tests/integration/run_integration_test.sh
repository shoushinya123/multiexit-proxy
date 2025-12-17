#!/bin/bash

# 集成测试运行脚本

echo "=========================================="
echo "  集成测试 - 端到端功能测试"
echo "=========================================="
echo ""

# 设置测试环境
export TEST_MODE=true

# 1. 加密往返测试
echo "测试 1: 加密往返测试"
go test -v -run TestEncryptionRoundTrip ./tests/integration/... 2>&1 | tee /tmp/integration_test.log

echo ""
echo "测试 2: 并发加密测试"
go test -v -run TestConcurrentEncryption ./tests/integration/... 2>&1 | tee -a /tmp/integration_test.log

echo ""
echo "测试 3: IP选择策略测试"
go test -v -run TestIPSelectorStrategies ./tests/integration/... 2>&1 | tee -a /tmp/integration_test.log

echo ""
echo "=========================================="
echo "  测试结果摘要"
echo "=========================================="

if grep -q "PASS" /tmp/integration_test.log; then
    echo "✅ 集成测试通过"
else
    echo "❌ 部分测试失败，请查看日志"
fi

echo ""
echo "完整日志: /tmp/integration_test.log"

