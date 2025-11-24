# 运维部署指南 (DevOps Deployment Guide)

本文档旨在说明 **Telegram To-Do Mini App** 的后端部署流程与架构，供运维同事配置 CI/CD 参考。

## 1. 项目概况

- **前端 (Frontend)**: 部署在 Vercel (由开发自行管理，无需运维介入)。
- **后端 (Backend)**: Go 语言编写，需部署到 Linux 服务器 (Test & Prod)。
- **开发环境 (Dev)**: 本地 Docker + Cloudflare Tunnel (开发自行管理)。

**运维主要职责**:
1. 配置 GitHub Actions 自动构建后端 Docker 镜像。
2. 配置 CI/CD 流程，将后端服务自动部署到 **测试环境** 和 **生产环境**。
3. 管理服务器上的数据库 (PostgreSQL) 和缓存 (Redis) 服务 (推荐使用 Docker Compose 一并启动)。

---

## 2. 关键文件结构说明

```text
.
├── server/                 # 后端 Go 源码
│   ├── cmd/api/            # API 服务入口 (主要部署对象)
│   ├── migrations/         # 数据库 SQL 迁移文件 (会被编译进二进制，启动时自动执行)
│   ├── Dockerfile          # 后端构建文件 (构建出 api 二进制)
│   └── go.mod              # 依赖定义
├── infra/                  # 基础设施配置
│   └── docker-compose.yml  # 服务编排参考 (含 DB, Redis, API)
└── .github/workflows/      # (待创建) CI/CD 流程文件
```

> **注意**: 目前 `cmd/bot` 模块尚在开发中，当前阶段仅需部署 `cmd/api` 服务。`server/Dockerfile` 目前也仅构建 `api` 服务。

---

## 3. 服务端环境依赖

服务端运行依赖以下组件：
- **PostgreSQL 15+**
- **Redis 7+**

推荐在服务器上使用 `docker-compose` 同时编排 `api`, `postgres`, `redis`，也可使用云厂商托管数据库。

---

## 4. 环境变量配置 (Environment Variables)

以下变量需要在 **GitHub Secrets** 或 **服务器 `.env` 文件** 中配置：

### 基础配置
| 变量名 | 说明 | 示例 |
| :--- | :--- | :--- |
| `SERVER_PORT` | 服务监听端口 | `8080` |
| `GIN_MODE` | Gin 框架模式 | `release` (生产) / `debug` (测试) |

### 数据库 & 缓存
| 变量名 | 说明 | 示例 |
| :--- | :--- | :--- |
| `DATABASE_DSN` | Postgres 连接串 | `postgres://user:pass@db:5432/dbname?sslmode=disable` |
| `REDIS_ADDR` | Redis 地址 | `redis:6379` |
| `REDIS_PASSWORD` | Redis 密码 | (留空或填写密码) |

### 第三方服务 (Secrets)
| 变量名 | 说明 |
| :--- | :--- |
| `TELEGRAM_BOT_TOKEN` | Telegram Bot Token (从 BotFather 获取) |
| `NOTION_CLIENT_ID` | Notion OAuth Client ID |
| `NOTION_CLIENT_SECRET` | Notion OAuth Secret |
| `NOTION_REDIRECT_URI` | Notion OAuth 回调地址 (需与 Notion 后台配置一致) |
| `ENCRYPTION_KEY` | 32字节 AES 加密密钥 (用于加密存储 Token) |

---

## 5. CI/CD 流程建议 (GitHub Actions)

请配置以下两条 Workflow：

### A. 测试环境部署 (Deploy to Test)
- **触发条件**: Push to `main` branch.
- **执行步骤**:
  1. **Build**: 使用 `server/Dockerfile` 构建 Docker 镜像。
  2. **Push**: 推送镜像到 Docker Hub / GHCR (Tag: `latest` 或 `commit-sha`)。
  3. **Deploy**: SSH 连接到测试服务器：
     - 拉取最新镜像。
     - 执行 `docker compose up -d api` 重启服务。
     - (可选) 执行数据库迁移 (程序启动时会自动尝试，但建议保留手动入口)。

### B. 生产环境部署 (Deploy to Prod)
- **触发条件**: Release Tag (e.g., `v1.0.0`).
- **执行步骤**:
  1. **Build**: 构建镜像并打上 Release Tag。
  2. **Push**: 推送镜像。
  3. **Deploy**: SSH 连接到生产服务器：
     - 更新镜像版本。
     - 平滑重启服务。

---

## 6. 部署命令参考

在服务器上，建议维护一个 `docker-compose.yml` (参考 `infra/docker-compose.yml`，但需移除 `web` 和 `bot` 部分，仅保留 `api`, `db`, `redis`)。

**启动/更新服务**:
```bash
# 拉取最新镜像
docker compose pull api

# 重启服务 (后台运行)
docker compose up -d api

# 查看日志
docker compose logs -f api
```

**健康检查**:
部署完成后，可通过访问 `/healthz` 接口验证服务状态：
```bash
curl http://localhost:8080/healthz
# 预期返回: {"status":"ok", "version":"...", "git_hash":"..."}
```
