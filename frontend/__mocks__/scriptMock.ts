import {
  IScriptBatchSummaryV2,
  IScriptResultResponse,
} from "services/entities/scripts";
import { IScript, IHostScript, ScriptBatchStatus } from "interfaces/script";

const DEFAULT_SCRIPT_MOCK: IScript = {
  id: 1,
  team_id: null,
  name: "test script",
  created_at: "2020-01-01T00:00:00.000Z",
  updated_at: "2020-01-01T00:00:00.000Z",
};

export const createMockScript = (overrides?: Partial<IScript>): IScript => {
  return { ...DEFAULT_SCRIPT_MOCK, ...overrides };
};

const DEFAULT_SCRIPT_RESULT_MOCK: IScriptResultResponse = {
  hostname: "Test Host",
  host_id: 1,
  execution_id: "123",
  script_contents: "ls /home/*\necho 'testing'\necho 'lines'\nexit $?",
  exit_code: 0,
  output: "test\nlines\n",
  message: "",
  runtime: 0,
  host_timeout: false,
  script_id: 1,
  created_at: "2020-01-01T00:00:00.000Z",
};

export const createMockScriptResult = (
  overrides?: Partial<IScriptResultResponse>
): IScriptResultResponse => {
  return { ...DEFAULT_SCRIPT_RESULT_MOCK, ...overrides };
};

const DEFAULT_HOST_SCRIPT_MOCK: IHostScript = {
  script_id: 1,
  name: "test script",
  last_execution: {
    execution_id: "123",
    executed_at: "2020-01-01T00:00:00.000Z",
    status: "ran",
  },
};

export const createMockHostScript = (
  overrides?: Partial<IHostScript>
): IHostScript => {
  return { ...DEFAULT_HOST_SCRIPT_MOCK, ...overrides };
};

const DEFAULT_SCRIPT_BATCH_SUMMARY_MOCK: IScriptBatchSummaryV2 = {
  created_at: "2025-07-01T10:00:00Z",
  batch_execution_id: "abc-def",
  // batch_execution_id: "6a3bb9a1-a303-4222-86a2-aab63999ce02", // a real one
  script_id: 1,
  script_name: "fake_batch_script.sh",
  team_id: 0,
  targeted_host_count: 10,
  ran_host_count: 6,
  pending_host_count: 1,
  errored_host_count: 1,
  incompatible_host_count: 1,
  canceled_host_count: 1,
  status: "finished" as ScriptBatchStatus,
  not_before: "2023-07-10T18:30:08Z",
  started_at: "2023-07-10T18:31:08Z",
  finished_at: "2023-07-10T18:32:08Z",
  canceled: false,
};

export const createMockBatchScriptSummary = (
  overrides?: Partial<IScriptBatchSummaryV2>
): IScriptBatchSummaryV2 => {
  return { ...DEFAULT_SCRIPT_BATCH_SUMMARY_MOCK, ...overrides };
};
