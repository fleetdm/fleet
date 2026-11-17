/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import { API_NO_TEAM_ID } from "interfaces/team";
import sendRequest from "services";
import endpoints from "utilities/endpoints";

export default {
  updateTeam: (teamId: number | undefined, agentOptions: any) => {
    if (!teamId || teamId <= API_NO_TEAM_ID) {
      return Promise.reject(
        new Error(
          `Invalid fleet id: ${teamId} must be greater than ${API_NO_TEAM_ID}`
        )
      );
    }
    const { TEAMS_AGENT_OPTIONS } = endpoints;

    return sendRequest("POST", TEAMS_AGENT_OPTIONS(teamId), agentOptions);
  },
};
