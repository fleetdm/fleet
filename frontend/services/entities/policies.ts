import sendRequest from "services";
import endpoints from "fleet/endpoints";
// import { IPolicyFormData, IPolicy } from "interfaces/policy";

export default {
  create: (query_id: number) => {
    const { POLICIES } = endpoints;

    return sendRequest("POST", POLICIES, { query_id });
  },
  destroy: (ids: number[]) => {
    const { POLICIES } = endpoints;
    const path = `${POLICIES}/delete`;

    return sendRequest("POST", path, { ids });
  },
  //   load: (id: string) => {
  //     const { POLICIES } = endpoints;
  //     const path = `${POLICIES}/${id}`;

  //     return sendRequest("GET", path);
  //   },
  loadAll: () => {
    const { POLICIES } = endpoints;

    return sendRequest("GET", POLICIES);
  },
  //   update: (id: number, updateParams: IPolicy) => {
  //     const { POLICIES } = endpoints;
  //     const path = `${POLICIES}/${id}`;

  //     return sendRequest("PATCH", path, JSON.stringify(updateParams));
  //   },
};
