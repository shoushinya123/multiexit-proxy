## MultiExit Proxy

一个高性能的多出口 IP 代理系统，支持 SNAT、多出口智能调度和可视化 Web 管理面板。

### 功能概览

- **多出口 IP**: 支持多公网 IP，按轮询、规则、地理位置、流量等策略分配出口。
- **协议支持**: SOCKS5、Trojan，支持 TCP/UDP。
- **健康检查与流量分析**: 自动剔除故障出口，提供连接/流量统计与异常分析。
- **Web 管理面板**: 通过浏览器管理配置、出口 IP、规则、版本回滚等。
- **安全特性**: TLS 加密、CSRF 防护、登录保护、IP 黑白名单、速率限制等。

### 环境要求

- Go 1.21+
- Node.js 20+ 和 npm / pnpm（用于前端）
- 可选：本地 PostgreSQL（如需启用数据库相关统计）

### 后端启动

```bash
cd /Users/shoushinya/123123
go run cmd/server/main.go -config configs/server.yaml
```

启动后：
- API 监听：`http://localhost:8080`
- 关键接口：`/api/status`, `/api/config`, `/api/ips`, `/api/rules`, `/api/traffic`

### 前端启动

```bash
cd /Users/shoushinya/123123/frontend-system-design
npm install
npm run dev -- -p 8081
```

然后在浏览器访问：
- 登录与面板：`http://localhost:8081`
- 默认管理账号密码：在 `configs/server.yaml` 的 `web.username` / `web.password` 中配置（示例为 `admin` / `admin123`）。

前端通过 Next.js `rewrites` 将 `/api/*` 代理到 `http://localhost:8080/api/*`，CSRF Token 会自动从后端获取并附加到请求中。

### 配置管理说明

- **配置加载**: 前端通过 `GET /api/config` 读取当前配置，并在页面中展示。
- **配置保存**: 前端通过 `POST /api/config` 提交完整配置，后端会进行校验并写入配置文件，同时创建配置版本，支持回滚。
- **出口 IP 管理**: 在「IP 管理」页面可以查看自动检测到的 IP 与配置中的 `exit_ips`，仅配置中的 IP 支持删除。

### 一键开发脚本

仓库中保留了用于本地一键开发的脚本：

- `start-dev.sh`：可选的开发启动脚本（如果你喜欢通过脚本一键启动后端/前端）。
- `scripts/deploy.sh`：一键部署相关脚本（根据需要自行修改）。

其他 `.sh` 脚本已按要求移除，如需自定义部署流程，建议基于上述脚本自行扩展。

### 贡献与问题反馈

- 提交 Pull Request 前，请确保：
  - `go test ./...` 通过；
  - 前端能在本地正常启动并完成基础操作（登录、查看配置、保存配置、查看 IP 与流量统计）。
- Bug 反馈与新功能建议可以直接在 GitHub Issues 中提交。


