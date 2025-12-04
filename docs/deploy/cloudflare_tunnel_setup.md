# Cloudflare Tunnel 开发环境配置指南

本指南用于将本地开发环境暴露至公网，支持 Telegram Mini App 调试。

## 1. 架构说明

我们采用 **混合开发模式** 以获得最佳体验：

- **后端 (API/DB)**: 运行在 Docker 中 (稳定、环境一致)。
- **前端 (Web)**: 运行在本地终端 (支持 HMR 热更新)。

| 服务    | 本地地址         | 公网域名 (示例)        | 备注                          |
| :------ | :--------------- | :--------------------- | :---------------------------- |
| **API** | `localhost:8080` | `ddddapi.zcvyzest.xyz` | 运行在 Docker (`infra-api-1`) |
| **Web** | `localhost:5173` | `ddddapp.zcvyzest.xyz` | 运行在本地 (`npm run dev`)    |

---

## 2. 前置准备

1.  **安装 Cloudflared**: `brew install cloudflared`
2.  **启动后端**: `cd infra && docker-compose up -d`
3.  **启动前端**: `cd web && npm run dev`
    - _注意：如果 Docker 中也启动了 web，请先停止它：`docker-compose stop web`_

---

## 3. Tunnel 配置 (Cloudflare Dashboard)

进入 [Zero Trust Dashboard](https://dash.cloudflare.com/) > **Networks** > **Tunnels**，配置 **Public Hostnames**：

### 规则 A：后端 API

- **Subdomain**: `ddddapi` (或自定义)
- **Domain**: `zcvyzest.xyz`
- **Path**: (留空)
- **Service**: `HTTP` -> `localhost:8080`

### 规则 B：前端 Web

- **Subdomain**: `ddddapp` (或自定义)
- **Domain**: `zcvyzest.xyz`
- **Path**: (留空)
- **Service**: `HTTP` -> `localhost:5173`

---

## 4. 关键配置 (必做)

为了让前端能通过公网域名访问，必须修改 `web/vite.config.ts`：

```typescript
// web/vite.config.ts
export default defineConfig({
  // ...
  server: {
    port: 5173,
    // 必须添加你的 Tunnel 域名，否则会出现 "Blocked request"
    allowedHosts: ["ddddapp.zcvyzest.xyz"],
  },
});
```

## 5. 验证

1.  **API**: 访问 `https://ddddapi.zcvyzest.xyz/healthz` → 返回 JSON 成功。
2.  **Web**: 访问 `https://ddddapp.zcvyzest.xyz` → 显示前端页面。
