# Server / Bot

Go 实现的 REST API 与 Telegram Bot：

- `cmd/api`：Mini App 访问的 HTTP 接口（目前以内存任务模拟数据，后续接入 Postgres）
- `cmd/bot`：Telegram Bot 占位程序，后续会监听消息并写入任务
- `internal/*`：配置、任务领域服务、HTTP 路由、数据库封装等
- `migrations/`：Postgres 迁移文件

## 常用命令

```bash
cd server
go mod tidy            # 首次执行以拉取 pgx/postgres 依赖
go test ./...
go run ./cmd/api
go run ./cmd/bot
make docker-build
```

如在 macOS 上遇到 `go test` 缓存权限问题，可通过 `GOCACHE=$(pwd)/.gocache go test ./...` 指定临时缓存目录。

## 配置说明

- `POSTGRES_URL`：Go API / Bot 统一使用的连接串
- `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS` / `DB_CONN_MAX_LIFETIME`：连接池参数，容器中已给出默认值
- 应用启动时会自动执行 `migrations/*.sql` 中的脚本初始化/更新数据表，并插入与 PRD 一致的示例任务，方便前端联调。
