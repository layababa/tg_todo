# 数据库数据结构说明

依据 `prd.md` 与 `api-by-page.md`，定义后端核心关系型数据模型。默认使用 UTC 时间，字段类型以 PostgreSQL 为例。

## 命名与通用字段
- 所有表默认字段：`id (uuid)`、`created_at timestamptz`、`updated_at timestamptz`、`deleted_at timestamptz null`（软删除）。
- 布尔字段默认 `false`，枚举推荐使用 PostgreSQL `ENUM`。
- 外键统一加索引。

## 表清单与字段

### 1) users
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | uuid | 主键 |
| tg_id | bigint | Telegram 用户 ID（不可变） |
| tg_username | text | Telegram Username（可变） |
| name | text | 展示名 |
| photo_url | text | Telegram 头像 URL |
| avatar | text | (Deprecated) 兼容旧设计，建议使用 photo_url |
| timezone | text | 用户时区（如 `UTC+8`） |
| notion_connected | boolean | 是否已绑定 Notion |

### 2) user_notion_tokens
存储用户绑定的 Notion OAuth 信息（加密）。
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | uuid | 主键 |
| user_id | uuid FK -> users.id | 对应用户 |
| access_token_enc | text | 加密后的 Notion Access Token |
| refresh_token_enc | text | 加密后的 Notion Refresh Token（如有） |
| expires_at | timestamptz | 过期时间 |
| workspace_id | text | Notion Workspace ID |
| workspace_name | text | Notion Workspace 名称 |

### 3) databases
Notion Database 信息缓存。
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | text | Database ID（Notion 原生 ID） |
| name | text | 名称 |
| workspace | text | 工作区名称 |
| icon | text | 图标标识 |
| is_personal | boolean | 是否个人库 |
| last_schema_checked_at | timestamptz | 最近一次字段检查时间 |

### 4) groups
Telegram 群组信息。
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | uuid | 主键 |
| tg_chat_id | bigint | Telegram Chat ID（含 Forum/Topic 时亦记录 thread_id） |
| title | text | 群名称 |
| status | enum('Connected','Unbound','Inactive') | 绑定状态（被踢、失效时为 Inactive） |

### 5) group_admins
群管理员列表，用于校验是否有权限绑定数据库。
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| group_id | uuid FK -> groups.id | 群 |
| user_id | uuid FK -> users.id | 管理员 |
| role | enum('Admin','Owner') | 角色 |

### 6) group_database_bindings
群聊 ↔ Notion Database 映射。
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | uuid | 主键 |
| group_id | uuid FK -> groups.id | 群 |
| database_id | text FK -> databases.id | 数据库 |
| status | enum('Connected','Inactive') | 连接状态 |
| bound_by | uuid FK -> users.id | 操作人 |
| bound_at | timestamptz | 绑定时间 |

### 7) tasks
任务主表，Notion 为真源，表中存缓存/本地状态。
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | uuid | 主键（本地） |
| notion_page_id | text | Notion Page ID |
| title | text | 标题 |
| status | enum('To Do','In Progress','Done') | 状态 |
| group_id | uuid FK -> groups.id | 来源群（可空，用于个人默认库） |
| database_id | text FK -> databases.id | 归属数据库 |
| topic | text | Topic/标签（可空） |
| due_at | timestamptz | 截止时间（可空） |
| creator_id | uuid FK -> users.id | 创建人 |
| chat_jump_url | text | Telegram 消息跳转链接 |
| notion_url | text | Notion 页面 URL |
| archived | boolean | 是否已归档/软删 |

### 8) task_assignees
支持多指派。
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| task_id | uuid FK -> tasks.id | 任务 |
| user_id | uuid FK -> users.id | 被指派人 |
| assigned_by | uuid FK -> users.id | 指派人 |
| assigned_at | timestamptz | 时间 |

### 9) task_descriptions
存储富文本块（可选，亦可直接透传 Notion）。
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| task_id | uuid FK -> tasks.id | 任务 |
| blocks | jsonb | Notion Rich Text Block 数组 |
| source | enum('Notion','Telegram') | 来源 |
| version | int | 版本号，用于幂等/合并 |

