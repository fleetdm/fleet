/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import { snakeCase, reduce } from "lodash";

import sendRequest from "services";
import endpoints from "utilities/endpoints";
import {
  ILoadTeamPoliciesResponse,
  IPolicyFormData,
  IPoliciesCountResponse,
  ILoadTeamPolicyResponse,
} from "interfaces/policy";
import { API_NO_TEAM_ID } from "interfaces/team";
import { buildQueryStringFromParams, QueryParams } from "utilities/url";

interface IPoliciesApiQueryParams {
  page?: number;
  perPage?: number;
  orderKey?: string;
  orderDirection?: "asc" | "desc";
  query?: string;
}

export interface IPoliciesApiParams extends IPoliciesApiQueryParams {
  teamId: number;
  mergeInherited?: boolean;
}

export interface ITeamPoliciesQueryKey extends IPoliciesApiParams {
  scope: "teamPolicies";
}

export interface ITeamPoliciesCountQueryKey
  extends Pick<IPoliciesApiParams, "query" | "teamId" | "mergeInherited"> {
  scope: "teamPoliciesCountMergeInherited" | "teamPoliciesCount";
}

interface IPoliciesCountApiParams {
  teamId: number;
  query?: string;
  mergeInherited?: boolean;
}

const ORDER_KEY = "name";
const ORDER_DIRECTION = "asc";

const convertParamsToSnakeCase = (params: IPoliciesApiQueryParams) => {
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
      software_title_id,
      // note absence of automations-related fields, which are only set by the UI via update
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
      software_title_id,
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
      // automations-related fields
      calendar_events_enabled,
      software_title_id,
      script_id,
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
      calendar_events_enabled,
      software_title_id,
      script_id,
    });
  },
  destroy: (teamId: number | undefined, ids: number[]) => {
    if (teamId === undefined || teamId < API_NO_TEAM_ID) {
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
  load: (team_id: number, id: number): Promise<ILoadTeamPolicyResponse> => {
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${team_id}/policies/${id}`;
    return sendRequest("GET", path);
  },
  loadAll: (team_id?: number): Promise<ILoadTeamPoliciesResponse> => {
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${team_id}/policies`;
    return sendRequest("GET", path);
  },
  loadAllNew: async ({
    teamId,
    page,
    perPage,
    orderKey = ORDER_KEY,
    orderDirection: orderDir = ORDER_DIRECTION,
    query,
    mergeInherited,
  }: IPoliciesApiParams): Promise<ILoadTeamPoliciesResponse> => {
    const { TEAMS } = endpoints;

    const queryParams = {
      page,
      perPage,
      orderKey,
      orderDirection: orderDir,
      query,
      mergeInherited,
    };

    const snakeCaseParams = convertParamsToSnakeCase(queryParams);
    const queryString = buildQueryStringFromParams(snakeCaseParams);
    const path = `${TEAMS}/${teamId}/policies?${queryString}`;
    return sendRequest("GET", path);
  },
  getCount: async ({
    query,
    teamId,
    mergeInherited = true,
  }: Pick<
    IPoliciesCountApiParams,
    "query" | "teamId" | "mergeInherited"
  >): Promise<IPoliciesCountResponse> => {
    const { TEAM_POLICIES } = endpoints;
    const path = `${TEAM_POLICIES(teamId)}/count`;
    const queryParams = {
      query,
      mergeInherited,
    };
    const snakeCaseParams = convertParamsToSnakeCase(queryParams);
    const queryString = buildQueryStringFromParams(snakeCaseParams);

    return sendRequest("GET", path.concat(`?${queryString}`));
  },
};
