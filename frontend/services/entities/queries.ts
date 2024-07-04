/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { getErrorReason } from "interfaces/errors";
import { ISelectedTargetsForApi } from "interfaces/target";
import {
  ICreateQueryRequestBody,
  IModifyQueryRequestBody,
} from "interfaces/schedulable_query";
import { buildQueryStringFromParams } from "utilities/url";

// Mock API requests to be used in developing FE for #7765 in parallel with BE development
// import { sendRequest } from "services/mock_service/service/service";

export default {
  create: (createQueryRequestBody: ICreateQueryRequestBody) => {
    const { QUERIES } = endpoints;

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
  loadAll: (teamId?: number, mergeInherited = false) => {
    const { QUERIES } = endpoints;
    const queryString = buildQueryStringFromParams({
      team_id: teamId,
      merge_inherited: mergeInherited || null,
    });
    const path = `${QUERIES}`;

    return sendRequest(
      "GET",
      queryString ? path.concat(`?${queryString}`) : path
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

    return sendRequest("PATCH", path, updateParams);
  },
};
