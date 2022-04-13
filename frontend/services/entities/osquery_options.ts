/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "fleet/endpoints";
import yaml from "js-yaml";

export default {
  // Unneeded for teams, but might need this for global
  loadAll: () => {
    const { OSQUERY_OPTIONS } = endpoints;

    return sendRequest("GET", OSQUERY_OPTIONS);
  },
  update: (osqueryOptionsData: any, endpoint: string) => {
    const yamlOptions = yaml.load(osqueryOptionsData.osquery_options);

    return sendRequest("POST", endpoint, yamlOptions);
  },
};
