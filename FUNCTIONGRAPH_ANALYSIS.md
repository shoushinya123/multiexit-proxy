# 华为云FunctionGraph适配分析

## 当前架构分析

### 核心组件
1. **TCP代理服务** - 长期运行，监听端口，处理TCP连接
2. **Web管理界面** - HTTP服务，提供管理API
3. **订阅服务** - HTTP API，提供订阅链接

### FunctionGraph限制

根据[华为云FunctionGraph文档](https://support.huaweicloud.com/devg-functiongraph/functiongraph_02_0504.html)：

1. **函数执行模式**：
   - 事件驱动：函数响应事件触发
   - HTTP函数：处理HTTP请求/响应
   - **不支持长期运行的TCP服务器**

2. **执行时间限制**：
   - 函数有超时时间限制
   - 不适合长期保持连接

3. **网络限制**：
   - 支持HTTP/HTTPS访问
   - 不支持自定义TCP端口监听

## 适配方案

### 方案1: 部分功能迁移（推荐）

将**Web管理界面**和**订阅API**迁移到FunctionGraph：

#### 可迁移的组件
- ✅ Web管理界面（HTTP函数）
- ✅ 订阅API（HTTP函数）
- ✅ 配置管理API（HTTP函数）

#### 不可迁移的组件
- ❌ TCP代理服务（需要长期运行）
- ❌ SNAT功能（需要系统级权限）

### 方案2: 混合架构

```
┌─────────────────┐
│  FunctionGraph  │
│  (Web管理+订阅)  │
└────────┬────────┘
         │
┌────────▼────────┐
│   ECS/CCE       │
│  (TCP代理服务)   │
└─────────────────┘
```

- FunctionGraph：处理Web管理和订阅
- ECS/CCE：运行TCP代理服务

### 方案3: 完全改造（不推荐）

将TCP代理改为HTTP代理，但会失去：
- SOCKS5协议支持
- 长连接性能优势
- 部分协议兼容性

## 推荐方案：部分迁移

### 迁移Web管理到FunctionGraph

#### 1. 创建HTTP函数

根据[HTTP函数文档](https://support.huaweicloud.com/devg-functiongraph/functiongraph_02_0603.html)：

```go
package main

import (
    "github.com/functiongraph/functiongraph-go"
    "net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
    // Web管理界面逻辑
    // 路由处理
    // API响应
}
```

#### 2. 函数结构

```
functiongraph-web/
├── bootstrap          # 入口文件
├── handler.go         # HTTP处理函数
├── go.mod
└── static/            # 静态文件
    └── index.html
```

#### 3. 配置要点

- **运行时**：Go 1.x
- **函数类型**：HTTP函数
- **触发器**：APIG触发器
- **超时时间**：30秒（API Gateway限制）

### 保留TCP代理在ECS

TCP代理服务继续在ECS上运行：
- 长期监听端口
- 处理TCP连接
- SNAT功能

## 实施步骤

### Step 1: 创建FunctionGraph HTTP函数

1. 登录华为云控制台
2. 创建函数工作流FunctionGraph
3. 选择Go运行时
4. 创建HTTP函数

### Step 2: 改造Web管理代码

将Web管理界面改造为FunctionGraph HTTP函数格式。

### Step 3: 配置APIG触发器

配置API Gateway触发器，暴露HTTP函数。

### Step 4: 部署TCP代理到ECS

TCP代理服务部署到ECS实例。

## 注意事项

1. **网络配置**：
   - FunctionGraph需要配置VPC访问
   - 确保FunctionGraph可以访问ECS上的TCP服务

2. **配置存储**：
   - 使用OBS存储配置文件
   - 或使用DCS/Redis共享配置

3. **认证机制**：
   - FunctionGraph函数间调用需要配置认证
   - 使用IAM或自定义token

4. **成本考虑**：
   - FunctionGraph按调用次数计费
   - ECS按实例计费
   - 评估混合架构成本

## 总结

**最佳实践**：
- ✅ Web管理界面 → FunctionGraph（HTTP函数）
- ✅ 订阅API → FunctionGraph（HTTP函数）
- ✅ TCP代理服务 → ECS（保持原架构）

这样既利用了FunctionGraph的弹性优势，又保持了TCP代理的性能和功能完整性。

