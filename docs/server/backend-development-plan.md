# 服务端开发计划（Telegram To-Do Mini App）
依据 `prd.md`、`docs/server/api-by-page.md`、`docs/server/db-schema.md` 与 `prototype/` 页面交互，制定端到端服务端开发顺序、验收标准与依赖。

## 技术栈与架构要求（MVP 版本）

### 核心技术栈
- **编程语言**: Go 1.21+
  - 利用 goroutine 处理并发（Webhook、通知、同步任务）
  - 标准项目结构（cmd/internal/pkg）
  - 使用 context 管理请求生命周期与超时
  
- **数据库**:
  - **PostgreSQL 15+**: 主存储，事务保证
    - JSONB 字段存储 Telegram init_data、上下文快照、Notion metadata
    - 唯一索引 + 外键约束保证数据一致性
  - **Redis 7+**: 缓存与幂等控制
    - 任务列表/库列表缓存（TTL 控制）
    - Telegram update 去重（幂等键）
    - 通知去重与速率限制计数器
    - Notion 同步分布式锁（防并发）

- **第三方 SDK**:
  - **Telegram Bot**: [mymmrac/telego](https://github.com/mymmrac/telego)
    - 支持 Webhook 和长轮询
    - 类型安全的 API 封装
    - Message Thread 支持（论坛 Topic）
  - **Notion API**: [dstotijn/go-notion](https://github.com/dstotijn/go-notion)
    - OAuth 2.0 流程
    - Database/Page/Comment CRUD
    - 字段类型映射与校验

### 框架与工具

- **Web 框架**: [gin-gonic/gin](https://github.com/gin-gonic/gin)
  - 高性能 HTTP 路由
  - 中间件：CORS、鉴权、日志、Recovery
  - JSON 绑定与验证

- **数据库访问**:
  - **ORM**: [gorm.io/gorm](https://gorm.io) v2
    - PostgreSQL Driver: [gorm.io/driver/postgres](https://gorm.io/driver/postgres)
    - Auto Migration 支持
    - Hook（自动时间戳）
  - **Redis Client**: [redis/go-redis](https://github.com/redis/go-redis) v9
    - 连接池管理
    - Pipeline/Transaction

- **配置管理**: [spf13/viper](https://github.com/spf13/viper)
  - 环境变量 + 配置文件（YAML/JSON）
  - 开发/生产环境隔离

- **日志**: [uber-go/zap](https://github.com/uber-go/zap)
  - 结构化 JSON 日志
  - 日志级别可配置（Debug/Info/Warn/Error）
  - 携带 request-id 追踪

- **验证**: [go-playground/validator](https://github.com/go-playground/validator) v10
  - Struct tag 验证
  - 自定义校验规则

- **API 文档**: [swaggo/swag](https://github.com/swaggo/swag)
  - OpenAPI 3.0 规范自动生成
  - Swagger UI 接口测试
  - 注解驱动（godoc 注释生成文档）
  - 前后端协作 API 契约

### 部署与监控

- **容器化**: Docker + Docker Compose
  - 多阶段构建（builder + runtime Alpine）
  - 环境变量注入配置
  - Docker Compose 编排：
    - Go 服务端容器
    - PostgreSQL 容器（数据持久化）
    - Redis 容器
    - **Nginx 容器**（生产环境）
      - HTTPS 终止（Let's Encrypt）
      - 反向代理 Go 服务
      - 静态资源服务（Mini App 前端）
      - Gzip 压缩、缓存控制
      - 速率限制

- **监控与健康检查**:
  - **健康检查**: `GET /healthz` 返回服务/DB/Redis 状态
  - **基础日志**: 结构化日志记录关键操作（请求、错误、外部调用）
  - **错误追踪**: request-id 贯穿整个调用链

- **安全**:
  - Notion token 加密存储（AES-256-GCM，使用 Go 标准库 `crypto/aes`）
  - Telegram init_data HMAC-SHA256 签名校验
  - 环境变量管理敏感配置（Bot Token、加密密钥、数据库密码）

### 测试策略（TDD）

- **单元测试**: `go test`
  - Mock 框架：[stretchr/testify](https://github.com/stretchr/testify/mock)
  - 重点覆盖：
    - 鉴权逻辑（Telegram 签名校验）
    - 幂等键生成与去重
    - 状态机流转（任务状态、群组绑定状态）
    - 权限判定（Admin/创建人/指派人）
    - 时区处理与时间计算
  - **覆盖率要求**: ≥ 60%（CI 门禁）

- **集成测试**: [testcontainers-go](https://github.com/testcontainers/testcontainers-go)
  - 真实容器环境：PostgreSQL + Redis
  - 测试范围：
    - GORM 迁移验证（表结构、索引、外键）
    - 事务一致性（并发写入、回滚）
    - Redis 幂等键 TTL 与分布式锁
    - Notion/Telegram Mock API 集成
  - 自动清理容器资源

- **E2E 测试**（手动/半自动）:
  - 关键路径：Onboarding → 绑定库 → 创建任务 → 通知 → 状态更新
  - Telegram Bot 真实交互测试
  - Notion 沙箱环境数据验证

- **TDD 流程**:
  1. 红：编写失败测试
  2. 绿：实现最小代码通过测试
  3. 重构：优化代码保持测试通过
  4. 门禁：单元测试 + 集成测试 + 覆盖率检查

### 代码规范

- 使用 `gofmt` / `goimports` 格式化
- 遵循 Go 社区最佳实践
- 公开接口/函数添加 godoc 注释

### 开发环境要求

- Go 1.21+
- Docker 24+ & Docker Compose v2
- PostgreSQL 15+ 客户端（psql，可选）
- Redis 7+ 客户端（redis-cli，可选）
- Git

### CI/CD

- **CI Pipeline**: GitHub Actions / GitLab CI
  - **代码检查**: `golangci-lint`（严格模式）
  - **单元测试**: `go test -v -race -coverprofile=coverage.out`
  - **覆盖率报告**: 低于 60% 失败（codecov/coveralls）
  - **集成测试**: testcontainers（需 Docker-in-Docker）
  - **安全扫描**: `gosec`（可选）
  - **Docker 构建**: 自动构建并推送镜像

- **CD Pipeline**（可选）:
  - 自动部署到预发布环境
  - 健康检查验证 `/healthz`
  - 失败自动回滚

## 总体原则
- 以 Notion Database 为数据源，服务端持久层做缓存/索引与审计，关键写操作同步 Notion。
- 每个功能模块独立验收，验收通过后再进入下一个模块（防止并行引入回滚成本）。
- 所有接口默认鉴权：验证 Telegram init data / 会话 token；写操作需幂等与审计。
- 错误返回统一格式：`{ success:false, error:{code,message}, meta:{} }`。

## 模块开发顺序与验收标准（含 TDD 提示）
遵循“基础设施 → 身份 → Bot 入口 → 业务核心 → 同步/通知 → 运营与可观测”。每模块建议先写测试（TDD），测试通过再进入下一模块。

### 1) 基础设施与骨架
- 内容：项目结构、配置加载、数据库连接池、迁移框架、统一响应/日志/错误中间件。
- TDD 要点：healthz handler、config 加载失败回退、panic recover 中间件的单测。
- Redis 备注：配置连接池、超时、命名空间前缀（环境隔离），预留健康检查。
- 验收：
  - `healthz` 返回 200，含版本/git hash。
  - 首次迁移创建核心表（见 `db-schema.md`）。
  - 日志含 request-id，panic/错误可追踪到调用链。

### 2) 身份与鉴权（Telegram Bootstrap）
- 内容：校验 `X-Telegram-Init-Data` 签名，创建/查找 `users`；dev 模式旁路。
- TDD 要点：签名校验真/伪请求；新用户创建；已存在用户幂等；无 token 返回 401。
- 验收：
  - `GET /auth/status` 返回用户、notion_connected。
  - 非法签名 401，并写审计。
  - 首次访问自动建 `users`（tg_id/tg_username/name/timezone 默认值）。

### 3) Telegram 更新接入与去重
- 内容：Bot Webhook/长轮询接入；存储原始 update 以便回溯；去重/重放保护；基础命令路由（/start, /help）。
- TDD 要点：重复 update_id 幂等、超时重试不重复；my_chat_member 状态变更；命令路由返回预期文案。
- Redis 用法：用 `SETNX update:{update_id}` 做幂等键，TTL 短期（如 10 分钟）；避免 DB 冲突日志噪音。
- 验收：
  - Webhook 回 200 < 500ms；重复 update_id 不产生重复 side-effect（表唯一幂等键）。
  - my_chat_member 事件可识别 bot 被拉入/踢出，写 group 状态变更审计。
  - /help 返回说明；/start 返回 Mini App 链接。

### 4) Telegram 任务创建与上下文抓取
- 覆盖 PRD 场景：S1/S2（Reply + @Bot 创建/多人指派）、S7（Forward 创建个人收件箱）、G1（群绑定引导）。
- 内容：解析 reply/forward/@Bot 文本，抽取标题/指派人；抓取触发消息前 10 条文本上下文；生成 `chat_jump_url`；创建任务（未绑定库时给出引导）。
- TDD 要点：不同触发源（reply/forward/@）解析正确；上下文 10 条截取与过滤；未绑定库时返回引导；多人 @ 产生多 assignee；生成 jump link。
- Redis 用法：可选缓存“待绑定群组引导态”或上下文临时片段，TTL 短期，避免 DB 写放大。
- 验收：
  - Reply + @Bot 在已绑定群创建任务，写入 tasks/task_context_snapshots；Notion Page 创建成功。
  - 多人 @ 时写多条 task_assignees；未 Start 过 Bot 的用户返回群内提示。
  - Forward 到私聊时写入个人默认库；若未配置默认库则回复选择提示。
  - 创建成功后群内回执含 deep link + jump 按钮；上下文仅包含文本占位。

### 5) Deep Link / Start Param 路由
- 覆盖 PRD S5/S6/S8：通知直达、跳转原消息、单项目专注。
- 内容：start_param 解析 (`task_{id}`, `settings`, `group_{id}`)，/bootstrap 返回路由；生成 deep link 用于通知与 Jump。
- TDD 要点：不同 start_param 路由解析；未授权用户跳转到 onboarding；jump deeplink 拼接。
- 验收：
  - 点击通知按钮直达 `detail.html?id=...`；jump 按钮可回到原消息。
  - 未登录/未授权 Notion 时返回 onboarding 路由提示。

### 6) Notion OAuth 与 Token 管理
- 内容：授权 URL、回调、token 加解密与刷新、workspace 缓存。
- TDD 要点：state 校验、防重放；token 加解密；过期刷新分支；无 token 时返回未授权。
- 验收：
  - `GET /auth/notion/url` 生效；`POST /auth/notion/callback` 存储 token。
  - 过期自动刷新；notion_connected 状态在 /auth/status 正确。

### 7) Notion 库资源管理（列表/校验/初始化）
- 内容：拉取授权库、搜索；必需字段校验与自动初始化（PRD G4）；个性化/工作区标记。
- TDD 要点：库搜索分页；缺字段检测；初始化创建缺失字段；限流/无权限错误分支。
- Redis 用法：可缓存库列表/校验结果，TTL（如 5-15 分钟）；命中失败可降级 DB/Notion。
- 验收：
  - `GET /databases` 分页/搜索返回库列表。
  - `GET /databases/{id}/validate` 返回缺失字段；`POST /groups/{group_id}/db/init` 可创建缺失字段。
  - 限流/无权限返回明确错误码；指数退避重试。

### 8) 群组管理与绑定
- 覆盖 PRD G1/G3/G4：绑定、更换、失效处理。
- 内容：群模型、管理员校验、绑定/解绑/更换库；状态机 Connected/Unbound/Inactive；论坛 Topic 记录 message_thread_id。
- TDD 要点：非管理员无法绑定；更换库后状态变更；被踢出群标记 Inactive；Topic 信息保留。
- 验收：
  - `GET /groups?role=admin` 返回绑定状态、数据库信息、Admin 角色。
  - `POST /groups/{group_id}/bind` 正常绑定；权限不足/被踢出返回业务错误码。
  - `POST /groups/refresh` 更新管理员和状态；数据库丢失标记 Inactive 并告警。

### 9) 任务核心 CRUD（含上下文）
- 内容：任务缓存/索引、Notion Page ID 对应、上下文 10 条、软删；列表分页与筛选。
- TDD 要点：列表过滤视图/数据库；空状态；软删不出现在列表；上下文随任务返回；描述字段透传。
- Redis 用法：可缓存任务列表第一页/任务详情，TTL 短期（数十秒-数分钟），写操作后主动失效；缓存缺失时回源 DB/Notion。
- 验收：
  - `GET /tasks` 支持 view=assigned/created/all、db_id 过滤、分页；空状态正确。
  - `GET /tasks/{id}` 返回标题/状态/指派/截止/跳转链接/context_snapshot/描述。
  - `DELETE /tasks/{id}` 软删并写 `task_events`；Notion 标记为 Archive/状态字段。

### 10) 指派、状态流转与截止时间
- 内容：多指派、状态更新（待办/进行中/已完成/重新打开）、截止日期；权限：Admin/创建人/指派人可改（PRD S3）。
- TDD 要点：权限判定；多指派去重；状态流转幂等；截止时间读写时区正确；过期标识。
- 验收：
  - `PATCH /tasks/{id}` 更新 status/assignee/due/title/description 幂等；Notion 同步成功。
  - 过期高亮数据正确（UTC 存储，前端本地转换）。
  - 未授权用户修改返回 403 + 提示。

### 11) 评论与嵌套回复
- 覆盖 PRD N3、原型 detail HUD 评论。
- 内容：树形评论、来源标记（Telegram/Notion）、@ 提及透传，Notion 评论同步。
- TDD 要点：嵌套结构渲染数据；parent_id 校验；Notion 写入失败入补偿队列；来源标记。
- 验收：
  - `GET /tasks/{id}/comments` 嵌套结构与原型一致；分页 cursor 可选。
  - `POST /tasks/{id}/comments` 支持 parent_id；写入 Notion；失败入补偿队列。

### 12) 通知体系（即时）
- 覆盖 PRD 通知场景：新任务指派、状态变更、重新打开、删除、评论、指派失败提醒。
- 内容：通知模板、操作者去重、不可达标记；deep link 按钮。
- TDD 要点：操作者去重；不可达标记；不同事件类型文案/按钮；重复发送幂等。
- Redis 用法：通知幂等键/去重（如 `notif:{user}:{task}:{event}` TTL 数分钟），避免重复推送；速率限制计数器。
- 验收：
  - 相关人收到，操作者不重复；Block 用户标记不可达。
  - `notifications` 记录送达状态；重试/补偿可查。

### 13) Notion ←→ Telegram 同步循环
- 内容：轮询/监听 Notion 更新（状态/评论/新建），同步到本地并推送 Telegram；缺字段/库丢失告警；限流退避。
- TDD 要点：模拟 Notion 回调/轮询增量；429/403/404 退避；库丢失转 Inactive；重复事件幂等。
- Redis 用法：轮询游标/offset、退避计数、分布式锁（同库/任务同步防并发），TTL 控制；异常时可回退到 DB 状态。
- 验收：
  - 状态/评论在 2 分钟内同步并触发通知；来源标记正确。
  - 429/403/404 有退避与告警；库丢失标记绑定 Inactive。

### 14) 每日摘要 / 运营推送
- 覆盖 PRD P1：每日 9:00 Digest。
- 内容：按用户时区生成今日截止/逾期/待办摘要，含 deep link。
- TDD 要点：不同用户时区的定时触发；禁用开关；摘要分组正确；发送幂等。
- Redis 用法：定时任务去重/锁（防重复跑），TTL 当日；队列积压监控计数。
- 验收：
  - 定时任务按时区发送；可关闭；发送记录写 notifications。
  - 内容正确分组（今日截止/逾期/待办）。

### 15) 个人设置与默认数据库
- 覆盖 PRD U1：默认个人数据库、时区设置、注销。
- 内容：`/me` 读取；`/me/settings` 更新 default_db_id/timezone；注销。
- TDD 要点：default_db_id 更新后 Forward 落库；幂等更新；注销清理会话。
- 验收：
  - `GET /me` 返回默认库、时区、群组数。
  - `PATCH /me/settings` 幂等；更新后 Forward 创建落入默认库。
  - `POST /auth/logout` 清理会话。

### 16) 可观测性与运维
- 内容：metrics（API 延迟/错误率、Notion 调用成功率、队列积压）、trace、结构化日志；告警。
- TDD 要点：metrics 注册与暴露；关键错误触发告警通路；trace 与日志关联 request-id。
- 验收：
  - Prometheus 指标齐全；SLO 警报触发（鉴权失败、Notion 429/403/404、数据库丢失、队列积压）。
  - 日志与 trace 可关联请求/消息。

## 依赖与前置
- 已有：`api-by-page.md`（接口）、`db-schema.md`（表结构）、`prototype/`（交互）、`prd.md`（业务规则）。
- 外部：Telegram Bot API、Notion API（OAuth + Database/Pages/Comments）、对象存储（可选用于日志/备份）。
- 开发前需要：Notion OAuth 凭证、Bot Token、数据库实例、迁移环境。

## 测试与验收策略
- 单元测试（模块内自测，门禁）：鉴权校验、状态机（任务/绑定）、字段校验、幂等键、权限判定、时间/时区处理。
- 集成测试（Mock 外部）：使用 Mock Notion API + Mock Telegram API，覆盖：任务 CRUD、多人指派、上下文抓取、评论嵌套、通知去重、库校验/初始化、绑定/解绑、Deep Link 路由。
- 端到端验收（沙箱）：按模块顺序在沙箱环境验证原型路径（onboarding -> index -> detail -> settings -> groups -> binding），含真实 Telegram Bot 与 Notion 沙箱库。
- 回归门禁：每模块完成后补充对应测试用例并纳入 CI；下一模块必须通过前面所有用例。关键路径（创建→指派→状态→评论→通知→跳转）需有 E2E 脚本。
- 失败补偿测试：模拟 Notion 429/403/404、Telegram block、库缺字段，验证退避/告警/补偿队列行为。

## 进度里程碑（建议）
1. 基础设施 + 鉴权（1-2）
2. Bot 入口 + 任务创建 + Deep Link（3-5）
3. Notion OAuth / 库管理 / 群组绑定（6-8）
4. 任务核心 + 指派 + 状态 + 评论（9-11）
5. 通知体系 + Notion 同步（12-13）
6. Digest + 个人设置（14-15）
7. 可观测性与收尾（16）

## 其他注意事项
- 幂等：Telegram update_id、Notion page_id + operation 作为幂等键；重试前检查唯一索引。
- 软删除：任务/绑定默认软删，防误删（PRD “软删除”）。
- 权限：Admin/创建人/指派人可改状态；无 Notion 权限返回 403 + 文案；未绑定 Notion 只能只读。
- 时区：UTC 存储；按用户时区展示；自然语言时间解析尽量保留时区。
- 限流/退避：Notion/Telegram 429 用指数退避；队列长度告警；最大重试次数后人工介入。
- 映射：Notion User ↔ Telegram User 建映射表；未匹配时在群内提示（PRD N1）。
- Topic 支持：记录 message_thread_id，通知/回复尽量落在原 Thread。
- 数据安全：token 加密；敏感字段遮蔽日志；最小权限（仅必要 scope）。
- 性能：列表分页默认 20/50；首页骨架屏可先返回缓存；可加本地只读缓存但以 Notion 为真源。
