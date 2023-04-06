/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import { API_NO_TEAM_ID } from "interfaces/team";
import sendRequest from "services";
import endpoints from "utilities/endpoints";

export default {
  // Unneeded for teams, but might need this for global
  loadAll: () => {
    const { OSQUERY_OPTIONS } = endpoints;

    return sendRequest("GET", OSQUERY_OPTIONS);
  },
  update: (agentOptions: any, endpoint: string) => {
    return sendRequest("POST", endpoint, agentOptions);
  },
  updateTeam: (teamId: number | undefined, agentOptions: any) => {
    if (!teamId || teamId <= API_NO_TEAM_ID) {
      return Promise.reject(
        new Error(
          `Invalid team id: ${teamId} must be greater than ${API_NO_TEAM_ID}`
        )
      );
    }
    const { TEAMS_AGENT_OPTIONS } = endpoints;

    return sendRequest("POST", TEAMS_AGENT_OPTIONS(teamId), agentOptions);
  },
};
