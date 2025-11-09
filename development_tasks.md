# 开发任务拆解与验收标准

## 1. 仓库基础架构与配置
- **任务**：创建 `web/`、`server/`、`infra/` 目录，分别放置 README 占位；补充 `.gitignore`、`.editorconfig`、`.env.example`（含 TELEGRAM_BOT_TOKEN、POSTGRES_URL、VERCEL_TOKEN 等）。
- **测试/验收**：
  - `tree -L 2` 可看到上述目录和文件。
  - `.env.example` 中变量命名与 PRD/AGENTS.md 描述一致，无敏感值。

## 2. 前端开发环境初始化
- **任务**：在 `web/` 下用 Vite + Vue 3 + TS 初始化；集成 Pinia、Vue Router、Tailwind、daisyUI，加载 `../daisyUI.template.css`；配置 ESLint、Prettier、Stylelint、Vitest、Cypress；在 `package.json` 添加 `dev/build/test/lint/e2e` 脚本。
- **测试/验收**：
  - `cd web && npm run lint`、`npm run test`、`npm run e2e:headless` 均成功（Cypress 可暂留占位命令）。
  - `npm run dev` 可访问默认页面，控制台无报错，Tailwind 主题变量生效。

## 3. 前端基础代码结构
- **任务**：按照 AGENTS.md 约定创建 `src/components/`, `src/views/`, `src/stores/`, `src/services/`, `src/assets/animations`, `src/router/`;提供示例组件（TodoList, TodoDetail）、Pinia store skeleton、Axios 服务封装、`MainButton` 控制封装、uiverse.io 动画样式引用。
- **测试/验收**：
  - `npm run type-check`（若启用 `vue-tsc`）通过。
  - `npm run test` 覆盖示例 store/组件，Vitest 报告 ≥80%。

## 4. 待办列表页开发
- **任务**：实现 Pending/Completed Tab、任务卡片（标题、创建人、时间、完成按钮）、空状态、完成按钮触发 store/API；集成 Telegram MainButton 状态同步与 uiverse.io 完成动画。
- **测试/验收**：
  - 提供 `TodoList.spec.ts` 单测覆盖状态切换。
  - `npm run dev` 手动切换 Tab、点击复选框，网络请求被 Mock（或走本地 API）且界面同步。
  - Cypress E2E：模拟 API 响应后验证列表渲染与完成动作。

## 5. 待办详情/编辑页开发
- **任务**：实现标题编辑、完成状态切换、创建/指派信息展示、原始消息 Deep Link、权限控制（创建人 vs 指派人）、删除按钮（含确认）；与 store/API 联动。
- **测试/验收**：
  - `TodoDetail.spec.ts` 覆盖权限分支与保存逻辑。
  - Cypress 用例覆盖编辑保存、删除、Deep Link 按钮可见性。
  - TWA SDK MainButton 设置生效（可在浏览器控制台看到调用日志或 Mock）。

## 6. 后端基础设施
- **任务**：在 `server/` 使用 Go modules 初始化；选择 Gin/Fiber；搭建目录 `cmd/bot/`, `cmd/api/`, `internal/{task,user,telegram}`，`pkg/db`, `migrations/`;配置 `Dockerfile` 与 Go 工具链。
- **测试/验收**：
  - `cd server && go test ./...` 初始通过。
  - `make lint`（若配置）运行 gofmt/golangci-lint。

## 7. 数据库与迁移
- **任务**：编写 PostgreSQL schema（tasks, assignees, users, telegram_messages）；使用 `golang-migrate` 或自定义脚本；在 `infra/docker-compose.yml` 配置 Postgres 与 Adminer。
- **测试/验收**：
  - `docker compose -f infra/docker-compose.yml up db` 成功启动。
  - `server/migrations` 可通过 `make migrate` 执行，无错误。

## 8. REST API 与权限
- **任务**：实现任务 CRUD、指派、状态切换；校验 Telegram `initData`；限制创建人/指派人权限；返回列表和详情所需字段。
- **测试/验收**：
  - Postman/HTTPie 脚本覆盖主要接口，返回 2xx。
  - Go 单测：表驱动测试 service 层逻辑，使用 `httptest` + fake DB。

## 9. Bot 集成与通知
- **任务**：实现 Telegram Bot 监听（转发、引用、@bot 场景），解析被 @ 用户；写入任务；任务完成时调用 Telegram API 向创建人和指派人推送消息。
- **测试/验收**：
  - 提供模拟更新的单元测试，验证解析逻辑。
  - 在沙箱群里手测：触发创建/完成动作后，数据库和通知同步。

## 10. 集成与部署
- **任务**：在 `infra/` 准备 `docker-compose.yaml`（Go API、Bot、Postgres、web 前端静态服务），提供本地启动脚本；准备 Vercel、Docker 部署说明；添加 GitHub Actions/CI（lint+test）。
- **测试/验收**：
  - `docker compose up --build` 可本地启动全套服务，前端可访问并与后端交互。
  - CI 工作流在本地（`act`）或云端通过，包含 lint/test。
  - Vercel 预览部署成功，可访问 Mini App 构建产物。

## 11. 文档与交付
- **任务**：更新 `AGENTS.md`、`README.md`，记录环境变量、启动命令、测试命令；补充前端/后端 API 说明与架构图。
- **测试/验收**：
  - 文档与实现一致，按步骤可复现本地启动流程。
  - 向团队成员验证：按文档操作可在 30 分钟内跑通开发环境。
