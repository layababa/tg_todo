import apiClient from "./client";
import type { DatabaseSummary } from "@/types/group";

export interface ListDatabasesResponse {
  success: boolean;
  data: {
    items: DatabaseSummary[];
  };
}

export const listDatabases = async (): Promise<DatabaseSummary[]> => {
  const res = await apiClient.get<ListDatabasesResponse>("/databases");
  return res.data.data.items;
};
