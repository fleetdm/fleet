/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { getErrorReason } from "interfaces/errors";
import { ISelectedTargetsForApi } from "interfaces/target";
import {
  ICreateQueryRequestBody,
  IModifyQueryRequestBody,
  IQueryKeyQueriesLoadAll,
  ISchedulableQuery,
} from "interfaces/schedulable_query";
import {
  buildQueryStringFromParams,
  convertParamsToSnakeCase,
} from "utilities/url";
import { SelectedPlatform } from "interfaces/platform";

export interface ILoadQueriesParams {
  teamId?: number;
  page?: number;
  perPage?: number;
  query?: string;
  orderDirection?: "asc" | "desc";
  orderKey?: string;
  mergeInherited?: boolean;
  targetedPlatform?: SelectedPlatform;
}
export interface IQueryKeyLoadQueries extends ILoadQueriesParams {
  scope: "queries";
}

export interface IQueriesResponse {
  queries: ISchedulableQuery[];
  count: number;
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

export default {
  create: (createQueryRequestBody: ICreateQueryRequestBody) => {
    const { QUERIES } = endpoints;
    if (createQueryRequestBody.name) {
      createQueryRequestBody.name = createQueryRequestBody.name.trim();
    }

    return sendRequest("POST", QUERIES, createQueryRequestBody);
  },
  destroy: (id: string | number) => {
    const { QUERIES } = endpoints;
    const path = `${QUERIES}/id/${id}`;

    return sendRequest("DELETE", path);
  },
  bulkDestroy: (ids: number[]) => {
    const { QUERIES } = endpoints;
    const path = `${QUERIES}/delete`;
    return sendRequest("POST", path, { ids });
  },
  load: (id: number) => {
    const { QUERIES } = endpoints;
    const path = `${QUERIES}/${id}`;

    return sendRequest("GET", path);
  },
  loadAll: ({
    teamId,
    page,
    perPage,
    query,
    orderDirection,
    orderKey,
    mergeInherited,
    // FE logic uses less ambiguous `targetedPlatform`, while API expects `platform` for alignment
    // with other API conventions and database `queries.platform` column
    targetedPlatform: platform,
  }: IQueryKeyQueriesLoadAll): Promise<IQueriesResponse> => {
    const { QUERIES } = endpoints;

    const snakeCaseParams = convertParamsToSnakeCase({
      teamId,
      page,
      perPage,
      query,
      orderDirection,
      orderKey,
      mergeInherited,
      platform,
    });

    // API expects "macos" instead of "darwin"
    if (snakeCaseParams.platform === "darwin") {
      snakeCaseParams.platform = "macos";
    }

    const queryString = buildQueryStringFromParams(snakeCaseParams);

    return sendRequest(
      "GET",
      queryString ? QUERIES.concat(`?${queryString}`) : QUERIES
    );
  },
  run: async ({
    query,
    queryId,
    selected,
  }: {
    query: string;
    queryId: number | null;
    selected: ISelectedTargetsForApi;
  }) => {
    const { LIVE_QUERY } = endpoints;

    try {
      const { campaign } = await sendRequest("POST", LIVE_QUERY, {
        query,
        query_id: queryId,
        selected,
      });
      return Promise.resolve({
        ...campaign,
        hosts_count: {
          successful: 0,
          failed: 0,
          total: 0,
        },
      });
    } catch (e) {
      throw new Error(
        getErrorReason(e) || `run query: parse server error ${e}`
      );
    }
  },
  update: (id: number, updateParams: IModifyQueryRequestBody) => {
    const { QUERIES } = endpoints;
    const path = `${QUERIES}/${id}`;
    if (updateParams.name) {
      updateParams.name = updateParams.name.trim();
    }

    return sendRequest("PATCH", path, updateParams);
  },
};
