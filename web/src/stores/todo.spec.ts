import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import type { Task } from '@/types/todo'

import { useTodoStore } from './todo'

const mocks = vi.hoisted(() => {
  const baseTask: Task = {
    id: '1',
    title: '测试任务',
    status: 'pending',
    createdAt: new Date().toISOString(),
    createdBy: { id: '1', displayName: '创建人' },
    assignees: [],
    permissions: { canEdit: true, canComplete: true, canDelete: true }
  }

  const cloneTask = (): Task => JSON.parse(JSON.stringify(baseTask))

  return {
    cloneTask,
    fetchTasksMock: vi.fn().mockResolvedValue([cloneTask()]),
    fetchTaskDetailMock: vi.fn().mockResolvedValue(cloneTask()),
    setTaskStatusMock: vi.fn().mockImplementation(async (id: string, status: Task['status']) => ({
      ...cloneTask(),
      id,
      status
    })),
    updateTaskMock: vi.fn().mockImplementation(async (id: string, payload: Partial<Task>) => ({
      ...cloneTask(),
      id,
      ...payload
    })),
    deleteTaskMock: vi.fn().mockResolvedValue(undefined)
  }
})

vi.mock('@/services/todo', () => ({
  fetchTasks: mocks.fetchTasksMock,
  fetchTaskDetail: mocks.fetchTaskDetailMock,
  setTaskStatus: mocks.setTaskStatusMock,
  updateTask: mocks.updateTaskMock,
  deleteTask: mocks.deleteTaskMock
}))

describe('useTodoStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    mocks.fetchTasksMock.mockResolvedValue([mocks.cloneTask()])
    mocks.fetchTaskDetailMock.mockResolvedValue(mocks.cloneTask())
  })

  it('fetches tasks', async () => {
    const store = useTodoStore()
    await store.fetchAll()
    expect(store.items).toHaveLength(1)
  })

  it('handles fetch error gracefully', async () => {
    const store = useTodoStore()
    mocks.fetchTasksMock.mockRejectedValueOnce(new Error('network'))
    await expect(store.fetchAll()).rejects.toThrow('network')
    expect(store.error).toBe('network')
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

  it('stores error when toggle fails', async () => {
    const store = useTodoStore()
    await store.fetchAll()
    mocks.setTaskStatusMock.mockRejectedValueOnce(new Error('fail'))
    await expect(store.toggleStatus('1', 'completed')).rejects.toThrow('fail')
    expect(store.error).toBe('fail')
  })
})
