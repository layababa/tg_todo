# Infra

存放 Docker Compose、部署脚本以及数据库迁移工具配置。

## 本地运行

```bash
cd infra
docker compose up --build
```

该命令会启动以下服务：

1. `db`：Postgres 15，默认账户 `tg_todo/change-me`
2. `api`：Go REST API（端口映射 `8080:8080`）
3. `bot`：Telegram Bot（监听 `/tasks`、`/ping` 等命令）
4. `web`：Vite 构建后的前端静态站点（端口映射 `4173:80`）

## 环境变量

Compose 默认会读取当前 Shell 中的以下变量：

| 变量名 | 作用 | 备注 |
| --- | --- | --- |
| `TELEGRAM_BOT_TOKEN` | 供 API 校验 `initData`、向 Telegram 发送通知，以及 Bot 监听消息 | 必须使用线上 Bot Token，测试/生产请分别配置 |
| `SERVICE_API_TOKEN` | 内部服务访问 REST API 时使用的 Bearer Token（Bot/脚本可复用） | 本地可设为任意随机串，但需要与请求头保持一致 |

示例：

```bash
export TELEGRAM_BOT_TOKEN=123456:abcd
export SERVICE_API_TOKEN=local-service-token
cd infra && docker compose up --build
```

Web 端若在 Telegram 外部调试，可在浏览器 LocalStorage 写入 `tg_todo_debug_init_data`（或在 `.env.development` 里设置 `VITE_TG_INIT_DATA`）以注入真实 `initData`。

> 注意：首次 `docker compose build` 会运行 `npm install`（web）与 `go mod download`（server），需具备外网访问 npm/Golang proxy 的能力。
