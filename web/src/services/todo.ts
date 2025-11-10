import axios from 'axios'

import type { Task, TaskStatus, UpdateTaskPayload } from '@/types/todo'

import http from './http'

const toFriendlyError = (error: unknown, fallback: string) => {
  if (axios.isAxiosError(error)) {
    const serverMessage = (error.response?.data as { error?: string } | undefined)?.error
    return new Error(serverMessage || error.message || fallback)
  }
  if (error instanceof Error) {
    return error
  }
  return new Error(fallback)
}

export const fetchTasks = async (): Promise<Task[]> => {
  try {
    const { data } = await http.get<Task[]>('/tasks')
    return data
  } catch (error) {
    throw toFriendlyError(error, '获取任务列表失败')
  }
}

export const fetchTaskDetail = async (taskId: string): Promise<Task> => {
  try {
    const { data } = await http.get<Task>(`/tasks/${taskId}`)
    return data
  } catch (error) {
    throw toFriendlyError(error, '加载任务详情失败')
  }
}

export const updateTask = async (taskId: string, payload: UpdateTaskPayload): Promise<Task> => {
  try {
    const { data } = await http.patch<Task>(`/tasks/${taskId}`, payload)
    return data
  } catch (error) {
    throw toFriendlyError(error, '更新任务失败')
  }
}

export const deleteTask = async (taskId: string): Promise<void> => {
  try {
    await http.delete(`/tasks/${taskId}`)
  } catch (error) {
    throw toFriendlyError(error, '删除任务失败')
  }
}

export const setTaskStatus = async (taskId: string, status: TaskStatus) => {
  return updateTask(taskId, { status })
}
