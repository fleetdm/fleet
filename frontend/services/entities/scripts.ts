import { IHostScript, IScript } from "interfaces/script";
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

/** Single script response from GET /script/:id */
export type IScriptResponse = IScript;

/** All scripts response from GET /scripts */
export interface IScriptsResponse {
  scripts: IScript[] | null;
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

export interface IListScriptsApiParams {
  page?: number;
  per_page?: number;
  team_id?: number;
}

export interface IListScriptsQueryKey extends IListScriptsApiParams {
  scope: "scripts";
}

/**
 * Script Result response from GET /scripts/results/:id
 */
export interface IScriptResultResponse {
  hostname: string;
  host_id: number;
  execution_id: string;
  script_contents: string;
  script_id: number | null; // null for ad-hoc script run via API
  exit_code: number | null;
  output: string;
  message: string;
  runtime: number;
  host_timeout: boolean;
  created_at: string;
}

/**
 * Request params for for GET /hosts/:id/scripts
 */
export interface IHostScriptsRequestParams {
  host_id: number;
  page?: number;
  per_page?: number;
}

export interface IHostScriptsQueryKey extends IHostScriptsRequestParams {
  scope: "host_scripts";
}

/**
 * Script response from GET /hosts/:id/scripts
 */
export interface IHostScriptsResponse {
  scripts: IHostScript[];
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

/**
 * Request body for POST /scripts/run
 *
 * https://github.com/fleetdm/fleet/blob/main/docs/Contributing/reference/api-for-contributors.md#run-script-asynchronously
 */
export interface IScriptRunRequest {
  host_id: number;
  script_id: number; // script_id is not required by the API currently, but we require it here to ensure it is always provided
  // script_contents: string; // script_contents is only supported for the CLI currently
}

/**
 * Response body for POST /scripts/run
 *
 * https://github.com/fleetdm/fleet/blob/main/docs/Contributing/reference/api-for-contributors.md#run-script-asynchronously
 */
export interface IScriptRunResponse {
  host_id: number;
  execution_id: string;
}

export interface IScriptBatchSupportedFilters {
  // a search string, not a Fleet.Query
  query?: string;
  label_id?: number;
  team_id?: number;
  status: any; // TODO - improve upstream typing
}
interface IRunScriptBatchRequestBase {
  script_id: number;
  not_before?: string; // ISO 8601 date-time string
}

interface IByFilters extends IRunScriptBatchRequestBase {
  host_ids?: never;
  filters: IScriptBatchSupportedFilters;
}

interface IByHostIds extends IRunScriptBatchRequestBase {
  host_ids: number[];
  filters?: never;
}
/** Request body for POST /scripts/run/batch */
export type IRunScriptBatchRequest = IByFilters | IByHostIds;

/** 202 successful response body for POST /scripts/run/batch */
export interface IRunScriptBatchResponse {
  batch_execution_id: string;
}
export interface IScriptBatchSummaryParams {
  batch_execution_id: string;
}
export interface IScriptBatchSummaryQueryKey extends IScriptBatchSummaryParams {
  scope: "script_batch_summary";
}

export interface IScriptBatchExecutionStatuses {
  ran: number;
  pending: number;
  errored: number;
}
export type ScriptBatchExecutionStatus = keyof IScriptBatchExecutionStatuses;
// 200 successful response
export interface IScriptBatchSummaryResponse
  extends IScriptBatchExecutionStatuses {
  team_id: number;
  script_name: string;
  created_at: string;
  // below fields not yet used by the UI
  canceled: number;
  targeted: number;
  script_id: number;
}
export default {
  getHostScripts({ host_id, page, per_page }: IHostScriptsRequestParams) {
    const { HOST_SCRIPTS } = endpoints;
    const path = `${HOST_SCRIPTS(host_id)}?${buildQueryStringFromParams({
      page,
      per_page,
    })}`;

    return sendRequest("GET", path);
  },

  getScripts(params: IListScriptsApiParams): Promise<IScriptsResponse> {
    const { SCRIPTS } = endpoints;
    const path = `${SCRIPTS}?${buildQueryStringFromParams({ ...params })}`;

    return sendRequest("GET", path);
  },

  getScript(id: number) {
    const { SCRIPT } = endpoints;
    return sendRequest("GET", SCRIPT(id));
  },

  uploadScript(file: File, teamId?: number) {
    const { SCRIPTS } = endpoints;

    const formData = new FormData();
    formData.append("script", file);

    if (teamId) {
      formData.append("team_id", teamId.toString());
    }

    return sendRequest("POST", SCRIPTS, formData);
  },

  downloadScript(id: number) {
    const { SCRIPT } = endpoints;
    const path = `${SCRIPT(id)}?${buildQueryStringFromParams({
      alt: "media",
    })}`;
    return sendRequest("GET", path);
  },

  updateScript(id: number, contents: string, name: string) {
    const { SCRIPT } = endpoints;
    const path = `${SCRIPT(id)}`;

    const file = new File([contents], name);
    const formData = new FormData();
    formData.append("script", file);

    return sendRequest("PATCH", path, formData);
  },

  deleteScript(id: number) {
    const { SCRIPT } = endpoints;
    return sendRequest("DELETE", SCRIPT(id));
  },

  getScriptResult(executionId: string) {
    const { SCRIPT_RESULT } = endpoints;
    return sendRequest("GET", SCRIPT_RESULT(executionId));
  },

  runScript(request: IScriptRunRequest): Promise<IScriptRunResponse> {
    const { SCRIPT_RUN } = endpoints;
    return sendRequest("POST", SCRIPT_RUN, request);
  },
  runScriptBatch(
    request: IRunScriptBatchRequest
  ): Promise<IRunScriptBatchResponse> {
    const { SCRIPT_RUN_BATCH } = endpoints;
    return sendRequest("POST", SCRIPT_RUN_BATCH, request);
  },
  getRunScriptBatchSummary({
    batch_execution_id,
  }: IScriptBatchSummaryParams): Promise<IScriptBatchSummaryResponse> {
    return sendRequest(
      "GET",
      `${endpoints.SCRIPT_RUN_BATCH_SUMMARY(batch_execution_id)}`
    );
  },
};
