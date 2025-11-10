# Telegram To-Do Mini App

Go + Vue 实现的 Telegram Mini App，用于管理群组任务。仓库同时包含 REST API、Telegram Bot、Vite 前端以及本地 Docker Compose 环境。

## 目录结构

```
.
├── AGENTS.md            # 开发规范
├── infra/               # docker compose、部署脚本
├── server/              # Go API + Bot
└── web/                 # Vite + Vue 3 Mini App
```

## 核心特性

- **真实任务写入**：Bot 将群聊消息解析后写入 Postgres，API/前端读取同一份数据。
- **严格鉴权**：所有 REST 调用必须携带 `X-Telegram-Init-Data`，或由内部服务使用 `SERVICE_API_TOKEN`。
- **生命周期通知**：任务完成 / 重新打开 / 删除时，自动向创建人和其他指派人推送 Telegram 消息。
- **CI 覆盖**：GitHub Actions 跑通 `go test`、ESLint、Vitest，避免回归。

## 环境变量

| 作用域 | 变量 | 说明 |
| --- | --- | --- |
| server/bot | `POSTGRES_URL` | 数据库连接串（`postgres://user:pass@host:5432/db?sslmode=disable`） |
| server/bot | `TELEGRAM_BOT_TOKEN` | Bot Token，用于校验 `initData` 及调用 Telegram API |
| server/bot | `TELEGRAM_API_URL` | Telegram API 地址（默认 `https://api.telegram.org`） |
| server/api/bot | `SERVICE_API_TOKEN` | 内部服务访问 REST API 的 Bearer Token，需要 client 同步携带 |
| web | `VITE_API_BASE_URL` | 前端请求的 API 根路径，默认为 `https://api.xwqpfzmlj.com` |
| web | `VITE_TG_INIT_DATA` | （本地调试可选）预置的 Telegram `initData` 字符串，会注入 `X-Telegram-Init-Data` |

> Tip：在浏览器控制台运行 `Telegram.WebApp.initData` 可复制真实 `initData`，也可以通过 `localStorage.setItem('tg_todo_debug_init_data', '...')` 长期缓存，便于在非 Telegram 环境调试。

## 本地启动

### 直接运行

```bash
# API
cd server
export POSTGRES_URL="postgres://tg_todo:change-me@localhost:5432/tg_todo?sslmode=disable"
export TELEGRAM_BOT_TOKEN="123456:abc"
export SERVICE_API_TOKEN="local-service-token"
go run ./cmd/api

# Bot
go run ./cmd/bot

# Web（另开终端）
cd web
npm install
npm run dev
```

### Docker Compose

```bash
export TELEGRAM_BOT_TOKEN="123456:abc"
export SERVICE_API_TOKEN="local-service-token"
cd infra
docker compose up --build
```

Compose 会同时启动 Postgres、API、Bot 与打包后的前端（`http://localhost:4173`）。

## 测试

```bash
# Go 单元测试
cd server
GOCACHE=$(pwd)/.gocache go test ./...

# 前端单元测试 + Lint
cd web
npm run lint
npm run test

# Cypress（需先启动 `npm run dev` 或部署版本）
npm run e2e:headless
```

CI 工作流位于 `.github/workflows/ci.yml`，默认在 Push/PR 触发上述 Go 测试与前端 lint/test。

## API 鉴权与通知流程

- 前端所有请求必须带上 `X-Telegram-Init-Data`，API 会使用 Bot Token 导出的 HMAC key 复验签名，校验通过后才能读取/写入任务。
- 内部任务（如 Bot 或运维脚本）可以携带 `Authorization: Bearer ${SERVICE_API_TOKEN}` 绕过 initData 校验，但该 Token 只应在受信任环境使用。
- 任务状态从 Pending→Completed、Completed→Pending 以及 Delete 时，API 通过 `TelegramNotifier` 触发通知：
  - `任务《{title}》已由 {actor} 标记完成。原始消息：{sourceUrl}`
  - `任务《{title}》已由 {actor} 重新打开。`
  - `任务《{title}》已被 {actor} 删除。`

如有新动作影响通知策略，请同步更新 `AGENTS.md` 与本文档。
