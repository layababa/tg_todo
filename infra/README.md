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

若需传入 Telegram Token，可在执行命令前导出 `TELEGRAM_BOT_TOKEN` 环境变量。

> 注意：首次 `docker compose build` 会运行 `npm install`（web）与 `go mod download`（server），需具备外网访问 npm/Golang proxy 的能力。
