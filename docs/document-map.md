# 📚 文档地图（Document Map）

> 目的：为 AI IDE 与协作者提供 docs/ 目录的统一索引，避免在任务启动前一次性读取所有 Markdown。按需阅读下方模块即可定位到最相关的内容。

## 快速索引

| 模块                   | 路径                                      | 主要内容                                                | 何时阅读/更新                                                |
| ---------------------- | ----------------------------------------- | ------------------------------------------------------- | ------------------------------------------------------------ |
| 产品需求（PRD）        | `docs/prd/prd.md`                         | 产品定位、痛点、解决策略、角色场景、关键触点            | 新增/调整产品能力、角色体验、跨端流程或验收标准              |
| 验收清单               | `docs/acceptance_checklist.md`            | 全面的用户验收测试案例（Happy Path + Edge Cases）       | 产品交付验收、QA 测试、Demo 演示前                           |
| 前端交互规范（FRD）    | `docs/frontend/frontend_requirements.md`  | 全局设计、页面级交互细节、组件状态、触感及动画          | 改动 UI/UX、页面结构、交互流程或主题变量                     |
| 后端 API（按页面）     | `docs/server/api-by-page.md`              | Mini App 每个页面所需接口、请求/响应样例、鉴权约束      | 新增页面、修改接口契约、调整字段/状态码                      |
| OpenAPI 契约           | `docs/server/openapi.yaml`                | 机器可读的后端 API 规范，含所有 path、schema 与中文备注 | 生成 SDK、编写联调脚本、更新接口字段时优先查看               |
| 服务端开发计划         | `docs/server/backend-development-plan.md` | 技术栈与架构、模块顺序、测试策略、CI/CD 要求            | 变动后端实现策略、引入新依赖、规划里程碑                     |
| 阶段规划地图（长期版） | `docs/project-roadmap.md`                 | 七个阶段的宏观交付目标、依赖与验收摘要                  | 需要了解整体路线或为工作日志制定计划时（仅在需求变化时更新） |
| 后端阶段成果档案       | `docs/server/backend-progress-*.md`       | Module 1-9 已完成交付、验收方式                         | 回顾历史交付、更新工作日志或准备新阶段时                     |
| 数据模型               | `docs/server/db-schema.md`                | PostgreSQL 表结构、字段定义、枚举、索引与关系           | 涉及数据模型、字段、关系或约束的修改                         |
| 部署与运维             | `docs/deploy/`（待补充）                  | 预留部署文档位置（Docker、环境变量、监控等）            | 当需要记录部署脚本、环境说明或发布流程时                     |

## 详细分区

### 1. `docs/prd/prd.md`

- **一、产品概述**：定位、目标人群、关键触点与部署方式。
- **二、核心问题与解决路径**：列举 Telegram/Notion 现状、数据割裂表现与应对策略。
- **三、用户角色与典型场景**：群主、被指派成员、Notion-first 成员、新成员等角色诉求。
- 后续章节（未展示）涵盖业务流程、里程碑与指标，如需扩展新版 PRD，沿用原有章节编号。

### 1.1 `docs/acceptance_checklist.md`

- **定位**: 配合 PRD 使用的执行级验收文档。
- **结构**: 分为 5 个阶段（新手引导、基础功能、深度同步、高级特性、异常处理）。
- **用法**: 每一项都对应具体的 User Story，通过勾选 `[ ]` 确认功能交付质量。

### 2. `docs/frontend/frontend_requirements.md`

- **全局设计规范**：主题、色彩、动画、触感、路由/启动参数、数据格式化。
- **页面需求**：按 `onboarding.html`、`index.html`、`detail.html`、`settings.html`、`binding.html`、`groups.html` 等拆解 UI 结构、元素、交互和状态逻辑。
- **组件/交互引用**：指定骨架屏、下拉刷新、HUD、Modal、Toast 等实现方式。

### 3. `docs/server/api-by-page.md`

- 依页面列出入口接口，涵盖 Onboarding、Task List、Task Detail、组/数据库绑定、设置页等。
- 对每个 API 提供 Method、路径、典型入参/出参及枚举定义，便于前后端联调。
- 包含通用模型（User/Database/Group/Task/Comment）与鉴权说明。

### 4. `docs/server/backend-development-plan.md`

- 记录后端技术栈（Go + Gin + GORM + Redis）、架构约束、安全策略、测试/CI 流程。
- 逐模块列出开发顺序、验收门槛与 TDD 提示，适合作为迭代排期和 code review 参照。

### 5. `docs/server/db-schema.md`

- 定义 PostgreSQL 表结构（users、tasks、comments 等 13 张表）、字段类型与说明。
- 提供关系概览、索引/约束建议、枚举列表与典型查询，帮助同步数据层变更。

### 6. `docs/deploy/`

- 当前为空，用于存放部署/运维相关文档（如 Docker Compose、环境变量模板、监控告警流程）。
- 当新增部署策略或 DevOps 手册时，请在该目录创建相应 `*.md` 并在上表补充说明。

### 7. `docs/server/backend-progress-2025-12-05.md`

- 记录 2025-12-05 前已完成的模块成果：基础设施、身份/Onboarding API、前端联调、环境变量与依赖清单。
- 当需要回顾既有交付、同步验收材料或精简《当前工作日志》历史信息时查阅/扩展。

### 8. `docs/project-roadmap.md`

- 长期规划地图，按照 7 个阶段（含模块 1-16）记录宏观目标、功能与验收要点。
- 更新频率低，仅在产品/战略方向调整时维护；日常工作请参考《当前工作日志.md》并引用本文件。

---

### 9. `docs/server/openapi.yaml`

- 提供全部 HTTP API（含 `/healthz`、`/auth/*`、任务/群组/设置页、更包括 `/webhook/telegram`）的 OpenAPI 3.1 规范，可直接导入 Postman/Insomnia/Stoplight。
- 每个接口都定义了详细的入参、出参 schema 与中文描述，便于追踪字段含义和错误码。
- 当新增/调整接口字段时，务必同步更新该规范，并在需要时生成 API 客户端或测试集合。

**使用建议**：任务启动前优先阅读本地图表，按需打开对应 Markdown；提交文档变更时，先更新相关文件，再在此地图新增或调整条目，确保知识结构对齐。
