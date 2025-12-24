import apiClient from "./client";
import type { Task, TaskDetail } from "@/types/task";

export interface ListTasksResponse {
  success: boolean;
  data: {
    items: TaskDetail[];
  };
}

export interface GetTaskResponse {
  success: boolean;
  data: Task;
}

export interface ListParams {
  view?: string;
  database_id?: string;
}

export const listTasks = async (params?: ListParams): Promise<TaskDetail[]> => {
  const res = await apiClient.get<ListTasksResponse>("/tasks", { params });
  return res.data.data.items;
};

export const getTask = async (id: string): Promise<Task> => {
  const res = await apiClient.get<GetTaskResponse>(`/tasks/${id}`);
  return res.data.data;
};

export interface PatchTaskRequest {
  title?: string;
  status?: string;
  due_at?: string | null;
}

export const patchTask = async (
  id: string,
  data: PatchTaskRequest
): Promise<Task> => {
  const res = await apiClient.patch<GetTaskResponse>(`/tasks/${id}`, data);
  return res.data.data;
};

export interface CreateTaskRequest {
  title: string;
  description?: string;
}

export const createTask = async (data: CreateTaskRequest): Promise<Task> => {
  const res = await apiClient.post<GetTaskResponse>("/tasks", data);
  return res.data.data;
};

export const deleteTask = async (id: string): Promise<void> => {
  await apiClient.delete(`/tasks/${id}`);
};