### 10) task_context_snapshots
任务上下文快照（创建时截取 10 条消息）。
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | uuid | 主键 |
| task_id | uuid FK -> tasks.id | 任务 |
| role | enum('me','other','system') | 角色 |
| author | text | 发送者名称（system 时可空） |
| text | text | 文本内容（过滤掉多媒体，仅保留占位） |
| tg_message_id | bigint | 原消息 ID（可空） |
| created_at | timestamptz | 消息时间 |

### 11) comments
任务评论（嵌套用 parent_id 自引用）。
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | uuid | 主键 |
| task_id | uuid FK -> tasks.id | 任务 |
| parent_id | uuid FK -> comments.id null | 父评论 |
| author_id | uuid FK -> users.id | 评论人 |
| text | text | 内容 |
| source | enum('Telegram','Notion') | 来源 |
| created_at | timestamptz | 时间 |

### 12) notifications
通知推送记录，用于去重与补偿。
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | uuid | 主键 |
| user_id | uuid FK -> users.id | 接收人 |
| task_id | uuid FK -> tasks.id null | 关联任务 |
| type | enum('Assign','StatusChanged','Comment','Deleted','Digest','Mention') | 通知类型 |
| payload | jsonb | 文案/按钮 deeplink 等 |
| delivered | boolean | 是否成功送达 |
| delivered_at | timestamptz | 送达时间 |

### 13) task_events (可选审计)
记录状态/指派/截止日期等变更。
| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | uuid | 主键 |
| task_id | uuid FK -> tasks.id | 任务 |
| actor_id | uuid FK -> users.id | 操作者 |
| event | enum('Create','Assign','Status','Due','Delete','Restore','Comment') | 事件类型 |
| before | jsonb | 变更前 |
| after | jsonb | 变更后 |
| created_at | timestamptz | 时间 |

## 关系概览
- user 1—N user_notion_tokens（通常最新一条有效）。
- group 1—N group_database_bindings；每组当前有效绑定可在业务层筛 `status='Connected' AND deleted_at IS NULL`。
- group N—N users (通过 group_admins) 用于权限校验。
- database 1—N tasks；group 1—N tasks（个人任务 group_id 可空）。
- tasks N—N users (通过 task_assignees)。
- tasks 1—N comments（自引用 parent_id 支持嵌套）。
- tasks 1—N task_context_snapshots。
- tasks 1—N task_events（审计）。
- users 1—N notifications。

## 索引与约束建议
- 唯一：`users.tg_id`；`group_admins (group_id,user_id)`；`task_assignees (task_id,user_id)`。
- 组合索引：`tasks(database_id,status,due_at)`、`comments(task_id,parent_id,created_at)`、`notifications(user_id,delivered,type)`.
- 外键全部 ON DELETE CASCADE（除审计/通知可保留）。

## 枚举汇总
- group.status: `Connected | Unbound | Inactive`
- tasks.status: `To Do | In Progress | Done`
- notifications.type: `Assign | StatusChanged | Comment | Deleted | Digest | Mention`
- task_context_snapshots.role: `me | other | system`
- description.source / comments.source: `Telegram | Notion`
- group_admins.role: `Admin | Owner`

## 典型查询示例
- 首页任务列表：`SELECT * FROM tasks t JOIN task_assignees a ON t.id=a.task_id WHERE a.user_id=:me AND t.deleted_at IS NULL ORDER BY t.status, t.due_at LIMIT :limit OFFSET :offset;`
- 任务详情（含上下文与评论）：用 task_id 分别查 `tasks`, `task_context_snapshots`, `comments`（按 parent_id 分组）。
- 群组列表（Admin）：`SELECT g.*, b.database_id, b.status FROM groups g JOIN group_admins ga ON ga.group_id=g.id AND ga.user_id=:me LEFT JOIN group_database_bindings b ON b.group_id=g.id AND b.deleted_at IS NULL;`
