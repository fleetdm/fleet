/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */

import { getErrorReason } from "interfaces/errors";
import { SelectedPlatform } from "interfaces/platform";
import {
  ICreateQueryFormData,
  IEditQueryFormData,
  IQueryKeyQueriesLoadAll,
  ISchedulableQuery,
} from "interfaces/schedulable_query";
import { ISelectedTargetsForApi } from "interfaces/target";
import sendRequest from "services";
import { MAX_SERVER_PER_PAGE } from "services/entities/common";
import endpoints from "utilities/endpoints";
import {
  buildQueryStringFromParams,
  convertParamsToSnakeCase,
} from "utilities/url";

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

interface ICreateQueryResponse {
  query: ISchedulableQuery;
}
export interface IQueriesResponse {
  queries: ISchedulableQuery[];
  count: number;
  inherited_query_count: number;
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

const loadAll = ({
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

  const { team_id, ...restParams } = snakeCaseParams;
  const queryString = buildQueryStringFromParams({
    ...restParams,
    fleet_id: team_id,
  });

  return sendRequest(
    "GET",
    queryString ? QUERIES.concat(`?${queryString}`) : QUERIES
  );
};

/** Fetches every matching query (report), page by page, following
 * `meta.has_next_results`. Use when the caller needs the complete list. */
const loadAllPages = async (
  params: Omit<IQueryKeyQueriesLoadAll, "page" | "perPage">
): Promise<IQueriesResponse> => {
  const queries: ISchedulableQuery[] = [];
  let page = 0;
  let response: IQueriesResponse;
  do {
    // pages must be fetched sequentially
    // eslint-disable-next-line no-await-in-loop
    response = await loadAll({
      ...params,
      page,
      perPage: MAX_SERVER_PER_PAGE,
    });
    queries.push(...(response.queries ?? []));
    page += 1;
  } while (response.meta?.has_next_results);

  return {
    ...response,
    queries,
    meta: { has_next_results: false, has_previous_results: false },
  };
};

export default {
  create: (
    createQueryRequestBody: ICreateQueryFormData
  ): Promise<ICreateQueryResponse> => {
    const { QUERIES } = endpoints;
    if (createQueryRequestBody.name) {
      createQueryRequestBody.name = createQueryRequestBody.name.trim();
    }

    return sendRequest("POST", QUERIES, createQueryRequestBody);
  },
  destroy: (id: string | number) => {
    const { QUERIES } = endpoints;
    const path = `${QUERIES}/${id}`;

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
  loadAll,
  loadAllPages,
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
        report_id: queryId,
        selected,
      });
      return campaign;
    } catch (e) {
      throw new Error(
        getErrorReason(e) || `run query: parse server error ${e}`
      );
    }
  },
  update: (id: number, updateParams: IEditQueryFormData) => {
    const { QUERIES } = endpoints;
    const path = `${QUERIES}/${id}`;
    if (updateParams.name) {
      updateParams.name = updateParams.name.trim();
    }

    return sendRequest("PATCH", path, updateParams);
  },
};
