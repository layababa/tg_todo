export interface UserProfile {
  id: string;
  tg_id?: number;
  name: string;
  photo_url?: string;
  timezone?: string;
  default_database_id?: string;
  notion_connected?: boolean;
}

export interface AuthStatusPayload {
  user: UserProfile;
  notion_connected: boolean;
  pending_sync_count?: number;
  redirect_hint?: string | null;
}

export interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: { code?: string; message?: string };
  meta?: Record<string, unknown>;
}
