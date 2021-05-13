import yaml from "js-yaml";

import endpoints from "kolide/endpoints";

export default (client) => {
  return {
    loadAll: () => {
      const { TEAMS_AGENT_OPTIONS } = endpoints;
      return client.authenticatedGet(client._endpoint(TEAMS_AGENT_OPTIONS));
    },
    update: (osqueryOptionsData, teamId) => {
      const { TEAMS_AGENT_OPTIONS } = endpoints;
      const yamlOptions = yaml.safeLoad(osqueryOptionsData.osquery_options);

      return client.authenticatedPost(
        client._endpoint(TEAMS_AGENT_OPTIONS(teamId)),
        JSON.stringify(yamlOptions)
      );
    },
  };
};
