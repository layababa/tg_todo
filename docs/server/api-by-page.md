# Telegram To-Do Mini App API（按页面）

依据 `prd.md` 与 `prototype/` 页面行为，列出 Mini App 各页面所需 API。默认响应包裹格式：

```json
{
  "success": true,
  "data": {},
  "error": null,
  "meta": {}
}
```

- 鉴权：前端在请求头附带 `X-Telegram-Init-Data`（Mini App init data）或用户会话 Token。后端验证 Telegram 签名，并映射到 user_id / chat_id。
- 通用枚举：`status` = `To Do` | `In Progress` | `Done`；时间均为 ISO8601 (UTC)，客户端按本地时区渲染。
- 通用模型：
  - **User** `{ id, tg_id, name, photo_url, notion_connected, timezone }`
  - **Database** `{ id, name, workspace, icon, is_personal }`
  - **Group** `{ id, title, status: Connected|Unbound|Inactive, db: Database|null, role: Admin|Member }`
  - **Task** `{ id, title, status, sync_status, group_id, group_title, db_id, topic, due_at, assignee: User, creator: User, notion_url, chat_jump_url, context_snapshot[] }`
  - **Comment** `{ id, author: User, text, created_at, replies: Comment[] }`

---

## 1) Onboarding / 授权页（`onboarding.html`）
用途：判断用户是否已绑定 Notion，提供 OAuth 入口，游客模式直达首页。

- `GET /auth/status`
  - 入参：`start_param`（可选，用于 deeplink 跳转 detail/group/settings）
  - 出参：
    ```json
    {
      "user": {
        "id": "u_1001",
        "name": "John Doe",
        "photo_url": "https://t.me/i/userpic/320/xxx.jpg",
        "notion_connected": false,
        "timezone": "UTC+8"
      },
      "notion_connected": false,
      "pending_sync_count": 5,
      "redirect_hint": null
    }
    ```
- `GET /auth/notion/url`
  - 入参：`redirect_uri`
  - 出参：`{ "url": "https://api.notion.com/oauth?...redirect_uri=..." }`
- `POST /auth/notion/callback`
  - 入参（JSON 或表单）：`code`, `state`
  - 出参：`{ "notion_connected": true }`

---

## 2) 任务列表首页（`index.html`）
需求：Tabs（指派给我/我创建的/全部）、按 Database 筛选、骨架屏加载、操作面板（标记完成/跳转/跟评/详情）。

- `GET /tasks`
  - Query：`view=assigned|created|all`，`db_id`（可选），`limit`（默认 20），`cursor`
  - 出参（示例，含待办与已完成，前端自行分组/折叠）：
    ```json
    {
      "items": [
        {
          "id": 2,
          "title": "修复 iOS 登录 Bug",
          "status": "To Do",
          "group_id": "g_dev",
          "group_title": "Dev Squad",
          "db_id": "db_dev",
          "topic": "Bugs",
          "due_at": "2023-11-20T10:00:00Z",
          "creator": { "id": "u_alice", "name": "alice" },
          "assignee": { "id": "u_me", "name": "me" },
          "assignee": { "id": "u_me", "name": "me" },
          "chat_jump_url": "https://t.me/c/123/456",
          "sync_status": "Pending"
        },
        {
          "id": 3,
          "title": "更新隐私政策",
          "status": "Done",
          "group_id": "g_personal",
          "group_title": "Personal Life",
          "db_id": "db_personal",
          "topic": null,
          "due_at": "2023-11-15T09:00:00Z",
          "creator": { "id": "u_me", "name": "me" },
          "assignee": { "id": "u_me", "name": "me" },
          "chat_jump_url": "https://t.me/c/123/789"
        }
      ],
      "next_cursor": null
    }
    ```
- `PATCH /tasks/{id}/status`
  - 入参：`{ "status": "Done" }`
  - 出参：`{ "id": 2, "status": "Done", "updated_by": "u_me" }`
- `GET /databases`
  - Query：`search`（可选，供筛选弹窗）
  - 出参：
    ```json
    {
      "items": [
        { "id": "db_marketing_q4", "name": "Marketing Q4", "workspace": "工作区 A", "icon": "ri-database-2-fill" },
        { "id": "db_dev_squad", "name": "Dev Squad", "workspace": "工作区 A", "icon": "ri-code-box-line" },
        { "id": "db_personal", "name": "Personal Life", "workspace": "个人", "icon": "ri-user-smile-line" }
      ]
    }
    ```
