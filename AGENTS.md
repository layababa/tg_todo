# Repository Guidelines

## 项目结构与模块组织
仓库根目录提供三份权威文档：`prd.md` 描述产品目标，`frontend_requirements.md` 定义前端交互细则，`当前工作日志.md` 记录实时进度与人工验证。界面原型集中在 `prototype/`：多页面 HTML（`index.html`、`detail.html`、`binding.html`、`settings.html`、`groups.html`、`onboarding.html`）、共用样式 `styles.css` 与交互脚本 `script.js`。新增素材或 JSON mock 请紧贴对应页面存放，确保设计师可以直接打开单个文件即可预览。

## 构建、测试与开发命令
- `cd prototype && python3 -m http.server 4173`：最轻量的本地静态服务，刷新即可看到改动。
- `open prototype/index.html`：无服务预览，适合快速像素校验。
- `NODE_ENV=development npx serve prototype`：需要 HTTPS 或路由实验时使用，可复用同一命令做演示。  
如引入打包器或构建脚本，请在此补充入口文件和 npm script。

## 代码风格与命名约定
HTML/CSS/JS 一律两空格缩进。JavaScript 使用 camelCase（如 `renderSkeletons`、`taskListEl`），共享配置用 SCREAMING_SNAKE_CASE（`MOCK_TASKS`），CSS 类名保持 kebab-case（`ptr-container`）。主题色、字体等 token 已在 `styles.css` 顶部定义，扩展新风格时先补充变量，避免散落魔法值。提交前运行 `npx prettier --write "prototype/**/*.{html,css,js}"` 统一格式。

## 测试规范
当前尚无自动化测试，功能调整后需人工冒烟：逐页点击，验证下拉刷新、骨架屏、HUD 展开/折叠，在移动 Safari 与桌面 Chrome 各走一次。如发现边缘问题，请在 `当前工作日志.md` 新增条目。未来若编写脚本化测试，请放入 `prototype/tests/`（示例：`pull-to-refresh.e2e.spec.ts`），并通过 `npm test` 暴露给 CI，逐步建立覆盖率指标。

## 提交与 Pull Request 指南
参考现有提交（如 `前端基础页面调整完毕,前端文档填写完毕`）：使用祈使句，≤72 字，必要时可中英双语但保持单行说明范围与影响。PR 描述需包含问题背景、UI 变更截图或录屏、可复现或预览命令以及关联需求/问题编号。同时在 `当前工作日志.md` 勾选 ✅/🔄/⏳，保证信息同步。

## 协作提示
开始或结束任何任务时更新 `当前工作日志.md`，继续沿用 ✅/🔄/⏳ 模块，便于后续 Agent 追踪。提交 UI 变更前对照 `prd.md` 与 `frontend_requirements.md` 的章节编号，在 PR 中标注引用，确保产品、设计和开发三方可快速定位决策背景。
