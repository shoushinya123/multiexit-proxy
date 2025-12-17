# 开发者指南

## 📚 目录

1. [代码结构](#代码结构)
2. [开发环境](#开发环境)
3. [代码规范](#代码规范)
4. [添加新功能](#添加新功能)
5. [调试技巧](#调试技巧)

---

## 代码结构

```
multiexit-proxy/
├── cmd/                    # 可执行程序入口
│   ├── server/            # 服务端入口
│   └── client/            # 客户端入口
├── internal/              # 内部包（不对外暴露）
│   ├── auth/              # 用户认证
│   ├── config/            # 配置管理
│   ├── monitor/           # 监控统计
│   ├── protocol/          # 协议实现
│   ├── proxy/             # 代理核心
│   ├── security/          # 安全功能
│   ├── snat/              # SNAT管理
│   ├── transport/         # 传输层
│   └── web/               # Web管理界面
├── pkg/                   # 公共包（可被外部使用）
│   ├── socks5/            # SOCKS5协议
│   └── subscribe/         # 订阅功能
├── tests/                 # 测试
│   ├── integration/       # 集成测试
│   └── performance/       # 性能测试
└── docs/                  # 文档
```

---

## 开发环境

### 1. 环境要求

- Go 1.21+
- Linux (SNAT功能需要)
- iptables, iproute2
- root权限（SNAT需要）

### 2. 设置开发环境

```bash
# 克隆仓库
git clone <repo>
cd multiexit-proxy

# 安装依赖
go mod download

# 运行测试
go test ./...

# 构建
go build ./cmd/server
go build ./cmd/client
```

---

## 代码规范

### 1. 命名规范

- **包名**: 小写，简短
- **函数名**: 驼峰式，公开函数首字母大写
- **变量名**: 驼峰式
- **常量**: 全大写，下划线分隔

### 2. 错误处理

```go
// 使用fmt.Errorf包装错误
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// 使用自定义错误类型
return &CustomError{Message: "error message"}
```

### 3. 注释规范

```go
// FunctionName 函数功能描述
// 参数说明
// 返回值说明
func FunctionName(param string) error {
    // ...
}
```

---

## 添加新功能

### 示例: 添加新的IP选择策略

1. **定义接口**（如果不存在）:
```go
// internal/snat/selector.go
type IPSelector interface {
    SelectIP(targetAddr string, targetPort int) (net.IP, error)
}
```

2. **实现选择器**:
```go
// internal/snat/custom_selector.go
type CustomSelector struct {
    ips []net.IP
}

func NewCustomSelector(ips []string) (*CustomSelector, error) {
    // 实现
}

func (c *CustomSelector) SelectIP(targetAddr string, targetPort int) (net.IP, error) {
    // 实现选择逻辑
}
```

3. **集成到服务端**:
```go
// internal/proxy/server.go
switch config.Strategy {
case "custom":
    ipSelector, err = snat.NewCustomSelector(config.ExitIPs)
}
```

4. **添加测试**:
```go
// internal/snat/custom_selector_test.go
func TestCustomSelector(t *testing.T) {
    // 测试代码
}
```

---

## 调试技巧

### 1. 日志调试

```go
import "github.com/sirupsen/logrus"

logrus.SetLevel(logrus.DebugLevel)
logrus.Debug("Debug message")
logrus.Info("Info message")
logrus.Error("Error message")
```

### 2. 使用调试器

```bash
# 使用dlv调试
go install github.com/go-delve/delve/cmd/dlv@latest
dlv debug ./cmd/server
```

### 3. 性能分析

```go
import _ "net/http/pprof"

// 在main函数中
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

然后访问: `http://localhost:6060/debug/pprof/`

---

## 贡献指南

1. Fork仓库
2. 创建功能分支: `git checkout -b feature/new-feature`
3. 提交更改: `git commit -am 'Add new feature'`
4. 推送分支: `git push origin feature/new-feature`
5. 创建Pull Request

---

## 测试要求

所有新功能必须包含：
- 单元测试
- 集成测试（如果适用）
- 性能基准测试（如果影响性能）

运行测试：
```bash
go test ./...
go test -cover ./...
go test -bench=. ./...
```

