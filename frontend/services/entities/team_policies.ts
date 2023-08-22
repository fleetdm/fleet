/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import { snakeCase, reduce } from "lodash";

import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { ILoadTeamPoliciesResponse, IPolicyFormData } from "interfaces/policy";
import { API_NO_TEAM_ID } from "interfaces/team";
import { buildQueryStringFromParams, QueryParams } from "utilities/url";

interface IPoliciesApiParams {
  team_id?: number;
  page?: number;
  perPage?: number;
  orderKey?: string;
  orderDirection?: "asc" | "desc";
  query?: string;
}

const ORDER_KEY = "name";
const ORDER_DIRECTION = "asc";

const convertParamsToSnakeCase = (params: IPoliciesApiParams) => {
  return reduce<typeof params, QueryParams>(
    params,
    (result, val, key) => {
      result[snakeCase(key)] = val;
      return result;
    },
    {}
  );
};

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
  destroy: (teamId: number | undefined, ids: number[]) => {
    if (!teamId || teamId <= API_NO_TEAM_ID) {
      return Promise.reject(
        new Error(
          `Invalid team id: ${teamId} must be greater than ${API_NO_TEAM_ID}`
        )
      );
    }
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${teamId}/policies/delete`;

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
  loadAllNew: async ({
    team_id,
    page,
    perPage,
    orderKey = ORDER_KEY,
    orderDirection: orderDir = ORDER_DIRECTION,
    query,
  }: IPoliciesApiParams): Promise<ILoadTeamPoliciesResponse> => {
    const { TEAMS } = endpoints;

    const queryParams = {
      page,
      perPage,
      orderKey,
      orderDirection: orderDir,
      query,
    };

    const snakeCaseParams = convertParamsToSnakeCase(queryParams);
    const queryString = buildQueryStringFromParams(snakeCaseParams);
    const path = `${TEAMS}/${team_id}/policies?${queryString}`;
    if (!team_id) {
      throw new Error("Invalid team id");
    }

    return sendRequest("GET", path);
  },
};
