import apiClient from "./client";
import type { Group, ValidationResult, InitResult } from "@/types/group";

export interface ListGroupsResponse {
  success: boolean;
  data: {
    items: Group[];
  };
}

export const listGroups = async (): Promise<Group[]> => {
  const res = await apiClient.get<ListGroupsResponse>("/groups");
  return res.data.data.items;
};

export const bindGroup = async (
  groupID: string,
  dbID: string
): Promise<Group> => {
  const res = await apiClient.post(`/groups/${groupID}/bind`, { db_id: dbID });
  return res.data.data;
};

export const unbindGroup = async (groupID: string): Promise<Group> => {
  const res = await apiClient.post(`/groups/${groupID}/unbind`);
  return res.data.data;
};

export const validateGroupDatabase = async (
  groupID: string,
  dbID: string
): Promise<ValidationResult> => {
  const res = await apiClient.post(`/groups/${groupID}/db/validate`, {
    db_id: dbID,
  });
  return res.data.data;
};

export const initGroupDatabase = async (
  groupID: string,
  dbID: string
): Promise<InitResult> => {
  const res = await apiClient.post(`/groups/${groupID}/db/init`, {
    db_id: dbID,
  });
  return res.data.data;
};
