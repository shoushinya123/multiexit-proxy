# 🚀 快速启动指南

## 一键启动（推荐）

```bash
# 1. 给脚本添加执行权限（首次运行）
chmod +x start-dev.sh stop-dev.sh

# 2. 启动前后端服务
./start-dev.sh
```

脚本会自动：
- ✅ 检查Go和Node.js环境
- ✅ 检查并创建配置文件
- ✅ 编译后端服务
- ✅ 安装前端依赖（如果需要）
- ✅ 启动后端服务（端口8080）
- ✅ 启动前端服务（端口8081）

## 访问地址

启动成功后，访问：

- **前端界面**: http://localhost:8081
- **后端API**: http://localhost:8080/api
- **Web管理界面**: http://localhost:8080

## 登录信息

- **用户名**: `admin`
- **密码**: `admin123`

## 停止服务

```bash
# 方式1: 使用停止脚本
./stop-dev.sh

# 方式2: 按 Ctrl+C（如果在启动脚本的终端中）
```

## 手动启动（如果需要）

### 启动后端

```bash
# 1. 编译
go build -o server ./cmd/server

# 2. 启动
./server -config configs/server.yaml
```

### 启动前端

```bash
# 1. 进入前端目录
cd frontend-system-design

# 2. 安装依赖（首次运行）
pnpm install  # 或 npm install

# 3. 启动开发服务器
pnpm dev  # 或 npm run dev
```

## 验证功能

详细的功能验证清单请查看：[VERIFICATION_GUIDE.md](./VERIFICATION_GUIDE.md)

## 常见问题

### 端口被占用

如果8080或8081端口被占用，可以：

1. **修改后端端口**: 编辑 `configs/server.yaml` 中的 `web.listen`
2. **修改前端端口**: 编辑 `frontend-system-design/package.json` 中的 `dev` 脚本，修改 `-p 8081` 为其他端口

### 前端无法连接后端

1. 确认后端已启动: `curl http://localhost:8080/api/status`
2. 检查浏览器控制台错误信息
3. 确认Next.js代理配置正确（已自动配置）

### 登录失败

1. 检查后端配置中的用户名和密码
2. 查看后端日志: `tail -f multiexit-proxy.log`
3. 清除浏览器缓存和localStorage

## 下一步

- 📖 查看 [功能验证指南](./VERIFICATION_GUIDE.md) 进行完整测试
- 📖 查看 [README.md](./README.md) 了解系统架构和配置
- 🐛 遇到问题？查看验证指南中的"常见问题"部分