- `POST /tasks/{id}/jump`（可选：用于校验/生成 Jump Link；也可前端直接用 `chat_jump_url`）
  - 入参：`{ "chat_id": "...", "message_id": "..." }`
  - 出参：`{ "deeplink": "https://t.me/c/123/456" }`

---

## 3) 任务详情页（`detail.html` / `detail copy.html`）
需求：读取任务详情、上下文快照(10条)、评论树、更新状态/指派/截止日期、删除任务、发评论、跳转 Notion/消息。

- `GET /tasks/{id}`
  - Query：`include=context,comments`
  - 出参：
    ```json
    {
      "id": 2,
      "title": "修复 iOS 登录 Bug",
      "status": "In Progress",
      "assignee": { "id": "u_felix", "name": "Felix" },
      "creator": { "id": "u_alice", "name": "Alice" },
      "due_at": "2023-11-20T10:00:00Z",
      "group_id": "g_dev",
      "group_title": "Dev Squad",
      "db_id": "db_dev",
      "notion_url": "https://notion.so/page/xxx",
      "chat_jump_url": "https://t.me/c/123/456",
      "context_snapshot": [
        { "role": "other", "author": "Alice", "text": "iOS 端登录好像又挂了？", "ts": "2023-11-18T02:01:00Z" },
        { "role": "me", "author": "me", "text": "看起来是 Refresh Token 没生效，我建个任务跟进下。", "ts": "2023-11-18T02:02:00Z" },
        { "role": "system", "text": "Bot created task via Reply", "ts": "2023-11-18T02:02:10Z" }
      ],
      "description": [
        { "type": "paragraph", "text": "用户反馈在 iOS 17.2 上无法完成登录流程，一直卡在 Loading 界面。" },
        { "type": "bullet", "text": "检查网络请求日志" }
      ],
      "comments": null
    }
    ```
- `GET /tasks/{id}/comments`
  - Query：`cursor`（可选）
  - 出参（嵌套）：
    ```json
    {
      "items": [
        {
          "id": 1,
          "author": { "id": "u_alice", "name": "Alice", "photo_url": "https://t.me/i/userpic/..." },
          "text": "我看了一下日志，确实是 Token 过期的问题。",
          "created_at": "2023-11-18T03:00:00Z",
          "replies": [
            {
              "id": 2,
              "author": { "id": "u_bob", "name": "Bob" },
              "text": "是 Refresh Token 没生效吗？",
              "created_at": "2023-11-18T03:30:00Z",
              "replies": []
            }
          ]
        },
        {
          "id": 3,
          "author": { "id": "u_charlie", "name": "Charlie" },
          "text": "我已经提交了修复补丁，正在跑 CI。",
          "created_at": "2023-11-18T04:00:00Z",
          "replies": []
        }
      ],
      "next_cursor": null
    }
    ```
- `POST /tasks/{id}/comments`
  - 入参：`{ "text": "修复补丁已发布", "parent_id": null }`
  - 出参：`{ "id": 9, "created_at": "2023-11-18T05:00:00Z" }`
- `PATCH /tasks/{id}`
  - 入参（任意字段可选）：`{ "title", "status", "assignee_id", "due_at", "description" }`
  - 出参：`{ "id": 2, "status": "Done", "assignee_id": "u_felix", "updated_at": "2023-11-18T05:10:00Z" }`
- `DELETE /tasks/{id}`
  - 语义：软删除/归档，遵循 PRD 的“防误删”
  - 出参：`{ "id": 2, "archived": true }`

---

## 4) 设置页（`settings.html`）
需求：展示用户信息、默认收集箱、时区、群组数量、刷新字段缓存、注销。

- `GET /me`
  - 出参：
    ```json
    {
      "id": "u_me",
      "name": "John Doe",
      "photo_url": "https://t.me/i/userpic/320/xxx.jpg",
      "notion_connected": true,
      "timezone": "UTC+8",
      "group_count": 3,
      "default_db": { "id": "db_personal", "name": "Personal Life" }
    }
    ```
- `PATCH /me/settings`
  - 入参：`{ "default_db_id": "db_personal", "timezone": "UTC+8" }`
  - 出参：`{ "updated": true }`
- `POST /databases/{id}/refresh-schema`
  - 作用：刷新字段缓存
  - 出参：`{ "status": "ok", "fields": ["Status", "Assignee", "Date"] }`
