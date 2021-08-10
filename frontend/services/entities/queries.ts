import sendRequest from "services";
import endpoints from "fleet/endpoints";
import { IQueryUpdate, IQuery } from "interfaces/query";

export default {
  create: ({ description, name, query, observer_can_run }: IQueryUpdate) => {
    const { QUERIES } = endpoints;

    return sendRequest("POST", QUERIES, { description, name, query, observer_can_run });
  },
  destroy: (id: string) => {
    const { QUERIES } = endpoints;
    const path = `${QUERIES}/id/${id}`;

    return sendRequest("DELETE", path);
  },
  load: (id: string) => {
    const { QUERIES } = endpoints;
    const path = `${QUERIES}/${id}`;
    
    return sendRequest("GET", path);
  },
  loadAll: () => {
    const { QUERIES } = endpoints;

    return sendRequest("GET", QUERIES);
  },
  run: ({ query, selected }: { query: string, selected: any }) => {
    const { RUN_QUERY } = endpoints;

    return sendRequest("POST", RUN_QUERY, { query, selected });
    // return client
    //   .authenticatedPost(
    //     client._endpoint(RUN_QUERY),
    //     JSON.stringify({ query, selected })
    //   )
    //   .then((response) => {
    //     const { campaign } = response;

    //     return {
    //       ...campaign,
    //       hosts_count: {
    //         successful: 0,
    //         failed: 0,
    //         total: 0,
    //       },
    //     };
    //   });
  },
  update: ({ id }: IQuery, updateParams: any) => {
    const { QUERIES } = endpoints;
    const path = `${QUERIES}/${id}`;

    return sendRequest("PATCH", path, JSON.stringify(updateParams));
  },
};