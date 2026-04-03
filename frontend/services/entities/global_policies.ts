/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */

import sendRequest from "services";
import endpoints from "utilities/endpoints";
import {
  IPolicyFormData,
  ILoadAllPoliciesResponse,
  IPoliciesCountResponse,
} from "interfaces/policy";
import {
  buildQueryStringFromParams,
  convertParamsToSnakeCase,
} from "utilities/url";
import { AutomationType } from "./team_policies";

export type GlobalPoliciesAutomationType = Exclude<
  AutomationType,
  "software" | "scripts" | "conditional_access" | "calendar"
>;

export interface IGlobalPoliciesApiQueryParams {
  page?: number;
  perPage?: number;
  orderKey?: string;
  orderDirection?: "asc" | "desc";
  query?: string;
  automationType?: GlobalPoliciesAutomationType;
}

export interface IPoliciesQueryKey extends IGlobalPoliciesApiQueryParams {
  scope: "globalPolicies";
}

export interface IPoliciesCountQueryKey
  extends Pick<IGlobalPoliciesApiQueryParams, "query" | "automationType"> {
  scope: "policiesCount";
}

const ORDER_KEY = "name";
const ORDER_DIRECTION = "asc";

export default {
  // TODO: How does the frontend need to support legacy policies?
  create: (data: IPolicyFormData) => {
    const { GLOBAL_POLICIES } = endpoints;

    return sendRequest("POST", GLOBAL_POLICIES, data);
  },
  destroy: (ids: number[]) => {
    const { GLOBAL_POLICIES } = endpoints;
    const path = `${GLOBAL_POLICIES}/delete`;

    return sendRequest("POST", path, { ids });
  },
  update: (id: number, data: IPolicyFormData) => {
    const { GLOBAL_POLICIES } = endpoints;
    const path = `${GLOBAL_POLICIES}/${id}`;

    return sendRequest("PATCH", path, data);
  },
  load: (id: number) => {
    const { GLOBAL_POLICIES } = endpoints;
    const path = `${GLOBAL_POLICIES}/${id}`;

    return sendRequest("GET", path);
  },
  loadAll: (): Promise<ILoadAllPoliciesResponse> => {
    const { GLOBAL_POLICIES } = endpoints;

    return sendRequest("GET", GLOBAL_POLICIES);
  },
  loadAllNew: ({
    page,
    perPage,
    orderKey = ORDER_KEY,
    orderDirection: orderDir = ORDER_DIRECTION,
    query,
    automationType,
  }: IGlobalPoliciesApiQueryParams): Promise<ILoadAllPoliciesResponse> => {
    const { GLOBAL_POLICIES } = endpoints;

    const queryParams = {
      page,
      perPage,
      orderKey,
      orderDirection: orderDir,
      query,
      automationType,
    };

    const snakeCaseParams = convertParamsToSnakeCase(queryParams);
    const queryString = buildQueryStringFromParams(snakeCaseParams);
    const path = `${GLOBAL_POLICIES}?${queryString}`;

    return sendRequest("GET", path);
  },
  getCount: ({
    query,
    automationType,
  }: Pick<
    IGlobalPoliciesApiQueryParams,
    "query" | "automationType"
  >): Promise<IPoliciesCountResponse> => {
    const { GLOBAL_POLICIES } = endpoints;
    const path = `${GLOBAL_POLICIES}/count`;
    const queryParams = {
      query,
      automationType,
    };
    const snakeCaseParams = convertParamsToSnakeCase(queryParams);
    const queryString = buildQueryStringFromParams(snakeCaseParams);

    return sendRequest("GET", path.concat(`?${queryString}`));
  },
};
