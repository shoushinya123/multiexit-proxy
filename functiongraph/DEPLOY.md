# FunctionGraph部署详细步骤

## 前提条件

1. 华为云账号
2. 已开通FunctionGraph服务
3. 已开通API Gateway服务（可选）

## 步骤1: 准备函数代码

```bash
cd functiongraph
go mod tidy
go build -o handler handler.go
chmod +x bootstrap
```

## 步骤2: 打包函数

```bash
# 创建函数包
zip -r function.zip bootstrap handler

# 如果使用定制运行时，需要包含所有依赖
```

## 步骤3: 创建FunctionGraph函数

### 3.1 登录控制台

访问：https://console.huaweicloud.com/functiongraph

### 3.2 创建函数

1. 点击"创建函数"
2. 选择"HTTP函数"
3. 配置基本信息：
   - **函数名称**：multiexit-web
   - **运行时**：Go 1.x
   - **函数代码**：上传function.zip
   - **入口函数**：Handler

### 3.3 配置环境变量

在"环境变量"中添加：
```
AUTH_KEY=your-auth-key-here
SERVER_ADDR=your-ecs-server-ip
WEB_USERNAME=admin
WEB_PASSWORD=your-password
```

### 3.4 配置函数设置

- **内存**：256MB（根据需求调整）
- **超时时间**：30秒（HTTP函数最大）
- **并发数**：根据需求配置

## 步骤4: 配置APIG触发器

### 4.1 创建触发器

1. 在函数详情页点击"触发器"
2. 选择"API Gateway"
3. 创建新的API或使用现有API

### 4.2 配置API

- **请求方法**：GET, POST
- **路径**：/{proxy+}
- **认证方式**：IAM认证或自定义认证

## 步骤5: 测试函数

```bash
# 获取函数URL
FUNCTION_URL="https://your-function-url"

# 测试订阅API
curl "${FUNCTION_URL}/api/subscribe?token=YOUR_TOKEN"

# 测试管理API
curl -u admin:password "${FUNCTION_URL}/api/subscription/link"
```

## 步骤6: 配置VPC（如需要）

如果函数需要访问ECS上的TCP服务：

1. 配置函数访问VPC
2. 选择VPC和子网
3. 配置安全组规则

## 步骤7: 配置固定公网IP（如需要）

1. 在VPC中创建NAT网关
2. 绑定弹性公网IP
3. 配置路由表

## 监控和日志

- **日志**：自动收集到LTS（日志服务）
- **监控**：在FunctionGraph控制台查看调用次数、错误率等
- **告警**：配置告警规则

## 成本估算

- **调用次数**：按实际调用计费
- **执行时间**：按GB-秒计费
- **公网流量**：按实际流量计费

## 故障排查

1. **函数无法启动**：检查bootstrap文件权限
2. **超时错误**：增加超时时间或优化代码
3. **网络不通**：检查VPC和安全组配置
4. **认证失败**：检查环境变量配置

## 参考文档

- [FunctionGraph开发指南](https://support.huaweicloud.com/devg-functiongraph/)
- [HTTP函数开发](https://support.huaweicloud.com/devg-functiongraph/functiongraph_02_0603.html)
- [Go函数开发](https://support.huaweicloud.com/devg-functiongraph/functiongraph_02_0504.html)



