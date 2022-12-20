/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { ILoadTeamPoliciesResponse, IPolicyFormData } from "interfaces/policy";

export default {
  create: (data: IPolicyFormData) => {
    const {
      name,
      description,
      query,
      team_id,
      resolution,
      platform,
      critical,
    } = data;
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${team_id}/policies`;

    return sendRequest("POST", path, {
      name,
      description,
      query,
      resolution,
      platform,
      critical,
    });
  },
  update: (id: number, data: IPolicyFormData) => {
    const {
      name,
      description,
      query,
      team_id,
      resolution,
      platform,
      critical,
    } = data;
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${team_id}/policies/${id}`;

    return sendRequest("PATCH", path, {
      name,
      description,
      query,
      resolution,
      platform,
      critical,
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
  loadAll: (team_id?: number): Promise<ILoadTeamPoliciesResponse> => {
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${team_id}/policies`;
    if (!team_id) {
      throw new Error("Invalid team id");
    }

    return sendRequest("GET", path);
  },
};
