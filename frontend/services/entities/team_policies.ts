/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";
import { IPolicyFormData } from "interfaces/policy";
// const endpoints = { TEAMS: "/v1/fleet/team" };

export default {
  // TODO: How does the frontend need to support legacy policies?
  create: (data: IPolicyFormData) => {
    const { name, description, query, team_id, resolution, platform } = data;
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${team_id}/policies`;

    return sendRequest("POST", path, {
      name,
      description,
      query,
      resolution,
      platform,
    });
  },
  update: (id: number, data: IPolicyFormData) => {
    const { name, description, query, team_id, resolution, platform } = data;
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${team_id}/policies/${id}`;

    return sendRequest("PATCH", path, {
      name,
      description,
      query,
      resolution,
      platform,
    });
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
