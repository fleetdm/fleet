import { SoftwareSource } from "./software";

export const SETUP_STEP_STATUSES = [
  "pending",
  "running",
  "success",
  "failure",
  "cancelled", // server should be aggregating cancelled installs with failed, check here just in case
] as const;

export type SetupStepStatus = typeof SETUP_STEP_STATUSES[number];

/** These type extends onto API returned software steps */
export const SETUP_STEP_TYPES = [
  "software_install", // API key: software
  "software_script_run", // API key: software, detected via source === "sh_packages" || "ps1_packages"
  "script_run", // API key: scripts
];

export type SetupStepType = typeof SETUP_STEP_TYPES[number];

export interface ISetupStep {
  name: string | null;
  status: SetupStepStatus;
  type: SetupStepType;
  error?: string | null;
  source?: SoftwareSource; // Software source (e.g., "sh_packages", "ps1_packages", "apps")
}
