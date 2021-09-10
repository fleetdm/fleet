import sendRequest from "services";
import endpoints from "fleet/endpoints";
import { IQueryFormData, IQuery } from "interfaces/query";

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
  destroy: (id: string) => {
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
    selected: any;
  }) => {
    const { RUN_QUERY } = endpoints;

    const { campaign } = await sendRequest("POST", RUN_QUERY, {
      query,
      query_id: queryId,
      selected,
    });
    return {
      ...campaign,
      hosts_count: {
        successful: 0,
        failed: 0,
        total: 0,
      },
    };
  },
  update: ({ id }: IQuery, updateParams: any) => {
    const { QUERIES } = endpoints;
    const path = `${QUERIES}/${id}`;

    return sendRequest("PATCH", path, JSON.stringify(updateParams));
  },
};
