# 未完成功能清单

1. **鉴权与安全**  
   - [x] Bot 与 API 统一校验 Telegram `initData`，防止伪造用户或越权调用。  
   - [x] 明确 Bot↔API 的鉴权机制（例如使用签名或 service token），当前仍是裸调用。

2. **任务生命周期通知**  
   - [x] 当 Mini App 标记完成/恢复或删除任务时，Bot 需向创建人及其他指派人推送通知。  
   - [x] 列出通知内容模板，区分个人/群组场景并写入 README。

3. **前端真实联调**  
   - [x] Mini App 改为调用真实 API（`GET /tasks`, `POST /tasks`, `PATCH /tasks/:id`, `DELETE /tasks/:id`），移除 mock 回退。  
   - [x] 补充前端 store / services 的错误提示与 MainButton 状态同步。  
   - [x] 提供至少一个 Vitest + 一个 Cypress 用例覆盖真实 API 交互（可使用 MSW/Mock Service Worker）。

4. **测试与稳定性**  
   - [x] 为 `task.Service`、`bot` 解析逻辑与 Telegram 通知编写单元测试。  
   - [x] 为 `task.Repository` 增加 table-driven Go 测试（覆盖扫描/归一化逻辑）。  
   - [x] 通过自定义 HTTP 客户端模拟 Telegram API，覆盖 `getUpdates` / `getChatMember` / `sendMessage` 关键分支。  
   - [x] CI 中加入 `go test`, `npm run lint`, `npm run test`。

5. **文档与运维**  
   - [x] 更新 `README.md`/`AGENTS.md`，说明任务创建链路、通知策略、环境变量（新增鉴权/通知相关配置）。  
   - [x] 在 `infra/` 的 compose 文件中串起 API + Bot + Web + Postgres，实现一键本地联调。
