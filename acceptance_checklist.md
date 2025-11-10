# Telegram To-Do Mini App 验收清单

## 0. 前置准备
- [ ] 在本地或 CI 保存 `.env`（或 `.env.local`）并确认包含 `POSTGRES_URL`、`TELEGRAM_BOT_TOKEN`、`VITE_API_BASE_URL`、`VERCEL_TOKEN` 等变量。
- [ ] 记录当前部署版本号（Git commit）并与 Vercel / Docker Compose 容器版本一致。
- [ ] 确认 Postgres、Go API、Bot、Mini App 均已在目标环境启动，无崩溃日志。

## 1. Telegram Bot 与 Mini App 入口
- [ ] 在 BotFather 执行 `/setdomain`，域名为 `checklist-one-sable.vercel.app`。
- [ ] 通过 BotFather 的 “Menu Button → Configure Web App” 绑定 Mini App URL，聊天菜单显示“打开待办”或自定义名称。
- [ ] 调用 `getChatMenuButton`（可用 [Bot API](https://api.telegram.org/bot<TOKEN>/getChatMenuButton)）确认默认返回中包含 `web_app.url`.
- [ ] 在手机 Telegram 中打开机器人私聊，点击左下角菜单验证 Mini App 能正常唤起，首次加载无白屏。
- [ ] 进入待办详情页后左上角按钮变为“返回”，点击可退回列表；再回到列表，按钮复为“关闭”。

## 2. Mini App UI 与交互
- [ ] 进入列表页默认展示 Pending Tab，切换 Completed Tab 不触发报错。
- [ ] Pending/Completed Tab 内容、空态文案与 PRD 描述一致。
- [ ] 点击任务卡片进入详情页，标题、指派人、创建人信息显示正确；返回列表后卡片保持最新状态。
- [ ] 切换任务完成状态时触发 uiverse.io 动画，状态在两个 Tab 之间同步。
- [ ] 可编辑任务标题（仅创建人），失焦或点击主按钮后保存并有成功反馈。
- [ ] “查看 Telegram 原始消息”按钮有效，能在 Telegram 打开对应消息（若有 deep link）。
- [ ] Telegram MainButton 仅在有编辑权限时出现，文字为“保存修改”；无权限时 MainButton/BackButton 均隐藏。

## 3. API 与数据校验
- [ ] 使用浏览器 DevTools 或代理确认前端每个 API 请求都带 `X-Telegram-Init-Data` 头。
- [ ] 后端对 `initData` 进行校验，伪造数据会返回 401/403（可用本地 curl 测试）。
- [ ] 执行 `go test ./...`（位于 `server/`）全部通过。
- [ ] 通过 API 创建/编辑/删除任务后，数据库表 (`tasks`, `task_assignees`, `users`) 数据与 PRD 权限模型一致。
- [ ] 删除任务需二次确认，执行后重新加载列表任务消失。

## 4. Bot 行为
- [ ] 使用 `/ping` 命令返回 “pong ok”，证明长轮询正常。
- [ ] `/tasks` 命令返回当前账号相关待办列表，内容与 Mini App 一致。
- [ ] 在群组触发 Bot（转发/引用/@bot）可创建任务，任务会立即出现在 Mini App。
- [ ] 任务完成后 Bot 会向创建人和所有指派人推送完成通知。

## 5. 部署与基础设施
- [ ] `cd web && npm run build` 在 CI 或本地通过且产物成功上传至 Vercel。
- [ ] `cd web && npm run lint && npm run test -- --coverage` 通过并达到 ≥80% 覆盖率。
- [ ] `cd server && go test ./...`、`make docker-build`（若配置）均成功。
- [ ] `cd infra && docker compose up --build` 可一键启动 db/api/bot/web，容器日志无错误。
- [ ] Vercel 生产访问性能正常（首屏加载 < 2s），Telegram WebView 中无 Mixed Content。

## 6. 安全与配置
- [ ] `.env.example` 包含所有必需变量，实际敏感信息未进入仓库。
- [ ] Telegram bot token、数据库密码仅在部署环境变量中配置，未硬编码。
- [ ] API 拒绝未验证 `initData` 的请求，日志中有失败记录。
- [ ] 若部署在公网服务器，确认 HTTPS 证书有效且自动续期。

## 7. 文档与交付
- [ ] `README.md` / `AGENTS.md` 已更新包含最新启动、测试、部署说明以及 Telegram 配置步骤。
- [ ] 将本验收清单链接或文件路径提供给团队，确保他们可复现上述步骤。
- [ ] 在内部任务/PR 中附上执行过的命令、测试截图、Telegram 菜单截图等佐证资料。

> 逐条勾选完毕后，即可判定当前部署满足 PRD 与 AGENTS.md 所列的最低交付要求。
