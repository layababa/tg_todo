import { defineStore } from 'pinia'

import type { Task, TaskStatus } from '@/types/todo'

import { deleteTask, fetchTaskDetail, fetchTasks, setTaskStatus, updateTask } from '@/services/todo'

interface TodoState {
  items: Task[]
  selectedTask?: Task
  activeTab: TaskStatus
  isLoading: boolean
  error?: string
}

export const useTodoStore = defineStore('todo', {
  state: (): TodoState => ({
    items: [],
    selectedTask: undefined,
    activeTab: 'pending',
    isLoading: false,
    error: undefined
  }),
  getters: {
    pendingTasks: state => state.items.filter(task => task.status === 'pending'),
    completedTasks: state => state.items.filter(task => task.status === 'completed'),
    activeTasks(state): Task[] {
      return state.activeTab === 'pending' ? this.pendingTasks : this.completedTasks
    }
  },
  actions: {
    async fetchAll() {
      this.isLoading = true
      this.error = undefined
      try {
        this.items = await fetchTasks()
      } catch (error) {
        this.error = error instanceof Error ? error.message : '加载失败'
        throw error
      } finally {
        this.isLoading = false
      }
    },
    async loadTask(taskId: string) {
      this.isLoading = true
      this.error = undefined
      try {
        this.selectedTask = await fetchTaskDetail(taskId)
      } catch (error) {
        this.error = error instanceof Error ? error.message : '加载失败'
        throw error
      } finally {
        this.isLoading = false
      }
    },
    async toggleStatus(taskId: string, status: TaskStatus) {
      this.error = undefined
      try {
        const updated = await setTaskStatus(taskId, status)
        this.items = this.items.map(item => (item.id === updated.id ? updated : item))
        if (this.selectedTask?.id === updated.id) {
          this.selectedTask = updated
        }
      } catch (error) {
        this.error = error instanceof Error ? error.message : '更新状态失败'
        throw error
      }
    },
    async updateTask(taskId: string, payload: Partial<Pick<Task, 'title'>>) {
      this.error = undefined
      try {
        const updated = await updateTask(taskId, payload)
        this.items = this.items.map(item => (item.id === updated.id ? updated : item))
        if (this.selectedTask?.id === updated.id) {
          this.selectedTask = updated
        }
      } catch (error) {
        this.error = error instanceof Error ? error.message : '更新任务失败'
        throw error
      }
    },
    async deleteTask(taskId: string) {
      this.error = undefined
      try {
        await deleteTask(taskId)
        this.items = this.items.filter(item => item.id !== taskId)
        if (this.selectedTask?.id === taskId) {
          this.selectedTask = undefined
        }
      } catch (error) {
        this.error = error instanceof Error ? error.message : '删除任务失败'
        throw error
      }
    },
    setActiveTab(tab: TaskStatus) {
      this.activeTab = tab
    }
  }
})
