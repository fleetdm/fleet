/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest, { getError } from "services";
import endpoints from "utilities/endpoints";
import { IQueryFormData } from "interfaces/query";
import { ISelectedTargets } from "interfaces/target";
import { AxiosResponse } from "axios";
import { buildQueryStringFromParams } from "utilities/url";

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
  loadAll: (teamId?: number) => {
    const { QUERIES } = endpoints;
    const queryString = buildQueryStringFromParams({ team_id: teamId });
    const path = `${QUERIES}`;

    // dummy
    return {
      queries: [
        {
          created_at: "2023-06-08T15:31:35Z",
          updated_at: "2023-06-08T15:31:35Z",
          id: 2,
          name: "test",
          description: "",
          query: "SELECT * FROM osquery_info;",
          team_id: 43,
          // saved: true,
          interval: 0,
          observer_can_run: false,
          author_id: 1,
          author_name: "Jacob",
          author_email: "jacob@fleetdm.com",
          packs: [],
          stats: {
            system_time_p50: null,
            system_time_p95: null,
            user_time_p50: null,
            user_time_p95: null,
            total_executions: 0,
          },
        },
      ],
    };
    // return sendRequest(
    //   "GET",
    //   queryString ? path.concat(`?${queryString}`) : path
    // );
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
