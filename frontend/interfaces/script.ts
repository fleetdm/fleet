export interface IScript {
  id: number;
  team_id: number | null;
  name: string;
  created_at: string;
  updated_at: string;
}

export const SCRIPT_SUPPORTED_PLATFORMS = ["darwin", "windows"] as const; // TODO: revisit this approach to white-list supported platforms (which would require a more robust approach to identifying linux flavors)

export type IScriptExecutionStatus = "ran" | "pending" | "error";

export interface ILastExecution {
  execution_id: string;
  executed_at: string;
  status: IScriptExecutionStatus;
}

export interface IHostScript {
  script_id: number;
  name: string;
  last_execution: ILastExecution | null;
}
