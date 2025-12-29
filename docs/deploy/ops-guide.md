# 生产环境运维指南

本文档记录 TG TODO 项目的生产环境操作命令和注意事项。

---

## 1. 连接生产服务器

```bash
ssh tg_todo_prod
```

> SSH Host 别名已配置在本地 `~/.ssh/config`

---

## 2. 服务管理

### 查看服务状态

```bash
cd /root/tg_todo/infra
docker compose -f docker-compose.prod.yml ps
```

### 查看日志

```bash
# 查看 API 日志（最近 100 行）
docker compose -f docker-compose.prod.yml logs --tail=100 api

# 实时跟踪日志
docker compose -f docker-compose.prod.yml logs -f api

# 查看所有服务日志
docker compose -f docker-compose.prod.yml logs --tail=50
```

### 重启服务

```bash
# 重启单个服务
docker compose -f docker-compose.prod.yml restart api

# 重启所有服务
docker compose -f docker-compose.prod.yml restart
```

### 拉取最新镜像并重新部署

```bash
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d
```

---

## 3. 数据库操作

### 进入数据库 CLI

```bash
docker compose -f docker-compose.prod.yml exec db psql -U tg_todo -d tg_todo
```

### 查看所有表

```sql
\dt
```

### 查看数据统计

```sql
SELECT
  (SELECT COUNT(*) FROM users) as users,
  (SELECT COUNT(*) FROM tasks) as tasks,
  (SELECT COUNT(*) FROM groups) as groups;
```

### ⚠️ 清空所有业务数据（危险操作）

```bash
docker compose -f docker-compose.prod.yml exec -T db psql -U tg_todo -d tg_todo -c "
TRUNCATE TABLE task_assignees CASCADE;
TRUNCATE TABLE pending_assignments CASCADE;
TRUNCATE TABLE task_comments CASCADE;
TRUNCATE TABLE task_context_snapshots CASCADE;
TRUNCATE TABLE task_events CASCADE;
TRUNCATE TABLE telegram_updates CASCADE;
TRUNCATE TABLE tasks CASCADE;
TRUNCATE TABLE user_groups CASCADE;
TRUNCATE TABLE groups CASCADE;
TRUNCATE TABLE user_notion_tokens CASCADE;
TRUNCATE TABLE users CASCADE;
"
```

> ⚠️ 此操作不可逆，执行前请确认！

---

## 4. 管理后台

### 访问地址

- **本地测试**: `http://localhost:9033/admin`
- **生产环境**: 需配置 Nginx 反向代理

### 默认管理员

- 用户名: `admin`
- 密码: `admin`

### 初始化 Go-Admin 系统表

```bash
# 在服务器上执行
cd /root/tg_todo/admin
docker exec -i tg_todo_prod_db psql -U tg_todo -d tg_todo < admin.pgsql
docker exec -i tg_todo_prod_db psql -U tg_todo -d tg_todo < menu_init.sql
```

---

## 5. CI/CD

### GitHub Actions 工作流

- 代码推送到 `main` 分支自动触发构建
- 镜像推送到 GHCR (GitHub Container Registry)
- 自动 SSH 部署到生产服务器

### 手动触发部署

1. 推送代码到 `main` 分支
2. 查看 GitHub Actions 状态
3. 等待部署完成（约 5-10 分钟）

---

## 6. 环境变量

生产环境变量文件: `/root/tg_todo/infra/.env.prod`

关键配置项：

- `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`
- `TELEGRAM_BOT_TOKEN`
- `NOTION_CLIENT_ID`, `NOTION_CLIENT_SECRET`
- `ENCRYPTION_KEY`
- `WEB_APP_URL`

---

## 7. 常见问题排查

### API 返回 500 错误

1. 查看日志: `docker compose logs --tail=100 api`
2. 检查数据库连接
3. 检查环境变量配置

### 数据库连接失败

```bash
# 检查数据库容器状态
docker compose ps db

# 检查数据库日志
docker compose logs db
```

### Telegram Webhook 不工作

1. 检查 Bot Token 配置
2. 查看 API 日志中的 webhook 请求
3. 确认服务器 SSL 证书有效
