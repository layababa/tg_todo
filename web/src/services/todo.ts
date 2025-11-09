import type { Task, TaskStatus, UpdateTaskPayload } from '@/types/todo'

import http from './http'

const buildMockTasks = (): Task[] => [
  {
    id: 'mock-1',
    title: '审核群组内被 @ 的待办项',
    description: '引用 + @bot + @成员 生成的任务示例',
    status: 'pending',
    createdAt: new Date().toISOString(),
    createdBy: {
      id: '1000',
      displayName: '产品经理',
      username: 'pm_lead'
    },
    assignees: [
      { id: '2000', displayName: '前端负责人', username: 'fe_lead' },
      { id: '2001', displayName: '后端负责人', username: 'be_lead' }
    ],
    sourceMessageUrl: 'https://t.me/c/12345/67890',
    permissions: {
      canEdit: true,
      canComplete: true,
      canDelete: true
    }
  },
  {
    id: 'mock-2',
    title: '为 Mini App 列表页增加 uiverse.io 完成动画',
    status: 'completed',
    createdAt: new Date(Date.now() - 3600_000).toISOString(),
    createdBy: {
      id: '1001',
      displayName: '设计师'
    },
    assignees: [{ id: '2000', displayName: '前端负责人' }],
    permissions: {
      canEdit: false,
      canComplete: true,
      canDelete: false
    }
  }
]

const fallbackTasks = buildMockTasks()

export const fetchTasks = async (): Promise<Task[]> => {
  try {
    const { data } = await http.get<Task[]>('/tasks')
    return data
  } catch (error) {
    // eslint-disable-next-line no-console
    console.warn('获取任务列表失败，使用 mock 数据', error)
    return fallbackTasks
  }
}

export const fetchTaskDetail = async (taskId: string): Promise<Task> => {
  try {
    const { data } = await http.get<Task>(`/tasks/${taskId}`)
    return data
  } catch {
    const task = fallbackTasks.find(item => item.id === taskId)
    if (!task) throw new Error('任务不存在')
    return task
  }
}

export const updateTask = async (taskId: string, payload: UpdateTaskPayload): Promise<Task> => {
  try {
    const { data } = await http.patch<Task>(`/tasks/${taskId}`, payload)
    return data
  } catch {
    const taskIndex = fallbackTasks.findIndex(item => item.id === taskId)
    if (taskIndex === -1) throw new Error('任务不存在')
    const updated: Task = { ...fallbackTasks[taskIndex], ...payload }
    fallbackTasks[taskIndex] = updated
    return updated
  }
}

export const deleteTask = async (taskId: string): Promise<void> => {
  try {
    await http.delete(`/tasks/${taskId}`)
  } catch {
    const index = fallbackTasks.findIndex(item => item.id === taskId)
    if (index >= 0) {
      fallbackTasks.splice(index, 1)
    }
  }
}

export const setTaskStatus = async (taskId: string, status: TaskStatus) => {
  return updateTask(taskId, { status })
}
