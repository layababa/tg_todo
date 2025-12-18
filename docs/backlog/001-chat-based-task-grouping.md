# 需求 001：按会话分组展示任务列表

> **状态**: 📋 待开发 (Backlog)  
> **优先级**: P1  
> **提出时间**: 2025-12-19  
> **依赖**: 当前迭代验收完成后开发

---

## 1. 需求背景

当前首页任务列表仅有一个固定分组标题「待办事项」，所有任务按时间线性排列。当用户同时参与多个群聊的任务时，难以快速定位「某个特定群聊中指派给我的任务」。

## 2. 需求目标

在保留现有 Tab 结构（指派给我 / 我创建的 / 全部任务）的前提下，**按来源会话对任务进行二级分组展示**，让用户一眼看清每个会话/群聊中的任务分布。

## 3. 详细设计

### 3.1 信息架构变更

```
┌─────────────────────────────────────────────────────────┐
│  Header (不变)                                          │
├─────────────────────────────────────────────────────────┤
│  Tab: [指派给我] | [我创建的] |[已完成][已废弃]            │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌─ 💬 项目Alpha群 ──────────────────────────── (3) ──┐ │
│  │  • 任务A                                           │ │
│  │  • 任务B                                           │ │
│  │  • 任务C                                           │ │
│  └────────────────────────────────────────────────────┘ │
│                                                         │
│  ┌─ 💬 营销团队 ─────────────────────────────── (2) ──┐ │
│  │  • 任务D                                           │ │
│  │  • 任务E                                           │ │
│  └────────────────────────────────────────────────────┘ │
│                                                         │
│  ┌─ 📥 个人收件箱 ───────────────────────────── (1) ──┐ │
│  │  • 任务F (转发创建)                                │ │
│  └────────────────────────────────────────────────────┘ │
│                                                         │
│   
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 3.2 分组规则

| 场景 | 分组标题 | 图标 |
|------|----------|------|
| 群聊中创建的任务 | Telegram 群聊标题（`groups.title`） | 💬 |
| 私聊 Bot 创建 / 转发创建 | 「个人收件箱」 | 📥 |

### 3.3 排序规则

1. **分组排序**：按该分组内最新任务的创建时间倒序（活跃群聊在前）
2. **组内任务排序**：按任务创建时间倒序
3. **特殊处理**：「个人收件箱」始终置底（或可配置）

### 3.4 已完成任务处理

- 已完成任务**不按会话分组**，保持当前折叠列表形式
- 点击展开后，线性展示所有已完成任务（或后续迭代增加分组）

---

## 4. 数据结构变更

### 4.1 后端 API 变更

**方案 A（推荐）：扩展 TaskDetail DTO**

```go
// service/task/service.go
type TaskDetail struct {
    Task       *repository.Task `json:"task"`
    GroupTitle string           `json:"group_title,omitempty"` // 新增：群聊标题
}
```

**或方案 B：扩展 Task 返回结构**

```go
// 在 ListByUser 查询时 JOIN groups 表
SELECT tasks.*, groups.title as group_title 
FROM tasks 
LEFT JOIN groups ON tasks.group_id = groups.id
WHERE ...
```

### 4.2 前端类型变更

```typescript
// web/src/types/task.ts
export interface Task {
  ID: string;
  Title: string;
  Status: string;
  SyncStatus: "Synced" | "Pending" | "Failed";
  DatabaseID?: string;
  GroupID?: string;      // 已存在，确保返回
  NotionURL?: string;
  CreatedAt: string;
  // ...
}

export interface TaskDetail {
  task: Task;
  group_title?: string;  // 新增
}
```

---

## 5. 前端实现要点

### 5.1 分组逻辑

```typescript
// HomePage.vue - computed
const groupedTasks = computed(() => {
  const groups = new Map<string, { title: string; tasks: TaskDetail[] }>()
  
  for (const item of activeTasks.value) {
    const groupId = item.task.GroupID || '__inbox__'
    const groupTitle = item.group_title || '个人收件箱'
    
    if (!groups.has(groupId)) {
      groups.set(groupId, { title: groupTitle, tasks: [] })
    }
    groups.get(groupId)!.tasks.push(item)
  }
  
  // 排序：按最新任务时间倒序，个人收件箱置底
  return Array.from(groups.entries())
    .sort((a, b) => {
      if (a[0] === '__inbox__') return 1
      if (b[0] === '__inbox__') return -1
      const aTime = new Date(a[1].tasks[0].task.CreatedAt).getTime()
      const bTime = new Date(b[1].tasks[0].task.CreatedAt).getTime()
      return bTime - aTime
    })
})
```

### 5.2 UI 变更

替换当前固定分组标题：

```vue
<!-- 当前实现 -->
<div class="px-5 py-3 text-[10px] font-mono text-primary uppercase tracking-widest flex items-center justify-between">
    待办事项 <span class="bg-base-content/10 px-1.5 py-0.5 rounded">{{ activeTasks.length }}</span>
</div>

<!-- 新实现 -->
<template v-for="[groupId, group] in groupedTasks" :key="groupId">
  <div class="px-5 py-3 text-[10px] font-mono text-primary uppercase tracking-widest flex items-center justify-between">
    <span class="flex items-center gap-1.5">
      <i :class="groupId === '__inbox__' ? 'ri-inbox-line' : 'ri-chat-3-line'"></i>
      {{ group.title }}
    </span>
    <span class="bg-base-content/10 px-1.5 py-0.5 rounded">{{ group.tasks.length }}</span>
  </div>
  
  <div v-for="{ task } in group.tasks" :key="task.ID" @click="goToDetail(task.ID)" class="task-card">
    <!-- 任务卡片内容不变 -->
  </div>
</template>
```

---

## 6. 验收标准

### 6.1 正常流程

- [ ] 「指派给我」Tab：任务按来源群聊分组展示，每组显示群聊标题和任务数量
- [ ] 「我创建的」Tab：同上逻辑
- [ ] 「全部任务」Tab：同上逻辑
- [ ] 私聊/转发创建的任务：归入「个人收件箱」分组
- [ ] 分组按活跃度排序，最近有新任务的群聊在前
- [ ] 已完成任务保持折叠列表，不按会话分组

### 6.2 边界情况

- [ ] 用户只有私聊任务（无群聊任务）：仅显示「个人收件箱」分组
- [ ] 某个群聊被删除/Bot被踢出：任务仍显示原群聊标题（来自 `groups.title` 缓存）
- [ ] 群聊改名：**显示最新名称**（实时同步，不做历史快照）✅ 已确认
- [ ] 空数据：显示「暂无任务」空状态

---

## 7. 开发预估

| 模块 | 工作量 | 说明 |
|------|--------|------|
| 后端 API 扩展 | 0.5d | 扩展 TaskDetail，JOIN groups 表 |
| 前端列表重构 | 1d | 分组逻辑、UI 调整、样式适配 |
| 联调测试 | 0.5d | 多场景验证 |
| **合计** | **2d** | |

---

## 8. 后续迭代（暂不实现）

- 分组可折叠/展开
- 「已完成」任务也按会话分组
- 分组支持拖拽排序（置顶常用群聊）
- 分组内任务支持拖拽排序

---

## 9. 关联文档

- PRD: `docs/prd/prd.md` - Story S8（Database 筛选器）
- 数据模型: `docs/server/db-schema.md` - tasks.group_id, groups.title
- 前端规范: `docs/frontend/frontend_requirements.md`

