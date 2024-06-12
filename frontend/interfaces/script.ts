import { HOST_LINUX_PLATFORMS } from "./platform";

export interface IScript {
  id: number;
  team_id: number | null;
  name: string;
  created_at: string;
  updated_at: string;
}

export const isScriptSupportedPlatform = (hostPlatform: string) =>
  ["darwin", "windows", ...HOST_LINUX_PLATFORMS].includes(hostPlatform); // excludes chrome, ios, ipados see also https://github.com/fleetdm/fleet/blob/5a21e2cfb029053ddad0508869eb9f1f23997bf2/server/fleet/hosts.go#L775

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
