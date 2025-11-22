# 🎨 Telegram To-Do Mini App 前端需求文档 (FRD)

> **文档依据**：[PRD V2.0](prd.md) | **原型参考**：[Prototype](prototype/)  
> **目标读者**：前端开发工程师、UI/UX 设计师  
> **核心目标**：基于已实现原型,明确 Mini App 的页面结构、交互逻辑与技术实现细节。

---

## 一、 全局设计规范 (Global Specs)

### 1.1 视觉风格
- **主题**：**Cyberpunk Web3** 风格,采用深色科技感设计
- **色彩**：
    - **主色**: 荧光绿 (`#00FF88`) 作为强调色和交互焦点
    - **背景**: 深黑 (`#050505`) + 渐变网格背景 (`.grid-bg`)
    - **面板**: 半透明黑色 (`rgba(10, 10, 10, 0.95)`) + 高斯模糊 (Glassmorphism)
    - **边框**: 暗灰 (`rgba(255, 255, 255, 0.1)`)
- **字体**：
    - **英文/数字**: Space Grotesk (标题), JetBrains Mono (代码/数字)
    - **中文**: Noto Sans SC
- **动画**：
    - 页面入场: `pageEntrance` 淡入 + 轻微位移
    - 扫描线动画: `.scan-line` 循环扫描
    - 按钮交互: `hover` 时荧光绿高亮

### 1.2 交互原则
- **导航**：使用自定义返回按钮,配合 `navigateTo()` 函数实现页面跳转
- **反馈**：
    - **Toast**: 轻量级提示 (调用 `showToast()`)
    - **Skeleton**: 骨架屏 (`.skeleton` + `.shimmer` 动画)
    - **Modal**: 底部弹窗 (`.modal-overlay` + `.modal-content`)
- **触感**：关键操作调用 `Telegram.WebApp.HapticFeedback` (轻度、成功、警告)

### 1.3 数据格式化
- **时间**：
    - 存储：UTC+0
    - 展示：相对时间 ("刚刚", "3小时前") 或 `MM-DD HH:mm`
- **富文本**：
    - 渲染 Notion Rich Text Block
    - 支持 Bold, Italic, Link, Code, List
    - 不支持的 Block 降级为纯文本

### 1.4 路由与启动参数
- **参数解析**：解析 `tgWebAppStartParam`
    - `task_{id}`: 直达任务详情
    - `settings`: 直达设置页
    - `group_{id}`: 直达特定群组任务视图
- **游客模式**：未绑定 Notion 时可只读访问,写操作时弹出授权引导

---

## 二、 页面详细需求 (Page Specifications)

### 2.1 启动/授权页 (`onboarding.html`)

**场景**：首次打开或未绑定 Notion 时展示。

**UI 元素**：
- **Logo**: `.hero-graphic` (大图标 + 脉冲动画)
- **标题**: "系统初始化"
- **副标题**: "建立 Telegram 与 Notion 的神经连接"
- **主按钮**: "连接 NOTION 工作区" (`.primary-btn`)
- **次要链接**: "[ 游客模式访问 ]" (`.secondary-link`)

**交互逻辑**：
1. 点击主按钮 → 跳转 Notion OAuth 授权
2. 点击游客模式 → 直达 `index.html` (只读)

---

### 2.2 任务列表首页 (`index.html`)

**场景**：主视图,展示任务列表。

**顶部导航栏 (`.header`)**：
- **品牌标签**: "系统.V2.0" (`.brand-tag`)
- **用户信息**: 
    - 标题: "你好, 约翰" (`.greeting-title`)
    - 状态: "系统在线" (`.status-indicator` + `.dot` 动画)
- **操作按钮**:
    - 筛选 (`#filterBtn`)
    - 设置 (跳转 `settings.html`)

**视图切换 (`.segmented-control`)**：
- 三个 Tab: "指派给我" | "我创建的" | "全部任务"
- 活动指示器: `.active-border` (动画滑动)

**筛选栏 (`.filter-bar`)**：
- 默认隐藏
- 显示当前筛选的数据库 (`#currentDbName`)
- "清除"按钮 (`#clearFilterBtn`)

**任务列表 (`.task-list`)**：
- **分组**:
    - **活跃任务**: 直接展示
    - **已完成**: 折叠在 `#doneTasksSection` (`.dimmed` 样式)
- **卡片 (`.task-card`)**:
    - 标题 (`.task-title`)
    - 状态标签 (`.task-status`)
    - 元信息 (`.task-meta`): 群组名 + Topic + 时间 + 创建者
    - **点击**: 弹出操作面板 (`#actionModal`),包含:
        - 跟评
        - 跳转到对应消息
        - 标记已完成
        - 查看详情 (跳转 `detail.html`)

**底部操作**：
- **FAB**: "+" 按钮 (`.fab-btn`)
- **数据库选择器 Modal** (`#dbModal`): 
    - 列表展示可用 database (`.db-item`)
    - 点击选择后更新筛选

---

### 2.3 任务详情页 (`detail.html`)

**场景**：查看/编辑任务,支持评论和属性修改。

**导航**：
- 顶部返回按钮 (`#backBtn`)
- 点击返回时**自动保存** (调用 `showToast('任务已自动保存')`)

**内容布局** (自上而下):
1. **任务标题** (`#taskTitle`): 可编辑 `textarea`
2. **上下文快照** (`.context-snapshot-section`):
    - 折叠式显示创建任务时的10条聊天记录
    - 气泡式对话 (`.chat-bubble.me` / `.chat-bubble.other`)
    - 点击 `#toggleContext` 展开/收起
