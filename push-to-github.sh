#!/bin/bash
# 推送到GitHub脚本

echo "=== 推送到GitHub ==="
echo ""

# 检查是否已配置远程仓库
if git remote get-url origin >/dev/null 2>&1; then
    echo "✅ 远程仓库已配置:"
    git remote get-url origin
    echo ""
    echo "开始推送..."
    git branch -M main 2>/dev/null || true
    git push -u origin main
else
    echo "⚠️  远程仓库未配置"
    echo ""
    echo "请先执行以下命令（替换YOUR_USERNAME为你的GitHub用户名）:"
    echo ""
    echo "  git remote add origin https://github.com/YOUR_USERNAME/multiexit-proxy.git"
    echo "  git branch -M main"
    echo "  git push -u origin main"
    echo ""
    echo "或者使用SSH:"
    echo "  git remote add origin git@github.com:YOUR_USERNAME/multiexit-proxy.git"
    echo "  git branch -M main"
    echo "  git push -u origin main"
fi