- `POST /auth/logout`（可选，供注销按钮）
  - 出参：`{ "success": true }`
- `POST /tasks/sync`
  - 作用：手动触发批量同步（将 Pending 任务写入 Notion）
  - 入参：`{ "target_db_id": "db_personal" }`（可选，若不传则尝试使用任务原 group 绑定或默认库）
  - 出参：`{ "synced_count": 3, "failed_count": 0 }`

---

## 5) 群组管理页（`groups.html`）
需求：列出管理员管理的群组，显示绑定状态/数据库，跳转到绑定页，刷新列表。

- `GET /groups?role=admin`
  - 出参：
    ```json
    {
      "items": [
        {
          "id": "g_marketing",
          "title": "Marketing Team",
          "status": "Connected",
          "db": { "id": "db_marketing_q4", "name": "Marketing Q4" },
          "role": "Admin"
        },
        {
          "id": "g_dev",
          "title": "Dev Squad",
          "status": "Connected",
          "db": { "id": "db_dev_tracker", "name": "Dev Tracker" },
          "role": "Admin"
        },
        {
          "id": "g_projectx",
          "title": "Project X",
          "status": "Unbound",
          "db": null,
          "role": "Admin"
        }
      ]
    }
    ```
- `POST /groups/refresh`（可选：刷新群组列表按钮）
  - 出参：`{ "refreshed_at": "2023-11-18T05:20:00Z" }`

---

## 6) 群组绑定引导页（`binding.html`）
需求：搜索/列出数据库、手动输入 ID 校验、字段检查、确认绑定、关闭时重置 Loading。

- `GET /databases`
  - Query：`search`（供顶部搜索框）
  - 出参：见「任务列表」中的 `GET /databases` 示例
- `GET /databases/{id}/validate`
  - 作用：手动输入 ID 时校验可用性与字段
  - 出参：
    ```json
    {
      "id": "db_manual_123",
      "name": "Product Roadmap",
      "workspace": "工作区 B",
      "required_fields": ["Status", "Assignee", "Date"],
      "missing_fields": []
    }
    ```
- `POST /groups/{group_id}/db/validate`
  - 入参：`{ "db_id": "db_marketing_q4" }`
  - 出参：
    ```json
    {
      "compatible": true,
      "missing_fields": [],
      "analysis": "结构完美匹配"
    }
    ```
- `POST /groups/{group_id}/bind`
  - 入参：`{ "db_id": "db_marketing_q4", "mode": "replace" }`
  - 出参：`{ "group_id": "g_marketing", "db_id": "db_marketing_q4", "status": "Connected" }`
- `POST /groups/{group_id}/db/init`
  - 作用：当存在缺失字段时一键初始化
  - 入参：`{ "db_id": "db_marketing_q4", "fields": ["Status", "Assignee", "Date"] }`
  - 出参：`{ "initialized": true, "created_fields": ["Status", "Assignee"] }`

---

## 7) Deep Link / Start Param 支持
- `GET /bootstrap`
  - Query：`tg_web_app_start_param`
  - 出参：`{ "route": "task_detail", "task_id": "task_123" }` 或 `{ "route": "settings" }` 等，前端据此路由跳转。

---

## 8) 错误格式示例
```json
{
  "success": false,
  "error": { "code": "NOTION_AUTH_REQUIRED", "message": "请先完成 Notion 授权" },
  "meta": {}
}
```

---

## 9) 小结：页面与必需 API 对照
- `onboarding.html`：`GET /auth/status`, `GET /auth/notion/url`, `POST /auth/notion/callback`
- `index.html`：`GET /tasks`, `PATCH /tasks/{id}/status`, `GET /databases`, （可选）`POST /tasks/{id}/jump`
- `detail.html` / `detail copy.html`：`GET /tasks/{id}`, `GET /tasks/{id}/comments`, `POST /tasks/{id}/comments`, `PATCH /tasks/{id}`, `DELETE /tasks/{id}`
- `settings.html`：`GET /me`, `PATCH /me/settings`, `POST /databases/{id}/refresh-schema`, `POST /auth/logout`
- `groups.html`：`GET /groups?role=admin`, `POST /groups/refresh`
- `binding.html`：`GET /databases`, `GET /databases/{id}/validate`, `POST /groups/{group_id}/db/validate`, `POST /groups/{group_id}/bind`, `POST /groups/{group_id}/db/init`
