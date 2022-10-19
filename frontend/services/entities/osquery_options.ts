/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import yaml from "js-yaml";

export default {
  // Unneeded for teams, but might need this for global
  loadAll: () => {
    const { OSQUERY_OPTIONS } = endpoints;

    return sendRequest("GET", OSQUERY_OPTIONS);
  },
  update: (agentOptions: any, endpoint: string) => {
    return sendRequest("POST", endpoint, agentOptions);
  },
};
