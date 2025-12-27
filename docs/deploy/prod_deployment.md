# 部署指南 (Production Deployment Guide)

## 1. SSH 连接

为了方便连接服务器，已在 `~/.ssh/config` 中配置了别名。

直接在终端输入以下命令即可登录：

```bash
ssh tg_todo_prod
```

---

## 2. 生产环境配置

项目采用 **Cloudflare Tunnel** 处理 HTTPS 流量，服务器无需开放 80/443 端口。

### 2.1 环境变量准备

在本地 `infra/.env.prod` 中配置真实的生产环境参数：

- `TELEGRAM_BOT_TOKEN`
- `POSTGRES_PASSWORD`
- `VITE_API_BASE_URL` (Cloudflare 绑定的前端域名)

### 2.2 部署步骤

1. **安装 Cloudflared**: 在控制台选择 Debian 64-bit 并运行生成的安装命令。
2. **连接验证**: 确保 Cloudflare Zero Trust 面板显示状态为 `HEALTHY`。
3. **域名绑定**: 在 Public Hostnames 中配置：
   - `api.yourdomain.com` -> `http://localhost:8080`
   - `app.yourdomain.com` -> `http://localhost:4173`
4. **发布代码**: (等待后续步骤)
