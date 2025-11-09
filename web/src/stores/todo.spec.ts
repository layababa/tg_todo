import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import type { Task } from '@/types/todo'

import { useTodoStore } from './todo'

vi.mock('@/services/todo', () => {
  const baseTask: Task = {
    id: '1',
    title: '测试任务',
    status: 'pending',
    createdAt: new Date().toISOString(),
    createdBy: { id: '1', displayName: '创建人' },
    assignees: [],
    permissions: { canEdit: true, canComplete: true, canDelete: true }
  }

  return {
    fetchTasks: vi.fn().mockResolvedValue([baseTask]),
    fetchTaskDetail: vi.fn().mockResolvedValue(baseTask),
    setTaskStatus: vi.fn().mockImplementation(async (id, status) => ({ ...baseTask, id, status })),
    updateTask: vi.fn().mockImplementation(async (id, payload) => ({ ...baseTask, id, ...payload })),
    deleteTask: vi.fn().mockResolvedValue(undefined)
  }
})

describe('useTodoStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('fetches tasks', async () => {
    const store = useTodoStore()
    await store.fetchAll()
    expect(store.items).toHaveLength(1)
  })

  it('switches tabs', () => {
    const store = useTodoStore()
    store.setActiveTab('completed')
    expect(store.activeTab).toBe('completed')
  })

  it('updates task status locally', async () => {
    const store = useTodoStore()
    await store.fetchAll()
    await store.toggleStatus('1', 'completed')
    expect(store.completedTasks).toHaveLength(1)
  })
})
