# 测试文档

## 测试结构

```
tests/
├── integration/          # 集成测试
│   ├── integration_test.go
│   └── run_integration_test.sh
└── performance/          # 性能测试
    ├── load_test.go
    └── benchmark.sh
```

---

## 运行测试

### 单元测试

```bash
# 运行所有单元测试
go test ./...

# 运行特定包的测试
go test ./internal/protocol/...
go test ./internal/snat/...

# 带覆盖率的测试
go test -cover ./...
```

### 集成测试

```bash
# 运行集成测试
./tests/integration/run_integration_test.sh

# 或直接运行
go test -v ./tests/integration/...
```

### 性能测试

```bash
# 运行性能基准测试
./tests/performance/benchmark.sh

# 或运行特定基准测试
go test -bench=. -benchmem ./tests/performance/...
```

---

## 测试覆盖率

```bash
# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

---

## 测试说明

### 集成测试

集成测试需要完整的系统环境，包括：
- 网络连接
- 可能需要的root权限（SNAT测试）
- 真实的IP地址（某些测试）

某些测试在CI/CD环境中可能会跳过。

### 性能测试

性能测试会进行：
- 加密/解密吞吐量测试
- 并发性能测试
- 负载测试
- 内存使用测试

### 基准测试

基准测试使用Go的benchmark工具，可以对比不同版本的性能。

---

## 持续集成

建议在CI/CD中运行：
1. 单元测试（所有环境）
2. 集成测试（特定环境）
3. 性能基准测试（定期运行）

