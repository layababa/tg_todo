# Repository Guidelines

## AI IDE 启动与协作流程
1. 进入任务后立即通读`.agent/rules/*.md`、`docs/document-map.md` 与 `当前工作日志.md`，掌握最新背景、知识结构与实时进度。
2. 面对多步骤或潜在风险的需求务必使用计划工具列出行动项；仅在极其简单的操作时才可跳过规划。
3. 任何与业务逻辑、页面或后端契约相关的开发，一旦新增或调整 API，必须立即同步更新 `docs/server/api-by-page.md` 与 `docs/server/openapi.yaml`；允许通过脚本/工具自动生成，但需在提交前人工校对，确保联调方可直接依赖。

