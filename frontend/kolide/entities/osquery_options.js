import yaml from "js-yaml";

import endpoints from "kolide/endpoints";

export default (client) => {
  return {
    // Unneeded for teams, but might need this for global
    loadAll: () => {
      const { OSQUERY_OPTIONS } = endpoints;
      return client.authenticatedGet(client._endpoint(OSQUERY_OPTIONS));
    },
    update: (osqueryOptionsData, teamId) => {
      // Both teams and global options route to this call
      const { TEAMS_AGENT_OPTIONS, OSQUERY_OPTIONS } = endpoints;
      const yamlOptions = yaml.safeLoad(osqueryOptionsData.osquery_options);
      const endpoint =
        teamId === undefined ? OSQUERY_OPTIONS : TEAMS_AGENT_OPTIONS(teamId);
      return client.authenticatedPost(
        client._endpoint(endpoint),
        JSON.stringify(yamlOptions)
      );
    },
  };
};
