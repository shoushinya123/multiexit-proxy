# FunctionGraph快速开始

## 当前状态

✅ **已完成**：FunctionGraph适配代码已准备
⚠️ **建议**：TCP代理服务保持ECS部署

## 如果决定迁移Web管理到FunctionGraph

### 1. 编译函数代码

```bash
cd functiongraph
go mod tidy
# FunctionGraph HTTP函数不需要编译，直接上传源码
```

### 2. 创建函数包

```bash
zip -r function.zip handler.go go.mod go.sum bootstrap
```

### 3. 在华为云控制台

1. 创建HTTP函数
2. 上传function.zip
3. 配置环境变量
4. 创建APIG触发器

### 4. 测试

```bash
curl "https://your-function-url/api/subscribe?token=YOUR_TOKEN"
```

## 注意事项

- FunctionGraph HTTP函数**不需要main函数**
- 只需要实现`Handler(w http.ResponseWriter, r *http.Request)`
- 函数会自动处理HTTP请求

