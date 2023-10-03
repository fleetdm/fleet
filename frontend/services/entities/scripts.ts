import { createMockScript } from "__mocks__/scriptMock";
import team from "interfaces/team";
import { create } from "lodash";
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

export default {
  getScripts(teamId?: number) {
    const { SCRIPTS } = endpoints;
    const path = teamId
      ? `${SCRIPTS}?${buildQueryStringFromParams({ team_id: teamId })}`
      : SCRIPTS;

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
};
