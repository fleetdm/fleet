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

interface IPoliciesApiParams {
  page?: number;
  perPage?: number;
  orderKey?: string;
  orderDirection?: "asc" | "desc";
  query?: string;
}

export interface IPoliciesQueryKey extends IPoliciesApiParams {
  scope: "globalPolicies";
}

export interface IPoliciesCountQueryKey
  extends Pick<IPoliciesApiParams, "query"> {
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
  }: IPoliciesApiParams): Promise<ILoadAllPoliciesResponse> => {
    const { GLOBAL_POLICIES } = endpoints;

    const queryParams = {
      page,
      perPage,
      orderKey,
      orderDirection: orderDir,
      query,
    };

    const snakeCaseParams = convertParamsToSnakeCase(queryParams);
    const queryString = buildQueryStringFromParams(snakeCaseParams);
    const path = `${GLOBAL_POLICIES}?${queryString}`;

    return sendRequest("GET", path);
  },
  getCount: ({
    query,
  }: Pick<IPoliciesApiParams, "query">): Promise<IPoliciesCountResponse> => {
    const { GLOBAL_POLICIES } = endpoints;
    const path = `${GLOBAL_POLICIES}/count`;
    const queryParams = {
      query,
    };
    const snakeCaseParams = convertParamsToSnakeCase(queryParams);
    const queryString = buildQueryStringFromParams(snakeCaseParams);

    return sendRequest("GET", path.concat(`?${queryString}`));
  },
};
