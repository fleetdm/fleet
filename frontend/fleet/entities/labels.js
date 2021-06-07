import endpoints from "fleet/endpoints";
import helpers from "fleet/helpers";

export default (client) => {
  return {
    create: ({ description, name, platform, query }) => {
      const { LABELS } = endpoints;

      return client
        .authenticatedPost(
          client._endpoint(LABELS),
          JSON.stringify({ description, name, platform, query })
        )
        .then((response) => {
          const { label } = response;

          return {
            ...label,
            slug: helpers.labelSlug(label),
            type: "custom",
          };
        });
    },
    destroy: (label) => {
      const { LABELS } = endpoints;
      const endpoint = client._endpoint(`${LABELS}/id/${label.id}`);

      return client.authenticatedDelete(endpoint);
    },
    loadAll: () => {
      const { LABELS } = endpoints;

      return client
        .authenticatedGet(client._endpoint(LABELS))
        .then((response) => helpers.formatLabelResponse(response));
    },
    update: (label, updateAttrs) => {
      const { LABELS } = endpoints;
      const endpoint = client._endpoint(`${LABELS}/${label.id}`);

      return client
        .authenticatedPatch(endpoint, JSON.stringify(updateAttrs))
        .then((response) => {
          const { label: updatedLabel } = response;

          return {
            ...updatedLabel,
            slug: helpers.labelSlug(updatedLabel),
            type: "custom",
          };
        });
    },
  };
};
