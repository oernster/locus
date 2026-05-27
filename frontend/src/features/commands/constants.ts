import type { StageId, Status } from "../../types/locus";

export const STAGES: StageId[] = ["PLAN", "EXECUTE", "CHECK", "DONE"];

export const DEFAULT_STAGE_LABELS: Record<StageId, string> = {
  PLAN: "Plan",
  EXECUTE: "Execute",
  CHECK: "Check",
  DONE: "Done",
};

export const BOTTOM_PANEL_LABELS: Record<StageId, string> = {
  PLAN: "EXPLORATION",
  EXECUTE: "DEEP WORK",
  CHECK: "ANALYSIS",
  DONE: "RETROSPECTIVE",
};

export const STATUSES: Status[] = [
  "Not Started",
  "In Progress",
  "Blocked",
  "Complete",
];
