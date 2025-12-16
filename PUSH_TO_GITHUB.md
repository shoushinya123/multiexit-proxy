# 推送到GitHub指南

## 步骤

### 1. 在GitHub创建仓库

1. 登录GitHub
2. 点击右上角 "+" → "New repository"
3. 仓库名称：`multiexit-proxy`
4. 描述：`A multi-exit IP proxy system with SNAT support, encryption, and traffic obfuscation`
5. 选择：**Public**（公共仓库）
6. **不要**初始化README、.gitignore或license（我们已经有了）
7. 点击"Create repository"

### 2. 推送代码到GitHub

执行以下命令（将YOUR_USERNAME替换为你的GitHub用户名）：

```bash
# 添加远程仓库
git remote add origin https://github.com/YOUR_USERNAME/multiexit-proxy.git

# 推送代码
git branch -M main
git push -u origin main
```

### 3. 如果需要使用SSH（推荐）

```bash
# 使用SSH URL
git remote set-url origin git@github.com:YOUR_USERNAME/multiexit-proxy.git

# 推送
git push -u origin main
```

## 仓库信息

- **仓库名**：`multiexit-proxy`
- **描述**：A multi-exit IP proxy system with SNAT support, encryption, and traffic obfuscation
- **标签建议**：`go`, `proxy`, `snat`, `tls`, `socks5`, `networking`, `multi-ip`, `vpn-alternative`

## 注意事项

1. 确保`.gitignore`已正确配置，避免提交敏感信息
2. 不要提交包含密钥的配置文件
3. 已排除的文件：
   - `*.pem`, `*.key` - 证书和密钥
   - `configs/server.yaml`, `configs/client.json` - 实际配置文件
   - `*.log` - 日志文件
   - `deploy/` - 部署目录

## 后续操作

推送成功后，可以：

1. 添加仓库描述和标签
2. 添加LICENSE文件（如MIT、Apache 2.0等）
3. 配置GitHub Actions进行CI/CD
4. 添加GitHub Pages文档（如果需要）

## 代码已准备好

✅ Git仓库已初始化
✅ .gitignore已配置
✅ README.md已更新
✅ 代码已提交到本地仓库

只需创建GitHub仓库并推送即可！

