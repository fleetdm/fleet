import { appendTargetTypeToTargets } from "redux/nodes/entities/targets/helpers";
import endpoints from "kolide/endpoints";

const defaultSelected = {
  hosts: [],
  labels: [],
};

export default (client) => {
  return {
    loadAll: (query, selected = defaultSelected) => {
      const { TARGETS } = endpoints;

      return client
        .authenticatedPost(
          client._endpoint(TARGETS),
          JSON.stringify({ query, selected })
        )
        .then((response) => {
          const { targets } = response;

          return {
            ...response,
            targets: [
              ...appendTargetTypeToTargets(targets.hosts, "hosts"),
              ...appendTargetTypeToTargets(targets.labels, "labels"),
            ],
          };
        });
    },
  };
};
