import apiClient from "./client";
import type { UserProfile } from "@/types/auth";

export interface Comment {
  id: string;
  task_id: string;
  parent_id?: string;
  user_id: string;
  content: string;
  created_at: string;
  user?: UserProfile;
}

export interface CreateCommentRequest {
  content: string;
  parent_id?: string;
}

export interface ListCommentsResponse {
  success: boolean;
  data: Comment[];
}

export const listComments = async (taskId: string): Promise<Comment[]> => {
  const res = await apiClient.get<ListCommentsResponse>(
    `/tasks/${taskId}/comments`
  );
  return res.data.data;
};

export interface CreateCommentResponse {
  success: boolean;
  data: Comment;
}

export const createComment = async (
  taskId: string,
  data: CreateCommentRequest
): Promise<Comment> => {
  const res = await apiClient.post<CreateCommentResponse>(
    `/tasks/${taskId}/comments`,
    data
  );
  return res.data.data;
};
