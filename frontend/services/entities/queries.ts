/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest, { getError } from "services";
import endpoints from "utilities/endpoints";
import { IQueryFormData } from "interfaces/query";
import { ISelectedTargets } from "interfaces/target";
import { AxiosResponse } from "axios";

export default {
  create: ({ description, name, query, observer_can_run }: IQueryFormData) => {
    const { QUERIES } = endpoints;

    return sendRequest("POST", QUERIES, {
      description,
      name,
      query,
      observer_can_run,
    });
  },
  destroy: (id: string | number) => {
    const { QUERIES } = endpoints;
    const path = `${QUERIES}/id/${id}`;

    return sendRequest("DELETE", path);
  },
  load: (id: number) => {
    const { QUERIES } = endpoints;
    const path = `${QUERIES}/${id}`;

    return sendRequest("GET", path);
  },
  loadAll: () => {
    const { QUERIES } = endpoints;

    return sendRequest("GET", QUERIES);
  },
  run: async ({
    query,
    queryId,
    selected,
  }: {
    query: string;
    queryId: number | null;
    selected: ISelectedTargets;
  }) => {
    const { RUN_QUERY } = endpoints;

    try {
      const { campaign } = await sendRequest("POST", RUN_QUERY, {
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
    } catch (response) {
      throw new Error(getError(response as AxiosResponse));
    }
  },
  update: (id: number, updateParams: IQueryFormData) => {
    const { QUERIES } = endpoints;
    const path = `${QUERIES}/${id}`;

    return sendRequest("PATCH", path, updateParams);
  },
};