3. **任务描述** (`.editor-content`):
    - 富文本内容区
    - 只读或简单文本编辑
4. **评论区** (`.comments-section`):
    - 标题: "评论 (3)"
    - 列表 (`#commentList`): 支持**嵌套回复**
    - 评论卡片 (`.comment-item` + `.nested-comments`)
    - 操作: "回复" / "点赞"

**底部 HUD (`.bottom-hud`)**:
- **智能折叠/展开**:
    - **默认**: 折叠 (`.hud-collapsed`), 仅显示评论输入框和操作图标
    - **滚动到底部时**: 自动展开,显示属性胶囊
- **Row 1 - 属性胶囊** (`.properties-row`, 可折叠):
    - 状态 (`#statusPill` → 点击弹出自定义状态选择器 `#statusModal`)
    - 指派人 (`#assigneePill`)
    - 截止日期 (`#datePill` → 点击弹出日期选择器 `#dateModal`)
- **Row 2 - 快捷操作** (`.actions-row`, 始终可见):
    - 打开 Notion (`#openNotionBtn`)
    - 跳转消息 (`#jumpBtn`)
    - 联系创建人
    - 更多操作 (`#moreBtn` → 弹出 `#moreActionsModal`,包含"删除任务")
- **Row 3 - 评论输入** (`.input-row`, 始终可见):
    - 输入框 (`#commentInput`)
    - 发送按钮 (`.hud-send-btn`)

**自定义 Modal**:
- **状态选择器** (`#statusModal`): 列表式选择 "待办" / "进行中" / "已完成"
- **日期选择器** (`#dateModal`): 快捷按钮 ("今天" / "明天" / "下周") + 日历控件
- **更多操作** (`#moreActionsModal`): "删除任务" (红色警告样式)
- **点击空白关闭**: 所有 modal 监听 overlay 点击事件

---

### 2.4 设置页 (`settings.html`)

**场景**：个人配置与连接管理。

**用户信息 (`.user-section`)**：
- 头像 (圆形)
- 用户名 ("John Doe")
- 状态: "已连接 Notion" (`.status-indicator`)

**设置项 (`.settings-section`)**：
1. **个人配置**:
    - **默认收集箱**: 选择默认 Database (`.db-tag`)
    - **时区设置**: 显示当前时区 (`UTC+8`)
2. **连接管理**:
    - **管理群组绑定**: 跳转 `groups.html` (显示已连接群组数量)
    - **刷新字段缓存**: 强制刷新 schema
3. **其他**:
    - **注销**: 顶部图标按钮
    - **版本信息**: "VERSION 2.0.1_BETA" (底部灰色小字)

---

### 2.5 群组管理页 (`groups.html`)

**场景**：查看所有管理的群组及其连接状态。

**顶部**:
- 返回按钮 (跳转 `settings.html`)
- 标题: "群组管理"
- 说明: "管理您作为管理员的群组连接"

**群组列表**:
- **已连接群组** (`.group-header`): 显示数量
    - 卡片 (`.task-card`): 群组名 + 绑定的 Database + "Admin" 标签
    - 点击 → 跳转 `binding.html` (带参数 `?group=xxx`)
- **未初始化群组**: 
    - 卡片样式: 灰色边框
    - 状态: "未连接"
    - 提示: "点击配置数据库"

**底部操作**:
- "刷新群组列表" 按钮

---

### 2.6 群组绑定引导页 (`binding.html`)

**场景**：为特定群组绑定 Notion Database。

**顶部**:
- 返回按钮
- 标题: "绑定数据库"
- 子标题: 为群组 "Marketing Team" 选择数据源

**搜索框**: 实时搜索 Database

**手动输入区**:
- 标题: "或者手动输入 ID"
- 输入框 (`#manualDbId`): 支持输入32位 Database ID
- 验证按钮: 点击调用 `checkManualDb()`

**Database 列表** (`#dbList`):
- `.db-item`: 图标 + 数据库名 + 工作区标签
- 点击 → 弹出 **字段检查 Modal** (`#schemaModal`):
    - **Loading 态**: 旋转图标 + "正在分析数据库结构..."
    - **成功态**: 绿色对勾 + "结构完美匹配" + "确认绑定" 按钮

---

## 三、 关键技术实现

### 3.1 HUD 驾驶舱交互

**核心思路**：将操作下沉到屏幕底部,最大化拇指可达区域。

**智能折叠逻辑**：滚动到页面底部时自动展开属性胶囊,向上滚动时收起,节省屏幕空间。

### 3.2 评论嵌套渲染

支持多层嵌套回复,通过递归渲染实现。评论卡片支持"回复"和"点赞"操作(mock)。

### 3.3 自定义 Modal 组件

所有选择器(状态、日期)采用底部弹窗设计,替代原生 Select/Date Picker,保持视觉一致性。支持点击空白区域关闭。

### 3.4 性能优化

- **缓存策略**: `localStorage` 缓存任务列表
- **分页**: 首屏加载 20 条, 触底加载更多
- **Debounce**: 搜索输入防抖 (300ms)

---

## 四、 API 需求摘要

1. **认证**: `GET /auth/status`, `POST /auth/notion`
2. **任务**: `GET/POST/PATCH/DELETE /tasks`
3. **评论**: `POST /tasks/:id/comments`, `POST /comments/:id/reply`
4. **群组**: `GET /groups`, `POST /groups/:id/bind`
5. **Database**: `GET /databases`, `POST /databases/:id/validate`
6. **设置**: `GET/PATCH /user/settings`

---

**文档版本**: V2.1  
**最后更新**: 2025-11-22  
**基于原型**: [Prototype Demo](prototype/)
