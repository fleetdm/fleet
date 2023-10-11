import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { buildQueryStringFromParams } from "utilities/url";

export interface IScript {
  id: number;
  team_id: number | null;
  name: string;
  created_at: string;
  updated_at: string;
}

/** Single script response from GET /script/:id */
export type IScriptResponse = IScript;

/** All scripts response from GET /scripts */
export interface IScriptsResponse {
  scripts: IScript[];
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
  exit_code: number | null;
  output: string;
  message: string;
  runtime: number;
  host_timeout: boolean;
}

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

export default {
  getHostScripts(id: number, page?: number) {
    const { HOST_SCRIPTS } = endpoints;

    let path = HOST_SCRIPTS(id);
    if (page) {
      path = `${path}?${buildQueryStringFromParams({ page })}`;
    }
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

  deleteScript(id: number) {
    const { SCRIPT } = endpoints;
    return sendRequest("DELETE", SCRIPT(id));
  },

  getScriptResult(executionId: string) {
    const { SCRIPT_RESULT } = endpoints;
    return sendRequest("GET", SCRIPT_RESULT(executionId));
  },

  runScript(id: number) {
    const { SCRIPT_RUN } = endpoints;
    return sendRequest("POST", SCRIPT_RUN, { script_id: id });
  },
};
