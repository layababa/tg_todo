export interface TaskContextSnapshot {
  ID: string;
  Role: "me" | "other" | "system";
  Author?: string;
  Text: string;
  CreatedAt: string;
}

export interface User {
  id: string;
  name: string;
  tg_username?: string;
  photo_url?: string;
}

export interface Task {
  ID: string;
  Title: string;
  Status: string;
  SyncStatus: "Synced" | "Pending" | "Failed";
  DatabaseID?: string;
  NotionURL?: string;
  CreatedAt: string;
  DueAt?: string | null;
  // Details
  Description?: string;
  ChatJumpURL?: string;
  Snapshots?: TaskContextSnapshot[];
  Assignees?: User[];
  Creator?: User;
  Group?: {
    id: string;
    title: string;
  };
}

export interface TaskDetail {
  task: Task;
}
