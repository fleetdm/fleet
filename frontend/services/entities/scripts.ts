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
  getScripts() {
    const { SCRIPTS } = endpoints;
    return sendRequest("GET", SCRIPTS);
  },

  getScript(id: number, teamId?: number) {
    const { SCRIPT } = endpoints;
    const path = teamId
      ? `${SCRIPT(id)}/${buildQueryStringFromParams({ team_id: teamId })}`
      : SCRIPT(id);

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
