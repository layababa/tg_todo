export interface Group {
  id: string;
  title: string;
  status: "Connected" | "Unbound" | "Inactive";
  db?: DatabaseSummary;
  role: "Admin" | "Member";
}

export interface DatabaseSummary {
  id: string;
  name: string;
  icon?: string;
}

export interface ValidationResult {
  valid: boolean;
  missing_properties: string[];
  name: string;
  reason?: string;
}

export interface InitResult {
  initialized: boolean;
  created_properties: string[];
}

export interface BindGroupRequest {
  db_id: string;
  mode?: string;
}
