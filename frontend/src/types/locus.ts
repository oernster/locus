// Locus domain types matching Go DTOs.

export type StageId = "PLAN" | "EXECUTE" | "CHECK" | "DONE";

export type Status =
  | "Not Started"
  | "In Progress"
  | "Blocked"
  | "Complete";

export interface CommandDTO {
  id: number;
  title: string;
  status: Status;
  stage_id: StageId;
  sort_index: number;
  created_at: string; // ISO 8601 UTC
}

export interface SessionDTO {
  active: boolean;
  id?: number;
  command_id?: number;
  stage_id?: StageId;
  started_at?: string; // ISO 8601 UTC
  ended_at?: string;   // ISO 8601 UTC
}

export interface OutcomeDTO {
  id: number;
  command_id: number;
  note: string;
  created_at: string;
}

export interface BoardDTO {
  name: string;
  user_named: boolean;
  is_new_unnamed: boolean;
  is_empty: boolean;
  stage_labels?: Record<string, string>;
}

export interface SnapshotDTO {
  id: number;
  name: string;
  saved_at: string;
}

export interface AppFocusDTO {
  exe_path: string;
  friendly_name: string;
  total_seconds: number;
  session_count: number;
}

export interface FocusDataDTO {
  available: boolean;
  stage_id: string;
  total_seconds: number;
  idle_seconds: number;
  deep_work_seconds: number;
  apps: AppFocusDTO[];
}
