/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";
// const endpoints = { TEAMS: "/v1/fleet/team" };

export default {
  create: (team_id: number, query_id: number) => {
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${team_id}/policies`;

    return sendRequest("POST", path, { query_id });
  },
  destroy: (team_id: number, ids: number[]) => {
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${team_id}/policies/delete`;

    return sendRequest("POST", path, { ids });
  },
  load: (team_id: number, id: number) => {
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${team_id}/policies/${id}`;

    return sendRequest("GET", path);
  },
  loadAll: (team_id: number) => {
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${team_id}/policies`;

    return sendRequest("GET", path);
  },
};
