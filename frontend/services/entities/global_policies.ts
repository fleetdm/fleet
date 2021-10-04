/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";
// import { IPolicyFormData, IPolicy } from "interfaces/policy";

export default {
  create: (query_id: number) => {
    const { GLOBAL_POLICIES } = endpoints;

    return sendRequest("POST", GLOBAL_POLICIES, { query_id });
  },
  destroy: (ids: number[]) => {
    const { GLOBAL_POLICIES } = endpoints;
    const path = `${GLOBAL_POLICIES}/delete`;

    return sendRequest("POST", path, { ids });
  },
  load: (id: number) => {
    const { GLOBAL_POLICIES } = endpoints;
    const path = `${GLOBAL_POLICIES}/${id}`;

    return sendRequest("GET", path);
  },
  loadAll: () => {
    const { GLOBAL_POLICIES } = endpoints;

    return sendRequest("GET", GLOBAL_POLICIES);
  },
};
