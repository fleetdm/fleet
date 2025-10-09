export const SETUP_STEP_STATUSES = [
  "pending",
  "running",
  "success",
  "failure",
  "cancelled", // server should be aggregating cancelled installs with failed, check here just in case
] as const;

export type SetupStepStatus = typeof SETUP_STEP_STATUSES[number];

export const SETUP_STEP_TYPES = [
  "software_install",
  "script_run",
  "software_script_run",
];

export type SetupStepType = typeof SETUP_STEP_TYPES[number];

export interface ISetupStep {
  name: string | null;
  status: SetupStepStatus;
  type?: SetupStepType;
}
