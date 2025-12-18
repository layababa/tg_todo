import apiClient from "./client";
import type { UserProfile } from "@/types/auth";

export interface GetMeResponse {
  success: boolean;
  data: UserProfile;
}

export const getMe = async (): Promise<UserProfile> => {
  const res = await apiClient.get<GetMeResponse>("/me");
  return res.data.data;
};

export interface UpdateSettingsRequest {
  timezone?: string;
  default_database_id?: string;
}

export const updateSettings = async (
  data: UpdateSettingsRequest
): Promise<UserProfile> => {
  const res = await apiClient.patch<GetMeResponse>("/me/settings", data);
  return res.data.data;
};
