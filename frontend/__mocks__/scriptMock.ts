import {
  IScriptBatchHostResult,
  IScriptBatchHostResultsResponse,
  IScriptBatchSummaryV2,
  IScriptResultResponse,
} from "services/entities/scripts";
import {
  IScript,
  IHostScript,
  ScriptBatchStatus,
  ScriptBatchHostStatus,
} from "interfaces/script";

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
  batch_execution_id: "2756fff7-9a0d-4d95-a893-ec5771e839d8",
  script_id: 1,
  script_name: "fake_batch_script.sh",
  team_id: 0,
  targeted_host_count: 100,
  ran_host_count: 50,
  pending_host_count: 15,
  errored_host_count: 15,
  incompatible_host_count: 10,
  canceled_host_count: 10,
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

const SCRIPT_BATCH_HOST_RESULTS_BY_STATUS: Record<
  ScriptBatchHostStatus,
  IScriptBatchHostResult
> = {
  ran: {
    id: 1,
    display_name: "Host 1",
    script_status: "ran",
    script_execution_id: "exec-1",
    script_executed_at: "2023-07-10T18:31:08Z",
    script_output_preview:
      "Output from Host 1Output from Host 1Output from Host 1Output from Host 1Output from Host 1Output from Host 1Output from Host 1Output from Host 1Output from Host 1Output from Host 1Output from Host 1Output from Host 1Output from Host 1Output from Host 1Output from Host 1Output from Host 1Output from Host 1Output from Host 1Output from Host 1Output from Host 1Output from Host 1",
  },
  errored: {
    id: 2,
    display_name: "Host 2",
    script_status: "errored",
    script_execution_id: "exec-2",
    script_executed_at: "2023-07-10T18:31:08Z",
    script_output_preview: "Error output from Host 1",
  },
  pending: {
    id: 3,
    display_name: "Host 3",
    script_status: "pending",
    script_execution_id: null,
    script_executed_at: null,
    script_output_preview: null,
  },
  incompatible: {
    id: 4,
    display_name: "Host 4",
    script_status: "incompatible",
    script_execution_id: null,
    script_executed_at: null,
    script_output_preview: null,
  },
  canceled: {
    id: 5,
    display_name: "Host 5",
    script_status: "canceled",
    script_execution_id: null,
    script_executed_at: null,
    script_output_preview: null,
  },
};

export const createMockScriptBatchHostResults = (
  status?: ScriptBatchHostStatus
): IScriptBatchHostResultsResponse => {
  return {
    meta: {
      has_next_results: false,
      has_previous_results: false,
    },
    count: 2,
    hosts: [
      SCRIPT_BATCH_HOST_RESULTS_BY_STATUS[status || "ran"],
      SCRIPT_BATCH_HOST_RESULTS_BY_STATUS[status || "ran"],
    ],
  };
};
