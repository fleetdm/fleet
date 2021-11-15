/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";
import { IPolicyFormData } from "interfaces/policy";

export default {
  create: (data: number | IPolicyFormData) => {
    const { GLOBAL_POLICIES } = endpoints;

    if (typeof data === "number") {
      return sendRequest("POST", GLOBAL_POLICIES, { query_id: data });
    }
    
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
  loadAll: () => {
    const { GLOBAL_POLICIES } = endpoints;

    return sendRequest("GET", GLOBAL_POLICIES);
  },
};
