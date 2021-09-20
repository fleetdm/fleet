import sendRequest from "services";
import endpoints from "fleet/endpoints";
// import { IPolicyFormData, IPolicy } from "interfaces/policy";

export default {
  create: (teamId: number, query_id: number) => {
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${teamId}`;

    return sendRequest("POST", path, { query_id });
  },
  destroy: (teamId: number, policyIds: number[]) => {
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${teamId}/policies/delete`;

    return sendRequest("POST", path, { policyIds });
  },
  load: (teamId: number, policyId: number) => {
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${teamId}/policies/${policyId}`;

    return sendRequest("GET", path);
  },
  loadAll: (teamId: number) => {
    const { TEAMS } = endpoints;
    const path = `${TEAMS}/${teamId}/policies`;


    return sendRequest("GET", path);
  },
};
