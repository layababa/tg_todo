export interface TaskContextSnapshot {
  ID: string;
  Role: "me" | "other" | "system";
  Author?: string;
  Text: string;
  CreatedAt: string;
}

export interface Task {
  ID: string;
  Title: string;
  Status: string;
  SyncStatus: "Synced" | "Pending" | "Failed";
  DatabaseID?: string;
  NotionURL?: string;
  CreatedAt: string;
  // Details
  Description?: string; // Currently missing in backend struct? Or is it handled elsewhere?
  ChatJumpURL?: string;
  Snapshots?: TaskContextSnapshot[];
  Assignees?: any[]; // Todo: Define User type
}

export interface TaskDetail {
  task: Task;
}
