# FunctionGraph适配总结

## 核心结论

**TCP代理服务不适合迁移到FunctionGraph**，但**Web管理界面和订阅API可以迁移**。

## 原因分析

### FunctionGraph特点（参考官方文档）

根据[华为云FunctionGraph文档](https://support.huaweicloud.com/devg-functiongraph/functiongraph_02_0504.html)：

1. **事件驱动模式**：函数响应事件触发执行
2. **HTTP函数**：处理HTTP请求/响应，不支持长期TCP连接
3. **执行时间限制**：函数有超时限制（HTTP函数最长30秒）
4. **无状态设计**：每次调用都是独立的，无法保持长连接

### 当前代理系统特点

1. **TCP长连接**：需要持续监听端口，保持连接
2. **长期运行**：服务需要7x24小时运行
3. **系统级操作**：需要root权限配置iptables SNAT

## 推荐架构

### 混合架构（最佳方案）

```
┌─────────────────────────────────────┐
│  华为云FunctionGraph                 │
│  ┌───────────────────────────────┐ │
│  │ HTTP函数                       │ │
│  │ - Web管理界面                  │ │
│  │ - 订阅API                      │ │
│  │ - 配置管理API                  │ │
│  └───────────────────────────────┘ │
└──────────────┬──────────────────────┘
               │ API调用
┌──────────────▼──────────────────────┐
│  华为云ECS                           │
│  ┌───────────────────────────────┐ │
│  │ TCP代理服务                    │ │
│  │ - 监听8443端口                 │ │
│  │ - 处理TCP连接                  │ │
│  │ - SNAT功能                     │ │
│  └───────────────────────────────┘ │
└─────────────────────────────────────┘
```

## 实施建议

### Phase 1: 保持现状（当前）

- TCP代理服务：ECS部署 ✅
- Web管理界面：ECS部署 ✅
- 订阅API：ECS部署 ✅

**优点**：简单、稳定、性能好

### Phase 2: 部分迁移（可选）

- Web管理界面 → FunctionGraph
- 订阅API → FunctionGraph  
- TCP代理服务 → 保持ECS

**优点**：
- Web管理弹性扩展
- 按调用计费，成本可控
- TCP代理保持高性能

**缺点**：
- 架构复杂度增加
- 需要配置VPC互通

### Phase 3: 完全迁移（不推荐）

将TCP代理改为HTTP代理，迁移到FunctionGraph。

**缺点**：
- 失去SOCKS5支持
- 性能下降
- 协议兼容性问题

## 代码说明

`functiongraph/` 目录包含：
- `handler.go` - FunctionGraph HTTP函数代码
- `bootstrap` - 入口文件（定制运行时）
- `README.md` - 部署说明
- `DEPLOY.md` - 详细部署步骤

**注意**：FunctionGraph HTTP函数不需要main函数，只需要Handler函数。

## 参考文档

- [Go函数开发概述](https://support.huaweicloud.com/devg-functiongraph/functiongraph_02_0504.html)
- [开发Go事件函数](https://support.huaweicloud.com/devg-functiongraph/functiongraph_02_0441.html)  
- [使用Go开发HTTP函数](https://support.huaweicloud.com/devg-functiongraph/functiongraph_02_0603.html)
- [将代码设计为云函数模式](https://support.huaweicloud.com/devg-functiongraph/functiongraph_01_0406.html)

## 结论

**建议**：保持当前ECS部署方案，FunctionGraph代码作为可选方案保留。

如果未来需要：
- Web管理界面需要弹性扩展 → 考虑迁移到FunctionGraph
- 订阅API需要高可用 → 考虑迁移到FunctionGraph
- TCP代理服务 → 继续使用ECS

