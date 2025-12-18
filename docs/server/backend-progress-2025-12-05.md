# 服务端阶段成果 · 2025-12-05

> 记录截至 2025-12-05 在 Module 1-2 期间的交付成果、验收方式与依赖，便于后续迭代溯源。原始记录来源于《当前工作日志》，本文件作为档案保存。

## 模块 1：基础设施 & 健康检查
- ✅ `cmd/api` 启动程序：加载配置、初始化 logger、中间件与 `/healthz` 路由。
- ✅ `internal/config`：`Load()` 支持配置文件 + 环境变量兜底，`config/default.yml` 提供示例。
- ✅ HTTP 中间件：RequestID 生成/透传 `X-Request-ID`，Recovery 输出 `{success:false,...}`。
- ✅ Healthz handler：注入 DB/Redis 依赖，返回版本、Git Hash、依赖状态。
- ✅ 数据库/Redis 连接封装可被后续模块复用，`server/pkg/db` & `pkg/redis` 已完成。
- ✅ Dockerfile 与 docker-compose 可构建/运行 API + Postgres + Redis；`curl http://localhost:8081/healthz` 验收通过。

## 模块 2：身份 & Onboarding API
### 步骤 1：数据模型与 Repository
- `internal/models/user.go` 定义 `User`、`UserNotionToken`。
- `internal/repository/user_repository.go` 实现 CRUD 与 token 查询。

### 步骤 2：Telegram Init Data 校验
- `pkg/telegramauth` 解析/验证 init data，覆盖签名、过期、User JSON 单测。

### 步骤 3：Token 加解密
- `pkg/crypto/aes` 提供 AES-256-GCM 加/解密与 key 生成，覆盖率 82.6%。

### 步骤 4：Notion OAuth Client
- `pkg/notion/oauth` 生成授权 URL、交换 token，配置映射到 `internal/config`。

### 步骤 5：Auth Middleware
- `internal/server/http/middleware/auth.go` 校验 `X-Telegram-Init-Data`、自动建用户、注入上下文。

### 步骤 6：Auth Handlers
- `/auth/status`、`/auth/notion/url`、`/auth/notion/callback` 实现，持久化加密后的 token 并更新 `notion_connected`。
- 集成 Gin 中间件，`cmd/api/main.go` 注册路由。

### 步骤 7：数据库迁移集成
- 引入 `migrations.Run()`（embed + golang-migrate），`cmd/api` 启动时自动执行 SQL。

### 步骤 8：集成测试
- `internal/server/http/handlers/auth/integration_test.go` 使用 testcontainers 验证完整 Onboarding 流程。
- `CGO_ENABLED=0 go test ./...`（server/）保持通过。

## 前端 Onboarding 配套
- 重建 `web/`（Vue + Pinia + Router + DaisyUI）。
- `src/pages/OnboardingPage.vue` 对接 `/auth/status`、`/auth/notion/url`，支持 `start_param`、游客路径、`window.tgTodo.setMockInitData()`。
- `src/store/auth.ts`、`src/utils/initData.ts`、`src/api/client.ts` 统一管理 init data、headers 与请求。
- `npm run build` 通过，确保与后端 `/auth/*` 契约一致。

## 环境变量基线
- `TELEGRAM_BOT_TOKEN`：来自 BotFather。
- `NOTION_CLIENT_ID`、`NOTION_CLIENT_SECRET`、`NOTION_REDIRECT_URI`：Notion OAuth 配置。
- `ENCRYPTION_KEY`：32 字节密钥（base64）。
- `DATABASE_DSN`、`REDIS_ADDR` 等基础配置写入 `.env.test`/`config/default.yml`。

## 依赖清单
- `gorm.io/gorm` + `gorm.io/driver/postgres`
- `github.com/golang-migrate/migrate/v4`
- `github.com/dstotijn/go-notion`
- `github.com/testcontainers/testcontainers-go`
- `github.com/redis/go-redis/v9`

## 文档与插入事项
- `docs/prd/prd.md`、`docs/frontend/frontend_requirements.md`、`docs/server/db-schema.md`、`docs/server/api-by-page.md` 已更新，强调「未绑定 Notion 仍可创建任务」。
- DevOps 待办（`docs/deploy/`、`web/Dockerfile` 缺失文件）保留在未来工作项中。
